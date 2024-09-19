package agentapi

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/agent/proto"
	"github.com/coder/coder/v2/coderd/database"
)

type WorkspacePrebuildsAPI struct {
	AgentFn                         func(ctx context.Context) (database.WorkspaceAgent, error)
	WorkspaceIDFn                   func(ctx context.Context, agent *database.WorkspaceAgent) (uuid.UUID, error)
	PublishWorkspacePrebuildReadyFn func(ctx context.Context, workspaceID uuid.UUID)
}

func (a *WorkspacePrebuildsAPI) MarkWorkspacePrebuildReady(ctx context.Context, _ *proto.MarkWorkspacePrebuildReadyRequest) (*proto.MarkWorkspacePrebuildReadyResponse, error) {
	agent, err := a.AgentFn(ctx)
	if err != nil {
		return nil, xerrors.Errorf("load agent: %w", err)
	}

	workspaceID, err := a.WorkspaceIDFn(ctx, &agent)
	if err != nil {
		return nil, xerrors.Errorf("load workspace ID: %w", err)
	}

	a.PublishWorkspacePrebuildReadyFn(ctx, workspaceID)
	return &proto.MarkWorkspacePrebuildReadyResponse{}, nil
}
