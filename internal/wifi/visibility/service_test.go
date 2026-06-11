package visibility_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

func mac(t *testing.T, s string) net.HardwareAddr {
	t.Helper()
	h, err := net.ParseMAC(s)
	if err != nil {
		t.Fatalf("ParseMAC(%q): %v", s, err)
	}
	return h
}

// beacon builds a synthetic beacon frame advertising one BSS.
func beacon(t *testing.T, bssid, ssid string, sec dot11.Security) *dot11.Frame {
	t.Helper()
	return &dot11.Frame{
		Kind:       dot11.KindBeacon,
		BSSID:      mac(t, bssid),
		Band:       dot11.Band24GHz,
		ChannelNum: 6,
		BSS: &dot11.BSS{
			SSID:     ssid,
			Security: sec,
			Standard: dot11.Standard80211ac,
		},
	}
}

func hasAnomaly(as []anomalyView, id string) bool {
	for _, a := range as {
		if a.DefKey == id {
			return true
		}
	}
	return false
}

// anomalyView is the minimal shape we assert on (mirrors anomaly.Anomaly).
type anomalyView = struct {
	DefKey string
}

func TestNewIsEmpty(t *testing.T) {
	svc, err := visibility.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := len(svc.Tree()); got != 0 {
		t.Errorf("fresh Tree len = %d, want 0", got)
	}
	if got := len(svc.Anomalies()); got != 0 {
		t.Errorf("fresh Anomalies len = %d, want 0", got)
	}
	st := svc.Status()
	if st.CaptureActive {
		t.Error("fresh service should not report capture active")
	}
}

func TestIngestAndEvaluateProducesAnomaly(t *testing.T) {
	svc, err := visibility.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Now()
	// An open network on 2.4 GHz channel 6: trips wifi-open-network (and is on a
	// valid non-overlapping channel, so no adjacent-channel noise).
	svc.Ingest(beacon(t, "00:11:22:33:44:55", "guest", dot11.SecurityOpen), now)
	svc.Evaluate(context.Background(), now)

	if len(svc.Tree()) == 0 {
		t.Fatal("Tree empty after ingesting a beacon")
	}
	got := svc.Anomalies()
	views := make([]anomalyView, len(got))
	for i, a := range got {
		views[i] = anomalyView{DefKey: a.DefKey}
	}
	if !hasAnomaly(views, wifianomaly.DefOpenNetwork) {
		t.Errorf("expected %s anomaly, got %+v", wifianomaly.DefOpenNetwork, got)
	}
	if svc.Status().Anomalies == 0 {
		t.Error("Status.Anomalies should be > 0 after evaluate")
	}
}

func TestEvaluateCoalesces(t *testing.T) {
	svc, err := visibility.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Now()
	b := beacon(t, "00:11:22:33:44:55", "guest", dot11.SecurityOpen)
	svc.Ingest(b, now)
	svc.Evaluate(context.Background(), now)
	svc.Ingest(b, now.Add(time.Second))
	svc.Evaluate(context.Background(), now.Add(time.Second))

	open := 0
	for _, a := range svc.Anomalies() {
		if a.DefKey == wifianomaly.DefOpenNetwork {
			open++
			if a.Count < 2 {
				t.Errorf("coalesced anomaly Count = %d, want >= 2", a.Count)
			}
		}
	}
	if open != 1 {
		t.Errorf("open-network instances = %d, want exactly 1 (coalesced)", open)
	}
}

func TestCustomDetectorAndOptions(t *testing.T) {
	// A tuned detector (co-channel threshold 2) plus the non-default options, to
	// exercise the option path and a multi-BSS evaluation.
	det := wifianomaly.NewDetector(wifianomaly.WithCoChannelThreshold(2))
	svc, err := visibility.New(
		visibility.WithDetector(det),
		visibility.WithRetention(time.Minute),
		visibility.WithCapabilities(wifianomaly.CapActiveTest),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Now()
	// Two WPA3/PMF BSSes sharing 2.4 GHz channel 6 → co-channel contention.
	for _, m := range []string{"00:11:22:33:44:01", "00:11:22:33:44:02"} {
		b := beacon(t, m, "corp", dot11.SecurityWPA3)
		b.BSS.PMFRequired = true
		svc.Ingest(b, now)
	}
	svc.Evaluate(context.Background(), now)

	found := false
	for _, a := range svc.Anomalies() {
		if a.DefKey == wifianomaly.DefCoChannelContention {
			found = true
		}
	}
	if !found {
		t.Errorf("expected co-channel-contention with threshold 2, got %+v", svc.Anomalies())
	}
	st := svc.Status()
	if st.BSSes != 2 || st.APs == 0 {
		t.Errorf("Status counts off: %+v", st)
	}
}

func TestSourceToggle(t *testing.T) {
	svc, err := visibility.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	svc.SetSource("monitor0")
	st := svc.Status()
	if !st.CaptureActive || st.Source != "monitor0" {
		t.Errorf("after SetSource: CaptureActive=%v Source=%q", st.CaptureActive, st.Source)
	}
	svc.ClearSource()
	if svc.Status().CaptureActive {
		t.Error("after ClearSource: capture should be inactive")
	}
}

func TestStartStopLifecycle(t *testing.T) {
	svc, err := visibility.New(visibility.WithEvalInterval(5 * time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if startErr := svc.Start(t.Context()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	svc.Ingest(beacon(t, "00:11:22:33:44:55", "guest", dot11.SecurityOpen), time.Now())
	// Give the ticker a few cycles to evaluate.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(svc.Anomalies()) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(svc.Anomalies()) == 0 {
		t.Error("background loop did not evaluate ingested frames")
	}
	if stopErr := svc.Stop(); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}
	// Stop is idempotent.
	if stopErr := svc.Stop(); stopErr != nil {
		t.Fatalf("second Stop: %v", stopErr)
	}
}

// fakeAnomalyStore is an in-memory anomaly.Store for the persistence tests.
type fakeAnomalyStore struct {
	mu       sync.Mutex
	rows     map[string]anomaly.Record
	resolved map[string]time.Time
	seed     []anomaly.Record // returned by LoadActive
}

func newFakeAnomalyStore() *fakeAnomalyStore {
	return &fakeAnomalyStore{rows: map[string]anomaly.Record{}, resolved: map[string]time.Time{}}
}

func (f *fakeAnomalyStore) Upsert(_ context.Context, recs []anomaly.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range recs {
		f.rows[r.ID] = r
	}
	return nil
}

func (f *fakeAnomalyStore) MarkResolved(_ context.Context, ids []string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range ids {
		f.resolved[id] = at
		delete(f.rows, id)
	}
	return nil
}

func (f *fakeAnomalyStore) LoadActive(_ context.Context) ([]anomaly.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seed, nil
}

func (f *fakeAnomalyStore) snapshot() []anomaly.Record {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]anomaly.Record, 0, len(f.rows))
	for _, r := range f.rows {
		out = append(out, r)
	}
	return out
}

// TestPersistsAnomaliesToStore asserts that with a store configured, evaluating
// an open-network beacon writes the detected anomaly through to the store tagged
// with the Wi-Fi source (ADR-0021 phase 3 producer).
func TestPersistsAnomaliesToStore(t *testing.T) {
	fake := newFakeAnomalyStore()
	svc, err := visibility.New(visibility.WithStore(fake))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Now()
	svc.Ingest(beacon(t, "00:11:22:33:44:55", "guest", dot11.SecurityOpen), now)
	svc.Evaluate(context.Background(), now)

	rows := fake.snapshot()
	if len(rows) == 0 {
		t.Fatal("no anomalies persisted after evaluate")
	}
	var foundOpen bool
	for _, r := range rows {
		if r.Source != anomaly.SourceWiFi {
			t.Errorf("persisted source = %q, want wifi", r.Source)
		}
		if r.Anomaly.DefKey == wifianomaly.DefOpenNetwork {
			foundOpen = true
		}
	}
	if !foundOpen {
		t.Errorf("expected a persisted %s anomaly, got %+v", wifianomaly.DefOpenNetwork, rows)
	}
}

// TestLoadOnStartRestoresAnomalies asserts Start repopulates the engine from the
// store's active instances so a restart does not lose live anomalies.
func TestLoadOnStartRestoresAnomalies(t *testing.T) {
	fake := newFakeAnomalyStore()
	t0 := time.Now().Add(-time.Minute)
	fake.seed = []anomaly.Record{{
		ID: wifianomaly.DefOpenNetwork + "|bssid|00:11:22:33:44:55", Source: anomaly.SourceWiFi,
		Anomaly: anomaly.Anomaly{
			DefKey:    wifianomaly.DefOpenNetwork,
			Severity:  anomaly.SeverityWarning,
			Subject:   anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "00:11:22:33:44:55"},
			FirstSeen: t0, LastSeen: t0, Count: 3,
		},
	}}
	// Long eval interval so the background loop does not evaluate during the test;
	// load-on-start runs synchronously inside Start.
	svc, err := visibility.New(visibility.WithStore(fake), visibility.WithEvalInterval(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if startErr := svc.Start(t.Context()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	defer func() { _ = svc.Stop() }()

	anoms := svc.Anomalies()
	if len(anoms) != 1 || anoms[0].DefKey != wifianomaly.DefOpenNetwork {
		t.Fatalf("restored anomalies = %+v, want one %s", anoms, wifianomaly.DefOpenNetwork)
	}
	if anoms[0].Count != 3 {
		t.Errorf("restored count = %d, want 3", anoms[0].Count)
	}
	if svc.Status().Anomalies != 1 {
		t.Errorf("Status.Anomalies = %d, want 1", svc.Status().Anomalies)
	}
}
