package workspaceprebuilds

import (
	"context"

	"cdr.dev/slog"
	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/rbac/policy"
	"github.com/coder/coder/v2/coderd/wsbuilder"
	"github.com/coder/coder/v2/codersdk"
)

// CreateNewWorkspacePrebuild creates a new workspace prebuild in the database and triggers the workspace builds.
// TODO: parameter matching arithmetic when creating new workspace which may match pubsub definition
func (m Controller) CreateNewWorkspacePrebuild(ctx context.Context, req codersdk.CreateWorkspacePrebuildRequest) (*database.WorkspacePrebuildPool, error) {
	// TODO: auditing
	// TODO: trigger workspace builds
	// TODO: validate required params are set

	// This field is nullable so that it can be set to NULL when the user who created the prebuild is deleted.
	// However, we should always enforce that this is set initially.
	if req.CreatedBy == uuid.Nil {
		return nil, xerrors.Errorf("prebuild creation requires a CreatedBy value")
	}

	pb, err := m.store.UpsertWorkspacePrebuildPool(ctx, database.UpsertWorkspacePrebuildPoolParams{
		ID:                uuid.New(),
		Name:              req.Name,
		Count:             req.Count,
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

func (m Controller) MarkWorkspacePrebuildReady(ctx context.Context, workspaceID uuid.UUID) error {
	// TODO: less broad context
	ctx = dbauthz.AsSystemRestricted(ctx)

	workspace, err := m.store.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return xerrors.Errorf("could not find given workspace %q: %w", workspaceID, err)
	}

	// Workspace is not associated to a prebuild, bail out.
	if workspace.PrebuildID.UUID == uuid.Nil {
		return nil
	}

	// Workspace has already been assigned, so we don't need to stop the workspace to prepare it for assignment.
	if workspace.PrebuildAssigned.Bool {
		return nil
	}

	prebuildPool, err := m.store.GetWorkspacePrebuildByID(ctx, workspace.PrebuildID.UUID)
	if err != nil {
		return xerrors.Errorf("could not find associated prebuild pool definition for workspace %q: %w", workspaceID, err)
	}

	m.logger.Info(ctx, "prebuild workspace is ready! stopping...", slog.F("prebuild_pool_id", prebuildPool.ID), slog.F("workspace_id", workspaceID))

	builder := wsbuilder.New(workspace, database.WorkspaceTransitionStop).
		Reason(database.BuildReasonPrebuildReady).
		Initiator(workspace.OwnerID). // TODO: system user?
		// ActiveVersion() // TODO: always active version?
		VersionID(prebuildPool.TemplateVersionID)
	// RichParameterValues(req.RichParameterValues)

	// We don't need the build or provisioner job here; this is a background process.
	_, _, err = builder.Build(
		ctx,
		m.store,
		func(action policy.Action, object rbac.Objecter) bool {
			// TODO: auth?
			return true
		},
		audit.WorkspaceBuildBaggage{}, // TODO: audit.WorkspaceBuildBaggageFromRequest(r)
	)
	return err
}

// Refresh will run a new Plan operation on all workspaces associated to the given prebuild in order to see if the associated
// template has been modified or the cache key (`coder_prebuild_cache_key` resource) has been invalidated. In either of
// these scenarios, the current workspaces should be updated.
func Refresh(ctx context.Context, prebuildID uuid.UUID) error {
	return nil
}