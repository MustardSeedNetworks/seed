package database_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// deliveryOn is alert's outcome on channel, nil when the channel was never
// offered it.
func deliveryOn(alert *alerts.Alert, channel alerts.Channel) *alerts.Delivery {
	for i := range alert.Deliveries {
		if alert.Deliveries[i].Channel == channel {
			return &alert.Deliveries[i]
		}
	}
	return nil
}

// The delivery outcomes carry #368's "visible failed status" clause, so they
// are only useful if both read paths see them — Get for the detail pane, List
// for the inbox the operator is actually scanning — and only if each channel
// keeps its own (#2997): a delivered webhook must not overwrite a failed email.
func TestAlertDeliveryStatusRoundTripsPerChannel(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	alert := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "Interface eth0 down on t-1", Message: "ifOperStatus up -> down",
		Source: "t-1", Rule: "iface.down",
		Deliveries: []alerts.Delivery{
			{Channel: alerts.ChannelWebhook, Status: alerts.DeliveryPending},
			{Channel: alerts.ChannelEmail, Status: alerts.DeliveryPending},
		},
	}
	if err := repo.Create(ctx, alert); err != nil {
		t.Fatalf("create: %v", err)
	}

	attemptedAt := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	const reason = `RCPT TO noc@example.test: 550 5.1.1 no such mailbox`
	if err := repo.RecordDelivery(
		ctx,
		alert.ID,
		alerts.ChannelEmail,
		alerts.DeliveryFailed,
		attemptedAt,
		reason,
	); err != nil {
		t.Fatalf("record email delivery: %v", err)
	}
	if err := repo.RecordDelivery(
		ctx,
		alert.ID,
		alerts.ChannelWebhook,
		alerts.DeliveryDelivered,
		attemptedAt,
		"",
	); err != nil {
		t.Fatalf("record webhook delivery: %v", err)
	}

	got, err := repo.Get(ctx, alert.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	listed, err := repo.List(ctx, alerts.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed %d alerts, want 1", len(listed))
	}
	for path, read := range map[string]*alerts.Alert{"Get": got, "List": listed[0]} {
		email := deliveryOn(read, alerts.ChannelEmail)
		if email == nil || email.Status != alerts.DeliveryFailed || email.Error != reason ||
			email.AttemptedAt == nil || !email.AttemptedAt.Equal(attemptedAt) {
			t.Errorf("%s email delivery = %+v, want failed at %v with %q", path, email, attemptedAt, reason)
		}
		webhook := deliveryOn(read, alerts.ChannelWebhook)
		if webhook == nil || webhook.Status != alerts.DeliveryDelivered || webhook.Error != "" {
			t.Errorf("%s webhook delivery = %+v, want delivered with no error", path, webhook)
		}
		if len(read.Deliveries) != 2 || read.Deliveries[0].Channel != alerts.ChannelEmail {
			t.Errorf("%s deliveries = %+v, want both channels in channel order", path, read.Deliveries)
		}
	}
}

// An install with no receiver configured must read as "delivery never applied",
// not as a failure: unset is silent by design, and a warning badge on every row
// of an air-gapped inbox would be worse than no status at all.
func TestAlertWithNoDeliveryReadsEmptyNotFailed(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	alert := &alerts.Alert{
		Type: alerts.TypeSystem, Severity: alerts.SeverityInfo,
		Title: "Storage 81% on t-1", Message: "usage crossed the warning threshold",
		Source: "t-1", Rule: "storage.high",
	}
	if err := repo.Create(ctx, alert); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, alert.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Deliveries != nil {
		t.Errorf("Deliveries = %+v, want none", got.Deliveries)
	}
}

// Deleting an alert takes its delivery outcomes with it, so retention leaves
// no orphan rows behind.
func TestDeletingAnAlertDeletesItsDeliveries(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	alert := &alerts.Alert{
		Type: alerts.TypeSystem, Severity: alerts.SeverityInfo, Title: "t", Message: "m",
		CreatedAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Deliveries: []alerts.Delivery{{Channel: alerts.ChannelEmail, Status: alerts.DeliveryPending}},
	}
	if err := repo.Create(ctx, alert); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.DeleteOlderThan(ctx, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("prune: %v", err)
	}
	var left int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM alert_deliveries`).Scan(&left); err != nil {
		t.Fatalf("count deliveries: %v", err)
	}
	if left != 0 {
		t.Errorf("%d delivery rows outlived their alert", left)
	}
}

// Retention prunes by age, so a delivery that exhausted its bounded retries can
// finish after the alert it was for was deleted. That write must not be an
// error: nothing an operator can see is wrong.
func TestRecordDeliveryForAPrunedAlertIsNotAnError(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := db.Alerts().RecordDelivery(
		ctx, 9999, alerts.ChannelWebhook, alerts.DeliveryFailed, time.Now().UTC(), "receiver unreachable",
	); err != nil {
		t.Fatalf("record delivery for a pruned alert: %v", err)
	}
}

// The slice's acceptance without a fake in the write path: the production
// Notifier, the production decorator and the production SQLite repository, and
// a receiver that refuses every POST. What an operator would then read in the
// inbox is what this asserts.
func TestFailedDeliveryIsVisibleOnTheStoredAlert(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	var posts atomic.Int64
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer receiver.Close()

	manager := delivery.NewManager(repo, slog.New(slog.DiscardHandler))
	manager.ApplyWebhook(delivery.WebhookConfig{
		URL:         receiver.URL,
		Secret:      "delivery-signing-material",
		MaxAttempts: 2, Backoff: time.Millisecond,
	})
	defer manager.Stop(ctx)

	// The composition root's own stacking (Server.alertStore): correlation
	// annotates inside delivery, and the pending stamp only survives if
	// correlation passes the same alert pointer through to the repository.
	store := delivery.WrapWriter(
		correlation.WrapWriter(repo, correlation.Config{}),
		manager,
	)
	alert := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "Interface eth0 down on t-1", Message: "ifOperStatus up -> down",
		Source: "t-1", Rule: "iface.down",
	}
	if createErr := store.Create(ctx, alert); createErr != nil {
		t.Fatalf("create through the delivering writer: %v", createErr)
	}

	deadline := time.Now().Add(10 * time.Second)
	var got *alerts.Delivery
	for {
		read, getErr := repo.Get(ctx, alert.ID)
		if getErr != nil {
			t.Fatalf("get: %v", getErr)
		}
		got = deliveryOn(read, alerts.ChannelWebhook)
		if (got != nil && got.Status == alerts.DeliveryFailed) || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got == nil || got.Status != alerts.DeliveryFailed {
		t.Fatalf("stored webhook delivery = %+v after %d refused posts, want %q",
			got, posts.Load(), alerts.DeliveryFailed)
	}
	if !strings.Contains(got.Error, "500") {
		t.Errorf("stored delivery error = %q, does not name the receiver's answer", got.Error)
	}
	if got.AttemptedAt == nil {
		t.Error("stored delivery has no attempt time")
	}
	t.Logf("inbox reads: status=%s error=%q attempts=%d", got.Status, got.Error, posts.Load())
}
