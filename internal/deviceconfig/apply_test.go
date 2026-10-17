package deviceconfig_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

const ncRunning = "<system><host-name>edge</host-name></system>\n"

// plannedPush is a fixture holding a ready NETCONF plan on dct-1.
func plannedPush(t *testing.T) (pushFixture, deviceconfig.ApplyRequest) {
	t.Helper()
	f := newPushFixture(t, ncRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformNETCONF))
	f.device.res = deviceconfig.DryRunResult{Diff: "-edge\n+core\n", Added: 1, Removed: 1}
	res, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge,
		"<system><host-name>core</host-name></system>"))
	require.NoError(t, err)
	require.Equal(t, deviceconfig.PlanReady, res.Outcome)
	return f, deviceconfig.ApplyRequest{
		ClientID: client, PlanID: res.PlanID, DiffSHA256: f.plans.plans[0].DiffSHA256,
		User: "admin", Remote: "192.0.2.10", UserAgent: "test",
	}
}

func (f pushFixture) stepsSincePlan() []string {
	f.device.mu.Lock()
	defer f.device.mu.Unlock()
	return append([]string(nil), f.device.steps[1:]...)
}

// TestApplyConfirmsAChangeThePostCheckFinds is the slice's acceptance at the
// unit level: apply re-reads the running configuration, commits under a
// device-side revert, proves the change from a new login, confirms it with
// the same commit id, and backs the result up.
func TestApplyConfirmsAChangeThePostCheckFinds(t *testing.T) {
	t.Parallel()
	f, req := plannedPush(t)
	f.fetch.answers["10.0.0.1"] = answer(ncRunning)

	res, err := f.pushes.Apply(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.ApplyConfirmed, res.Outcome, res.Reason)
	require.Equal(t, []string{"commit", "dry-run", "confirm"}, f.stepsSincePlan())

	commit := f.device.requests[1]
	require.Equal(t, "pusher", commit.User, "apply logs in with the push credential")
	require.Equal(t, "<system><host-name>core</host-name></system>", commit.Candidate)
	require.True(t, strings.HasPrefix(commit.CommitID, "seed-"), commit.CommitID)
	require.Equal(t, commit.CommitID, f.device.requests[3].CommitID, "the confirm names the commit")
	require.NotEmpty(t, res.BackupID)

	stored := f.plans.plans[0]
	require.NotNil(t, stored.AppliedAt)
	require.Equal(t, "admin", stored.AppliedBy)
	require.Equal(t, deviceconfig.ApplyConfirmed, stored.ApplyOutcome)
	require.Equal(t, []string{
		"device_config.push.plan_created", "device_config.push.apply_started", "device_config.push.confirmed",
	}, f.audit.actions())
	for _, e := range f.audit.events {
		for _, v := range e.Detail {
			require.NotContains(t, fmt.Sprint(v), "core", "configuration text never enters the audit log")
		}
	}

	_, err = f.pushes.Apply(context.Background(), req)
	require.ErrorIs(t, err, deviceconfig.ErrPlanSpent, "a plan is applied at most once")
}

// TestApplyAnIOSMerge: an IOS plan, predicted from the backup, applies
// through the same commit, post-check and confirm as a device-diffed one.
func TestApplyAnIOSMerge(t *testing.T) {
	t.Parallel()
	f := newPushFixture(t, iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
	plan, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge,
		"interface Gi1/0/1\n description uplink to core\n"))
	require.NoError(t, err)
	require.Equal(t, deviceconfig.PlanReady, plan.Outcome)

	res, err := f.pushes.Apply(context.Background(), deviceconfig.ApplyRequest{
		ClientID: client, PlanID: plan.PlanID, DiffSHA256: f.plans.plans[0].DiffSHA256, User: "admin",
	})
	require.NoError(t, err)
	require.Equal(t, deviceconfig.ApplyConfirmed, res.Outcome, res.Reason)
	require.Equal(t, []string{"commit", "dry-run", "confirm"}, f.device.steps)
	require.Equal(t, deviceconfig.PlatformCiscoIOS, f.device.requests[0].Platform)
}

func TestApplyOutcomes(t *testing.T) {
	t.Parallel()
	unreachable := fmt.Errorf("%w: connection reset", deviceconfig.ErrUnreachable)
	tests := []struct {
		name    string
		arrange func(f pushFixture)
		outcome deviceconfig.ApplyOutcome
		steps   []string
		reason  string
	}{
		{
			name: "the configuration changed after the plan",
			arrange: func(f pushFixture) {
				f.fetch.answers["10.0.0.1"] = answer("<system><host-name>other</host-name></system>\n")
			},
			outcome: deviceconfig.ApplyDrifted,
			reason:  "plan again",
		},
		{
			name:    "the device refuses the commit",
			arrange: func(f pushFixture) { f.device.commitErr = fmt.Errorf("%w: commit: in-use", deviceconfig.ErrCommand) },
			outcome: deviceconfig.ApplyFailed,
			steps:   []string{"commit"},
			reason:  "reverts by itself within 5m0s",
		},
		{
			name: "the change is not on the device afterwards",
			arrange: func(f pushFixture) {
				f.device.afterRes = deviceconfig.DryRunResult{Diff: "-edge\n+core\n", Added: 1, Removed: 1}
			},
			outcome: deviceconfig.ApplyRolledBack,
			steps:   []string{"commit", "dry-run", "rollback"},
			reason:  "2 lines still to change",
		},
		{
			name: "the push cut Seed off",
			arrange: func(f pushFixture) {
				f.device.afterErr, f.device.rollbackErr = unreachable, unreachable
			},
			outcome: deviceconfig.ApplyReverting,
			steps:   []string{"commit", "dry-run", "rollback"},
			reason:  "the device reverts by itself",
		},
		{
			name: "the startup configuration is not saved",
			arrange: func(f pushFixture) {
				f.device.confirmErr = fmt.Errorf("%w: write memory: %% flash full", deviceconfig.ErrStartupNotSaved)
			},
			outcome: deviceconfig.ApplyConfirmed,
			steps:   []string{"commit", "dry-run", "confirm"},
			reason:  "a reload loses the change until it is saved",
		},
		{
			name:    "the confirm fails",
			arrange: func(f pushFixture) { f.device.confirmErr = unreachable },
			outcome: deviceconfig.ApplyReverting,
			steps:   []string{"commit", "dry-run", "confirm"},
			reason:  "the confirm failed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, req := plannedPush(t)
			tc.arrange(f)
			res, err := f.pushes.Apply(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, tc.outcome, res.Outcome, res.Reason)
			require.Contains(t, res.Reason, tc.reason)
			require.Equal(t, tc.steps, f.stepsSincePlan())
			require.Equal(t, tc.outcome, f.plans.plans[0].ApplyOutcome)
			require.Equal(t, "device_config.push."+string(tc.outcome), f.audit.actions()[2])
		})
	}
}

func TestApplyRefusesBeforeALogin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		arrange func(f pushFixture, req *deviceconfig.ApplyRequest)
		check   func(t *testing.T, err error)
	}{
		{
			name:    "a hash that is not the plan's",
			arrange: func(_ pushFixture, req *deviceconfig.ApplyRequest) { req.DiffSHA256 = strings.Repeat("0", 64) },
			check:   validation("not the hash of this plan's diff"),
		},
		{
			name:    "an unknown plan",
			arrange: func(_ pushFixture, req *deviceconfig.ApplyRequest) { req.PlanID = "dcp-000000000099" },
			check:   func(t *testing.T, err error) { t.Helper(); require.ErrorIs(t, err, deviceconfig.ErrNotFound) },
		},
		{
			name: "an expired plan",
			arrange: func(f pushFixture, _ *deviceconfig.ApplyRequest) {
				f.plans.plans[0].ExpiresAt = time.Now().Add(-time.Second)
			},
			check: func(t *testing.T, err error) { t.Helper(); require.ErrorIs(t, err, deviceconfig.ErrPlanSpent) },
		},
		{
			name: "pushing turned off since the plan",
			arrange: func(f pushFixture, _ *deviceconfig.ApplyRequest) {
				tg := f.repo.targets["dct-1"]
				tg.PushCredentialsID = ""
				f.repo.targets["dct-1"] = tg
			},
			check: validation("not enabled"),
		},
		{
			name: "the target turned read-only since the plan",
			arrange: func(f pushFixture, _ *deviceconfig.ApplyRequest) {
				tg := f.repo.targets["dct-1"]
				tg.Platform = deviceconfig.PlatformCiscoNXOS
				f.repo.targets["dct-1"] = tg
			},
			check: validation("read-only"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, req := plannedPush(t)
			tc.arrange(f, &req)
			require.Error(t, f.pushes.CheckApply(context.Background(), req))
			_, err := f.pushes.Apply(context.Background(), req)
			tc.check(t, err)
			require.Empty(t, f.stepsSincePlan(), "nothing is sent to the device")
			require.Nil(t, f.plans.plans[0].AppliedAt, "a refused apply leaves the plan applicable")
		})
	}
}

func validation(msg string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		var ve deviceconfig.ValidationError
		require.ErrorAs(t, err, &ve)
		require.Contains(t, ve.Msg, msg)
	}
}
