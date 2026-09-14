package delivery_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// These are #368's "visible failed status" clause as tests: the outcome of a
// delivery has to reach the alert row, because the alert row is the only place
// an operator looks. The recorder here stands in for the alert repository; the
// repository's own write is covered in internal/database.

// recordingRecorder captures the per-alert write-backs the Notifier makes.
type recordingRecorder struct {
	mu      sync.Mutex
	writes  []recordedDelivery
	failErr error
}

type recordedDelivery struct {
	alertID     int64
	status      string
	attemptedAt time.Time
	errText     string
}

func (r *recordingRecorder) RecordDelivery(
	_ context.Context,
	alertID int64,
	status string,
	attemptedAt time.Time,
	deliveryErr string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes = append(r.writes, recordedDelivery{alertID, status, attemptedAt, deliveryErr})
	return r.failErr
}

func (r *recordingRecorder) last() (recordedDelivery, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.writes) == 0 {
		return recordedDelivery{}, false
	}
	return r.writes[len(r.writes)-1], true
}

func (r *recordingRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.writes)
}

func TestDeliveredOutcomeReachesTheAlertRow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	recorder := &recordingRecorder{}
	store := &recordingStore{}
	writer := newTestWriter(t, store, recorder, delivery.Config{URL: srv.URL, Secret: signingKey})

	alert := testAlert()
	alert.ID = 0 // the store assigns it, as the repository does
	if err := writer.Create(context.Background(), alert); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The row is written once, with the insert, so the inbox never shows an
	// alert whose delivery state is missing rather than pending.
	if alert.DeliveryStatus != alerts.DeliveryPending {
		t.Errorf("stored alert DeliveryStatus = %q, want %q", alert.DeliveryStatus, alerts.DeliveryPending)
	}

	if !waitFor(t, func() bool {
		w, ok := recorder.last()
		return ok && w.status == alerts.DeliveryDelivered
	}) {
		got, _ := recorder.last()
		t.Fatalf("delivery outcome never recorded; last write = %+v", got)
	}

	w, _ := recorder.last()
	if w.alertID != alert.ID {
		t.Errorf("recorded against alert %d, want %d", w.alertID, alert.ID)
	}
	if w.errText != "" {
		t.Errorf("successful delivery recorded an error: %q", w.errText)
	}
	if w.attemptedAt.IsZero() {
		t.Error("successful delivery recorded no attempt time")
	}
}

func TestFailedDeliveryLeavesAVisibleReasonOnTheAlert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	recorder := &recordingRecorder{}
	store := &recordingStore{}
	writer := newTestWriter(t, store, recorder, delivery.Config{
		URL: srv.URL, Secret: signingKey, MaxAttempts: 2, Backoff: time.Millisecond,
	})

	alert := testAlert()
	alert.ID = 0
	if err := writer.Create(context.Background(), alert); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !waitFor(t, func() bool {
		w, ok := recorder.last()
		return ok && w.status == alerts.DeliveryFailed
	}) {
		got, _ := recorder.last()
		t.Fatalf("exhausted retries never recorded a failure; last write = %+v", got)
	}

	w, _ := recorder.last()
	if !strings.Contains(w.errText, "500") {
		t.Errorf("recorded error %q does not name the receiver's status", w.errText)
	}
}

func TestQueueFullIsRecordedAsDroppedNotPending(t *testing.T) {
	// A receiver that never answers holds the single worker, so the second
	// alert fills the one-deep queue and the third has nowhere to go.
	block := make(chan struct{})
	defer close(block)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	recorder := &recordingRecorder{}
	store := &recordingStore{}
	writer := newTestWriter(t, store, recorder, delivery.Config{
		URL: srv.URL, Secret: signingKey, QueueSize: 1,
	})

	var dropped *alerts.Alert
	for range 8 {
		alert := testAlert()
		alert.ID = 0
		if err := writer.Create(context.Background(), alert); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if w, ok := recorder.last(); ok && w.status == alerts.DeliveryDropped && w.alertID == alert.ID {
			dropped = alert
			break
		}
	}
	if dropped == nil {
		t.Fatal("a full queue never produced a dropped delivery record")
	}
	// Dropped must be recorded synchronously: the worker is blocked, so
	// nothing else will ever revisit this alert and a pending row would stay
	// pending forever.
	if dropped.DeliveryStatus != alerts.DeliveryPending {
		t.Errorf("the stored row read %q, want the insert to carry %q",
			dropped.DeliveryStatus, alerts.DeliveryPending)
	}
}

func TestNoReceiverConfiguredRecordsNothing(t *testing.T) {
	recorder := &recordingRecorder{}
	store := &recordingStore{}

	// WrapWriter with no notifier is what the composition root builds when
	// SEED_ALERT_WEBHOOK_URL is unset — the air-gapped default.
	writer := delivery.WrapWriter(store, nil)

	alert := testAlert()
	alert.ID = 0
	if err := writer.Create(context.Background(), alert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if alert.DeliveryStatus != "" {
		t.Errorf("an install with no receiver stamped DeliveryStatus = %q, want empty",
			alert.DeliveryStatus)
	}
	if recorder.count() != 0 {
		t.Errorf("an install with no receiver made %d delivery writes, want 0", recorder.count())
	}
}

// newTestWriter builds a started Notifier over cfg and returns the wrapped
// store, stopping the notifier when the test ends.
func newTestWriter(
	t *testing.T,
	store delivery.Writer,
	recorder delivery.Recorder,
	cfg delivery.Config,
) delivery.Writer {
	t.Helper()
	cfg.Recorder = recorder
	cfg.Logger = slog.New(slog.DiscardHandler)
	notifier, err := delivery.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	notifier.Start()
	t.Cleanup(func() { notifier.Stop(context.Background()) })
	return delivery.WrapWriter(store, notifier)
}
