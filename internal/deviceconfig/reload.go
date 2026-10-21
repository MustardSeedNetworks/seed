package deviceconfig

// Reloading a device onto a staged image (P-D3, #3499, ADR-0035 §3). A reload
// is its own admin action, never part of staging. It checks that the device
// boots the image next, tells it to reload, and then waits for it to come back
// and reads which software it came back on. Seed cannot undo a bad image from
// here: the fallback boot entry is the rollback.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReloadOutcome is how a reload job ended.
type ReloadOutcome string

// Reload outcomes. Refused means a check stopped the reload before the device
// was told to reload. WrongImage is a device that came back on other software
// than the staged image, its fallback most likely. Unreachable is a device
// that went down and did not answer again within the reload window.
const (
	ReloadVerified    ReloadOutcome = "verified"
	ReloadRefused     ReloadOutcome = "refused"
	ReloadFailed      ReloadOutcome = "failed"
	ReloadWrongImage  ReloadOutcome = "wrong_image"
	ReloadUnreachable ReloadOutcome = "unreachable"
)

const (
	// reloadWindow is how long a device is given to come back: a Junos
	// upgrade installs the new release during the boot, which takes tens of
	// minutes on a small switch.
	reloadWindow = 45 * time.Minute
	// reloadPoll is how often a reloading device is tried. It is well under
	// the time any switch or router takes to boot, so the device is seen
	// down before it is seen back.
	reloadPoll = 15 * time.Second
)

// ErrUnsavedChanges is a device whose running configuration has changes
// that are not saved. A reload would lose them, and saving them would keep
// changes Seed did not make, so the reload is refused.
var ErrUnsavedChanges = errors.New("the device's running configuration has unsaved changes")

// ReloadRequest is an admin's request to reload a target onto a staged image.
// User, Remote and UserAgent are for the audit log.
type ReloadRequest struct {
	ClientID  string
	ImageID   string
	TargetID  string
	User      string
	Remote    string
	UserAgent string
}

// ReloadResult is a reload job's outcome, persisted by the job store.
// FromVersion is what the device ran before the reload and Version what it
// came back on.
type ReloadResult struct {
	Outcome      ReloadOutcome `json:"outcome"`
	Reason       string        `json:"reason,omitempty"`
	FromVersion  string        `json:"fromVersion,omitempty"`
	Version      string        `json:"version,omitempty"`
	RunningImage string        `json:"runningImage,omitempty"`
}

// CheckReload refuses a reload request that can be refused without a login,
// as CheckStage does. The reload job checks again.
func (p *Pushes) CheckReload(ctx context.Context, req ReloadRequest) error {
	_, _, err := p.checkStage(ctx, StageRequest{ClientID: req.ClientID, ImageID: req.ImageID, TargetID: req.TargetID})
	return err
}

// Reload reloads a target onto a staged image and waits for it to come back.
// The returned error is a request refused before the device was touched;
// every reload that claimed the target ends in an outcome, recorded in the
// audit log.
func (p *Pushes) Reload(ctx context.Context, req ReloadRequest) (ReloadResult, error) {
	img, t, err := p.checkStage(ctx, StageRequest{ClientID: req.ClientID, ImageID: req.ImageID, TargetID: req.TargetID})
	if err != nil {
		return ReloadResult{}, err
	}
	if !p.claim(t.ID) {
		return ReloadResult{}, ErrPushBusy
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
	note(ctx, "reload_requested", map[string]any{"name": img.Name})
	if err = errors.Join(auditErrs...); err != nil {
		return ReloadResult{}, fmt.Errorf("audit reload: %w", err)
	}

	res := p.reload(ctx, t, img, note)
	// The device may be reloading whatever the job's context says.
	ctx = context.WithoutCancel(ctx)
	note(ctx, "reload_"+string(res.Outcome), map[string]any{
		"reason": res.Reason, "from_version": res.FromVersion, "version": res.Version,
	})
	if err = errors.Join(auditErrs...); err != nil {
		return res, fmt.Errorf("audit reload of %s: %w", t.ID, err)
	}
	return res, nil
}

func (p *Pushes) reload(
	ctx context.Context, t Target, img Image, note func(context.Context, string, map[string]any),
) ReloadResult {
	req, err := p.deviceRequest(ctx, t, "", "")
	if err != nil {
		return failedReload(ReloadResult{}, "prepare the push login", err)
	}
	before, err := p.readFacts(ctx, req)
	if err != nil {
		return failedReload(ReloadResult{}, "read the device's version and boot image", err)
	}
	res := ReloadResult{FromVersion: before.Version}
	if reason := refuseReload(before, img); reason != "" {
		res.Outcome, res.Reason = ReloadRefused, reason
		return res
	}
	err = onDeviceFor(ctx, deviceTimeout, func(ctx context.Context) error { return p.device.Reload(ctx, req) })
	if errors.Is(err, ErrUnsavedChanges) {
		res.Outcome, res.Reason = ReloadRefused, err.Error()+"; save or discard them on the device, then reload"
		return res
	}
	if err != nil {
		return failedReload(res, "reload the device", err)
	}
	note(ctx, "reload_sent", map[string]any{"from_version": before.Version})
	return p.awaitReload(ctx, req, img, res)
}

// awaitReload tries the device every reloadPoll until it is back. A device
// still answering on the version it ran has not gone down yet, unless it was
// seen down: then it came back on that version.
func (p *Pushes) awaitReload(ctx context.Context, req DeviceRequest, img Image, res ReloadResult) ReloadResult {
	deadline := time.Now().Add(p.reloadWindow)
	seenDown := false
	for {
		select {
		case <-ctx.Done():
			res.Outcome, res.Reason = ReloadFailed, "the wait for the device was cancelled after it was told to reload"
			return res
		case <-time.After(p.reloadPoll):
		}
		after, err := p.readFacts(ctx, req)
		switch {
		case err != nil:
			seenDown = true
		case reloadedOnto(img, res.FromVersion, after):
			res.Outcome, res.Version, res.RunningImage = ReloadVerified, after.Version, after.RunningImage
			return res
		case seenDown || after.Version != res.FromVersion:
			res.Outcome, res.Version, res.RunningImage = ReloadWrongImage, after.Version, after.RunningImage
			res.Reason = "the device came back on " + after.Version + ", not " + img.Name +
				"; it booted its fallback or other software"
			return res
		}
		if time.Now().After(deadline) {
			if !seenDown {
				res.Outcome, res.Reason = ReloadFailed, "the device took the reload but did not restart within "+
					p.reloadWindow.String()
				return res
			}
			res.Outcome, res.Reason = ReloadUnreachable, "the device did not answer within "+p.reloadWindow.String()+
				" of the reload; check it from its console"
			return res
		}
	}
}

func (p *Pushes) readFacts(ctx context.Context, req DeviceRequest) (ImageFacts, error) {
	var facts ImageFacts
	err := onDeviceFor(ctx, deviceTimeout, func(ctx context.Context) error {
		var err error
		facts, err = p.device.ImageFacts(ctx, req)
		return err
	})
	return facts, err
}

// refuseReload is the pre-check: the reason reloading a device with these
// facts onto img would be wrong, or "". Junos does not name the software it
// boots next; staging installed it, and the version after the reload says
// whether it took.
func refuseReload(facts ImageFacts, img Image) string {
	switch {
	case img.Platform == PlatformJuniperJunos:
		if strings.Contains(img.Name, facts.Version) {
			return "the device already runs Junos " + facts.Version
		}
	case img.Platform == PlatformCiscoIOS && fileName(facts.RunningImage) == img.Name:
		return "the device already runs " + img.Name
	case fileName(facts.BootImage) != img.Name:
		boot := facts.BootImage
		if boot == "" {
			boot = "no image named in its boot configuration"
		}
		return "the device boots " + boot + " next, not " + img.Name + "; stage the image first"
	}
	return ""
}

// reloadedOnto reports whether a device that ran fromVersion now runs img.
// IOS names its running image; EOS names only its version, which a new
// image changes; a Junos package is named for its release.
func reloadedOnto(img Image, fromVersion string, after ImageFacts) bool {
	switch img.Platform {
	case PlatformCiscoIOS:
		return fileName(after.RunningImage) == img.Name
	case PlatformJuniperJunos:
		return after.Version != fromVersion && strings.Contains(img.Name, after.Version)
	case PlatformAristaEOS:
		return after.Version != fromVersion && fileName(after.BootImage) == img.Name
	case PlatformCiscoNXOS, PlatformNETCONF:
	}
	return false
}

func failedReload(res ReloadResult, step string, err error) ReloadResult {
	res.Outcome, res.Reason = ReloadFailed, step+": "+err.Error()
	return res
}
