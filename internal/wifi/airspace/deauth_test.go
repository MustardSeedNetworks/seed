package airspace_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
)

// deauth builds a deauthentication management frame from an AP (Address3=BSSID).
func deauth(t *testing.T, bssid string) *dot11.Frame {
	t.Helper()
	return &dot11.Frame{
		Kind:      dot11.KindDeauth,
		BSSID:     mac(t, bssid),
		Band:      dot11.Band24GHz,
		SignalDBm: -55,
	}
}

func bssRecentDeauths(t *testing.T, a *airspace.Airspace, bssid string) (int, bool) {
	t.Helper()
	for _, g := range a.Tree() {
		for _, ap := range g.APs {
			for _, b := range ap.BSSes {
				if b.BSSID == bssid {
					return b.RecentDeauths, true
				}
			}
		}
	}
	return 0, false
}

func TestObserveCountsDeauths(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	const bssid = "00:11:22:33:44:01"
	a.Observe(beacon(t, bssid, "CorpWiFi", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime())
	for i := range 3 {
		a.Observe(deauth(t, bssid), baseTime().Add(time.Duration(i)*time.Second))
	}

	got, ok := bssRecentDeauths(t, a, bssid)
	if !ok {
		t.Fatalf("BSS %s not in tree", bssid)
	}
	if got != 3 {
		t.Errorf("RecentDeauths = %d, want 3", got)
	}
}

func TestDeauthOnlyBSSAppearsThenAgesOut(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	const bssid = "aa:bb:cc:00:00:01"
	// No beacon ever decoded — a pure deauth burst still surfaces the BSSID.
	a.Observe(deauth(t, bssid), baseTime())
	a.Observe(deauth(t, bssid), baseTime().Add(time.Second))

	if got, ok := bssRecentDeauths(t, a, bssid); !ok || got != 2 {
		t.Fatalf("RecentDeauths = %d (ok=%v), want 2", got, ok)
	}

	// Prune with a cutoff after the deauths: the window empties and the
	// beacon-less, station-less entry ages out entirely.
	a.Prune(baseTime().Add(time.Minute))
	if _, ok := bssRecentDeauths(t, a, bssid); ok {
		t.Error("deauth-only BSS should age out once its window empties")
	}
}

func TestPruneSlidesDeauthWindowOnBeaconedBSS(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	const bssid = "00:11:22:33:44:01"
	a.Observe(beacon(t, bssid, "CorpWiFi", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime())
	a.Observe(deauth(t, bssid), baseTime())                     // old
	a.Observe(deauth(t, bssid), baseTime().Add(10*time.Second)) // recent
	a.Observe(beacon(t, bssid, "CorpWiFi", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime().Add(20*time.Second))

	// Cutoff drops the first deauth only; the beaconed BSS survives with the
	// remaining one (the window slides even though the BSS itself stays alive).
	a.Prune(baseTime().Add(5 * time.Second))
	got, ok := bssRecentDeauths(t, a, bssid)
	if !ok {
		t.Fatalf("beaconed BSS %s should survive prune", bssid)
	}
	if got != 1 {
		t.Errorf("RecentDeauths after slide = %d, want 1", got)
	}
}
