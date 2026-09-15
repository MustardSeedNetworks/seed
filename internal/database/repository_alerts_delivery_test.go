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

// The delivery columns carry #368's "visible failed status" clause, so they are
// only useful if both read paths see them — Get for the detail pane, List for
// the inbox the operator is actually scanning.
func TestAlertDeliveryStatusRoundTrips(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	alert := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "Interface eth0 down on t-1", Message: "ifOperStatus up -> down",
		Source: "t-1", Rule: "iface.down",
		DeliveryStatus: alerts.DeliveryPending,
	}
	if err := repo.Create(ctx, alert); err != nil {
		t.Fatalf("create: %v", err)
	}

	attemptedAt := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	const reason = `receiver answered 500 for alert 1`
	if err := repo.RecordDelivery(ctx, alert.ID, alerts.DeliveryFailed, attemptedAt, reason); err != nil {
		t.Fatalf("record delivery: %v", err)
	}

	got, err := repo.Get(ctx, alert.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.DeliveryStatus != alerts.DeliveryFailed {
		t.Errorf("Get DeliveryStatus = %q, want %q", got.DeliveryStatus, alerts.DeliveryFailed)
	}
	if got.DeliveryError != reason {
		t.Errorf("Get DeliveryError = %q, want %q", got.DeliveryError, reason)
	}
	if got.DeliveryAttemptedAt == nil || !got.DeliveryAttemptedAt.Equal(attemptedAt) {
		t.Errorf("Get DeliveryAttemptedAt = %v, want %v", got.DeliveryAttemptedAt, attemptedAt)
	}

	listed, err := repo.List(ctx, alerts.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed %d alerts, want 1", len(listed))
	}
	if listed[0].DeliveryStatus != alerts.DeliveryFailed {
		t.Errorf("List DeliveryStatus = %q, want %q", listed[0].DeliveryStatus, alerts.DeliveryFailed)
	}
	if listed[0].DeliveryError != reason {
		t.Errorf("List DeliveryError = %q, want %q", listed[0].DeliveryError, reason)
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
	if got.DeliveryStatus != "" {
		t.Errorf("DeliveryStatus = %q, want empty", got.DeliveryStatus)
	}
	if got.DeliveryAttemptedAt != nil {
		t.Errorf("DeliveryAttemptedAt = %v, want nil", got.DeliveryAttemptedAt)
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
		ctx, 9999, alerts.DeliveryFailed, time.Now().UTC(), "receiver unreachable",
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
	manager.Apply(delivery.Config{
		URL:         receiver.URL,
		Secret:      "delivery-signing-material",
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
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
	var got *alerts.Alert
	for {
		var getErr error
		got, getErr = repo.Get(ctx, alert.ID)
		if getErr != nil {
			t.Fatalf("get: %v", getErr)
		}
		if got.DeliveryStatus == alerts.DeliveryFailed || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got.DeliveryStatus != alerts.DeliveryFailed {
		t.Fatalf("stored alert DeliveryStatus = %q after %d refused posts, want %q",
			got.DeliveryStatus, posts.Load(), alerts.DeliveryFailed)
	}
	if !strings.Contains(got.DeliveryError, "500") {
		t.Errorf("stored DeliveryError = %q, does not name the receiver's answer", got.DeliveryError)
	}
	if got.DeliveryAttemptedAt == nil {
		t.Error("stored alert has no delivery attempt time")
	}
	t.Logf("inbox reads: status=%s error=%q attempts=%d",
		got.DeliveryStatus, got.DeliveryError, posts.Load())
}
