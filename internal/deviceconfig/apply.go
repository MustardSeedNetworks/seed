package deviceconfig

// Configuration push, apply half (P-D3, #3499, ADR-0035 §2). Apply commits a
// plan's candidate only under a commit the device reverts by itself, then
// proves the change from a new login before confirming it. Seed rolls back
// explicitly only once it knows the commit landed: before that, undoing
// "the last commit" could undo someone else's.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ApplyOutcome is how an apply ended.
type ApplyOutcome string

// Apply outcomes. Reverting means Seed could not confirm or roll back, and
// the device's own timer undoes the commit when ConfirmWindow runs out.
const (
	ApplyConfirmed  ApplyOutcome = "confirmed"
	ApplyRolledBack ApplyOutcome = "rolled_back"
	ApplyReverting  ApplyOutcome = "reverting"
	ApplyDrifted    ApplyOutcome = "config_drifted"
	ApplyFailed     ApplyOutcome = "failed"
)

// ErrStartupNotSaved is a confirmed push whose startup configuration could
// not be written: the change runs, and a reload loses it.
var ErrStartupNotSaved = errors.New("the startup configuration was not saved")

// ErrPlanSpent is a plan that was already applied or has expired. A plan is
// applied at most once; the admin plans again.
var ErrPlanSpent = errors.New("the push plan was already applied or has expired; plan again")

// ApplyRequest is an admin's request to apply a plan. DiffSHA256 is the hash
// of the diff the admin was shown. User, Remote and UserAgent are for the
// audit log.
type ApplyRequest struct {
	ClientID   string
	PlanID     string
	DiffSHA256 string
	User       string
	Remote     string
	UserAgent  string
}

// ApplyResult is an apply job's outcome. Like PlanResult it is persisted by
// the job store, so it carries no configuration text. BackupID names the
// backup taken after a confirmed push.
type ApplyResult struct {
	Outcome  ApplyOutcome `json:"outcome"`
	Reason   string       `json:"reason,omitempty"`
	BackupID string       `json:"backupId,omitempty"`
}

// CheckApply refuses an apply that can be refused without a login: an
// unknown plan, a hash that is not the plan's, a spent plan, and a target
// that can no longer be pushed to. The apply job checks again.
func (p *Pushes) CheckApply(ctx context.Context, req ApplyRequest) error {
	_, _, err := p.checkApply(ctx, req)
	return err
}

func (p *Pushes) checkApply(ctx context.Context, req ApplyRequest) (Plan, Target, error) {
	plan, err := p.plans.GetPlan(ctx, req.ClientID, req.PlanID)
	if err != nil {
		return Plan{}, Target{}, err
	}
	switch {
	case req.DiffSHA256 != plan.DiffSHA256:
		return Plan{}, Target{}, ValidationError{Msg: "planSha256 is not the hash of this plan's diff"}
	case plan.AppliedAt != nil, !p.backups.now().Before(plan.ExpiresAt):
		return Plan{}, Target{}, ErrPlanSpent
	}
	t, err := p.pushTarget(ctx, req.ClientID, plan.TargetID)
	if err != nil {
		return Plan{}, Target{}, err
	}
	return plan, t, nil
}

// Apply applies a plan once. The returned error is a request that was
// refused before anything was sent to the device; every apply that claimed
// the plan ends in an outcome, recorded on the plan and in the audit log.
func (p *Pushes) Apply(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	plan, t, err := p.checkApply(ctx, req)
	if err != nil {
		return ApplyResult{}, err
	}
	if !p.claim(t.ID) {
		return ApplyResult{}, ErrPushBusy
	}
	defer p.release(t.ID)
	if err = p.plans.ClaimApply(ctx, req.ClientID, plan.ID, req.User, p.backups.now().UTC()); err != nil {
		return ApplyResult{}, err
	}

	event := PushAuditEvent{User: req.User, TargetID: t.ID, Remote: req.Remote, UserAgent: req.UserAgent}
	detail := map[string]any{
		"plan_id": plan.ID, "mode": plan.Mode, "diff_sha256": plan.DiffSHA256,
		"added": plan.Added, "removed": plan.Removed,
	}
	event.Action, event.Detail = "device_config.push.apply_started", detail
	if err = p.audit(ctx, event); err != nil {
		return ApplyResult{}, fmt.Errorf("audit push apply: %w", err)
	}

	res := p.apply(ctx, t, plan)
	// The outcome is recorded even when the job is being cancelled: the
	// device may already hold the change.
	ctx = context.WithoutCancel(ctx)
	finishErr := p.plans.FinishApply(ctx, req.ClientID, plan.ID, res.Outcome)
	event.Action, event.Detail = "device_config.push."+string(res.Outcome), map[string]any{
		"plan_id": plan.ID, "outcome": res.Outcome, "reason": res.Reason, "backup_id": res.BackupID,
	}
	if err = errors.Join(finishErr, p.audit(ctx, event)); err != nil {
		return res, fmt.Errorf("record push apply %s: %w", plan.ID, err)
	}
	return res, nil
}

func (p *Pushes) apply(ctx context.Context, t Target, plan Plan) ApplyResult {
	before, err := p.backups.backupOne(ctx, t)
	switch {
	case err != nil:
		return failed("back up the running configuration first", err)
	case before.Status != StatusOK:
		return ApplyResult{Outcome: ApplyFailed, Reason: fmt.Sprintf(
			"back up the running configuration first: %s: %s", before.Status, before.Error)}
	}
	running, err := p.backups.GetBackup(ctx, t.ClientID, before.BackupID)
	if err != nil {
		return failed("read the running configuration", err)
	}
	if running.SHA256 != plan.RunningSHA256 {
		return ApplyResult{
			Outcome: ApplyDrifted, Reason: "the running configuration changed after the plan; plan again",
		}
	}

	candidate, err := p.backups.cipher.DecryptValue(plan.CandidateCT)
	if err != nil {
		return failed("decrypt the candidate", err)
	}
	req, err := p.deviceRequest(ctx, t, plan.Mode, candidate)
	if err != nil {
		return failed("prepare the push login", err)
	}
	if req.CommitID, err = commitID(); err != nil {
		return failed("name the commit", err)
	}

	window := ConfirmWindow.String()
	if err = p.onDevice(ctx, req, p.device.Commit); err != nil {
		return ApplyResult{Outcome: ApplyFailed, Reason: fmt.Sprintf(
			"the commit did not complete: %v; if the device made it, it reverts by itself within %s", err, window)}
	}

	// The post-check is a dry run of the same candidate from a new login: it
	// proves management access survived, and a device that holds the change
	// has nothing left to change.
	var check DryRunResult
	err = p.onDevice(ctx, req, func(ctx context.Context, req DeviceRequest) error {
		var runErr error
		check, runErr = p.device.DryRun(ctx, req)
		return runErr
	})
	if err == nil && check.Added+check.Removed == 0 {
		return p.confirm(ctx, t, req)
	}
	reason := fmt.Sprintf("the post-check found %d lines still to change", check.Added+check.Removed)
	if err != nil {
		reason = fmt.Sprintf("the post-check could not log in: %v", err)
	}
	if err = p.onDevice(ctx, req, p.device.Rollback); err != nil {
		return ApplyResult{Outcome: ApplyReverting, Reason: fmt.Sprintf(
			"%s; the rollback failed (%v), so the device reverts by itself within %s", reason, err, window)}
	}
	return ApplyResult{Outcome: ApplyRolledBack, Reason: reason}
}

// confirm makes the commit permanent and backs the result up, so the stored
// configuration is the device's and P-D2 alerts on the change.
func (p *Pushes) confirm(ctx context.Context, t Target, req DeviceRequest) ApplyResult {
	var notes []string
	switch err := p.onDevice(ctx, req, p.device.Confirm); {
	case errors.Is(err, ErrStartupNotSaved):
		notes = append(notes, fmt.Sprintf("%v; a reload loses the change until it is saved", err))
	case err != nil:
		return ApplyResult{Outcome: ApplyReverting, Reason: fmt.Sprintf(
			"the confirm failed: %v; the device reverts by itself within %s", err, ConfirmWindow)}
	}
	res := ApplyResult{Outcome: ApplyConfirmed}
	after, err := p.backups.backupOne(ctx, t)
	switch {
	case err != nil:
		notes = append(notes, "the backup after it failed: "+err.Error())
	case after.Status != StatusOK:
		notes = append(notes, fmt.Sprintf("the backup after it failed: %s: %s", after.Status, after.Error))
	default:
		res.BackupID = after.BackupID
	}
	if len(notes) > 0 {
		res.Reason = "confirmed, but " + strings.Join(notes, "; ")
	}
	return res
}

// onDevice runs one device step under the per-device timeout.
func (p *Pushes) onDevice(
	ctx context.Context, req DeviceRequest, step func(context.Context, DeviceRequest) error,
) error {
	ctx, cancel := context.WithTimeout(ctx, deviceTimeout)
	defer cancel()
	return step(ctx, req)
}

func failed(step string, err error) ApplyResult {
	return ApplyResult{Outcome: ApplyFailed, Reason: step + ": " + err.Error()}
}

// commitID names one apply's commit. It is a NETCONF persist token, so only
// the session that knows it can confirm or cancel the commit.
func commitID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "seed-" + hex.EncodeToString(b[:]), nil
}
