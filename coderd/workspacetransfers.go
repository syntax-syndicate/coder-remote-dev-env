package coderd

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/rbac/policy"
	"github.com/coder/coder/v2/coderd/wsbuilder"
	"github.com/coder/coder/v2/codersdk"
)

type authFunc func(action policy.Action, object rbac.Objecter) bool

// TODO: should this be in codersdk?
// Probably not, this is just used to collapse all the various args into one struct for a cleaner API.
type workspaceTransferRequest struct {
	workspaceID uuid.UUID
	newOwnerID  uuid.UUID
	richParams  []codersdk.WorkspaceBuildParameter
	// latestBuild  database.WorkspaceBuild
	auditBaggage audit.WorkspaceBuildBaggage
}

// transferWorkspace MUST be called in a transaction.
func transferWorkspace(ctx context.Context, dbInTx database.Store, req workspaceTransferRequest, authFunc authFunc) (*database.WorkspaceBuild, *database.ProvisionerJob, error) {
	// Execute transfer of ownership.
	workspace, err := dbInTx.TransferWorkspaceOwnership(ctx, database.TransferWorkspaceOwnershipParams{
		TargetUser:  req.newOwnerID,
		WorkspaceID: req.workspaceID,
	})
	if err != nil {
		return nil, nil, xerrors.Errorf("transfer workspace ownership: %w", err)
	}

	// Rebuild workspace with new ownership.
	builder := wsbuilder.New(workspace, database.WorkspaceTransitionStop).
		Reason(database.BuildReasonTransfer).
		Initiator(req.newOwnerID).
		ActiveVersion(). // TODO: always require active version?
		RichParameterValues(req.richParams)

	wb, pj, err := builder.Build(
		ctx,
		dbInTx,
		authFunc,
		req.auditBaggage,
	)
	if err != nil {
		return nil, nil, xerrors.Errorf("rebuild workspace with new ownership: %w", err)
	}

	return wb, pj, err
}
