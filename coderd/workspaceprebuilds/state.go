package workspaceprebuilds

import (
	"context"

	"cdr.dev/slog"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

// ReconcileState reads the current state of a prebuild and attempts to reconcile it against its definition.
func (m Controller) ReconcileState(ctx context.Context, prebuildID uuid.UUID) error {
	// TODO: add exclusive advisory lock in database to prevent replicas from performing the same action.
	// TODO: validate that all required params are set if template changes.

	// fetch prebuild
	// check if has desired number of UNASSIGNED replicas
	//		if not, schedule workspace builds
	// remember: workspaces are created before workspace builds...
	prebuild, err := m.store.GetWorkspacePrebuildByID(ctx, prebuildID)
	if err != nil {
		return xerrors.Errorf("failed to load prebuild by ID %q: %w", prebuildID.String(), err)
	}

	// TODO: also check for workspaces which are in a transitionary state (i.e. created but building or failed)
	// WITHOUT THIS, WE WILL REPORT INCORRECT FIGURES AND PROVISION TOO MANY REPLICAS.
	workspaces, err := m.store.GetPrebuildsByPoolID(ctx, prebuildID)
	if err != nil {
		return xerrors.Errorf("failed to load prebuild workspaces by ID %q: %w", prebuildID.String(), err)
	}

	logger := m.logger.With(slog.F("expected_count", prebuild.Count), slog.F("actual_count", len(workspaces)))
	if len(workspaces) < prebuild.Count {
		logger.Warn(ctx, "prebuild is missing workspaces, provisioning...")
		// TODO: prevent further calls in case this takes long
		// add replicas
		for i := 0; i < prebuild.Count - len(workspaces); i++ {
			m.provisionPrebuildWorkspace(ctx, prebuildID)
		}
	} else if len(workspaces) > prebuild.Count {
		// TODO: nominate replicas to be deleted
		logger.Warn(ctx, "too many replicas found")
	} else {
		logger.Debug(ctx, "prebuild is in expected state")
	}

	// check if template has changed -> trigger update of workspaces
	// check if any associated (unassigned) workspaces are in a failed state, attempt to restart them and only do this 5x before logging error.
	return nil
}
