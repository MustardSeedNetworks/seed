package delivery_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// The Manager is what makes #2605's "no daemon restart" clause true: the store
// chain is wired once, and the receiver behind it is swapped in place.

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// countingReceiver is an httptest receiver that reports how many POSTs arrived.
func countingReceiver(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestManagerDeliversAfterApplyWithoutRestart(t *testing.T) {
	srv, hits := countingReceiver(t)
	m := delivery.NewManager(nil, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })

	// Before Apply the manager is the air-gapped default: no receiver, so the
	// alert is stored and nothing is sent.
	store := &recordingStore{}
	writer := delivery.WrapWriter(store, m)
	if err := writer.Create(context.Background(), &alerts.Alert{ID: 1}); err != nil {
		t.Fatalf("Create before Apply: %v", err)
	}
	if got := hits.Load(); got != 0 {
		t.Fatalf("unconfigured manager made %d requests, want 0", got)
	}
	if got := store.created[0].DeliveryStatus; got != "" {
		t.Fatalf("delivery status %q on an unconfigured manager, want empty "+
			"(empty means nobody ever tried to send this)", got)
	}

	m.Apply(delivery.Config{URL: srv.URL, Secret: "s3cret"})

	if err := writer.Create(context.Background(), &alerts.Alert{ID: 2}); err != nil {
		t.Fatalf("Create after Apply: %v", err)
	}
	if got := store.created[1].DeliveryStatus; got != alerts.DeliveryPending {
		t.Fatalf("delivery status %q after Apply, want %q", got, alerts.DeliveryPending)
	}
	if !waitFor(t, func() bool { return hits.Load() == 1 }) {
		t.Fatalf("timed out waiting for the alert to reach the receiver")
	}
}

func TestManagerApplyEmptyURLStopsDelivery(t *testing.T) {
	srv, hits := countingReceiver(t)
	m := delivery.NewManager(nil, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })
	store := &recordingStore{}
	writer := delivery.WrapWriter(store, m)

	m.Apply(delivery.Config{URL: srv.URL, Secret: "s3cret"})
	if err := writer.Create(context.Background(), &alerts.Alert{ID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !waitFor(t, func() bool { return hits.Load() == 1 }) {
		t.Fatalf("timed out waiting for the first alert to arrive")
	}

	m.Apply(delivery.Config{})
	if err := writer.Create(context.Background(), &alerts.Alert{ID: 2}); err != nil {
		t.Fatalf("Create after clearing the URL: %v", err)
	}
	if got := store.created[1].DeliveryStatus; got != "" {
		t.Fatalf("delivery status %q after the URL was cleared, want empty", got)
	}
	// Give a would-be delivery room to arrive before asserting it did not.
	time.Sleep(100 * time.Millisecond)
	if got := hits.Load(); got != 1 {
		t.Fatalf("receiver saw %d requests after the URL was cleared, want 1", got)
	}
}

func TestManagerApplyRepointsToTheNewReceiver(t *testing.T) {
	first, firstHits := countingReceiver(t)
	second, secondHits := countingReceiver(t)
	m := delivery.NewManager(nil, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })
	writer := delivery.WrapWriter(&recordingStore{}, m)

	m.Apply(delivery.Config{URL: first.URL, Secret: "s3cret"})
	m.Apply(delivery.Config{URL: second.URL, Secret: "s3cret"})
	if err := writer.Create(context.Background(), &alerts.Alert{ID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !waitFor(t, func() bool { return secondHits.Load() == 1 }) {
		t.Fatalf("timed out waiting for the alert to reach the second receiver")
	}
	if got := firstHits.Load(); got != 0 {
		t.Fatalf("the replaced receiver saw %d requests, want 0", got)
	}
}

func TestManagerApplyUnusableConfigDisablesDelivery(t *testing.T) {
	srv, hits := countingReceiver(t)
	m := delivery.NewManager(nil, quietLogger())
	t.Cleanup(func() { m.Stop(context.Background()) })
	writer := delivery.WrapWriter(&recordingStore{}, m)

	m.Apply(delivery.Config{URL: srv.URL, Secret: "s3cret"})
	// A URL with no signing material cannot produce a verifiable request. The
	// receiver must go away rather than keep receiving under the old secret.
	m.Apply(delivery.Config{URL: srv.URL})
	if err := writer.Create(context.Background(), &alerts.Alert{ID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if got := hits.Load(); got != 0 {
		t.Fatalf("receiver saw %d requests after an unusable Apply, want 0", got)
	}
	if m.Enabled() {
		t.Fatal("manager reports enabled after an unusable Apply")
	}
}
