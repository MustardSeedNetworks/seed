package delivery_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// This is the row's acceptance in a test: a real threshold breach, through the
// real observation pipeline, arrives at an external receiver as a signed POST.
// Only the SNMP observations and the alert store are fakes — the rule
// evaluation, the alert it produces, the store decorator and the HTTP transport
// are the production code paths.

type recordingStore struct {
	mu      sync.Mutex
	created []*alerts.Alert
}

func (r *recordingStore) Create(_ context.Context, a *alerts.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, a)
	return nil
}

type staticObservations struct {
	rows []*observation.SNMPObservation
}

func (s *staticObservations) List(
	_ context.Context,
	opts observation.ListOptions,
) ([]*observation.SNMPObservation, error) {
	out := make([]*observation.SNMPObservation, 0, len(s.rows))
	for _, row := range s.rows {
		if opts.Kind != "" && row.Kind != opts.Kind {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

type memorySettings struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySettings) GetWithDefault(_ context.Context, key, def string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.values[key]; ok {
		return v, nil
	}
	return def, nil
}

func (m *memorySettings) Set(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}

func storageObservation(observed time.Time, usedBytes int) *observation.SNMPObservation {
	payload, err := json.Marshal(map[string]any{"Storage": []map[string]any{
		{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": usedBytes},
	}})
	if err != nil {
		panic(err)
	}
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: "t-1", Kind: "host_resources",
		ObservedAt: observed, PayloadJSON: string(payload),
	}
}

func TestThresholdBreachReachesAnExternalReceiver(t *testing.T) {
	type delivered struct {
		body      []byte
		signature string
		timestamp string
	}
	got := make(chan delivered, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		got <- delivered{
			body:      body,
			signature: r.Header.Get("X-Seed-Signature"),
			timestamp: r.Header.Get("X-Seed-Timestamp"),
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	notifier, err := delivery.New(delivery.Config{
		URL:    srv.URL,
		Secret: signingKey,
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	notifier.Start()
	defer notifier.Stop(context.Background())

	store := &recordingStore{}
	now := func() time.Time { return time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC) }

	p, err := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: &staticObservations{rows: []*observation.SNMPObservation{
			storageObservation(now().Add(-time.Minute), 800), // 80% — under the threshold
			storageObservation(now(), 970),                   // 97% — breach
		}},
		Alerts:   delivery.WrapWriter(store, notifier),
		Settings: &memorySettings{values: map[string]string{}},
		Logger:   slog.New(slog.DiscardHandler),
		Now:      now,
	})
	if err != nil {
		t.Fatalf("NewObservationPipeline: %v", err)
	}
	if scanErr := p.ScanOnce(context.Background()); scanErr != nil {
		t.Fatalf("ScanOnce: %v", scanErr)
	}

	store.mu.Lock()
	stored := len(store.created)
	store.mu.Unlock()
	if stored != 1 {
		t.Fatalf("pipeline stored %d alerts, want 1", stored)
	}

	select {
	case d := <-got:
		mac := hmac.New(sha256.New, []byte(signingKey))
		mac.Write([]byte(d.timestamp + "." + string(d.body)))
		if want := "sha256=" + hex.EncodeToString(mac.Sum(nil)); d.signature != want {
			t.Errorf("receiver could not verify the signature: got %q want %q", d.signature, want)
		}
		var payload struct {
			Alert alerts.Alert `json:"alert"`
		}
		if unmarshalErr := json.Unmarshal(d.body, &payload); unmarshalErr != nil {
			t.Fatalf("unmarshal envelope: %v", unmarshalErr)
		}
		if payload.Alert.Severity != alerts.SeverityCritical {
			t.Errorf("delivered severity = %q, want critical", payload.Alert.Severity)
		}
		if payload.Alert.Title == "" {
			t.Error("delivered alert has no title")
		}
		t.Logf("receiver got: severity=%s title=%q message=%q",
			payload.Alert.Severity, payload.Alert.Title, payload.Alert.Message)
	case <-time.After(5 * time.Second):
		t.Fatal("the threshold breach never reached the receiver")
	}
}
