// Package escalation re-sends an alert nobody has acknowledged (P-B2, #3032).
//
// An operator gives a rule a ladder: stages that each fire a fixed time after
// the alert was raised, on the channels the stage names, and an optional
// repeat of the last stage. Acknowledging or resolving the alert stops the
// ladder. Delivery itself stays in internal/alerts/delivery; this package only
// decides when an alert is due and records that it went.
//
// The stage an alert has reached is stored on the alert, not held in memory,
// so a restart neither re-sends a stage nor forgets one. Advancing it is a
// compare-and-set that also requires the alert to be open, so an alert
// acknowledged between the read and the send is not escalated.
package escalation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// EngineName is the engine registry identifier.
const EngineName = "alert-escalation"

// Bounds on a ladder. A stage sooner than a minute is a duplicate of the
// alert itself, and more stages than five is a paging policy, not a ladder.
const (
	MinDelay  = time.Minute
	MaxStages = 5

	defaultInterval = 15 * time.Second
)

// ErrInvalidLadder wraps every reason Validate refuses a ladder.
var ErrInvalidLadder = errors.New("invalid escalation ladder")

// Stage is one step of a ladder.
type Stage struct {
	// After is measured from when the alert was raised, not from the
	// previous stage, so the ladder reads as a timeline.
	After    time.Duration
	Channels []alerts.Channel
}

// Ladder is the escalation policy for one rule.
type Ladder struct {
	// Rule is the alert's rule identifier: a built-in such as "iface.down",
	// or "db.<id>" for an operator rule.
	Rule   string
	Stages []Stage
	// Repeat re-sends the last stage at this period until the alert is
	// acknowledged or resolved. Zero sends the last stage once.
	Repeat time.Duration
}

// Validate reports why l cannot run, or nil.
func (l Ladder) Validate() error {
	if l.Rule == "" {
		return fmt.Errorf("%w: rule is required", ErrInvalidLadder)
	}
	if len(l.Stages) == 0 || len(l.Stages) > MaxStages {
		return fmt.Errorf("%w: %s needs 1 to %d stages", ErrInvalidLadder, l.Rule, MaxStages)
	}
	var previous time.Duration
	for i, s := range l.Stages {
		if s.After < MinDelay {
			return fmt.Errorf("%w: %s stage %d must come at least %s after the alert",
				ErrInvalidLadder, l.Rule, i+1, MinDelay)
		}
		if s.After <= previous {
			return fmt.Errorf("%w: %s stage %d must come after stage %d", ErrInvalidLadder, l.Rule, i+1, i)
		}
		previous = s.After
		if len(s.Channels) == 0 {
			return fmt.Errorf("%w: %s stage %d names no channel", ErrInvalidLadder, l.Rule, i+1)
		}
		for _, c := range s.Channels {
			if c != alerts.ChannelWebhook && c != alerts.ChannelEmail && c != alerts.ChannelSyslog {
				return fmt.Errorf("%w: %s stage %d: unknown channel %q", ErrInvalidLadder, l.Rule, i+1, c)
			}
		}
	}
	if l.Repeat != 0 && l.Repeat < MinDelay {
		return fmt.Errorf("%w: %s repeat must be 0 or at least %s", ErrInvalidLadder, l.Rule, MinDelay)
	}
	return nil
}

// Step is one escalation that is due: the stage to record and the channels
// to send it on.
type Step struct {
	Stage    int
	Channels []alerts.Channel
}

// Due reports the escalation a, if open, owes at now under l.
//
// A ladder never replays stages it slept through: after a restart that missed
// stages 1 and 2, the alert goes straight to stage 2 rather than paging twice
// in two ticks.
func (l Ladder) Due(a *alerts.Alert, now time.Time) (Step, bool) {
	if a.Acknowledged || a.Resolved || len(l.Stages) == 0 {
		return Step{}, false
	}
	reached := 0
	for i, s := range l.Stages {
		if !now.Before(a.CreatedAt.Add(s.After)) {
			reached = i + 1
		}
	}
	if reached > a.EscalationStage {
		return Step{Stage: reached, Channels: l.Stages[reached-1].Channels}, true
	}
	last := len(l.Stages)
	if l.Repeat > 0 && a.EscalationStage >= last && a.EscalatedAt != nil &&
		!now.Before(a.EscalatedAt.Add(l.Repeat)) {
		return Step{Stage: last, Channels: l.Stages[last-1].Channels}, true
	}
	return Step{}, false
}

// Store is the alert-store surface the escalator needs.
type Store interface {
	// ListEscalating returns the open alerts raised by any of rules.
	ListEscalating(ctx context.Context, rules []string) ([]*alerts.Alert, error)
	// AdvanceEscalation moves alert id from the stage and time the caller
	// read to stage, sent at at, only while the alert is still open and nothing
	// else has moved it. It reports whether it did.
	AdvanceEscalation(
		ctx context.Context,
		id int64,
		fromStage int,
		fromAt *time.Time,
		stage int,
		at time.Time,
	) (bool, error)
}

// Sender hands an escalated alert to the named channels' receivers.
type Sender interface {
	Escalate(ctx context.Context, alert *alerts.Alert, channels []alerts.Channel)
}

// Config wires an Escalator.
type Config struct {
	Store  Store
	Sender Sender
	// Ladders returns the current policy. It is read every tick, so a
	// settings write takes effect without a restart.
	Ladders  func() []Ladder
	Logger   *slog.Logger
	Now      func() time.Time
	Interval time.Duration
}

// Escalator is the engine that walks open alerts up their ladders.
type Escalator struct {
	store    Store
	sender   Sender
	ladders  func() []Ladder
	logger   *slog.Logger
	now      func() time.Time
	interval time.Duration

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New returns an unstarted Escalator.
func New(cfg Config) (*Escalator, error) {
	if cfg.Store == nil || cfg.Sender == nil || cfg.Ladders == nil {
		return nil, errors.New("escalation: store, sender and ladders are required")
	}
	e := &Escalator{
		store:    cfg.Store,
		sender:   cfg.Sender,
		ladders:  cfg.Ladders,
		logger:   cfg.Logger,
		now:      cfg.Now,
		interval: cfg.Interval,
	}
	if e.logger == nil {
		e.logger = slog.Default()
	}
	if e.now == nil {
		e.now = time.Now
	}
	if e.interval <= 0 {
		e.interval = defaultInterval
	}
	return e, nil
}

// Name implements engine.Engine.
func (*Escalator) Name() string { return EngineName }

// Start implements engine.Engine.
func (e *Escalator) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.started = true
	e.wg.Go(func() { e.loop(loopCtx) })
	return nil
}

// Stop implements engine.Engine.
func (e *Escalator) Stop(ctx context.Context) error {
	e.mu.Lock()
	cancel := e.cancel
	e.started = false
	e.cancel = nil
	e.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Escalator) loop(ctx context.Context) {
	t := time.NewTicker(e.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := e.Tick(ctx); err != nil {
				e.logger.WarnContext(ctx, "alert escalation pass failed", "error", err)
			}
		}
	}
}

// Tick escalates every alert that is due now and returns how many it sent.
func (e *Escalator) Tick(ctx context.Context) (int, error) {
	byRule := map[string]Ladder{}
	for _, l := range e.ladders() {
		byRule[l.Rule] = l
	}
	if len(byRule) == 0 {
		return 0, nil
	}
	rules := make([]string, 0, len(byRule))
	for r := range byRule {
		rules = append(rules, r)
	}
	slices.Sort(rules)

	open, err := e.store.ListEscalating(ctx, rules)
	if err != nil {
		return 0, fmt.Errorf("list escalating alerts: %w", err)
	}
	now := e.now().UTC().Truncate(time.Second)
	sent := 0
	for _, a := range open {
		step, due := byRule[a.Rule].Due(a, now)
		if !due {
			continue
		}
		moved, advErr := e.store.AdvanceEscalation(ctx, a.ID, a.EscalationStage, a.EscalatedAt, step.Stage, now)
		if advErr != nil {
			e.logger.WarnContext(ctx, "could not record alert escalation", "alert_id", a.ID, "error", advErr)
			continue
		}
		if !moved {
			continue
		}
		a.EscalationStage = step.Stage
		a.EscalatedAt = &now
		e.sender.Escalate(ctx, a, step.Channels)
		e.logger.InfoContext(ctx, "alert escalated",
			"alert_id", a.ID, "rule", a.Rule, "stage", step.Stage, "channels", step.Channels)
		sent++
	}
	return sent, nil
}
