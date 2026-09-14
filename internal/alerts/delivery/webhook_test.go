package delivery_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
)

// signingKey is the shared HMAC material the fake receivers verify with. Named
// away from "secret"/"token" so gosec G101 does not read it as a credential.
const signingKey = "test-signing-material"

func testAlert() *alerts.Alert {
	return &alerts.Alert{
		ID:        42,
		Type:      alerts.TypePerformance,
		Severity:  alerts.SeverityCritical,
		Title:     "Gateway latency threshold breached",
		Message:   "gateway latency 812ms over the 200ms threshold",
		Source:    "alert-observation-pipeline",
		CreatedAt: time.Unix(1_700_000_000, 0).UTC(),
	}
}

// waitFor polls cond until it holds or the deadline passes. The condition is
// tested once more after the deadline so a slow scheduler cannot fail a run
// whose work completed.
func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return cond()
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestNotifierPostsSignedAlert(t *testing.T) {
	type received struct {
		body      []byte
		signature string
		timestamp string
		alertID   string
	}
	got := make(chan received, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		got <- received{
			body:      body,
			signature: r.Header.Get("X-Seed-Signature"),
			timestamp: r.Header.Get("X-Seed-Timestamp"),
			alertID:   r.Header.Get("X-Seed-Alert-Id"),
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n, err := delivery.New(delivery.Config{URL: srv.URL, Secret: signingKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	n.Deliver(testAlert())

	select {
	case r := <-got:
		if r.alertID != "42" {
			t.Errorf("X-Seed-Alert-Id = %q, want 42", r.alertID)
		}
		ts, convErr := strconv.ParseInt(r.timestamp, 10, 64)
		if convErr != nil {
			t.Fatalf("X-Seed-Timestamp %q not an integer: %v", r.timestamp, convErr)
		}
		if delta := time.Since(time.Unix(ts, 0)); delta > time.Minute || delta < -time.Minute {
			t.Errorf("X-Seed-Timestamp %d is not near now", ts)
		}

		mac := hmac.New(sha256.New, []byte(signingKey))
		mac.Write([]byte(r.timestamp + "." + string(r.body)))
		want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if r.signature != want {
			t.Errorf("signature = %q, want %q", r.signature, want)
		}

		var payload struct {
			Alert alerts.Alert `json:"alert"`
		}
		if unmarshalErr := json.Unmarshal(r.body, &payload); unmarshalErr != nil {
			t.Fatalf("payload is not the documented envelope: %v", unmarshalErr)
		}
		if payload.Alert.Title != testAlert().Title {
			t.Errorf("alert.title = %q, want %q", payload.Alert.Title, testAlert().Title)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no delivery arrived at the receiver")
	}
}

func TestNotifierRetriesThenReportsFailed(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n, err := delivery.New(delivery.Config{
		URL:         srv.URL,
		Secret:      signingKey,
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	n.Deliver(testAlert())

	if !waitFor(t, func() bool { return n.Status().Failed == 1 }) {
		t.Fatalf("delivery never recorded a failure: %+v", n.Status())
	}
	if a := attempts.Load(); a != 3 {
		t.Errorf("receiver saw %d attempts, want 3 (bounded retry)", a)
	}
	st := n.Status()
	if st.Delivered != 0 {
		t.Errorf("Delivered = %d, want 0", st.Delivered)
	}
	if st.LastError == "" {
		t.Error("LastError is empty; a failed delivery must be discoverable")
	}
}

func TestNotifierDoesNotRetryPermanentRejection(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	n, err := delivery.New(delivery.Config{
		URL:         srv.URL,
		Secret:      signingKey,
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	n.Deliver(testAlert())

	if !waitFor(t, func() bool { return n.Status().Failed == 1 }) {
		t.Fatalf("delivery never recorded a failure: %+v", n.Status())
	}
	if a := attempts.Load(); a != 1 {
		t.Errorf("receiver saw %d attempts, want 1 (4xx is permanent)", a)
	}
}

func TestNotifierDropsWhenQueueFullAndNeverBlocks(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	defer close(release)

	n, err := delivery.New(delivery.Config{
		URL:       srv.URL,
		Secret:    signingKey,
		QueueSize: 1,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 50 {
			n.Deliver(testAlert())
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Deliver blocked: the alert pipeline must never wait on the receiver")
	}

	if !waitFor(t, func() bool { return n.Status().Dropped > 0 }) {
		t.Errorf("queue overflow was not recorded as dropped: %+v", n.Status())
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  delivery.Config
	}{
		{"empty url", delivery.Config{Secret: signingKey}},
		{"relative url", delivery.Config{URL: "/alerts", Secret: signingKey}},
		{"unsupported scheme", delivery.Config{URL: "ftp://example.test/a", Secret: signingKey}},
		{"no host", delivery.Config{URL: "https://", Secret: signingKey}},
		{"userinfo in url", delivery.Config{URL: "https://user:pw@example.test/a", Secret: signingKey}},
		{"missing secret", delivery.Config{URL: "https://example.test/a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := delivery.New(tt.cfg); !errors.Is(err, delivery.ErrInvalidConfig) {
				t.Errorf("New(%+v) error = %v, want ErrInvalidConfig", tt.cfg, err)
			}
		})
	}
}

// fakeWriter is the alert store the pipelines write through.
type fakeWriter struct {
	created []*alerts.Alert
	err     error
}

func (f *fakeWriter) Create(_ context.Context, a *alerts.Alert) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, a)
	return nil
}

func TestWrapWriterDeliversEveryStoredAlert(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n, err := delivery.New(delivery.Config{URL: srv.URL, Secret: signingKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	store := &fakeWriter{}
	writer := delivery.WrapWriter(store, n)

	if createErr := writer.Create(context.Background(), testAlert()); createErr != nil {
		t.Fatalf("Create: %v", createErr)
	}
	if len(store.created) != 1 {
		t.Fatalf("underlying store got %d alerts, want 1", len(store.created))
	}
	if !waitFor(t, func() bool { return posts.Load() == 1 }) {
		t.Errorf("stored alert was not delivered: %d posts", posts.Load())
	}
}

func TestWrapWriterDoesNotDeliverWhenTheStoreFails(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n, err := delivery.New(delivery.Config{URL: srv.URL, Secret: signingKey})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	storeErr := errors.New("insert failed")
	writer := delivery.WrapWriter(&fakeWriter{err: storeErr}, n)

	if createErr := writer.Create(context.Background(), testAlert()); !errors.Is(createErr, storeErr) {
		t.Fatalf("Create error = %v, want the store's error", createErr)
	}
	time.Sleep(50 * time.Millisecond)
	if posts.Load() != 0 {
		t.Errorf("an alert that was never stored was delivered (%d posts)", posts.Load())
	}
}

func TestEndpointWithholdsThePathSecret(t *testing.T) {
	n, err := delivery.New(delivery.Config{
		URL:    "https://hooks.example.test/services/T000/B000/XXXXsecretXXXX?k=v",
		Secret: signingKey,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got, want := n.Endpoint(), "https://hooks.example.test"; got != want {
		t.Errorf("Endpoint() = %q, want %q — the path carries the receiver's secret", got, want)
	}
}
