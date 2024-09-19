package coderd

import (
	"context"
	"math/rand"
	"net/http"

	"cdr.dev/slog"
	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/db2sdk"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/rbac/policy"
	"github.com/coder/coder/v2/coderd/workspaceprebuilds"
	"github.com/coder/coder/v2/codersdk"
)

func (api *API) postWorkspacePrebuilds(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	apiKey := httpmw.APIKey(r)
	org := httpmw.OrganizationParam(r)

	// prebuild := httpmw.WorkspacePrebuildParam(r)
	var createPrebuild codersdk.CreateWorkspacePrebuildPoolRequest
	if !httpapi.Read(ctx, rw, r, &createPrebuild) {
		return
	}

	createPrebuild.CreatedBy = apiKey.UserID
	createPrebuild.OrganizationID = org.ID

	pb, err := api.PrebuildsController.CreateNewWorkspacePrebuild(ctx, createPrebuild)
	if err != nil || pb == nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Internal error creating workspace prebuild.",
			Detail:  err.Error(),
		})
		return
	}

	apiPb := db2sdk.WorkspacePrebuildPool(*pb)
	httpapi.Write(ctx, rw, http.StatusCreated, apiPb)
}

func (api *API) publishWorkspacePrebuildReady(ctx context.Context, workspaceID uuid.UUID) {
	if err := api.Pubsub.Publish(workspaceprebuilds.PrebuildReadyChannel(), []byte(workspaceID.String())); err != nil {
		api.Logger.Warn(ctx, "failed to publish workspace prebuild ready event", slog.Error(err), slog.F("workspace_id", workspaceID))
	}
}

func (api *API) assignPrebuildToUser(ctx context.Context, r *http.Request, req codersdk.CreateWorkspaceRequest, owner workspaceOwner) (*database.WorkspaceBuild, *database.ProvisionerJob, bool, error) {
	// Check for, and optionally use, a prebuilt workspace.
	// TODO: handle req.TemplateID being set but not req.TemplateVersionID
	prebuildPool, err := api.findMatchingPrebuild(ctx, req.TemplateVersionID, req.RichParameterValues) // TODO: what if no TemplateVersionID given?
	if err != nil || prebuildPool == nil {
		api.Logger.Warn(ctx, "failed to find matching prebuilds", slog.Error(err))
		return nil, nil, false, nil
	}

	logger := api.Logger.With(slog.F("prebuild_pool_id", prebuildPool.ID))

	if !req.UsePrebuild {
		logger.Info(ctx, "prebuild found, but not used - per request")
		return nil, nil, false, nil
	}

	// nominate prebuilt workspace for transfer
	workspaces, err := api.Database.GetUnassignedPrebuildsByPoolID(ctx, prebuildPool.ID)
	if err != nil {
		logger.Warn(ctx, "failed to load workspaces for prebuild", slog.Error(err))
		return nil, nil, false, nil
	}

	if len(workspaces) == 0 {
		logger.Warn(ctx, "no available prebuild workspaces")
		return nil, nil, false, nil
	}

	// pick random victim workspace
	victim := workspaces[rand.Intn(len(workspaces))]

	// transfer victim to new owner
	var (
		workspaceBuild *database.WorkspaceBuild
		provisionerJob *database.ProvisionerJob
	)

	err = api.Database.InTx(func(db database.Store) error {
		// Prospectively mark the prebuild as reassigned to the new owner.
		// If the transfer fails to complete, this will get rolled back.
		err = api.Database.MarkWorkspacePrebuildAssigned(ctx, victim.ID)
		if err != nil {
			return xerrors.Errorf("mark prebuild workspace as reassigned: %w", err)
		}

		// Transfer the prebuild workspace to the new owner.
		workspaceBuild, provisionerJob, err = transferWorkspace(ctx, db, workspaceTransferRequest{
			workspaceID:  victim.ID,
			newOwnerID:   owner.ID,
			richParams:   nil, // TODO: params
			auditBaggage: audit.WorkspaceBuildBaggageFromRequest(r),
		}, func(action policy.Action, object rbac.Objecter) bool {
			return api.Authorize(r, action, object)
		})
		if err != nil {
			return xerrors.Errorf("transfer workspace: %w", err)
		}

		return nil
	}, nil)

	if err != nil {
		return nil, nil, true, err
	}

	// Wait until tx completes because a) renaming mustn't fail the tx and b) we need the auth context to update so it works
	_, err = api.Database.UpdateWorkspace(ctx, database.UpdateWorkspaceParams{
		ID:   victim.ID,
		Name: req.Name,
	})
	if err != nil {
		// Don't fail the operation, this is just a renaming.
		api.Logger.Warn(ctx, "failed to rename prebuild workspace",
			slog.F("prebuild_id", victim.ID.String()), slog.F("name", req.Name))
	}

	logger.Info(ctx, "prebuild assigning to new owner", slog.F("new_owner_id", owner.ID))

	// Trigger a reconciliation.
	_ = api.Pubsub.Publish(workspaceprebuilds.PrebuildReconcileChannel(), []byte(prebuildPool.ID.String()))

	return workspaceBuild, provisionerJob, true, err
}
