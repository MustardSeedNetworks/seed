package escalation_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/escalation"
)

func webhook() []alerts.Channel { return []alerts.Channel{alerts.ChannelWebhook} }

func both() []alerts.Channel {
	return []alerts.Channel{alerts.ChannelWebhook, alerts.ChannelEmail}
}

func twoStages(repeat time.Duration) escalation.Ladder {
	return escalation.Ladder{
		Rule: "iface.down",
		Stages: []escalation.Stage{
			{After: 5 * time.Minute, Channels: webhook()},
			{After: 15 * time.Minute, Channels: both()},
		},
		Repeat: repeat,
	}
}

func TestLadderValidate(t *testing.T) {
	stage := func(after time.Duration, ch ...alerts.Channel) escalation.Stage {
		return escalation.Stage{After: after, Channels: ch}
	}
	tests := []struct {
		name   string
		ladder escalation.Ladder
		ok     bool
	}{
		{"two stages", twoStages(0), true},
		{"two stages with repeat", twoStages(30 * time.Minute), true},
		{"no rule", escalation.Ladder{Stages: []escalation.Stage{stage(time.Minute, alerts.ChannelWebhook)}}, false},
		{"no stage", escalation.Ladder{Rule: "r"}, false},
		{"six stages", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(1*time.Minute, alerts.ChannelWebhook), stage(2*time.Minute, alerts.ChannelWebhook),
			stage(3*time.Minute, alerts.ChannelWebhook), stage(4*time.Minute, alerts.ChannelWebhook),
			stage(5*time.Minute, alerts.ChannelWebhook), stage(6*time.Minute, alerts.ChannelWebhook),
		}}, false},
		{"stage sooner than a minute", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(59*time.Second, alerts.ChannelWebhook),
		}}, false},
		{"stages out of order", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(10*time.Minute, alerts.ChannelWebhook), stage(5*time.Minute, alerts.ChannelEmail),
		}}, false},
		{"two stages at one time", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(5*time.Minute, alerts.ChannelWebhook), stage(5*time.Minute, alerts.ChannelEmail),
		}}, false},
		{"stage with no channel", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(5 * time.Minute),
		}}, false},
		{"unknown channel", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(5*time.Minute, "pager"),
		}}, false},
		{"repeat sooner than a minute", escalation.Ladder{Rule: "r", Stages: []escalation.Stage{
			stage(5*time.Minute, alerts.ChannelWebhook),
		}, Repeat: 30 * time.Second}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ladder.Validate()
			if tt.ok && err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
			if !tt.ok && !errors.Is(err, escalation.ErrInvalidLadder) {
				t.Fatalf("Validate() = %v, want ErrInvalidLadder", err)
			}
		})
	}
}

func TestLadderDue(t *testing.T) {
	raised := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	at := raised.Add
	tests := []struct {
		name      string
		ladder    escalation.Ladder
		stage     int
		escalated time.Duration // since raised; 0 = never escalated
		acked     bool
		resolved  bool
		now       time.Duration
		wantStage int // 0 = nothing due
		wantCh    []alerts.Channel
	}{
		{name: "before the first stage", ladder: twoStages(0), now: 4*time.Minute + 59*time.Second},
		{name: "first stage on time", ladder: twoStages(0), now: 5 * time.Minute, wantStage: 1, wantCh: webhook()},
		{
			name:      "first stage already sent",
			ladder:    twoStages(0),
			stage:     1,
			escalated: 5 * time.Minute,
			now:       14 * time.Minute,
		},
		{
			name: "second stage on time", ladder: twoStages(0), stage: 1, escalated: 5 * time.Minute,
			now: 15 * time.Minute, wantStage: 2, wantCh: both(),
		},
		{
			name:      "slept through both stages sends only the second",
			ladder:    twoStages(0),
			now:       time.Hour,
			wantStage: 2,
			wantCh:    both(),
		},
		{
			name:      "last stage sent, no repeat",
			ladder:    twoStages(0),
			stage:     2,
			escalated: 15 * time.Minute,
			now:       24 * time.Hour,
		},
		{
			name: "repeat not yet due", ladder: twoStages(30 * time.Minute), stage: 2, escalated: 15 * time.Minute,
			now: 44 * time.Minute,
		},
		{
			name: "repeat due", ladder: twoStages(30 * time.Minute), stage: 2, escalated: 15 * time.Minute,
			now: 45 * time.Minute, wantStage: 2, wantCh: both(),
		},
		{name: "acknowledged never escalates", ladder: twoStages(0), acked: true, now: time.Hour},
		{name: "resolved never escalates", ladder: twoStages(0), resolved: true, now: time.Hour},
		{
			name: "acknowledged stops the repeat", ladder: twoStages(30 * time.Minute), stage: 2,
			escalated: 15 * time.Minute, acked: true, now: 24 * time.Hour,
		},
		{name: "no ladder for the rule", ladder: escalation.Ladder{}, now: time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &alerts.Alert{
				ID: 1, Rule: "iface.down", CreatedAt: raised, EscalationStage: tt.stage,
				Acknowledged: tt.acked, Resolved: tt.resolved,
			}
			if tt.escalated != 0 {
				e := at(tt.escalated)
				a.EscalatedAt = &e
			}
			step, due := tt.ladder.Due(a, at(tt.now))
			if due != (tt.wantStage != 0) {
				t.Fatalf("due = %v, want stage %d", due, tt.wantStage)
			}
			if step.Stage != tt.wantStage || !slices.Equal(step.Channels, tt.wantCh) {
				t.Fatalf("Due() = stage %d on %v, want stage %d on %v",
					step.Stage, step.Channels, tt.wantStage, tt.wantCh)
			}
		})
	}
}

// fakeStore holds open alerts and applies AdvanceEscalation's compare-and-set
// the way the repository does.
type fakeStore struct {
	alerts   map[int64]*alerts.Alert
	listed   []string
	advances int
}

func (s *fakeStore) ListEscalating(_ context.Context, rules []string) ([]*alerts.Alert, error) {
	s.listed = rules
	var out []*alerts.Alert
	for _, a := range s.alerts {
		if !a.Acknowledged && !a.Resolved && slices.Contains(rules, a.Rule) {
			c := *a
			out = append(out, &c)
		}
	}
	return out, nil
}

func (s *fakeStore) AdvanceEscalation(
	_ context.Context, id int64, fromStage int, fromAt *time.Time, stage int, at time.Time,
) (bool, error) {
	a := s.alerts[id]
	sameAt := (fromAt == nil) == (a.EscalatedAt == nil) && (fromAt == nil || fromAt.Equal(*a.EscalatedAt))
	if a.Acknowledged || a.Resolved || a.EscalationStage != fromStage || !sameAt {
		return false, nil
	}
	s.advances++
	a.EscalationStage = stage
	a.EscalatedAt = &at
	return true, nil
}

type sent struct {
	id       int64
	stage    int
	channels []alerts.Channel
}

type fakeSender struct{ sent []sent }

func (f *fakeSender) Escalate(_ context.Context, a *alerts.Alert, channels []alerts.Channel) {
	f.sent = append(f.sent, sent{a.ID, a.EscalationStage, channels})
}

func newEscalator(
	t *testing.T,
	store escalation.Store,
	sender escalation.Sender,
	now *time.Time,
	ladders ...escalation.Ladder,
) *escalation.Escalator {
	t.Helper()
	e, err := escalation.New(escalation.Config{
		Store: store, Sender: sender,
		Ladders: func() []escalation.Ladder { return ladders },
		Now:     func() time.Time { return *now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestTickSendsOnlyRulesWithALadder(t *testing.T) {
	raised := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{alerts: map[int64]*alerts.Alert{
		1: {ID: 1, Rule: "iface.down", CreatedAt: raised},
		2: {ID: 2, Rule: "bgp.flap", CreatedAt: raised},
	}}
	sender := &fakeSender{}
	now := raised.Add(5 * time.Minute)
	e := newEscalator(t, store, sender, &now, twoStages(0))

	n, err := e.Tick(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("Tick() = %d, %v; want 1, nil", n, err)
	}
	if !slices.Equal(store.listed, []string{"iface.down"}) {
		t.Fatalf("listed rules %v, want only the laddered one", store.listed)
	}
	if len(sender.sent) != 1 || sender.sent[0].id != 1 || sender.sent[0].stage != 1 ||
		!slices.Equal(sender.sent[0].channels, webhook()) {
		t.Fatalf("sent %+v, want alert 1 at stage 1", sender.sent)
	}
	if got := store.alerts[1]; got.EscalationStage != 1 || got.EscalatedAt == nil || !got.EscalatedAt.Equal(now) {
		t.Fatalf("stored stage %d at %v, want 1 at %v", got.EscalationStage, got.EscalatedAt, now)
	}

	// The same instant again: the stage is recorded, so nothing is re-sent.
	if n, err = e.Tick(context.Background()); err != nil || n != 0 {
		t.Fatalf("second Tick() = %d, %v; want 0, nil", n, err)
	}
}

// A lost compare-and-set means another pass sent the stage or the operator
// acknowledged the alert after it was read; either way it must not be sent.
type racingStore struct{ fakeStore }

func (s *racingStore) AdvanceEscalation(
	context.Context, int64, int, *time.Time, int, time.Time,
) (bool, error) {
	return false, nil
}

func TestTickDoesNotSendWhenTheAdvanceIsLost(t *testing.T) {
	raised := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := &racingStore{fakeStore{alerts: map[int64]*alerts.Alert{
		1: {ID: 1, Rule: "iface.down", CreatedAt: raised},
	}}}
	sender := &fakeSender{}
	now := raised.Add(time.Hour)
	e := newEscalator(t, store, sender, &now, twoStages(0))

	if n, err := e.Tick(context.Background()); err != nil || n != 0 {
		t.Fatalf("Tick() = %d, %v; want 0, nil", n, err)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("sent %+v after a lost advance", sender.sent)
	}
}

func TestNewRequiresItsPorts(t *testing.T) {
	if _, err := escalation.New(escalation.Config{}); err == nil {
		t.Fatal("New with no store, sender or ladders succeeded")
	}
}

// syncSender reports each escalation on a channel, for the engine loop test.
type syncSender struct{ got chan int64 }

func (s syncSender) Escalate(_ context.Context, a *alerts.Alert, _ []alerts.Channel) {
	s.got <- a.ID
}

// The engine runs Tick on its own interval between Start and Stop, and Stop
// waits for the loop to exit.
func TestEngineEscalatesOnItsIntervalUntilStopped(t *testing.T) {
	raised := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{alerts: map[int64]*alerts.Alert{1: {ID: 1, Rule: "iface.down", CreatedAt: raised}}}
	sender := syncSender{got: make(chan int64, 1)}
	e, err := escalation.New(escalation.Config{
		Store: store, Sender: sender,
		Ladders:  func() []escalation.Ladder { return []escalation.Ladder{twoStages(0)} },
		Now:      func() time.Time { return raised.Add(time.Hour) },
		Interval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.Name() != escalation.EngineName {
		t.Fatalf("Name() = %q", e.Name())
	}
	ctx := context.Background()
	if err = e.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err = e.Start(ctx); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	select {
	case id := <-sender.got:
		if id != 1 {
			t.Fatalf("escalated alert %d, want 1", id)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the engine never escalated")
	}
	if err = e.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err = e.Stop(ctx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}
