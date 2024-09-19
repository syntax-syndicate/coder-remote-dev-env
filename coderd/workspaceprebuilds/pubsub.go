package workspaceprebuilds

import (
	"context"
	"errors"

	"cdr.dev/slog"
	"github.com/google/uuid"

	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/pubsub"
)

func PrebuildCreatedChannel() string {
	return "prebuild-created"
}

func PrebuildReadyChannel() string {
	return "prebuild-ready"
}

func (m Controller) prebuildCreatedListener(ctx context.Context, message []byte, err error) {
	if errors.Is(err, pubsub.ErrDroppedMessages) {
		m.logger.Warn(ctx, "pubsub may have dropped prebuild creation events")
		// TODO: run slow check to fetch all prebuilds and reconcile state
		return
	}

	prebuildID, err := uuid.ParseBytes(message)
	if err != nil {
		m.logger.Error(ctx, "failed to parse prebuild ID", slog.F("prebuild_id", message), slog.Error(err))
		return
	}

	if err = m.ReconcileState(dbauthz.AsSystemRestricted(ctx), prebuildID); err != nil {
		m.logger.Error(ctx, "failed to reconcile prebuild state", slog.F("prebuild_id", message), slog.Error(err))
		return
	}
}

func (m Controller) prebuildReadyListener(ctx context.Context, message []byte, err error) {
	if errors.Is(err, pubsub.ErrDroppedMessages) {
		m.logger.Warn(ctx, "pubsub may have dropped prebuild ready events")
		return
	}

	workspaceID, err := uuid.ParseBytes(message)
	if err != nil {
		m.logger.Error(ctx, "failed to parse workspace ID", slog.F("workspace_id", message), slog.Error(err))
		return
	}

	if err = m.MarkWorkspacePrebuildReady(ctx, workspaceID); err != nil {
		m.logger.Error(ctx, "failed to mark workspace prebuild as ready", slog.F("workspace_id", message), slog.Error(err))
	}
}
