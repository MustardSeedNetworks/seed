package deviceconfig_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

const stagedImage = "cat9k_iosxe.17.12.04.SPA.bin"

// iosFacts is a Catalyst running 17.09 from flash with room for the image.
func iosFacts() deviceconfig.ImageFacts {
	return deviceconfig.ImageFacts{
		Version: "17.09.04a", RunningImage: "flash:cat9k_iosxe.17.09.04a.SPA.bin", Dir: "flash:", FreeBytes: 1 << 30,
	}
}

// stageFixture holds an IOS push target dct-1 and an uploaded IOS image.
func stageFixture(t *testing.T) (pushFixture, deviceconfig.StageRequest) {
	t.Helper()
	f := newPushFixture(t, iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS))
	f.device.facts = iosFacts()
	img, err := f.images.Upload(context.Background(),
		upload(stagedImage, "cisco-ios", sha512Hex(imageBytes), imageBytes))
	require.NoError(t, err)
	return f, deviceconfig.StageRequest{
		ClientID: client, ImageID: img.ID, TargetID: "dct-1", User: "admin", Remote: "192.0.2.10", UserAgent: "test",
	}
}

func (f pushFixture) stageEvents() []string {
	var out []string
	for _, a := range f.audit.actions() {
		if a != "device_config.image.uploaded" {
			out = append(out, strings.TrimPrefix(a, "device_config.image."))
		}
	}
	return out
}

// TestStageSetsTheBootImage is the slice's acceptance at the unit level: the
// device is checked first, the image's bytes are copied under its name next
// to the running image, the device verifies the copy with SHA-512, and the
// copy becomes the boot image with the running image as the fallback, each
// step audited.
func TestStageSetsTheBootImage(t *testing.T) {
	t.Parallel()
	f, req := stageFixture(t)

	res, err := f.pushes.Stage(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StageResult{
		Outcome: deviceconfig.StageBootSet, RunningVersion: "17.09.04a",
		RunningImage: iosFacts().RunningImage, BootImage: "flash:" + stagedImage,
	}, res)
	require.Equal(t, []string{
		"facts",
		"copy flash:" + stagedImage,
		"verify " + sha512Hex(imageBytes)[:8],
		"boot flash:" + stagedImage + " then " + iosFacts().RunningImage,
	}, f.device.steps)
	require.Equal(t, imageBytes, f.device.copied)
	for _, r := range f.device.requests {
		require.Equal(t, "pusher", r.User, "staging logs in with the push credential")
	}
	require.Equal(t, []string{"stage_started", "staged", "verified", "boot_set"}, f.stageEvents())
	for _, e := range f.audit.events[1:] {
		require.Equal(t, req.ImageID, e.ResourceID)
		require.Equal(t, "dct-1", e.Detail["target_id"])
	}
}

// TestStageOutcomes drives staging through each way it can stop: a
// pre-check that refuses before anything is copied, and a failed step.
func TestStageOutcomes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(*memDevice)
		outcome deviceconfig.StageOutcome
		reason  string
		steps   int
	}{
		{
			name:    "install mode",
			setup:   func(d *memDevice) { d.facts.InstallMode, d.facts.RunningImage = true, "flash:packages.conf" },
			outcome: deviceconfig.StageRefused, reason: "install mode", steps: 1,
		},
		{
			name:    "already running",
			setup:   func(d *memDevice) { d.facts.RunningImage = "bootflash:" + stagedImage },
			outcome: deviceconfig.StageRefused, reason: "the device already runs " + stagedImage, steps: 1,
		},
		{
			name:    "flash too small",
			setup:   func(d *memDevice) { d.facts.FreeBytes = int64(len(imageBytes)) - 1 },
			outcome: deviceconfig.StageRefused, reason: "flash: has 17 bytes free and the image needs 18", steps: 1,
		},
		{
			name:    "device unreadable",
			setup:   func(d *memDevice) { d.factsErr = deviceconfig.ErrUnreachable },
			outcome: deviceconfig.StageFailed, reason: "read the device's version and flash", steps: 1,
		},
		{
			name:    "copy fails",
			setup:   func(d *memDevice) { d.copyErr = errors.New("scp: the device refused the copy") },
			outcome: deviceconfig.StageFailed, reason: "copy the image to flash:" + stagedImage, steps: 2,
		},
		{
			name:    "copy corrupt",
			setup:   func(d *memDevice) { d.verifyErr = deviceconfig.ErrImageCorrupt },
			outcome: deviceconfig.StageFailed, reason: "does not match the image, so the copy was deleted", steps: 3,
		},
		{
			name:    "boot not set",
			setup:   func(d *memDevice) { d.bootErr = deviceconfig.ErrCommand },
			outcome: deviceconfig.StageFailed, reason: "set the boot image", steps: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, req := stageFixture(t)
			tt.setup(f.device)

			res, err := f.pushes.Stage(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, tt.outcome, res.Outcome)
			require.Contains(t, res.Reason, tt.reason)
			require.Empty(t, res.BootImage)
			require.Len(t, f.device.steps, tt.steps)
			events := f.stageEvents()
			require.Equal(t, "stage_"+string(tt.outcome), events[len(events)-1])
		})
	}
}

// TestStageRefusesBeforeTheDevice: a request that can be refused without a
// login never reaches the device.
func TestStageRefusesBeforeTheDevice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		target deviceconfig.Target
		req    func(*deviceconfig.StageRequest)
		want   string
		is     error
	}{
		{
			name:   "unknown image",
			target: pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS),
			req:    func(r *deviceconfig.StageRequest) { r.ImageID = "dim-000000000099" },
			is:     deviceconfig.ErrNotFound,
		},
		{
			name:   "another client's image",
			target: pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS),
			req:    func(r *deviceconfig.StageRequest) { r.ClientID = "tenant-x" },
			is:     deviceconfig.ErrNotFound,
		},
		{
			name:   "another platform",
			target: pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformAristaEOS),
			want:   "the image is for cisco-ios and the target is arista-eos",
		},
		{
			name: "pushing off",
			target: func() deviceconfig.Target {
				tg := pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformCiscoIOS)
				tg.PushCredentialsID = ""
				return tg
			}(),
			want: "pushing is not enabled on this target",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newPushFixture(t, iosRunning, tt.target)
			img, err := f.images.Upload(context.Background(),
				upload(stagedImage, "cisco-ios", sha512Hex(imageBytes), imageBytes))
			require.NoError(t, err)
			req := deviceconfig.StageRequest{ClientID: client, ImageID: img.ID, TargetID: "dct-1"}
			if tt.req != nil {
				tt.req(&req)
			}

			err = f.pushes.CheckStage(context.Background(), req)
			if tt.is != nil {
				require.ErrorIs(t, err, tt.is)
			} else {
				var ve deviceconfig.ValidationError
				require.ErrorAs(t, err, &ve)
				require.Equal(t, tt.want, ve.Msg)
			}
			_, stageErr := f.pushes.Stage(context.Background(), req)
			require.Equal(t, err, stageErr, "the job checks again")
			require.Empty(t, f.device.steps)
		})
	}
}

// junosStageFixture holds a Junos push target dct-1 running 21.4R3-S5.4 and
// an uploaded Junos package of release name.
func junosStageFixture(t *testing.T, name string) (pushFixture, deviceconfig.StageRequest) {
	t.Helper()
	f := newPushFixture(t, iosRunning, pushTarget("dct-1", "10.0.0.1", deviceconfig.PlatformJuniperJunos))
	f.device.facts = deviceconfig.ImageFacts{Version: "21.4R3-S5.4", Dir: "/var/tmp/", FreeBytes: 1 << 30}
	img, err := f.images.Upload(context.Background(),
		upload(name, "juniper-junos", sha256Hex(imageBytes), imageBytes))
	require.NoError(t, err)
	return f, deviceconfig.StageRequest{ClientID: client, ImageID: img.ID, TargetID: "dct-1", User: "admin"}
}

// TestStageInstallsTheJunosPackage is slice 5b-3 at the unit level: the
// package goes to /var/tmp, the audit records the SHA-256 the device
// checks, and the install is the boot step, with no fallback image to name.
func TestStageInstallsTheJunosPackage(t *testing.T) {
	t.Parallel()
	f, req := junosStageFixture(t, junosPackage)

	res, err := f.pushes.Stage(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StageResult{
		Outcome: deviceconfig.StageBootSet, RunningVersion: "21.4R3-S5.4", BootImage: "/var/tmp/" + junosPackage,
	}, res)
	require.Equal(t, []string{
		"facts",
		"copy /var/tmp/" + junosPackage,
		"verify " + sha512Hex(imageBytes)[:8],
		"boot /var/tmp/" + junosPackage + " then ",
	}, f.device.steps)
	require.Equal(t, []string{"stage_started", "staged", "verified", "boot_set"}, f.stageEvents())
	verified := f.audit.events[3]
	require.Equal(t, sha256Hex(imageBytes), verified.Detail["sha256"])
	require.NotContains(t, verified.Detail, "sha512")
}

// TestStageRefusesTheRunningJunosRelease: a package named for the release
// the device runs is refused before anything is copied.
func TestStageRefusesTheRunningJunosRelease(t *testing.T) {
	t.Parallel()
	f, req := junosStageFixture(t, "junos-install-ex-x86-64-21.4R3-S5.4.tgz")

	res, err := f.pushes.Stage(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, deviceconfig.StageRefused, res.Outcome)
	require.Equal(t, "the device already runs Junos 21.4R3-S5.4", res.Reason)
	require.Equal(t, []string{"facts"}, f.device.steps)
}

// TestStageOneJobPerTarget: staging shares the push's one-job-per-target
// rule, so nothing else runs on a device while an image is copied to it.
func TestStageOneJobPerTarget(t *testing.T) {
	t.Parallel()
	f, req := stageFixture(t)
	f.device.gate = make(chan struct{})
	first := make(chan error, 1)
	go func() {
		_, err := f.pushes.Stage(context.Background(), req)
		first <- err
	}()
	require.Eventually(t, func() bool {
		f.device.mu.Lock()
		defer f.device.mu.Unlock()
		return len(f.device.steps) == 2
	}, 5*time.Second, 10*time.Millisecond)

	_, err := f.pushes.Stage(context.Background(), req)
	require.ErrorIs(t, err, deviceconfig.ErrPushBusy)
	_, err = f.pushes.Plan(context.Background(), planRequest("dct-1", deviceconfig.PushMerge, "hostname core"))
	require.ErrorIs(t, err, deviceconfig.ErrPushBusy)
	close(f.device.gate)
	require.NoError(t, <-first)
}
