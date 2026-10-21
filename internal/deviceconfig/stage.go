package deviceconfig

// Operating system image staging (P-D3, #3499, ADR-0035 §3). Staging checks
// the device first, copies an uploaded image to it, has the device verify its
// copy against the image's hash, and makes the copy the image the device
// boots next, keeping the running software as the fallback. It never reloads
// the device: reloading is its own confirmed action.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// StageOutcome is how a staging job ended.
type StageOutcome string

// Staging outcomes. Refused means a pre-check stopped staging before anything
// was copied.
const (
	StageBootSet StageOutcome = "boot_set"
	StageRefused StageOutcome = "refused"
	StageFailed  StageOutcome = "failed"
)

const (
	// minCopyRate is the slowest management link a copy is given time for,
	// 5 Mbit/s, the rate an upload is given time for.
	minCopyRate = 625_000
	// minHashRate is the slowest flash a device is given time to hash its
	// copy at.
	minHashRate = 2 << 20
	// junosInstallTimeout is how long a Junos "request system software add"
	// is given to validate and unpack a package, which takes minutes.
	junosInstallTimeout = 30 * time.Minute
)

// ErrImageCorrupt is a copy whose hash on the device is not the image's.
var ErrImageCorrupt = errors.New("the device's hash of its copy does not match the image")

// ImageFacts is what staging checks on the device before it copies anything.
type ImageFacts struct {
	Version string
	// RunningImage is the image the device runs, as the device names it
	// ("flash:c2960x-universalk9-mz.152-7.E4.bin"). Junos runs installed
	// packages rather than an image file, so it is empty there.
	RunningImage string
	// Dir is the file system the running image is on, where the new image is
	// staged ("flash:"; "/var/tmp/" on Junos), and FreeBytes its free space.
	Dir       string
	FreeBytes int64
	// InstallMode is an IOS-XE device booted from packages.conf, which takes
	// a new image with "install add" rather than a boot statement.
	InstallMode bool
	// BootImage is the image the device boots next: IOS's first boot
	// statement, EOS's boot configuration. Junos names none.
	BootImage string
}

// DeviceImage is an image as it is staged on one device.
type DeviceImage struct {
	Dir    string
	Name   string
	Size   int64
	SHA256 string
	SHA512 string
	// Fallback is the image the device runs now, kept to boot from if the new
	// one does not come up. Junos has none: the install keeps the running
	// software for "request system software rollback".
	Fallback string
}

// Path is the image's file on the device ("flash:new.bin").
func (d DeviceImage) Path() string { return d.Dir + d.Name }

// StageRequest is an admin's request to stage an image on a target. User,
// Remote and UserAgent are for the audit log.
type StageRequest struct {
	ClientID  string
	ImageID   string
	TargetID  string
	User      string
	Remote    string
	UserAgent string
}

// StageResult is a staging job's outcome, persisted by the job store.
// RunningVersion is the version before staging, which the reload checks the
// device has left.
type StageResult struct {
	Outcome        StageOutcome `json:"outcome"`
	Reason         string       `json:"reason,omitempty"`
	RunningVersion string       `json:"runningVersion,omitempty"`
	RunningImage   string       `json:"runningImage,omitempty"`
	BootImage      string       `json:"bootImage,omitempty"`
}

// CheckStage refuses a staging request that can be refused without a login:
// an unknown image, a target that cannot be pushed to, and an image for
// another platform. The staging job checks again.
func (p *Pushes) CheckStage(ctx context.Context, req StageRequest) error {
	_, _, err := p.checkStage(ctx, req)
	return err
}

func (p *Pushes) checkStage(ctx context.Context, req StageRequest) (Image, Target, error) {
	img, err := p.images.store.GetImage(ctx, req.ClientID, req.ImageID)
	if err != nil {
		return Image{}, Target{}, err
	}
	t, err := p.pushTarget(ctx, req.ClientID, req.TargetID)
	if err != nil {
		return Image{}, Target{}, err
	}
	if t.Platform != img.Platform {
		return Image{}, Target{}, ValidationError{
			Msg: fmt.Sprintf("the image is for %s and the target is %s", img.Platform, t.Platform),
		}
	}
	return img, t, nil
}

// Stage stages an image on a target. The returned error is a request refused
// before the device was touched; every stage that claimed the target ends in
// an outcome, recorded in the audit log step by step.
func (p *Pushes) Stage(ctx context.Context, req StageRequest) (StageResult, error) {
	img, t, err := p.checkStage(ctx, req)
	if err != nil {
		return StageResult{}, err
	}
	if !p.claim(t.ID) {
		return StageResult{}, ErrPushBusy
	}
	defer p.release(t.ID)

	var auditErrs []error
	note := func(ctx context.Context, action string, detail map[string]any) {
		detail["target_id"] = t.ID
		auditErrs = append(auditErrs, p.audit(ctx, PushAuditEvent{
			Action: "device_config.image." + action, User: req.User, ResourceType: ImageAuditResource,
			ResourceID: img.ID, Remote: req.Remote, UserAgent: req.UserAgent, Detail: detail,
		}))
	}
	note(ctx, "stage_started", map[string]any{"name": img.Name, "size": img.Size})
	if err = errors.Join(auditErrs...); err != nil {
		return StageResult{}, fmt.Errorf("audit image staging: %w", err)
	}

	res := p.stage(ctx, t, img, note)
	// The outcome is recorded even when the job is being cancelled: the
	// device may already hold the copy.
	ctx = context.WithoutCancel(ctx)
	if res.Outcome != StageBootSet {
		note(ctx, "stage_"+string(res.Outcome), map[string]any{
			"reason": res.Reason, "running_version": res.RunningVersion,
		})
	}
	if err = errors.Join(auditErrs...); err != nil {
		return res, fmt.Errorf("audit image staging of %s: %w", img.ID, err)
	}
	return res, nil
}

func (p *Pushes) stage(
	ctx context.Context, t Target, img Image, note func(context.Context, string, map[string]any),
) StageResult {
	req, err := p.deviceRequest(ctx, t, "", "")
	if err != nil {
		return failedStage(StageResult{}, "prepare the push login", err)
	}
	var facts ImageFacts
	err = onDeviceFor(ctx, deviceTimeout, func(ctx context.Context) error {
		var factsErr error
		facts, factsErr = p.device.ImageFacts(ctx, req)
		return factsErr
	})
	if err != nil {
		return failedStage(StageResult{}, "read the device's version and flash", err)
	}
	res := StageResult{RunningVersion: facts.Version, RunningImage: facts.RunningImage}
	if reason := refuseStage(facts, img); reason != "" {
		res.Outcome, res.Reason = StageRefused, reason
		return res
	}

	di := DeviceImage{
		Dir: facts.Dir, Name: img.Name, Size: img.Size, SHA256: img.SHA256, SHA512: img.SHA512,
		Fallback: facts.RunningImage,
	}
	_, f, err := p.images.open(ctx, img.ClientID, img.ID)
	if err != nil {
		return failedStage(res, "open the image", err)
	}
	err = onDeviceFor(ctx, deviceTimeout+rateTime(img.Size, minCopyRate), func(ctx context.Context) error {
		return p.device.CopyImage(ctx, req, di, f)
	})
	_ = f.Close()
	if err != nil {
		return failedStage(res, "copy the image to "+di.Path(), err)
	}
	note(ctx, "staged", map[string]any{"path": di.Path(), "size": img.Size})

	err = onDeviceFor(ctx, deviceTimeout+rateTime(img.Size, minHashRate), func(ctx context.Context) error {
		return p.device.VerifyImage(ctx, req, di)
	})
	if errors.Is(err, ErrImageCorrupt) {
		res.Outcome, res.Reason = StageFailed, fmt.Sprintf(
			"the device's hash of %s does not match the image, so the copy was deleted; stage it again",
			di.Path())
		return res
	}
	if err != nil {
		return failedStage(res, "verify the copy", err)
	}
	// The hash in the audit is the one the device computed.
	verified, bootTimeout := map[string]any{"path": di.Path(), "sha512": img.SHA512}, deviceTimeout
	if t.Platform == PlatformJuniperJunos {
		verified, bootTimeout = map[string]any{"path": di.Path(), "sha256": img.SHA256}, junosInstallTimeout
	}
	note(ctx, "verified", verified)

	err = onDeviceFor(ctx, bootTimeout, func(ctx context.Context) error { return p.device.SetBoot(ctx, req, di) })
	if err != nil {
		return failedStage(res, "set the boot image", err)
	}
	note(ctx, "boot_set", map[string]any{"boot_image": di.Path(), "fallback": di.Fallback})
	res.Outcome, res.BootImage = StageBootSet, di.Path()
	return res
}

// refuseStage is the pre-check: the reason staging img on a device with
// these facts would be wrong, or "".
func refuseStage(facts ImageFacts, img Image) string {
	switch {
	case facts.InstallMode:
		return "the device boots in install mode (packages.conf), which takes an image with \"install add\"; " +
			"stage it from the device's CLI"
	case fileName(facts.RunningImage) == img.Name:
		return "the device already runs " + img.Name
	// A Junos package is named for the release it installs
	// ("junos-install-ex-x86-64-21.4R3-S5.4.tgz").
	case img.Platform == PlatformJuniperJunos && strings.Contains(img.Name, facts.Version):
		return "the device already runs Junos " + facts.Version
	case facts.FreeBytes < img.Size:
		return fmt.Sprintf("%s has %d bytes free and the image needs %d; free space on the device first",
			facts.Dir, facts.FreeBytes, img.Size)
	}
	return ""
}

// fileName is the file part of a device path ("flash:/EOS.swi" is "EOS.swi").
func fileName(path string) string { return path[strings.LastIndexAny(path, ":/")+1:] }

// rateTime is how long size bytes take at rate bytes a second.
func rateTime(size, rate int64) time.Duration {
	return time.Duration(size/rate+1) * time.Second
}

// onDeviceFor runs one device step under timeout.
func onDeviceFor(ctx context.Context, timeout time.Duration, step func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return step(ctx)
}

func failedStage(res StageResult, step string, err error) StageResult {
	res.Outcome, res.Reason = StageFailed, step+": "+err.Error()
	return res
}
