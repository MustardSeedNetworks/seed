package deviceconfig

// Configuration push, plan half (P-D3, #3499, ADR-0035). A plan logs in,
// takes a fresh backup through the backup path, and asks the device what the
// candidate would change without changing anything. The diff is what an admin
// is shown and what a later apply is bound to by its hash; nothing in this
// file writes to a device.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// PushMode is how a candidate meets the running configuration.
type PushMode string

// Push modes: merge adds the candidate to the running configuration; replace
// makes the candidate the whole configuration.
const (
	PushMerge   PushMode = "merge"
	PushReplace PushMode = "replace"
)

// DiffSource says who produced a plan's diff: the device, from its own dry
// run, or Seed, predicting it from the fresh backup where the platform offers
// no dry run.
type DiffSource string

// Diff sources.
const (
	DiffFromDevice DiffSource = "device"
	DiffPredicted  DiffSource = "predicted"
)

// PlanOutcome is how a planning job ended.
type PlanOutcome string

// Planning outcomes. Only PlanReady stores a plan.
const (
	PlanReady    PlanOutcome = "ready"
	PlanNoChange PlanOutcome = "no_change"
	PlanRefused  PlanOutcome = "refused"
)

const (
	// PlanLifetime is how long a plan may be applied: long enough to read the
	// diff, short enough that the device it describes is still the device.
	PlanLifetime = 15 * time.Minute
	// MaxCandidateBytes caps a candidate. A whole chassis configuration for
	// replace fits.
	MaxCandidateBytes = 2 << 20
	// ConfirmWindow is how long a device waits for an applied commit to be
	// confirmed before reverting it: room for the post-check's login and
	// read, and short enough that a push which cut Seed off is soon undone.
	ConfirmWindow = 5 * time.Minute
	// planRetention is how many plans a target keeps.
	planRetention = 50
)

// ErrPushBusy is a second push job on a target that already has one running.
var ErrPushBusy = errors.New("deviceconfig: a push job is already running on this target")

// PlanRefusedError is a device or platform that cannot take a push safely,
// with the reason an admin can act on. ADR-0035 has no best-effort mode.
type PlanRefusedError struct{ Reason string }

func (e *PlanRefusedError) Error() string { return "push refused: " + e.Reason }

// Plan is a stored dry run. Candidate and Diff are set only when one plan is
// read; CandidateCT and DiffCT are the stored ciphertext and never leave the
// service.
type Plan struct {
	ID            string     `json:"id"`
	ClientID      string     `json:"clientId"`
	TargetID      string     `json:"targetId"`
	CreatedBy     string     `json:"createdBy"`
	CreatedAt     time.Time  `json:"createdAt"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	Mode          PushMode   `json:"mode"`
	DiffSource    DiffSource `json:"diffSource"`
	BackupID      string     `json:"backupId"`
	RunningSHA256 string     `json:"runningSha256"`
	// DiffSHA256 is the hash of Diff; apply must present it, so what is
	// applied is what the admin was shown.
	DiffSHA256  string `json:"diffSha256"`
	Added       int    `json:"added"`
	Removed     int    `json:"removed"`
	Candidate   string `json:"candidate,omitempty"`
	Diff        string `json:"diff,omitempty"`
	CandidateCT string `json:"-"`
	DiffCT      string `json:"-"`
	// AppliedAt is set when an apply claims the plan, which happens once.
	// ApplyOutcome stays empty while that apply runs.
	AppliedAt    *time.Time   `json:"appliedAt,omitempty"`
	AppliedBy    string       `json:"appliedBy,omitempty"`
	ApplyOutcome ApplyOutcome `json:"applyOutcome,omitempty"`
}

// PlanStore keeps push plans.
type PlanStore interface {
	// InsertPlan stores a plan, generating its id.
	InsertPlan(ctx context.Context, p *Plan) error
	// GetPlan returns one plan with CandidateCT and DiffCT set.
	GetPlan(ctx context.Context, clientID, id string) (Plan, error)
	// ListPlans returns a target's plans newest first, without ciphertext.
	ListPlans(ctx context.Context, clientID, targetID string, limit int) ([]Plan, error)
	// PrunePlans keeps the target's newest keep plans.
	PrunePlans(ctx context.Context, clientID, targetID string, keep int) error
	// ClaimApply marks a plan applied by user at now, and answers
	// ErrPlanSpent unless the plan was unapplied and unexpired at now.
	ClaimApply(ctx context.Context, clientID, id, user string, now time.Time) error
	// FinishApply records how an applied plan's apply ended.
	FinishApply(ctx context.Context, clientID, id string, outcome ApplyOutcome) error
}

// DeviceRequest is one login to the device a push is for.
type DeviceRequest struct {
	Host          string
	Port          int
	User          string
	Password      string
	PinnedHostKey string
	Platform      Platform
	Mode          PushMode
	Candidate     string
	// CommitID names an applied commit, so a NETCONF confirmed commit can be
	// confirmed or cancelled from a later session (RFC 6241 §8.4 persist).
	CommitID string
}

// DryRunResult is the device's account of what the candidate would change.
type DryRunResult struct {
	Diff    string
	Added   int
	Removed int
}

// PushDevice is the device side of a push. Each method is one login.
type PushDevice interface {
	// DryRun asks what the candidate would change and changes nothing. A
	// device that cannot take a push safely answers *PlanRefusedError.
	DryRun(ctx context.Context, req DeviceRequest) (DryRunResult, error)
	// Commit applies the candidate under a confirmed commit that the device
	// reverts by itself unless Confirm arrives within ConfirmWindow.
	Commit(ctx context.Context, req DeviceRequest) error
	// Confirm makes the pending commit permanent.
	Confirm(ctx context.Context, req DeviceRequest) error
	// Rollback undoes the pending commit now rather than at the timer.
	Rollback(ctx context.Context, req DeviceRequest) error
}

// PushAuditEvent is one push action for the audit log. Detail never carries
// configuration text: the audit log is read more widely than the backups.
type PushAuditEvent struct {
	Action    string
	User      string
	TargetID  string
	Remote    string
	UserAgent string
	Detail    map[string]any
}

// PushAuditResource is the audit log resource type of every push event.
const PushAuditResource = "device_config_push"

// PlanRequest is an admin's request to plan a push. User, Remote and
// UserAgent are for the audit log.
type PlanRequest struct {
	ClientID  string
	TargetID  string
	Mode      PushMode
	Candidate string
	User      string
	Remote    string
	UserAgent string
}

// PlanResult is a planning job's outcome. It is the job's result, which the
// job store persists, so it carries no configuration text.
type PlanResult struct {
	Outcome PlanOutcome `json:"outcome"`
	PlanID  string      `json:"planId,omitempty"`
	Reason  string      `json:"reason,omitempty"`
	Added   int         `json:"added"`
	Removed int         `json:"removed"`
}

// Pushes is the configuration push use-case. It shares the backup use-case's
// targets, vault and cipher, and takes its fresh backup through it.
type Pushes struct {
	backups *Service
	plans   PlanStore
	device  PushDevice
	audit   func(context.Context, PushAuditEvent) error

	mu   sync.Mutex
	busy map[string]struct{}
}

// NewPushes builds the use-case. Every dependency is required.
func NewPushes(
	backups *Service, plans PlanStore, device PushDevice, audit func(context.Context, PushAuditEvent) error,
) (*Pushes, error) {
	if backups == nil || plans == nil || device == nil || audit == nil {
		return nil, ErrUnavailable
	}
	return &Pushes{backups: backups, plans: plans, device: device, audit: audit, busy: map[string]struct{}{}}, nil
}

// Check refuses a plan request that can be refused without a login: an
// unknown target, one with pushing off or no pinned key, a platform push is
// not offered on, and a malformed candidate. The planning job checks again,
// since the target can change after the request was accepted.
func (p *Pushes) Check(ctx context.Context, req PlanRequest) error {
	_, err := p.check(ctx, req)
	return err
}

func (p *Pushes) check(ctx context.Context, req PlanRequest) (Target, error) {
	t, err := p.pushTarget(ctx, req.ClientID, req.TargetID)
	if err != nil {
		return Target{}, err
	}
	switch {
	case req.Mode != PushMerge && req.Mode != PushReplace:
		return Target{}, ValidationError{Msg: "mode must be merge or replace"}
	case strings.TrimSpace(req.Candidate) == "":
		return Target{}, ValidationError{Msg: "candidate is required"}
	case len(req.Candidate) > MaxCandidateBytes:
		return Target{}, ValidationError{Msg: fmt.Sprintf("candidate exceeds %d bytes", MaxCandidateBytes)}
	}
	switch t.Platform {
	case PlatformNETCONF:
		err = checkXMLCandidate(req.Candidate)
	case PlatformCiscoIOS, PlatformJuniperJunos, PlatformAristaEOS:
		err = checkCLICandidate(t.Platform, req.Candidate)
	case PlatformCiscoNXOS:
	}
	if err != nil {
		return Target{}, ValidationError{Msg: err.Error()}
	}
	return t, nil
}

// pushTarget returns a target that may be pushed to: pushing on, its host key
// pinned, and not NX-OS.
func (p *Pushes) pushTarget(ctx context.Context, clientID, targetID string) (Target, error) {
	t, err := p.backups.repo.GetTarget(ctx, clientID, targetID)
	if errors.Is(err, ErrNotFound) {
		return Target{}, ValidationError{Msg: fmt.Sprintf("no backup target %q", targetID)}
	}
	if err != nil {
		return Target{}, err
	}
	switch {
	case t.PushCredentialsID == "":
		return Target{}, ValidationError{Msg: "pushing is not enabled on this target"}
	case t.HostKeySHA256 == "":
		return Target{}, ValidationError{Msg: "the target's host key must be confirmed before a push"}
	case t.Platform == PlatformCiscoNXOS:
		return Target{}, ValidationError{
			Msg: "cisco-nxos is read-only: it has no device-side revert, so a push that cut Seed off could not be undone",
		}
	}
	return t, nil
}

// Plan runs a dry run and stores the plan when it changes something. The
// returned error is a failure to plan; a refusal or an empty diff is an
// outcome.
func (p *Pushes) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	if !p.claim(req.TargetID) {
		return PlanResult{}, ErrPushBusy
	}
	defer p.release(req.TargetID)

	res, err := p.plan(ctx, req)
	if cerr := ctx.Err(); cerr != nil {
		return PlanResult{}, cerr
	}
	action, detail := "device_config.push.plan_created", map[string]any{
		"mode": req.Mode, "outcome": res.Outcome, "plan_id": res.PlanID, "added": res.Added, "removed": res.Removed,
	}
	switch {
	case err != nil:
		action, detail = "device_config.push.plan_failed", map[string]any{"mode": req.Mode, "error": err.Error()}
	case res.Outcome == PlanRefused:
		action, detail = "device_config.push.plan_refused", map[string]any{"mode": req.Mode, "reason": res.Reason}
	}
	if auditErr := p.audit(ctx, PushAuditEvent{
		Action: action, User: req.User, TargetID: req.TargetID, Remote: req.Remote, UserAgent: req.UserAgent,
		Detail: detail,
	}); auditErr != nil {
		return PlanResult{}, errors.Join(err, fmt.Errorf("audit push plan: %w", auditErr))
	}
	return res, err
}

func (p *Pushes) plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	t, err := p.check(ctx, req)
	if err != nil {
		return PlanResult{}, err
	}
	backup, err := p.freshBackup(ctx, t)
	if err != nil {
		return PlanResult{}, err
	}

	var (
		dr     DryRunResult
		source DiffSource
	)
	if t.Platform == PlatformCiscoIOS {
		source = DiffPredicted
		dr, err = predictIOS(backup, req.Mode, req.Candidate)
	} else {
		source = DiffFromDevice
		dr, err = p.deviceDryRun(ctx, t, req)
	}
	if refused, ok := errors.AsType[*PlanRefusedError](err); ok {
		return PlanResult{Outcome: PlanRefused, Reason: refused.Reason}, nil
	}
	if err != nil {
		return PlanResult{}, err
	}
	if dr.Added+dr.Removed == 0 {
		return PlanResult{Outcome: PlanNoChange}, nil
	}

	plan, err := p.store(ctx, t, req, backup, source, dr)
	if err != nil {
		return PlanResult{}, err
	}
	return PlanResult{Outcome: PlanReady, PlanID: plan.ID, Added: plan.Added, Removed: plan.Removed}, nil
}

// freshBackup backs the target up through the backup path, so the
// configuration the plan starts from is stored and diffable like any other,
// and returns it decrypted.
func (p *Pushes) freshBackup(ctx context.Context, t Target) (Backup, error) {
	r, err := p.backups.backupOne(ctx, t)
	if err != nil {
		return Backup{}, err
	}
	if r.Status != StatusOK {
		return Backup{}, fmt.Errorf("back up the running configuration first: %s: %s", r.Status, r.Error)
	}
	return p.backups.GetBackup(ctx, t.ClientID, r.BackupID)
}

func (p *Pushes) deviceDryRun(ctx context.Context, t Target, req PlanRequest) (DryRunResult, error) {
	dr, err := p.deviceRequest(ctx, t, req.Mode, req.Candidate)
	if err != nil {
		return DryRunResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, deviceTimeout)
	defer cancel()
	return p.device.DryRun(ctx, dr)
}

// deviceRequest is a login to t with its push credential, never the backup's.
func (p *Pushes) deviceRequest(ctx context.Context, t Target, mode PushMode, candidate string) (DeviceRequest, error) {
	cred, err := p.backups.creds.Get(ctx, t.PushCredentialsID, t.ClientID)
	if err != nil {
		return DeviceRequest{}, fmt.Errorf("read push credential: %w", err)
	}
	if cred.Kind != polling.CredentialKindSSH {
		return DeviceRequest{}, fmt.Errorf("push credential %s is %s, not SSH", cred.ID, cred.Kind)
	}
	password, err := p.backups.cipher.DecryptValue(cred.SSHPasswordCT)
	if err != nil {
		return DeviceRequest{}, fmt.Errorf("decrypt SSH password: %w", err)
	}
	return DeviceRequest{
		Host:          t.Host,
		Port:          t.Port,
		User:          cred.SSHUser,
		Password:      password,
		PinnedHostKey: t.HostKeySHA256,
		Platform:      t.Platform,
		Mode:          mode,
		Candidate:     candidate,
	}, nil
}

func (p *Pushes) store(
	ctx context.Context, t Target, req PlanRequest, backup Backup, source DiffSource, dr DryRunResult,
) (Plan, error) {
	candidateCT, err := p.backups.cipher.EncryptValue(req.Candidate)
	if err != nil {
		return Plan{}, fmt.Errorf("encrypt candidate: %w", err)
	}
	diffCT, err := p.backups.cipher.EncryptValue(dr.Diff)
	if err != nil {
		return Plan{}, fmt.Errorf("encrypt diff: %w", err)
	}
	now := p.backups.now().UTC().Truncate(time.Second)
	sum := sha256.Sum256([]byte(dr.Diff))
	plan := Plan{
		ClientID:      t.ClientID,
		TargetID:      t.ID,
		CreatedBy:     req.User,
		CreatedAt:     now,
		ExpiresAt:     now.Add(PlanLifetime),
		Mode:          req.Mode,
		DiffSource:    source,
		BackupID:      backup.ID,
		RunningSHA256: backup.SHA256,
		DiffSHA256:    hex.EncodeToString(sum[:]),
		Added:         dr.Added,
		Removed:       dr.Removed,
		CandidateCT:   candidateCT,
		DiffCT:        diffCT,
	}
	if err = p.plans.InsertPlan(ctx, &plan); err != nil {
		return Plan{}, fmt.Errorf("record push plan for %s: %w", t.ID, err)
	}
	if err = p.plans.PrunePlans(ctx, t.ClientID, t.ID, planRetention); err != nil {
		return Plan{}, fmt.Errorf("prune push plans of %s: %w", t.ID, err)
	}
	return plan, nil
}

// ListPlans returns a target's plans, newest first, without their text.
func (p *Pushes) ListPlans(ctx context.Context, clientID, targetID string, limit int) ([]Plan, error) {
	return p.plans.ListPlans(ctx, clientID, targetID, limit)
}

// GetPlan returns one plan with its candidate and diff decrypted.
func (p *Pushes) GetPlan(ctx context.Context, clientID, id string) (Plan, error) {
	plan, err := p.plans.GetPlan(ctx, clientID, id)
	if err != nil {
		return Plan{}, err
	}
	if plan.Candidate, err = p.backups.cipher.DecryptValue(plan.CandidateCT); err != nil {
		return Plan{}, fmt.Errorf("decrypt candidate %s: %w", id, err)
	}
	if plan.Diff, err = p.backups.cipher.DecryptValue(plan.DiffCT); err != nil {
		return Plan{}, fmt.Errorf("decrypt diff %s: %w", id, err)
	}
	plan.CandidateCT, plan.DiffCT = "", ""
	return plan, nil
}

func (p *Pushes) claim(targetID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.busy[targetID]; ok {
		return false
	}
	p.busy[targetID] = struct{}{}
	return true
}

func (p *Pushes) release(targetID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.busy, targetID)
}

// lineDiff renders the change from one configuration text to another as a
// unified diff and counts its lines.
func lineDiff(fromName, from, toName, to string) (DryRunResult, error) {
	a, b := difflib.SplitLines(from), difflib.SplitLines(to)
	text, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: a, B: b, FromFile: fromName, ToFile: toName, Context: diffContext,
	})
	if err != nil {
		return DryRunResult{}, fmt.Errorf("render push diff: %w", err)
	}
	res := DryRunResult{Diff: text}
	res.Added, res.Removed = countChanges(a, b)
	return res, nil
}
