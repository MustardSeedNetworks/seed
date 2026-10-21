package deviceconfig_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

// reloadFixture is stageFixture after staging: the IOS target boots the
// staged image next and still runs 17.09. The wait for the reloaded device
// is shortened to milliseconds.
func reloadFixture(t *testing.T) (pushFixture, deviceconfig.ReloadRequest) {
	t.Helper()
	f, stage := stageFixture(t)
	f.device.facts.BootImage = "flash:" + stagedImage
	f.pushes.SetReloadPace(200*time.Millisecond, time.Millisecond)
	return f, deviceconfig.ReloadRequest{
		ClientID: client, ImageID: stage.ImageID, TargetID: "dct-1", User: "admin", Remote: "192.0.2.10",
		UserAgent: "test",
	}
}

// onStaged is the IOS target after it booted the staged image.
func onStaged() factsAnswer {
	facts := iosFacts()
	facts.Version, facts.RunningImage, facts.BootImage = "17.12.04", "flash:"+stagedImage, "flash:"+stagedImage
	return factsAnswer{facts: facts}
}

func (f pushFixture) reloadEvents() []string {
	var out []string
	for _, e := range f.stageEvents() {
		if e != "stage_started" {
			out = append(out, e)
		}
	}
	return out
}

// TestReloadVerifiesTheStagedImage is the slice's acceptance at the unit
// level: the device is checked to boot the staged image next, told to
// reload, seen down, and seen back running the staged image, each step
// audited with the versions before and after.
func TestReloadVerifiesTheStagedImage(t *testing.T) {
	t.Parallel()
	f, req := reloadFixture(t)
	down := factsAnswer{err: deviceconfig.ErrUnreachable}
	f.device.reloaded = []factsAnswer{down, down, onStaged()}

	res, err := f.pushes.Reload(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.ReloadResult{
		Outcome: deviceconfig.ReloadVerified, FromVersion: "17.09.04a", Version: "17.12.04",
		RunningImage: "flash:" + stagedImage,
	}, res)
	require.Equal(t, []string{"facts", "reload", "facts", "facts", "facts"}, f.device.steps)
	for _, r := range f.device.requests {
		require.Equal(t, "pusher", r.User, "a reload logs in with the push credential")
	}
	require.Equal(t, []string{"reload_requested", "reload_sent", "reload_verified"}, f.reloadEvents())
	last := f.audit.events[len(f.audit.events)-1]
	require.Equal(t, req.ImageID, last.ResourceID)
	require.Equal(t, "dct-1", last.Detail["target_id"])
	require.Equal(t, "17.09.04a", last.Detail["from_version"])
	require.Equal(t, "17.12.04", last.Detail["version"])
}

// TestReloadOutcomes drives a reload through each way it can end other than
// verified.
func TestReloadOutcomes(t *testing.T) {
	t.Parallel()
	down := factsAnswer{err: deviceconfig.ErrUnreachable}
	old := factsAnswer{facts: iosFacts()}
	tests := []struct {
		name     string
		setup    func(*memDevice)
		outcome  deviceconfig.ReloadOutcome
		reason   string
		reloaded bool
	}{
		{
			name:    "not staged",
			setup:   func(d *memDevice) { d.facts.BootImage = iosFacts().RunningImage },
			outcome: deviceconfig.ReloadRefused,
			reason: "the device boots flash:cat9k_iosxe.17.09.04a.SPA.bin next, not " + stagedImage +
				"; stage the image first",
		},
		{
			name:    "already running",
			setup:   func(d *memDevice) { d.facts.RunningImage = "flash:" + stagedImage },
			outcome: deviceconfig.ReloadRefused, reason: "the device already runs " + stagedImage,
		},
		{
			name:    "device unreadable",
			setup:   func(d *memDevice) { d.factsErr = deviceconfig.ErrUnreachable },
			outcome: deviceconfig.ReloadFailed, reason: "read the device's version and boot image",
		},
		{
			name:     "unsaved changes",
			setup:    func(d *memDevice) { d.reloadErr = deviceconfig.ErrUnsavedChanges },
			outcome:  deviceconfig.ReloadRefused,
			reason:   "unsaved changes; save or discard them on the device, then reload",
			reloaded: true,
		},
		{
			name:    "reload rejected",
			setup:   func(d *memDevice) { d.reloadErr = deviceconfig.ErrCommand },
			outcome: deviceconfig.ReloadFailed, reason: "reload the device",
			reloaded: true,
		},
		{
			name:     "fallback booted",
			setup:    func(d *memDevice) { d.reloaded = []factsAnswer{down, old} },
			outcome:  deviceconfig.ReloadWrongImage,
			reason:   "the device came back on 17.09.04a, not " + stagedImage,
			reloaded: true,
		},
		{
			name:    "never back",
			setup:   func(d *memDevice) { d.reloaded = []factsAnswer{down} },
			outcome: deviceconfig.ReloadUnreachable, reason: "the device did not answer within 200ms",
			reloaded: true,
		},
		{
			name:    "never down",
			setup:   func(d *memDevice) { d.reloaded = []factsAnswer{old} },
			outcome: deviceconfig.ReloadFailed, reason: "the device took the reload but did not restart",
			reloaded: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, req := reloadFixture(t)
			tt.setup(f.device)

			res, err := f.pushes.Reload(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, tt.outcome, res.Outcome)
			require.Contains(t, res.Reason, tt.reason)
			require.Equal(t, tt.reloaded, len(f.device.steps) > 1 && f.device.steps[1] == "reload")
			events := f.reloadEvents()
			require.Equal(t, "reload_"+string(tt.outcome), events[len(events)-1])
		})
	}
}

// TestReloadJunosAndEOS: Junos is verified by the release its package is
// named for and EOS by a new version on the boot image, since neither names
// a running image file.
func TestReloadJunosAndEOS(t *testing.T) {
	t.Parallel()

	t.Run("junos", func(t *testing.T) {
		t.Parallel()
		f, stage := junosStageFixture(t, junosPackage)
		f.pushes.SetReloadPace(time.Second, time.Millisecond)
		f.device.reloaded = []factsAnswer{
			{err: deviceconfig.ErrUnreachable},
			{facts: deviceconfig.ImageFacts{Version: "23.4R2.13", Dir: "/var/tmp/", FreeBytes: 1 << 30}},
		}
		res, err := f.pushes.Reload(context.Background(), deviceconfig.ReloadRequest{
			ClientID: client, ImageID: stage.ImageID, TargetID: "dct-1", User: "admin",
		})
		require.NoError(t, err)
		require.Equal(t, deviceconfig.ReloadVerified, res.Outcome, res.Reason)
		require.Equal(t, "23.4R2.13", res.Version)
	})

	t.Run("junos already on the release", func(t *testing.T) {
		t.Parallel()
		f, stage := junosStageFixture(t, "junos-install-ex-x86-64-21.4R3-S5.4.tgz")
		res, err := f.pushes.Reload(context.Background(), deviceconfig.ReloadRequest{
			ClientID: client, ImageID: stage.ImageID, TargetID: "dct-1", User: "admin",
		})
		require.NoError(t, err)
		require.Equal(t, deviceconfig.ReloadRefused, res.Outcome)
		require.Equal(t, "the device already runs Junos 21.4R3-S5.4", res.Reason)
	})

	t.Run("eos", func(t *testing.T) {
		t.Parallel()
		const name = "EOS-4.32.1F.swi"
		f := newPushFixture(t, iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformAristaEOS))
		f.pushes.SetReloadPace(time.Second, time.Millisecond)
		img, err := f.images.Upload(context.Background(), upload(name, "arista-eos", sha512Hex(imageBytes), imageBytes))
		require.NoError(t, err)
		staged := deviceconfig.ImageFacts{
			Version: "4.31.2F", RunningImage: "flash:/" + name, BootImage: "flash:/" + name, Dir: "flash:",
		}
		f.device.facts = staged
		after := staged
		after.Version = "4.32.1F"
		f.device.reloaded = []factsAnswer{{err: deviceconfig.ErrUnreachable}, {facts: after}}

		res, err := f.pushes.Reload(context.Background(), deviceconfig.ReloadRequest{
			ClientID: client, ImageID: img.ID, TargetID: "dct-1", User: "admin",
		})
		require.NoError(t, err)
		require.Equal(t, deviceconfig.ReloadVerified, res.Outcome, res.Reason)
		require.Equal(t, "4.31.2F", res.FromVersion)
		require.Equal(t, "4.32.1F", res.Version)
	})
}

// TestReloadOneJobPerTarget: a reload holds the target for the whole wait,
// so no push or staging runs on a device while it is rebooting.
func TestReloadOneJobPerTarget(t *testing.T) {
	t.Parallel()
	f, req := reloadFixture(t)
	f.pushes.SetReloadPace(time.Minute, 10*time.Millisecond)
	f.device.reloaded = []factsAnswer{{err: deviceconfig.ErrUnreachable}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan deviceconfig.ReloadResult, 1)
	go func() {
		res, _ := f.pushes.Reload(ctx, req)
		done <- res
	}()
	require.Eventually(t, func() bool {
		f.device.mu.Lock()
		defer f.device.mu.Unlock()
		return len(f.device.steps) >= 3
	}, 5*time.Second, 10*time.Millisecond)

	_, err := f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "hostname core"))
	require.ErrorIs(t, err, deviceconfig.ErrPushBusy)
	cancel()
	res := <-done
	require.Equal(t, deviceconfig.ReloadFailed, res.Outcome)
	require.Contains(t, res.Reason, "cancelled after it was told to reload")
	events := f.reloadEvents()
	require.Equal(t, "reload_failed", events[len(events)-1], "the outcome is audited after the cancel")
}

// TestReloadRefusesBeforeTheDevice: the reload checks its request as staging
// does, before any login.
func TestReloadRefusesBeforeTheDevice(t *testing.T) {
	t.Parallel()
	f, req := reloadFixture(t)
	req.ImageID = "dim-000000000099"
	require.ErrorIs(t, f.pushes.CheckReload(context.Background(), req), deviceconfig.ErrNotFound)
	_, err := f.pushes.Reload(context.Background(), req)
	require.ErrorIs(t, err, deviceconfig.ErrNotFound)
	require.Empty(t, f.device.steps)
}
