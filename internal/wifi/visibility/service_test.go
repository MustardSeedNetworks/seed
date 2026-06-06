package visibility_test

import (
	"net"
	"testing"
	"time"

	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/dot11"
	"github.com/krisarmstrong/seed/internal/wifi/visibility"
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
	svc.Evaluate(now)

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
	svc.Evaluate(now)
	svc.Ingest(b, now.Add(time.Second))
	svc.Evaluate(now.Add(time.Second))

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
	svc.Evaluate(now)

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
