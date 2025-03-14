package vpn
import (
	"errors"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"google.golang.org/protobuf/proto"
	"cdr.dev/slog"
)
type SpeakerRole string
type rpcMessage interface {
	proto.Message
	GetRpc() *RPC
	// EnsureRPC isn't autogenerated, but we'll manually add it for RPC types so that the speaker
	// can allocate the RPC.
	EnsureRPC() *RPC
}
func (t *TunnelMessage) EnsureRPC() *RPC {
	if t.Rpc == nil {
		t.Rpc = &RPC{}
	}
	return t.Rpc
}
func (m *ManagerMessage) EnsureRPC() *RPC {
	if m.Rpc == nil {
		m.Rpc = &RPC{}
	}
	return m.Rpc
}
// receivableRPCMessage is an rpcMessage that we can receive, and unmarshal, using generics, from a
// byte stream.  proto.Unmarshal requires us to have already allocated the memory for the message
// type we are unmarshalling.  All our message types are pointers like *TunnelMessage, so to
// allocate, the compiler needs to know:
//
// a) that the type is a pointer type
// b) what type it is pointing to
//
// So, this generic interface requires that the message is a pointer to the type RR.  Then, we pass
// both the receivableRPCMessage and RR as type constraints, so that we'll have access to the
// underlying type when it comes time to allocate it.  It's a bit messy, but the alternative is
// reflection, which has its own challenges in understandability.
type receivableRPCMessage[RR any] interface {
	rpcMessage
	*RR
}
const (
	SpeakerRoleManager SpeakerRole = "manager"
	SpeakerRoleTunnel  SpeakerRole = "tunnel"
)
// speaker is an implementation of the CoderVPN protocol. It handles unary RPCs and their responses,
// as well as the low-level serialization & deserialization to the ReadWriteCloser (rwc).
//
//	      ┌────────┐                                                             sendCh
//	◄─────│        ◄──────────────────────────────────────────────────────────────────        ◄┐
//	      │        │                                                ▲ rpc requests
//	rwc   │ serdes │                                                │                          │ sendReply()
//	      │        │        ┌───────────────────┐            ┌──────┼──────┐
//	──────►        ┼────────►  recvFromSerdes() │ rpc        │rpc handling │                   │
//	      └────────┘ recvCh │                   ┼────────────►             ◄──── unaryRPC()
//	                        │                   │ responses  │             │                   │
//	                        │                   │            │             │
//	                        │                   │            └─────────────┘              ┌ ─ ─│─ ─ ─ ─ ─ ─ ─ ┐
//	                        │                   ┼──────────────────────────────────────────► request handling
//	                        └───────────────────┘                               requests     (outside speaker)
//	                                                                                      └ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘
//
// speaker is implemented as a generic type that accepts the type of message we send (S), the type we receive (R), and
// the underlying type that R points to (RR).  The speaker is intended to be wrapped by another, non-generic type for
// the role (manager or tunnel).  E.g. Tunnel from this package.
//
// The serdes handles SERialiazation and DESerialization of the low level message types. The wrapping type may send
// non-RPC messages (that is messages that don't expect an explicit reply) by sending on the sendCh.
//
// Unary RPCs are handled by the unaryRPC() function, which handles sending the message and waiting for the response.
//
// recvFromSerdes() reads all incoming messages from the serdes. If they are RPC responses, it dispatches them to the
// waiting unaryRPC() function call, if any.  If they are RPC requests or non-RPC messages, it wraps them in a request
// struct and sends them over the requests chan.  The manager/tunnel role type must read from this chan and handle
// the requests.  If they are RPC types, it should call sendReply() on the request with the reply message.
type speaker[S rpcMessage, R receivableRPCMessage[RR], RR any] struct {
	serdes    *serdes[S, R, RR]
	requests  chan *request[S, R]
	logger    slog.Logger
	nextMsgID uint64
	ctx    context.Context
	cancel context.CancelFunc
	sendCh       chan<- S
	recvCh       <-chan R
	recvLoopDone chan struct{}
	mu            sync.Mutex
	responseChans map[uint64]chan R
}
// newSpeaker creates a new protocol speaker.
func newSpeaker[S rpcMessage, R receivableRPCMessage[RR], RR any](
	ctx context.Context, logger slog.Logger, conn io.ReadWriteCloser,
	me, them SpeakerRole,
) (
	*speaker[S, R, RR], error,
) {
	ctx, cancel := context.WithCancel(ctx)
	if err := handshake(ctx, conn, logger, me, them); err != nil {
		cancel()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}
	sendCh := make(chan S)
	recvCh := make(chan R)
	s := &speaker[S, R, RR]{
		serdes:        newSerdes(ctx, logger, conn, sendCh, recvCh),
		logger:        logger,
		requests:      make(chan *request[S, R]),
		responseChans: make(map[uint64]chan R),
		nextMsgID:     1,
		ctx:           ctx,
		cancel:        cancel,
		sendCh:        sendCh,
		recvCh:        recvCh,
		recvLoopDone:  make(chan struct{}),
	}
	return s, nil
}
// start starts the serialzation/deserialization.  It's important this happens
// after any assignments of the speaker to its owning Tunnel or Manager, since
// the mutex is copied and that is not threadsafe.
// nolint: revive
func (s *speaker[_, _, _]) start() {
	s.serdes.start()
	go s.recvFromSerdes()
}
func (s *speaker[S, R, _]) recvFromSerdes() {
	defer close(s.recvLoopDone)
	defer close(s.requests)
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug(s.ctx, "recvFromSerdes context done while waiting for proto", slog.Error(s.ctx.Err()))
			return
		case msg, ok := <-s.recvCh:
			if !ok {
				s.logger.Debug(s.ctx, "recvCh is closed")
				return
			}
			rpc := msg.GetRpc()
			if rpc != nil && rpc.ResponseTo != 0 {
				// this is a unary response
				s.tryToDeliverResponse(msg)
				continue
			}
			req := &request[S, R]{
				ctx:     s.ctx,
				msg:     msg,
				replyCh: s.sendCh,
			}
			select {
			case <-s.ctx.Done():
				s.logger.Debug(s.ctx, "recvFromSerdes context done while waiting for request handler", slog.Error(s.ctx.Err()))
				return
			case s.requests <- req:
			}
		}
	}
}
// Close closes the speaker
// nolint: revive
func (s *speaker[_, _, _]) Close() error {
	s.cancel()
	err := s.serdes.Close()
	return err
}
// unaryRPC sends a request/response style RPC over the protocol, waits for the response, then
// returns the response
func (s *speaker[S, R, _]) unaryRPC(ctx context.Context, req S) (resp R, err error) {
	rpc := req.EnsureRPC()
	msgID, respCh := s.newRPC()
	rpc.MsgId = msgID
	logger := s.logger.With(slog.F("msg_id", msgID))
	select {
	case <-ctx.Done():
		return resp, ctx.Err()
	case <-s.ctx.Done():
		return resp, fmt.Errorf("vpn protocol closed: %w", s.ctx.Err())
	case <-s.recvLoopDone:
		logger.Debug(s.ctx, "recvLoopDone while sending request")
		return resp, io.ErrUnexpectedEOF
	case s.sendCh <- req:
		logger.Debug(s.ctx, "sent rpc request", slog.F("req", req))
	}
	select {
	case <-ctx.Done():
		s.rmResponseChan(msgID)
		return resp, ctx.Err()
	case <-s.ctx.Done():
		s.rmResponseChan(msgID)
		return resp, fmt.Errorf("vpn protocol closed: %w", s.ctx.Err())
	case <-s.recvLoopDone:
		logger.Debug(s.ctx, "recvLoopDone while waiting for response")
		return resp, io.ErrUnexpectedEOF
	case resp = <-respCh:
		logger.Debug(s.ctx, "got response", slog.F("resp", resp))
		return resp, nil
	}
}
func (s *speaker[_, R, _]) newRPC() (uint64, chan R) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgID := s.nextMsgID
	s.nextMsgID++
	c := make(chan R)
	s.responseChans[msgID] = c
	return msgID, c
}
func (s *speaker[_, _, _]) rmResponseChan(msgID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.responseChans, msgID)
}
func (s *speaker[_, R, _]) tryToDeliverResponse(resp R) {
	msgID := resp.GetRpc().GetResponseTo()
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.responseChans[msgID]
	if ok {
		c <- resp
		// Remove the channel since we delivered a response. This ensures that each response channel
		// gets _at most_ one response.  Since the channels are buffered with size 1, send will
		// never block.
		delete(s.responseChans, msgID)
	}
}
// handshake performs the initial CoderVPN protocol handshake over the given conn
func handshake(
	ctx context.Context, conn io.ReadWriteCloser, logger slog.Logger, me, them SpeakerRole,
) error {
	// read and write simultaneously to avoid deadlocking if the conn is not buffered
	errCh := make(chan error, 2)
	go func() {
		ours := headerString(me, CurrentSupportedVersions)
		_, err := conn.Write([]byte(ours))
		logger.Debug(ctx, "wrote out header")
		if err != nil {
			err = fmt.Errorf("write header: %w", err)
		}
		errCh <- err
	}()
	headerCh := make(chan string, 1)
	go func() {
		// we can't use bufio.Scanner here because we need to ensure we don't read beyond the
		// first newline. So, we'll read one byte at a time. It's inefficient, but the initial
		// header is only a few characters, so we'll keep this code simple.
		buf := make([]byte, 256)
		have := 0
		for {
			_, err := conn.Read(buf[have : have+1])
			if err != nil {
				errCh <- fmt.Errorf("read header: %w", err)
				return
			}
			if buf[have] == '\n' {
				logger.Debug(ctx, "got newline header delimiter")
				// use have (not have+1) since we don't want the delimiter for verification.
				headerCh <- string(buf[:have])
				return
			}
			have++
			if have >= len(buf) {
				errCh <- fmt.Errorf("header malformed or too large: %s", string(buf))
				return
			}
		}
	}()
	writeOK := false
	theirHeader := ""
	readOK := false
	for !(readOK && writeOK) {
		select {
		case <-ctx.Done():
			_ = conn.Close() // ensure our read/write goroutines get a chance to clean up
			return ctx.Err()
		case err := <-errCh:
			if err == nil {
				// write goroutine sends nil when completing successfully.
				logger.Debug(ctx, "write ok")
				writeOK = true
				continue
			}
			_ = conn.Close()
			return err
		case theirHeader = <-headerCh:
			logger.Debug(ctx, "read ok")
			readOK = true
		}
	}
	logger.Debug(ctx, "handshake read/write complete", slog.F("their_header", theirHeader))
	gotVersion, err := validateHeader(theirHeader, them, CurrentSupportedVersions)
	if err != nil {
		return fmt.Errorf("validate header (%s): %w", theirHeader, err)
	}
	logger.Debug(ctx, "handshake validated", slog.F("common_version", gotVersion))
	// TODO: actually use the common version to perform different behavior once
	// we have multiple versions
	return nil
}
const headerPreamble = "codervpn"
func headerString(role SpeakerRole, versions RPCVersionList) string {
	return fmt.Sprintf("%s %s %s\n", headerPreamble, role, versions.String())
}
func validateHeader(header string, expectedRole SpeakerRole, supportedVersions RPCVersionList) (RPCVersion, error) {
	parts := strings.Split(header, " ")
	if len(parts) != 3 {
		return RPCVersion{}, errors.New("wrong number of parts")
	}
	if parts[0] != headerPreamble {
		return RPCVersion{}, errors.New("invalid preamble")
	}
	if parts[1] != string(expectedRole) {
		return RPCVersion{}, errors.New("unexpected role")
	}
	otherVersions, err := ParseRPCVersionList(parts[2])
	if err != nil {
		return RPCVersion{}, fmt.Errorf("parse version list %q: %w", parts[2], err)
	}
	compatibleVersion, ok := supportedVersions.IsCompatibleWith(otherVersions)
	if !ok {
		return RPCVersion{},
			fmt.Errorf("current supported versions %q is not compatible with peer versions %q", supportedVersions.String(), otherVersions.String())
	}
	return compatibleVersion, nil
}
type request[S rpcMessage, R rpcMessage] struct {
	ctx     context.Context
	msg     R
	replyCh chan<- S
}
func (r *request[S, _]) sendReply(reply S) error {
	rrpc := reply.EnsureRPC()
	mrpc := r.msg.GetRpc()
	if mrpc == nil {
		return fmt.Errorf("message didn't want a reply")
	}
	rrpc.ResponseTo = mrpc.MsgId
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	case r.replyCh <- reply:
	}
	return nil
}
