package workspaceprebuilds

import (
	"context"

	"cdr.dev/slog"
	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/codersdk"
)

// CreateNewWorkspacePrebuild creates a new workspace prebuild in the database and triggers the workspace builds.
// TODO: parameter matching arithmetic when creating new workspace which may match pubsub definition
func (m Controller) CreateNewWorkspacePrebuild(ctx context.Context, req codersdk.CreateWorkspacePrebuildRequest) (*database.WorkspacePrebuild, error) {
	// TODO: auditing
	// TODO: trigger workspace builds
	// TODO: validate required params are set

	// This field is nullable so that it can be set to NULL when the user who created the prebuild is deleted.
	// However, we should always enforce that this is set initially.
	if req.CreatedBy == uuid.Nil {
		return nil, xerrors.Errorf("prebuild creation requires a CreatedBy value")
	}

	pb, err := m.store.UpsertWorkspacePrebuild(ctx, database.UpsertWorkspacePrebuildParams{
		ID:                uuid.New(),
		Name:              req.Name,
		Replicas:          req.Replicas,
		OrganizationID:    req.OrganizationID,
		TemplateID:        req.TemplateID,
		TemplateVersionID: req.TemplateVersionID,
		CreatedBy:         uuid.NullUUID{UUID: req.CreatedBy, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	if err = m.pubsub.Publish(PrebuildCreatedChannel(), []byte(pb.ID.String())); err != nil {
		m.logger.Warn(ctx, "failed to publish prebuild creation message", slog.F("prebuild_id", pb.ID.String()))
	}

	return &pb, nil
}

// Refresh will run a new Plan operation on all workspaces associated to the given prebuild in order to see if the associated
// template has been modified or the cache key (`coder_prebuild_cache_key` resource) has been invalidated. In either of
// these scenarios, the current workspaces should be updated.
func Refresh(ctx context.Context, prebuildID uuid.UUID) error {
	return nil
}

func Transfer(ctx context.Context, prebuildID uuid.UUID, request codersdk.CreateWorkspaceRequest) error {
	// use workspace name from request
	// rebuild workspace with additional params from request
	// transfer to new owner
	return nil
}
