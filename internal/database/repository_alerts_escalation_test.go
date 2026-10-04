package database_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/escalation"
)

// webhookPost is what the receiver saw: which alert, at which stage.
type webhookPost struct {
	id    int64
	stage int
}

// P-B2's acceptance as a test: an unacknowledged alert escalates through two
// stages on schedule and stops on acknowledge; a resolved alert never
// escalates. Everything but the clock is production code — the alert
// repository on SQLite, the delivery Manager, the signed webhook transport and
// an HTTP receiver.
func TestUnacknowledgedAlertEscalatesTwoStagesAndStopsOnAcknowledge(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	posts := escalationReceiver(t)

	logger := slog.New(slog.DiscardHandler)
	manager := delivery.NewManager(repo, logger)
	manager.ApplyWebhook(delivery.WebhookConfig{URL: posts.url, Secret: "escalation-test-signing-key"})
	defer manager.Stop(ctx)
	manager.ApplyEscalations([]escalation.Ladder{{
		Rule: "iface.down",
		Stages: []escalation.Stage{
			{After: 5 * time.Minute, Channels: []alerts.Channel{alerts.ChannelWebhook}},
			{After: 15 * time.Minute, Channels: []alerts.Channel{alerts.ChannelWebhook}},
		},
		// The repeat proves acknowledging stops the ladder for good, not
		// merely after its last stage.
		Repeat: 30 * time.Minute,
	}})

	raised := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	now := raised
	esc, err := escalation.New(escalation.Config{
		Store: repo, Sender: manager, Ladders: manager.Escalations, Logger: logger,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}

	writer := delivery.WrapWriter(repo, manager)
	open := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "Interface eth0 down on t-1", Source: "t-1", Rule: "iface.down", CreatedAt: raised,
	}
	resolved := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "Interface eth1 down on t-1", Source: "t-1", Rule: "iface.down", CreatedAt: raised,
	}
	for _, a := range []*alerts.Alert{open, resolved} {
		if err = writer.Create(ctx, a); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	if err = repo.Resolve(ctx, resolved.ID); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// The two first sends, as raised.
	got := map[webhookPost]bool{posts.next(t): true, posts.next(t): true}
	if !got[webhookPost{open.ID, 0}] || !got[webhookPost{resolved.ID, 0}] {
		t.Fatalf("first sends %v, want both alerts at stage 0", got)
	}

	tick := func(at time.Duration, want int) {
		t.Helper()
		now = raised.Add(at)
		n, tickErr := esc.Tick(ctx)
		if tickErr != nil || n != want {
			t.Fatalf("Tick at +%s = %d, %v; want %d sent", at, n, tickErr, want)
		}
	}

	tick(4*time.Minute, 0)
	tick(5*time.Minute, 1)
	if p := posts.next(t); p != (webhookPost{open.ID, 1}) {
		t.Fatalf("at +5m the receiver got %+v, want alert %d at stage 1", p, open.ID)
	}
	tick(10*time.Minute, 0)
	tick(15*time.Minute, 1)
	if p := posts.next(t); p != (webhookPost{open.ID, 2}) {
		t.Fatalf("at +15m the receiver got %+v, want alert %d at stage 2", p, open.ID)
	}

	stored, err := repo.Get(ctx, open.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.EscalationStage != 2 || stored.EscalatedAt == nil ||
		!stored.EscalatedAt.Equal(raised.Add(15*time.Minute)) {
		t.Fatalf("stored escalation stage %d at %v, want 2 at +15m", stored.EscalationStage, stored.EscalatedAt)
	}

	if err = repo.Acknowledge(ctx, open.ID, "noc"); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	tick(45*time.Minute, 0)
	tick(24*time.Hour, 0)

	never, err := repo.Get(ctx, resolved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if never.EscalationStage != 0 || never.EscalatedAt != nil {
		t.Fatalf("resolved alert escalated to stage %d at %v", never.EscalationStage, never.EscalatedAt)
	}
	posts.none(t)
}

// receiver is an HTTP webhook receiver that reports each alert it is sent.
type receiver struct {
	url   string
	posts chan webhookPost
}

func escalationReceiver(t *testing.T) *receiver {
	t.Helper()
	r := &receiver{posts: make(chan webhookPost, 16)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		var envelope struct {
			Alert alerts.Alert `json:"alert"`
		}
		if err = json.Unmarshal(body, &envelope); err != nil {
			t.Errorf("decode body: %v", err)
		}
		r.posts <- webhookPost{envelope.Alert.ID, envelope.Alert.EscalationStage}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	r.url = srv.URL
	return r
}

func (r *receiver) next(t *testing.T) webhookPost {
	t.Helper()
	select {
	case p := <-r.posts:
		return p
	case <-time.After(5 * time.Second):
		t.Fatal("the receiver got nothing")
		return webhookPost{}
	}
}

func (r *receiver) none(t *testing.T) {
	t.Helper()
	select {
	case p := <-r.posts:
		t.Fatalf("receiver got %+v after the alert was acknowledged", p)
	case <-time.After(200 * time.Millisecond):
	}
}

// The repeat is measured from the last send, and the compare-and-set on the
// stored stage keeps two passes that read the same row from both sending it.
func TestAdvanceEscalationIsACompareAndSet(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	a := &alerts.Alert{Type: alerts.TypeSystem, Severity: alerts.SeverityWarning, Title: "t", Rule: "r"}
	if err := repo.Create(ctx, a); err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 10, 4, 12, 5, 0, 0, time.UTC)

	moved, err := repo.AdvanceEscalation(ctx, a.ID, 0, nil, 1, first)
	if err != nil || !moved {
		t.Fatalf("first advance = %v, %v; want moved", moved, err)
	}
	if moved, err = repo.AdvanceEscalation(ctx, a.ID, 0, nil, 1, first); err != nil || moved {
		t.Fatalf("stale advance = %v, %v; want not moved", moved, err)
	}
	// A repeat of the last stage keeps the stage and needs the time it read.
	repeat := first.Add(30 * time.Minute)
	if moved, err = repo.AdvanceEscalation(ctx, a.ID, 1, &repeat, 1, repeat); err != nil || moved {
		t.Fatalf("repeat from the wrong time = %v, %v; want not moved", moved, err)
	}
	if moved, err = repo.AdvanceEscalation(ctx, a.ID, 1, &first, 1, repeat); err != nil || !moved {
		t.Fatalf("repeat = %v, %v; want moved", moved, err)
	}

	open, err := repo.ListEscalating(ctx, []string{"r", "other"})
	if err != nil || len(open) != 1 || open[0].EscalationStage != 1 || !open[0].EscalatedAt.Equal(repeat) {
		t.Fatalf("ListEscalating = %+v, %v; want the alert at stage 1, %v", open, err, repeat)
	}
	if err = repo.Acknowledge(ctx, a.ID, "noc"); err != nil {
		t.Fatal(err)
	}
	if moved, err = repo.AdvanceEscalation(ctx, a.ID, 1, &repeat, 2, repeat.Add(time.Hour)); err != nil || moved {
		t.Fatalf("advance after acknowledge = %v, %v; want not moved", moved, err)
	}
	if open, err = repo.ListEscalating(ctx, []string{"r"}); err != nil || len(open) != 0 {
		t.Fatalf("ListEscalating after acknowledge = %+v, %v; want none", open, err)
	}
}
