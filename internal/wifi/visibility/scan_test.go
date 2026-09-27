package visibility_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// Documentation-range BSSIDs (RFC 7042); the repo is public.
const (
	scanBSSID    = "00:00:5e:00:53:10"
	capturedBSID = "00:00:5e:00:53:20"
)

func scannedBSS(bssid, ssid, security string) airspace.BSSView {
	return airspace.BSSView{
		BSSID:    bssid,
		SSID:     ssid,
		Band:     dot11.Band24GHz.String(),
		Channel:  6,
		Security: security,
		Stations: []airspace.StationView{},
	}
}

func bssids(tree []airspace.SSIDGroup) map[string]airspace.BSSView {
	out := map[string]airspace.BSSView{}
	for _, g := range tree {
		for _, ap := range g.APs {
			for _, b := range ap.BSSes {
				out[b.BSSID] = b
			}
		}
	}
	return out
}

// TestScanOnlyAirspaceRaisesAnomalies is #2351's point: with no capture source
// at all, a scanned transition-mode BSS still reaches the rules.
func TestScanOnlyAirspaceRaisesAnomalies(t *testing.T) {
	store := newFakeAnomalyStore()
	svc := visibility.New(visibility.WithCoordinator(wifiCoord(t, store)))
	now := time.Now()

	svc.IngestScan([]airspace.BSSView{scannedBSS(scanBSSID, "corp", "WPA2/WPA3")}, now)
	svc.Evaluate(t.Context(), now)

	if got := bssids(svc.Tree()); len(got) != 1 {
		t.Fatalf("Tree holds %d BSSes, want the 1 scanned", len(got))
	}
	st := svc.Status()
	if st.CaptureActive || st.BSSes != 1 || !st.LastScan.Equal(now) {
		t.Errorf("Status = %+v, want captureActive=false, bsses=1, lastScan=%v", st, now)
	}
	persisted := recordsToAnomalies(store.snapshot())
	if !hasAnomaly(persisted, wifianomaly.DefWPA3TransitionDowngrade) {
		t.Errorf("persisted anomalies = %v, want %s", persisted, wifianomaly.DefWPA3TransitionDowngrade)
	}
}

// TestCapturedBSSWinsOverScan keeps the stations only capture can see, and
// still adds the BSSes only the scan saw.
func TestCapturedBSSWinsOverScan(t *testing.T) {
	svc := visibility.New()
	now := time.Now()
	f := beacon(t, capturedBSID, "corp", dot11.SecurityWPA2)
	svc.Ingest(f, now)
	svc.Ingest(&dot11.Frame{
		Kind:        dot11.KindData,
		ToDS:        true,
		Receiver:    mac(t, capturedBSID),
		Transmitter: mac(t, "00:00:5e:00:53:99"),
		BSSID:       mac(t, capturedBSID),
	}, now)

	svc.IngestScan([]airspace.BSSView{
		scannedBSS(capturedBSID, "corp", "Open"),
		scannedBSS(scanBSSID, "lab", "WPA2"),
	}, now)

	got := bssids(svc.Tree())
	if len(got) != 2 {
		t.Fatalf("Tree holds %d BSSes, want 2 (one captured, one scanned)", len(got))
	}
	c := got[capturedBSID]
	if c.Security != "WPA2" || len(c.Stations) != 1 {
		t.Errorf("captured BSS = security %q, %d stations; want the capture's WPA2 and 1 station",
			c.Security, len(c.Stations))
	}
	if _, ok := got[scanBSSID]; !ok {
		t.Error("scan-only BSS missing from the merged tree")
	}
}

// TestScanSnapshotExpires drops a snapshot no scan has refreshed within the
// retention window, so a radio that stopped scanning does not pin old BSSes.
func TestScanSnapshotExpires(t *testing.T) {
	svc := visibility.New(visibility.WithRetention(time.Minute))
	start := time.Now()
	svc.IngestScan([]airspace.BSSView{scannedBSS(scanBSSID, "corp", "WPA2")}, start)

	svc.Evaluate(t.Context(), start.Add(59*time.Second))
	if len(svc.Tree()) == 0 {
		t.Fatal("snapshot expired inside the retention window")
	}
	svc.Evaluate(t.Context(), start.Add(61*time.Second))
	if n := len(svc.Tree()); n != 0 {
		t.Errorf("Tree has %d SSIDs after the retention window, want 0", n)
	}
}

// TestRunScansUntilCapture scans at start and on the interval, keeps the last
// good snapshot when a scan fails, and stands down while capture feeds frames.
func TestRunScansUntilCapture(t *testing.T) {
	var calls atomic.Int32
	var fail atomic.Bool
	svc := visibility.New(
		visibility.WithEvalInterval(time.Hour),
		visibility.WithScanInterval(5*time.Millisecond),
	)
	svc.SetScanSource(func() ([]airspace.BSSView, error) {
		calls.Add(1)
		if fail.Load() {
			return nil, errors.New("radio gone")
		}
		return []airspace.BSSView{scannedBSS(scanBSSID, "corp", "WPA2")}, nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	waitFor(t, "a first scan", func() bool { return len(svc.Tree()) == 1 })

	fail.Store(true)
	after := calls.Load()
	waitFor(t, "a failed scan", func() bool { return calls.Load() > after+1 })
	if len(svc.Tree()) != 1 {
		t.Error("a failed scan discarded the last good snapshot")
	}

	svc.SetSource("mon0")
	time.Sleep(20 * time.Millisecond) // let an in-flight scan finish
	stopped := calls.Load()
	time.Sleep(50 * time.Millisecond)
	if n := calls.Load(); n != stopped {
		t.Errorf("scanned %d times while capture was active, want 0", n-stopped)
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func recordsToAnomalies(recs []anomaly.Record) []anomaly.Anomaly {
	out := make([]anomaly.Anomaly, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.Anomaly)
	}
	return out
}
