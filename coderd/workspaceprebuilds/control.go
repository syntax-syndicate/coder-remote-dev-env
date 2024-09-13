package workspaceprebuilds

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/codersdk"
)

// TODO:
//		-> parameter matching arithmetic
//		-> pubsub notifications

type Store interface {
	GetWorkspacePrebuildByID(ctx context.Context, id uuid.UUID) (database.WorkspacePrebuild, error)
	GetWorkspacePrebuildParameters(ctx context.Context, workspacePrebuildID uuid.UUID) ([]database.WorkspacePrebuildParameter, error)
	GetWorkspacePrebuilds(ctx context.Context) ([]database.WorkspacePrebuild, error)
	UpsertWorkspacePrebuild(ctx context.Context, arg database.UpsertWorkspacePrebuildParams) (database.WorkspacePrebuild, error)
}

// CreateNewWorkspacePrebuild creates a new workspace prebuild in the database and triggers the workspace builds.
func CreateNewWorkspacePrebuild(ctx context.Context, store Store, req codersdk.CreateWorkspacePrebuildRequest) (*database.WorkspacePrebuild, error) {
	// TODO: auditing
	// TODO: trigger workspace builds

	// This field is nullable so that it can be set to NULL when the user who created the prebuild is deleted.
	// However, we should always enforce that this is set initially.
	if req.CreatedBy == uuid.Nil {
		return nil, xerrors.Errorf("prebuild creation requires a CreatedBy value")
	}

	pb, err := store.UpsertWorkspacePrebuild(ctx, database.UpsertWorkspacePrebuildParams{
		ID:                uuid.New(),
		Name:              req.Name,
		Replicas:          int32(req.Replicas),
		OrganizationID:    req.OrganizationID,
		TemplateID:        req.TemplateID,
		TemplateVersionID: req.TemplateVersionID,
		CreatedBy:         uuid.NullUUID{UUID: req.CreatedBy, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &pb, nil
}

// ReconcileState reads the current state of a prebuild and attempts to reconcile it against its definition.
func ReconcileState(ctx context.Context, prebuildID uuid.UUID) error {
	// fetch prebuild
	// check if has desired number of replicas
	//		if not, schedule workspace builds
	// check if template has changed -> trigger update of workspaces
	// check if any associated (unassigned) workspaces are in a failed state, attempt to restart them and only do this 5x before logging error.
	return nil
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
