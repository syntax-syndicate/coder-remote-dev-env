package prebuilds_test

import (
	"context"
	"database/sql"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/serpent"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/rbac"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/coderd/coderdenttest"
	"github.com/coder/coder/v2/enterprise/coderd/license"
	"github.com/coder/coder/v2/enterprise/coderd/prebuilds"
	"github.com/coder/coder/v2/provisioner/echo"
	"github.com/coder/coder/v2/provisionersdk/proto"
	"github.com/coder/coder/v2/testutil"
)

type storeSpy struct {
	database.Store

	claims           *atomic.Int32
	claimParams      *atomic.Pointer[database.ClaimPrebuildParams]
	claimedWorkspace *atomic.Pointer[database.ClaimPrebuildRow]
}

func newStoreSpy(db database.Store) *storeSpy {
	return &storeSpy{
		Store:            db,
		claims:           &atomic.Int32{},
		claimParams:      &atomic.Pointer[database.ClaimPrebuildParams]{},
		claimedWorkspace: &atomic.Pointer[database.ClaimPrebuildRow]{},
	}
}

func (m *storeSpy) InTx(fn func(store database.Store) error, opts *database.TxOptions) error {
	// Pass spy down into transaction store.
	return m.Store.InTx(func(store database.Store) error {
		spy := newStoreSpy(store)
		spy.claims = m.claims
		spy.claimParams = m.claimParams
		spy.claimedWorkspace = m.claimedWorkspace

		return fn(spy)
	}, opts)
}

func (m *storeSpy) ClaimPrebuild(ctx context.Context, arg database.ClaimPrebuildParams) (database.ClaimPrebuildRow, error) {
	m.claims.Add(1)
	m.claimParams.Store(&arg)
	result, err := m.Store.ClaimPrebuild(ctx, arg)
	m.claimedWorkspace.Store(&result)
	return result, err
}

func TestClaimPrebuild(t *testing.T) {
	t.Parallel()

	// Setup. // TODO: abstract?

	ctx := testutil.Context(t, testutil.WaitSuperLong)
	db, pubsub := dbtestutil.NewDB(t)
	spy := newStoreSpy(db)

	client, _, _, owner := coderdenttest.NewWithAPI(t, &coderdenttest.Options{
		Options: &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			Database:                 spy,
			Pubsub:                   pubsub,
		},

		LicenseOptions: &coderdenttest.LicenseOptions{
			Features: license.Features{
				codersdk.FeatureWorkspacePrebuilds: 1,
			},
		},
	})

	controller := prebuilds.NewStoreReconciler(spy, pubsub, codersdk.PrebuildsConfig{}, testutil.Logger(t))

	const (
		desiredInstances = 1
		presetCount      = 2
	)

	version := coderdtest.CreateTemplateVersion(t, client, owner.OrganizationID, templateWithAgentAndPresetsWithPrebuilds(desiredInstances))
	_ = coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	coderdtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
	presets, err := client.TemplateVersionPresets(ctx, version.ID)
	require.NoError(t, err)
	require.Len(t, presets, presetCount)

	userClient, user := coderdtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleMember())

	ctx = dbauthz.AsSystemRestricted(ctx)

	// Given: a reconciliation completes.
	require.NoError(t, controller.ReconcileAll(ctx))

	// Given: a set of running, eligible prebuilds eventually starts up.
	runningPrebuilds := make(map[uuid.UUID]database.GetRunningPrebuildsRow, desiredInstances*presetCount)
	require.Eventually(t, func() bool {
		rows, err := spy.GetRunningPrebuilds(ctx)
		require.NoError(t, err)

		for _, row := range rows {
			runningPrebuilds[row.CurrentPresetID.UUID] = row

			agents, err := db.GetWorkspaceAgentsInLatestBuildByWorkspaceID(ctx, row.WorkspaceID)
			require.NoError(t, err)

			for _, agent := range agents {
				require.NoError(t, db.UpdateWorkspaceAgentLifecycleStateByID(ctx, database.UpdateWorkspaceAgentLifecycleStateByIDParams{
					ID:             agent.ID,
					LifecycleState: database.WorkspaceAgentLifecycleStateReady,
					StartedAt:      sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true},
					ReadyAt:        sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true},
				}))
			}
		}

		t.Logf("found %d running prebuilds so far, want %d", len(runningPrebuilds), desiredInstances*presetCount)

		return len(runningPrebuilds) == (desiredInstances * presetCount)
	}, testutil.WaitSuperLong, testutil.IntervalSlow)

	// When: a user creates a new workspace with a preset for which prebuilds are configured.
	workspaceName := strings.ReplaceAll(testutil.GetRandomName(t), "_", "-")
	params := database.ClaimPrebuildParams{
		NewUserID: user.ID,
		NewName:   workspaceName,
		PresetID:  presets[0].ID,
	}
	userWorkspace, err := userClient.CreateUserWorkspace(ctx, user.Username, codersdk.CreateWorkspaceRequest{
		TemplateVersionID:        version.ID,
		Name:                     workspaceName,
		TemplateVersionPresetID:  presets[0].ID,
		ClaimPrebuildIfAvailable: true, // TODO: doesn't do anything yet; it probably should though.
	})
	require.NoError(t, err)
	coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, userWorkspace.LatestBuild.ID)

	// TODO: this feels... wrong; we should probably be injecting an implementation of prebuilds.Claimer.
	// Then: a prebuild should have been claimed.
	require.EqualValues(t, spy.claims.Load(), 1)
	require.NotNil(t, spy.claims.Load())
	require.EqualValues(t, *spy.claimParams.Load(), params)
	require.NotNil(t, spy.claimedWorkspace.Load())
	claimed := *spy.claimedWorkspace.Load()
	require.NotEqual(t, claimed, uuid.Nil)

	// Then: the claimed prebuild must now be owned by the requester.
	workspace, err := spy.GetWorkspaceByID(ctx, claimed.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, workspace.OwnerID)

	// Then: the number of running prebuilds has changed since one was claimed.
	currentPrebuilds, err := spy.GetRunningPrebuilds(ctx)
	require.NoError(t, err)
	require.NotEqual(t, len(currentPrebuilds), len(runningPrebuilds))

	// Then: the claimed prebuild is now missing from the running prebuilds set.
	current, err := spy.GetRunningPrebuilds(ctx)
	require.NoError(t, err)

	var found bool
	for _, prebuild := range current {
		if prebuild.WorkspaceID == claimed.ID {
			found = true
			break
		}
	}
	require.False(t, found, "claimed prebuild should not still be considered a running prebuild")
}

func templateWithAgentAndPresetsWithPrebuilds(desiredInstances int32) *echo.Responses {
	return &echo.Responses{
		Parse: echo.ParseComplete,
		ProvisionPlan: []*proto.Response{
			{
				Type: &proto.Response_Plan{
					Plan: &proto.PlanComplete{
						Resources: []*proto.Resource{
							{
								Type: "compute",
								Name: "main",
								Agents: []*proto.Agent{
									{
										Name:            "smith",
										OperatingSystem: "linux",
										Architecture:    "i386",
									},
								},
							},
						},
						Presets: []*proto.Preset{
							{
								Name: "preset-a",
								Parameters: []*proto.PresetParameter{
									{
										Name:  "k1",
										Value: "v1",
									},
								},
								Prebuild: &proto.Prebuild{
									Instances: desiredInstances,
								},
							},
							{
								Name: "preset-b",
								Parameters: []*proto.PresetParameter{
									{
										Name:  "k1",
										Value: "v2",
									},
								},
								Prebuild: &proto.Prebuild{
									Instances: desiredInstances,
								},
							},
						},
					},
				},
			},
		},
		ProvisionApply: []*proto.Response{
			{
				Type: &proto.Response_Apply{
					Apply: &proto.ApplyComplete{
						Resources: []*proto.Resource{
							{
								Type: "compute",
								Name: "main",
								Agents: []*proto.Agent{
									{
										Name:            "smith",
										OperatingSystem: "linux",
										Architecture:    "i386",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// TODO(dannyk): test claiming a prebuild causes a replacement to be provisioned.
// TODO(dannyk): test that prebuilds are only attempted to be claimed for net-new workspace builds
