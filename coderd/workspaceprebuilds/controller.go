package workspaceprebuilds

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"cdr.dev/slog"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbtime"
	"github.com/coder/coder/v2/coderd/database/provisionerjobs"
	"github.com/coder/coder/v2/coderd/database/pubsub"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/coderd/rbac/policy"
	"github.com/coder/coder/v2/coderd/wsbuilder"
)

// TODO: use different name to describe the prebuild definition and the workspace prebuild; terminology is currently confusing.

// Controller TODO
type Controller struct {
	stopping chan any

	store      database.Store
	pubsub     pubsub.Pubsub
	authorizer authorizeFunc

	logger slog.Logger
}

type authorizeFunc func(r *http.Request, action policy.Action, object rbac.Objecter) bool

func NewController(store database.Store, ps pubsub.Pubsub, authorizer authorizeFunc, logger slog.Logger) *Controller {
	return &Controller{
		stopping: make(chan any, 1),

		store:      store,
		pubsub:     ps,
		authorizer: authorizer,

		logger: logger,
	}
}

func (m Controller) provisionPrebuildWorkspace(ctx context.Context, prebuildID uuid.UUID) {
	// TODO: auditing
	aReq := &audit.Request[database.Workspace]{}

	prebuild, err := m.store.GetWorkspacePrebuildByID(ctx, prebuildID)
	if err != nil {
		m.logger.Error(ctx, "failed to create new prebuild workspace", slog.F("prebuild_id", prebuildID), slog.Error(err))
		return
	}

	logger := m.logger.With(slog.F("prebuild_id", prebuildID))

	// TODO: who will the actor be??
	initiator := prebuild.CreatedBy.UUID
	// TODO: who will the owner be??
	owner := prebuild.CreatedBy.UUID
	// TODO: rich params

	//nolint:gocritic // System needs to be able to get owner roles.
	roles, err := m.store.GetAuthorizationUserRoles(dbauthz.AsSystemRestricted(ctx), owner)
	if err != nil {
		logger.Error(ctx, "failed to load prebuild owner details", slog.F("owner_id", owner), slog.Error(err))
		return
	}

	roleNames, err := roles.RoleNames()
	if err != nil {
		logger.Error(ctx, "failed to load prebuild owner roles", slog.F("owner_id", owner), slog.Error(err))
		return
	}

	subject := rbac.Subject{
		ID:     owner.String(),
		Roles:  rbac.RoleIdentifiers(roleNames),
		Groups: roles.Groups,
		Scope:  rbac.ScopeAll,
	}.WithCachedASTValue()

	userCtx := dbauthz.As(ctx, subject)

	template, err := m.store.GetTemplateByID(ctx, prebuild.TemplateID)
	if err != nil {
		logger.Error(ctx, "failed to create fetch prebuild template", slog.F("template_id", prebuild.TemplateID), slog.Error(err))
		return
	}

	var (
		workspace      database.Workspace
		provisionerJob *database.ProvisionerJob
		workspaceBuild *database.WorkspaceBuild
	)
	err = m.store.InTx(func(db database.Store) error {
		now := dbtime.Now()
		// Workspaces are created without any versions.
		workspace, err = db.InsertWorkspace(userCtx, database.InsertWorkspaceParams{
			ID:               uuid.New(),
			CreatedAt:        now,
			UpdatedAt:        now,
			OwnerID:          owner,
			OrganizationID:   template.OrganizationID,
			TemplateID:       template.ID,
			Name:             m.generatePrebuildWorkspaceName(prebuild.Name),
			LastUsedAt:       dbtime.Now(),
			AutomaticUpdates: database.AutomaticUpdatesAlways, // TODO: ?
			PrebuildID:       uuid.NullUUID{UUID: prebuildID, Valid: true},
			PrebuildAssigned: sql.NullBool{Bool: false, Valid: true},
		})
		if err != nil {
			return xerrors.Errorf("insert workspace: %w", err)
		}

		builder := wsbuilder.New(workspace, database.WorkspaceTransitionStart).
			Reason(database.BuildReasonInitiator).
			Initiator(initiator).
			// ActiveVersion() // TODO: always active version?
			VersionID(prebuild.TemplateVersionID)
		// RichParameterValues(req.RichParameterValues)

		workspaceBuild, provisionerJob, err = builder.Build(
			ctx,
			db,
			func(action policy.Action, object rbac.Objecter) bool {
				// TODO: auth?
				return true
			},
			audit.WorkspaceBuildBaggage{}, // TODO: audit.WorkspaceBuildBaggageFromRequest(r)
		)
		return err
	}, nil)
	var bldErr wsbuilder.BuildError
	if xerrors.As(err, &bldErr) {
		logger.Error(ctx, "prebuild workspace build failure", slog.F("status", bldErr.Status), slog.F("msg", bldErr.Message), slog.Error(bldErr))
		return
	}
	if err != nil {
		logger.Error(ctx, "prebuild workspace build failure", slog.Error(err))
		return
	}
	err = provisionerjobs.PostJob(m.pubsub, *provisionerJob)
	if err != nil {
		logger.Error(ctx, "failed to post provisioner job to pubsub", slog.Error(err))
	}
	aReq.New = workspace

	// TODO: telemetry?
	// api.Telemetry.Report(&telemetry.Snapshot{
	// 	Workspaces:      []telemetry.Workspace{telemetry.ConvertWorkspace(workspace)},
	// 	WorkspaceBuilds: []telemetry.WorkspaceBuild{telemetry.ConvertWorkspaceBuild(*workspaceBuild)},
	// })

	logger.Info(ctx, "prebuild workspace created!", slog.F("workspace_build_id", workspaceBuild.ID.String()))
}

func (m Controller) generatePrebuildWorkspaceName(base string) string {
	hash := md5.Sum([]byte(uuid.New().String()))
	return fmt.Sprintf("%s-%x", base, hash[:6])
}

// Run subscribes to various pubsub channels to implement a control loop. Run blocks until the given context is canceled
// or Stop is called.
func (m Controller) Run(ctx context.Context) error {
	cancelCreated, err := m.pubsub.SubscribeWithErr(PrebuildCreatedChannel(), m.prebuildCreatedListener)
	defer cancelCreated()
	if err != nil {
		m.logger.Warn(ctx, "failed to subscribe to prebuild creations", slog.Error(err))
		return err
	}

	cancelReady, err := m.pubsub.SubscribeWithErr(PrebuildReadyChannel(), m.prebuildReadyListener)
	defer cancelReady()
	if err != nil {
		m.logger.Warn(ctx, "failed to subscribe to prebuild creations", slog.Error(err))
		return err
	}

	// Reconcile state every 30s as a backup mechanism (state should already be reconciled using pubsub).
	go m.reconcileLoop(ctx, time.Second*30)

	// Wait until context is canceled or Manager is stopped.
	select {
	case <-m.stopping:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m Controller) reconcileLoop(ctx context.Context, dur time.Duration) {
	tick := time.NewTicker(dur)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			m.logger.Debug(ctx, "reconciling state")
			prebuilds, err := m.store.GetWorkspacePrebuilds(dbauthz.AsSystemRestricted(ctx))
			if err != nil {
				m.logger.Warn(ctx, "cannot run periodic state reconciliation", slog.Error(err))
				continue
			}

			var wg errgroup.Group
			for _, pb := range prebuilds {
				wg.Go(func() error {
					return m.ReconcileState(ctx, pb.ID)
				})
			}

			err = wg.Wait()
			if err != nil {
				m.logger.Warn(ctx, "periodic state reconciliation failed", slog.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m Controller) Stop(ctx context.Context) error {
	m.logger.Info(ctx, "stop requested")
	close(m.stopping)
	return nil
}
