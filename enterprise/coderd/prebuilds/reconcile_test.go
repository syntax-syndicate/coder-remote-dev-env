package prebuilds_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/quartz"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"tailscale.com/types/ptr"

	"github.com/coder/serpent"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/database/pubsub"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/coderd/prebuilds"
	"github.com/coder/coder/v2/testutil"
)

func TestNoReconciliationActionsIfNoPresets(t *testing.T) {
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres")
	}

	// Scenario: No reconciliation actions are taken if there are no presets
	t.Parallel()

	clock := quartz.NewMock(t)
	ctx := testutil.Context(t, testutil.WaitLong)
	db, pubsub := dbtestutil.NewDB(t)
	cfg := codersdk.PrebuildsConfig{
		ReconciliationInterval: serpent.Duration(testutil.WaitLong),
	}
	logger := testutil.Logger(t)
	controller := prebuilds.NewStoreReconciler(db, pubsub, cfg, logger, quartz.NewMock(t))

	// given a template version with no presets
	org := dbgen.Organization(t, db, database.Organization{})
	user := dbgen.User(t, db, database.User{})
	template := dbgen.Template(t, db, database.Template{
		CreatedBy:      user.ID,
		OrganizationID: org.ID,
	})
	templateVersion := dbgen.TemplateVersion(t, db, database.TemplateVersion{
		TemplateID:     uuid.NullUUID{UUID: template.ID, Valid: true},
		OrganizationID: org.ID,
		CreatedBy:      user.ID,
	})
	// verify that the db state is correct
	gotTemplateVersion, err := db.GetTemplateVersionByID(ctx, templateVersion.ID)
	require.NoError(t, err)
	require.Equal(t, templateVersion, gotTemplateVersion)

	// when we trigger the reconciliation loop for all templates
	require.NoError(t, controller.ReconcileAll(ctx))

	// then no reconciliation actions are taken
	// because without presets, there are no prebuilds
	// and without prebuilds, there is nothing to reconcile
	jobs, err := db.GetProvisionerJobsCreatedAfter(ctx, clock.Now().Add(earlier))
	require.NoError(t, err)
	require.Empty(t, jobs)
}

func TestNoReconciliationActionsIfNoPrebuilds(t *testing.T) {
	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres")
	}

	// Scenario: No reconciliation actions are taken if there are no prebuilds
	t.Parallel()

	clock := quartz.NewMock(t)
	ctx := testutil.Context(t, testutil.WaitLong)
	db, pubsub := dbtestutil.NewDB(t)
	cfg := codersdk.PrebuildsConfig{
		ReconciliationInterval: serpent.Duration(testutil.WaitLong),
	}
	logger := testutil.Logger(t)
	controller := prebuilds.NewStoreReconciler(db, pubsub, cfg, logger, quartz.NewMock(t))

	// given there are presets, but no prebuilds
	org := dbgen.Organization(t, db, database.Organization{})
	user := dbgen.User(t, db, database.User{})
	template := dbgen.Template(t, db, database.Template{
		CreatedBy:      user.ID,
		OrganizationID: org.ID,
	})
	templateVersion := dbgen.TemplateVersion(t, db, database.TemplateVersion{
		TemplateID:     uuid.NullUUID{UUID: template.ID, Valid: true},
		OrganizationID: org.ID,
		CreatedBy:      user.ID,
	})
	preset, err := db.InsertPreset(ctx, database.InsertPresetParams{
		TemplateVersionID: templateVersion.ID,
		Name:              "test",
	})
	require.NoError(t, err)
	_, err = db.InsertPresetParameters(ctx, database.InsertPresetParametersParams{
		TemplateVersionPresetID: preset.ID,
		Names:                   []string{"test"},
		Values:                  []string{"test"},
	})
	require.NoError(t, err)

	// verify that the db state is correct
	presetParameters, err := db.GetPresetParametersByTemplateVersionID(ctx, templateVersion.ID)
	require.NoError(t, err)
	require.NotEmpty(t, presetParameters)

	// when we trigger the reconciliation loop for all templates
	require.NoError(t, controller.ReconcileAll(ctx))

	// then no reconciliation actions are taken
	// because without prebuilds, there is nothing to reconcile
	// even if there are presets
	jobs, err := db.GetProvisionerJobsCreatedAfter(ctx, clock.Now().Add(earlier))
	require.NoError(t, err)
	require.Empty(t, jobs)
}

func TestPrebuildReconciliation(t *testing.T) {
	t.Parallel()

	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres")
	}

	type testCase struct {
		name                      string
		prebuildLatestTransitions []database.WorkspaceTransition
		prebuildJobStatuses       []database.ProvisionerJobStatus
		templateVersionActive     []bool
		shouldCreateNewPrebuild   *bool
		shouldDeleteOldPrebuild   *bool
	}

	testCases := []testCase{
		{
			name: "never create prebuilds for inactive template versions",
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
				database.WorkspaceTransitionStop,
				database.WorkspaceTransitionDelete,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusSucceeded,
				database.ProvisionerJobStatusCanceled,
				database.ProvisionerJobStatusFailed,
				database.ProvisionerJobStatusPending,
				database.ProvisionerJobStatusRunning,
				database.ProvisionerJobStatusCanceling,
			},
			templateVersionActive:   []bool{false},
			shouldCreateNewPrebuild: ptr.To(false),
		},
		{
			name: "no need to create a new prebuild if one is already running",
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusSucceeded,
			},
			templateVersionActive:   []bool{true},
			shouldCreateNewPrebuild: ptr.To(false),
		},
		{
			name: "don't create a new prebuild if one is queued to build or already building",
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusPending,
				database.ProvisionerJobStatusRunning,
			},
			templateVersionActive:   []bool{true},
			shouldCreateNewPrebuild: ptr.To(false),
		},
		{
			name: "create a new prebuild if one is in a state that disqualifies it from ever being claimed",
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStop,
				database.WorkspaceTransitionDelete,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusPending,
				database.ProvisionerJobStatusRunning,
				database.ProvisionerJobStatusCanceling,
				database.ProvisionerJobStatusSucceeded,
			},
			templateVersionActive:   []bool{true},
			shouldCreateNewPrebuild: ptr.To(true),
		},
		{
			// See TestFailedBuildBackoff for the start/failed case.
			name: "create a new prebuild if one is in any kind of exceptional state",
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStop,
				database.WorkspaceTransitionDelete,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusCanceled,
			},
			templateVersionActive:   []bool{true},
			shouldCreateNewPrebuild: ptr.To(true),
		},
		{
			name: "never attempt to interfere with active builds",
			// The workspace builder does not allow scheduling a new build if there is already a build
			// pending, running, or canceling. As such, we should never attempt to start, stop or delete
			// such prebuilds. Rather, we should wait for the existing build to complete and reconcile
			// again in the next cycle.
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
				database.WorkspaceTransitionStop,
				database.WorkspaceTransitionDelete,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusPending,
				database.ProvisionerJobStatusRunning,
				database.ProvisionerJobStatusCanceling,
			},
			templateVersionActive:   []bool{true, false},
			shouldDeleteOldPrebuild: ptr.To(false),
		},
		{
			name: "never delete prebuilds in an exceptional state",
			// We don't want to destroy evidence that might be useful to operators
			// when troubleshooting issues. So we leave these prebuilds in place.
			// Operators are expected to manually delete these prebuilds.
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
				database.WorkspaceTransitionStop,
				database.WorkspaceTransitionDelete,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusCanceled,
				database.ProvisionerJobStatusFailed,
			},
			templateVersionActive:   []bool{true, false},
			shouldDeleteOldPrebuild: ptr.To(false),
		},
		{
			name: "delete running prebuilds for inactive template versions",
			// We only support prebuilds for active template versions.
			// If a template version is inactive, we should delete any prebuilds
			// that are running.
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusSucceeded,
			},
			templateVersionActive:   []bool{false},
			shouldDeleteOldPrebuild: ptr.To(true),
		},
		{
			name: "don't delete running prebuilds for active template versions",
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStart,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusSucceeded,
			},
			templateVersionActive:   []bool{true},
			shouldDeleteOldPrebuild: ptr.To(false),
		},
		{
			name: "don't delete stopped or already deleted prebuilds",
			// We don't ever stop prebuilds. A stopped prebuild is an exceptional state.
			// As such we keep it, to allow operators to investigate the cause.
			prebuildLatestTransitions: []database.WorkspaceTransition{
				database.WorkspaceTransitionStop,
				database.WorkspaceTransitionDelete,
			},
			prebuildJobStatuses: []database.ProvisionerJobStatus{
				database.ProvisionerJobStatusSucceeded,
			},
			templateVersionActive:   []bool{true, false},
			shouldDeleteOldPrebuild: ptr.To(false),
		},
	}
	for _, tc := range testCases {
		tc := tc
		for _, templateVersionActive := range tc.templateVersionActive {
			templateVersionActive := templateVersionActive
			for _, prebuildLatestTransition := range tc.prebuildLatestTransitions {
				prebuildLatestTransition := prebuildLatestTransition
				for _, prebuildJobStatus := range tc.prebuildJobStatuses {
					t.Run(fmt.Sprintf("%s - %s - %s", tc.name, prebuildLatestTransition, prebuildJobStatus), func(t *testing.T) {
						t.Parallel()
						clock := quartz.NewMock(t)
						ctx := testutil.Context(t, testutil.WaitShort)
						cfg := codersdk.PrebuildsConfig{}
						logger := slogtest.Make(
							t, &slogtest.Options{IgnoreErrors: true},
						).Leveled(slog.LevelDebug)
						db, pubsub := dbtestutil.NewDB(t)
						controller := prebuilds.NewStoreReconciler(db, pubsub, cfg, logger, quartz.NewMock(t))

						orgID, userID, templateID := setupTestDBTemplate(t, db)
						templateVersionID := setupTestDBTemplateVersion(t, ctx, clock, db, pubsub, orgID, userID, templateID)
						_, prebuild := setupTestDBPrebuild(
							t,
							ctx,
							clock,
							db,
							pubsub,
							prebuildLatestTransition,
							prebuildJobStatus,
							orgID,
							templateID,
							templateVersionID,
						)

						if !templateVersionActive {
							// Create a new template version and mark it as active
							// This marks the template version that we care about as inactive
							setupTestDBTemplateVersion(t, ctx, clock, db, pubsub, orgID, userID, templateID)
						}

						// Run the reconciliation multiple times to ensure idempotency
						// 8 was arbitrary, but large enough to reasonably trust the result
						for i := 1; i <= 8; i++ {
							require.NoErrorf(t, controller.ReconcileAll(ctx), "failed on iteration %d", i)

							if tc.shouldCreateNewPrebuild != nil {
								newPrebuildCount := 0
								workspaces, err := db.GetWorkspacesByTemplateID(ctx, templateID)
								require.NoError(t, err)
								for _, workspace := range workspaces {
									if workspace.ID != prebuild.ID {
										newPrebuildCount++
									}
								}
								// This test configures a preset that desires one prebuild.
								// In cases where new prebuilds should be created, there should be exactly one.
								require.Equal(t, *tc.shouldCreateNewPrebuild, newPrebuildCount == 1)
							}

							if tc.shouldDeleteOldPrebuild != nil {
								builds, err := db.GetWorkspaceBuildsByWorkspaceID(ctx, database.GetWorkspaceBuildsByWorkspaceIDParams{
									WorkspaceID: prebuild.ID,
								})
								require.NoError(t, err)
								if *tc.shouldDeleteOldPrebuild {
									require.Equal(t, 2, len(builds))
									require.Equal(t, database.WorkspaceTransitionDelete, builds[0].Transition)
								} else {
									require.Equal(t, 1, len(builds))
									require.Equal(t, prebuildLatestTransition, builds[0].Transition)
								}
							}
						}
					})
				}
			}
		}
	}
}

func TestFailedBuildBackoff(t *testing.T) {
	t.Parallel()

	if !dbtestutil.WillUsePostgres() {
		t.Skip("This test requires postgres")
	}
	ctx := testutil.Context(t, testutil.WaitSuperLong)

	// Setup.
	clock := quartz.NewMock(t)
	backoffInterval := time.Minute
	cfg := codersdk.PrebuildsConfig{
		// Given: explicitly defined backoff configuration to validate timings.
		ReconciliationBackoffLookback: serpent.Duration(muchEarlier * -10), // Has to be positive.
		ReconciliationBackoffInterval: serpent.Duration(backoffInterval),
		ReconciliationInterval:        serpent.Duration(time.Second),
	}
	logger := slogtest.Make(
		t, &slogtest.Options{IgnoreErrors: true},
	).Leveled(slog.LevelDebug)
	db, ps := dbtestutil.NewDB(t)
	reconciler := prebuilds.NewStoreReconciler(db, ps, cfg, logger, clock)

	// Given: an active template version with presets and prebuilds configured.
	orgID, userID, templateID := setupTestDBTemplate(t, db)
	templateVersionID := setupTestDBTemplateVersion(t, ctx, clock, db, ps, orgID, userID, templateID)
	preset, _ := setupTestDBPrebuild(t, ctx, clock, db, ps, database.WorkspaceTransitionStart, database.ProvisionerJobStatusFailed, orgID, templateID, templateVersionID)

	// When: determining what actions to take next, backoff is calculated because the prebuild is in a failed state.
	state, err := reconciler.SnapshotState(ctx, db)
	require.NoError(t, err)
	require.Len(t, state.Presets, 1)
	presetState, err := state.FilterByPreset(preset.ID)
	require.NoError(t, err)
	actions, err := reconciler.DetermineActions(ctx, *presetState)
	require.NoError(t, err)
	// Then: the backoff time is in the future, and no prebuilds are running.
	require.EqualValues(t, 0, actions.Actual)
	require.EqualValues(t, 1, actions.Desired)
	require.True(t, clock.Now().Before(actions.BackoffUntil))

	// Then: the backoff time is roughly equal to one backoff interval, since only one build has failed so far.
	require.NotNil(t, presetState.Backoff)
	require.EqualValues(t, 1, presetState.Backoff.NumFailed)
	require.EqualValues(t, backoffInterval*time.Duration(presetState.Backoff.NumFailed), clock.Until(actions.BackoffUntil).Truncate(backoffInterval))

	// When: advancing beyond the backoff time.

	clock.Advance(clock.Until(actions.BackoffUntil.Add(time.Second)))

	// Then: we will attempt to create a new prebuild.
	state, err = reconciler.SnapshotState(ctx, db)
	require.NoError(t, err)
	presetState, err = state.FilterByPreset(preset.ID)
	require.NoError(t, err)
	actions, err = reconciler.DetermineActions(ctx, *presetState)
	require.NoError(t, err)
	require.EqualValues(t, 0, actions.Actual)
	require.EqualValues(t, 1, actions.Desired)
	require.EqualValues(t, 1, actions.Create)

	// When: a new prebuild is provisioned but it fails again.
	_ = setupTestWorkspaceBuild(t, ctx, clock, db, ps, database.WorkspaceTransitionStart, database.ProvisionerJobStatusFailed, orgID, preset, templateID, templateVersionID)

	// Then: the backoff time is roughly equal to two backoff intervals, since another build has failed.
	state, err = reconciler.SnapshotState(ctx, db)
	require.NoError(t, err)
	presetState, err = state.FilterByPreset(preset.ID)
	require.NoError(t, err)
	actions, err = reconciler.DetermineActions(ctx, *presetState)
	require.NoError(t, err)
	require.EqualValues(t, 0, actions.Actual)
	require.EqualValues(t, 1, actions.Desired)
	require.EqualValues(t, 0, actions.Create)
	require.EqualValues(t, 2, presetState.Backoff.NumFailed)
	require.EqualValues(t, backoffInterval*time.Duration(presetState.Backoff.NumFailed), clock.Until(actions.BackoffUntil).Truncate(backoffInterval))
}

func setupTestDBTemplate(
	t *testing.T,
	db database.Store,
) (
	orgID uuid.UUID,
	userID uuid.UUID,
	templateID uuid.UUID,
) {
	t.Helper()
	org := dbgen.Organization(t, db, database.Organization{})
	user := dbgen.User(t, db, database.User{})

	template := dbgen.Template(t, db, database.Template{
		CreatedBy:      user.ID,
		OrganizationID: org.ID,
		CreatedAt:      time.Now().Add(muchEarlier),
	})

	return org.ID, user.ID, template.ID
}

const (
	earlier     = -time.Hour
	muchEarlier = -time.Hour * 2
)

func setupTestDBTemplateVersion(
	t *testing.T,
	ctx context.Context,
	clock quartz.Clock,
	db database.Store,
	pubsub pubsub.Pubsub,
	orgID uuid.UUID,
	userID uuid.UUID,
	templateID uuid.UUID,
) uuid.UUID {
	t.Helper()
	templateVersionJob := dbgen.ProvisionerJob(t, db, pubsub, database.ProvisionerJob{
		ID:             uuid.New(),
		CreatedAt:      clock.Now().Add(muchEarlier),
		CompletedAt:    sql.NullTime{Time: clock.Now().Add(earlier), Valid: true},
		OrganizationID: orgID,
		InitiatorID:    userID,
	})
	templateVersion := dbgen.TemplateVersion(t, db, database.TemplateVersion{
		TemplateID:     uuid.NullUUID{UUID: templateID, Valid: true},
		OrganizationID: orgID,
		CreatedBy:      userID,
		JobID:          templateVersionJob.ID,
		CreatedAt:      time.Now().Add(muchEarlier),
	})
	require.NoError(t, db.UpdateTemplateActiveVersionByID(ctx, database.UpdateTemplateActiveVersionByIDParams{
		ID:              templateID,
		ActiveVersionID: templateVersion.ID,
	}))
	return templateVersion.ID
}

func setupTestDBPrebuild(
	t *testing.T,
	ctx context.Context,
	clock quartz.Clock,
	db database.Store,
	pubsub pubsub.Pubsub,
	transition database.WorkspaceTransition,
	prebuildStatus database.ProvisionerJobStatus,
	orgID uuid.UUID,
	templateID uuid.UUID,
	templateVersionID uuid.UUID,
) (
	preset database.TemplateVersionPreset,
	prebuild database.WorkspaceTable,
) {
	t.Helper()
	preset, err := db.InsertPreset(ctx, database.InsertPresetParams{
		TemplateVersionID: templateVersionID,
		Name:              "test",
		CreatedAt:         clock.Now().Add(muchEarlier),
	})
	require.NoError(t, err)
	_, err = db.InsertPresetParameters(ctx, database.InsertPresetParametersParams{
		TemplateVersionPresetID: preset.ID,
		Names:                   []string{"test"},
		Values:                  []string{"test"},
	})
	require.NoError(t, err)
	_, err = db.InsertPresetPrebuild(ctx, database.InsertPresetPrebuildParams{
		ID:               uuid.New(),
		PresetID:         preset.ID,
		DesiredInstances: 1,
	})
	require.NoError(t, err)

	return preset, setupTestWorkspaceBuild(t, ctx, clock, db, pubsub, transition, prebuildStatus, orgID, preset, templateID, templateVersionID)
}

func setupTestWorkspaceBuild(
	t *testing.T,
	ctx context.Context,
	clock quartz.Clock,
	db database.Store,
	pubsub pubsub.Pubsub,
	transition database.WorkspaceTransition,
	prebuildStatus database.ProvisionerJobStatus,
	orgID uuid.UUID,
	preset database.TemplateVersionPreset,
	templateID uuid.UUID,
	templateVersionID uuid.UUID) (workspaceID database.WorkspaceTable) {

	cancelledAt := sql.NullTime{}
	completedAt := sql.NullTime{}

	startedAt := sql.NullTime{}
	if prebuildStatus != database.ProvisionerJobStatusPending {
		startedAt = sql.NullTime{Time: clock.Now().Add(muchEarlier), Valid: true}
	}

	buildError := sql.NullString{}
	if prebuildStatus == database.ProvisionerJobStatusFailed {
		completedAt = sql.NullTime{Time: clock.Now().Add(earlier), Valid: true}
		buildError = sql.NullString{String: "build failed", Valid: true}
	}

	switch prebuildStatus {
	case database.ProvisionerJobStatusCanceling:
		cancelledAt = sql.NullTime{Time: clock.Now().Add(earlier), Valid: true}
	case database.ProvisionerJobStatusCanceled:
		completedAt = sql.NullTime{Time: clock.Now().Add(earlier), Valid: true}
		cancelledAt = sql.NullTime{Time: clock.Now().Add(earlier), Valid: true}
	case database.ProvisionerJobStatusSucceeded:
		completedAt = sql.NullTime{Time: clock.Now().Add(earlier), Valid: true}
	default:
	}

	workspace := dbgen.Workspace(t, db, database.WorkspaceTable{
		TemplateID:     templateID,
		OrganizationID: orgID,
		OwnerID:        prebuilds.OwnerID,
		Deleted:        false,
	})
	job := dbgen.ProvisionerJob(t, db, pubsub, database.ProvisionerJob{
		InitiatorID:    prebuilds.OwnerID,
		CreatedAt:      clock.Now().Add(muchEarlier),
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		CanceledAt:     cancelledAt,
		OrganizationID: orgID,
		Error:          buildError,
	})
	dbgen.WorkspaceBuild(t, db, database.WorkspaceBuild{
		WorkspaceID:             workspace.ID,
		InitiatorID:             prebuilds.OwnerID,
		TemplateVersionID:       templateVersionID,
		JobID:                   job.ID,
		TemplateVersionPresetID: uuid.NullUUID{UUID: preset.ID, Valid: true},
		Transition:              transition,
		CreatedAt:               clock.Now(),
	})

	return workspace
}

// TODO (sasswart): test mutual exclusion
