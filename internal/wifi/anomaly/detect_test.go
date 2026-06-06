package wifianomaly_test

import (
	"sort"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/airspace"
	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/dot11"
)

// bss is a terse builder for an airspace.BSSView in tests.
func bss(bssid, ssid, security, band string, channel int) airspace.BSSView {
	return airspace.BSSView{
		BSSID:    bssid,
		SSID:     ssid,
		Security: security,
		Band:     band,
		Channel:  channel,
		Standard: "802.11ac (Wi-Fi 5)",
	}
}

// bssStd builds a clean WPA2/PMF BSS on a given 802.11 standard, for the
// standard-mismatch test.
func bssStd(bssid, ssid, std string, ch int) airspace.BSSView {
	b := bss(bssid, ssid, "WPA2", "5 GHz", ch)
	b.Standard = std
	b.PMFRequired = true
	return b
}

// tree wraps a single SSID served by one AP (one vendor) holding the given BSSes.
func tree(ssid string, bsses ...airspace.BSSView) []airspace.SSIDGroup {
	return []airspace.SSIDGroup{{
		SSID:    ssid,
		APCount: 1,
		APs:     []airspace.APGroup{{Key: "ap1", Vendor: "Cisco", BSSes: bsses}},
	}}
}

// defKeys returns the sorted set of def keys present in the detections.
func defKeys(dets []anomaly.Detection) []string {
	seen := map[string]bool{}
	for _, d := range dets {
		seen[d.DefKey] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func hasDef(dets []anomaly.Detection, id string) bool {
	for _, d := range dets {
		if d.DefKey == id {
			return true
		}
	}
	return false
}

func TestCatalogBuildsAndCoversEveryDefID(t *testing.T) {
	cat, err := wifianomaly.Catalog()
	if err != nil {
		t.Fatalf("Catalog() error: %v", err)
	}
	ids := []string{
		wifianomaly.DefOpenNetwork,
		wifianomaly.DefWEPInUse,
		wifianomaly.DefWPSEnabled,
		wifianomaly.DefPMFNotRequired,
		wifianomaly.DefSecurityMismatch,
		wifianomaly.DefEvilTwin,
		wifianomaly.DefCoChannelContention,
		wifianomaly.DefAdjacentChannelOverlap,
		wifianomaly.DefHiddenSSID,
		wifianomaly.DefCountryConflict,
		wifianomaly.DefStandardMismatch,
		wifianomaly.DefWPA3TransitionDowngrade,
		wifianomaly.DefDefaultSSIDName,
		wifianomaly.DefSSIDSprawl,
		wifianomaly.DefInconsistentRoaming,
		wifianomaly.DefRegulatoryViolation,
		wifianomaly.DefBSSLoadSaturation,
		wifianomaly.DefWideChannel24GHz,
		wifianomaly.DefChannelWidthMismatch,
		wifianomaly.DefDeauthFlood,
		wifianomaly.DefRogueAPOnLAN,
	}
	if cat.Len() != len(ids) {
		t.Errorf("catalog Len = %d, want %d (every exported def ID, no extras)", cat.Len(), len(ids))
	}
	for _, id := range ids {
		if _, ok := cat.Lookup(id); !ok {
			t.Errorf("catalog missing def %q", id)
		}
	}
}

// TestDetectionsAreAllCatalogued proves every rule emits only catalogued defs:
// feeding all detections through the real engine must never error.
func TestDetectionsAreAllCatalogued(t *testing.T) {
	cat, err := wifianomaly.Catalog()
	if err != nil {
		t.Fatalf("Catalog(): %v", err)
	}
	eng := anomaly.NewEngine(cat)
	dets := wifianomaly.NewDetector().Detect(everyAnomalyTree())
	if len(dets) == 0 {
		t.Fatal("expected detections from the kitchen-sink tree, got none")
	}
	now := time.Now()
	for _, d := range dets {
		if obsErr := eng.Observe(d, now); obsErr != nil {
			t.Errorf("engine rejected detection %+v: %v", d, obsErr)
		}
	}
}

// everyAnomalyTree is a synthetic airspace engineered to trip every rule.
func everyAnomalyTree() []airspace.SSIDGroup {
	return []airspace.SSIDGroup{
		// SSID with a strong + weak BSS (security-mismatch) served by two
		// different vendors (evil-twin) on different standards (standard-mismatch).
		{
			SSID:    "corp",
			APCount: 2,
			APs: []airspace.APGroup{
				{Key: "a", Vendor: "Cisco", BSSes: []airspace.BSSView{{
					BSSID: "00:11:22:00:00:01", SSID: "corp", Security: "WPA2",
					Band: "2.4 GHz", Channel: 6, Standard: "802.11ax (Wi-Fi 6)",
					CountryCode: "US",
				}}},
				{Key: "b", Vendor: "Netgear", BSSes: []airspace.BSSView{{
					BSSID: "aa:bb:cc:00:00:02", SSID: "corp", Security: "Open",
					Band: "2.4 GHz", Channel: 3, Standard: "802.11n (Wi-Fi 4)",
					CountryCode: "DE",
				}}},
			},
		},
		// A WEP BSS, a WPS BSS, an RSN-without-PMF BSS, a hidden BSS — and four
		// BSSes packed onto 2.4 GHz channel 6 (co-channel-contention).
		{
			SSID:    "lab",
			APCount: 1,
			APs: []airspace.APGroup{{Key: "c", Vendor: "Aruba", BSSes: []airspace.BSSView{
				{
					BSSID:      "00:00:00:00:00:10",
					SSID:       "lab",
					Security:   "WEP",
					Band:       "2.4 GHz",
					Channel:    6,
					Standard:   "802.11g",
					WPSEnabled: true,
				},
				{
					BSSID:       "00:00:00:00:00:11",
					SSID:        "lab",
					Security:    "WPA2",
					Band:        "2.4 GHz",
					Channel:     6,
					Standard:    "802.11ac (Wi-Fi 5)",
					PMFRequired: false,
				},
				{
					BSSID:       "00:00:00:00:00:12",
					SSID:        "lab",
					Security:    "WPA3",
					Band:        "2.4 GHz",
					Channel:     6,
					Standard:    "802.11ax (Wi-Fi 6)",
					PMFRequired: true,
				},
				{
					BSSID:       "00:00:00:00:00:13",
					SSID:        "lab",
					Security:    "WPA2",
					Band:        "2.4 GHz",
					Channel:     6,
					Standard:    "802.11ac (Wi-Fi 5)",
					PMFRequired: true,
				},
			}}},
		},
		// A cloaked SSID (hidden).
		{
			SSID:   "",
			Hidden: true,
			APs: []airspace.APGroup{{Key: "d", Vendor: "Ubiquiti", BSSes: []airspace.BSSView{
				{
					BSSID:       "00:00:00:00:00:20",
					SSID:        "",
					Hidden:      true,
					Security:    "WPA2",
					Band:        "5 GHz",
					Channel:     36,
					Standard:    "802.11ac (Wi-Fi 5)",
					PMFRequired: true,
				},
			}}},
		},
	}
}

func TestOpenAndWEPAndWPS(t *testing.T) {
	det := wifianomaly.NewDetector()

	open := det.Detect(tree("guest", bss("00:00:00:00:00:01", "guest", "Open", "5 GHz", 36)))
	if !hasDef(open, wifianomaly.DefOpenNetwork) {
		t.Errorf("open network not detected: %v", defKeys(open))
	}

	wep := det.Detect(tree("legacy", bss("00:00:00:00:00:02", "legacy", "WEP", "2.4 GHz", 6)))
	if !hasDef(wep, wifianomaly.DefWEPInUse) {
		t.Errorf("WEP not detected: %v", defKeys(wep))
	}

	b := bss("00:00:00:00:00:03", "home", "WPA2", "5 GHz", 36)
	b.WPSEnabled = true
	b.PMFRequired = true
	wps := det.Detect(tree("home", b))
	if !hasDef(wps, wifianomaly.DefWPSEnabled) {
		t.Errorf("WPS not detected: %v", defKeys(wps))
	}
}

func TestPMFNotRequiredOnlyForRSN(t *testing.T) {
	det := wifianomaly.NewDetector()

	// WPA2 without PMF → flagged.
	weak := bss("00:00:00:00:00:01", "corp", "WPA2", "5 GHz", 36)
	weak.PMFRequired = false
	if !hasDef(det.Detect(tree("corp", weak)), wifianomaly.DefPMFNotRequired) {
		t.Error("WPA2 without PMF should be flagged")
	}

	// Open network is not RSN — PMF is undefined, must NOT be flagged.
	if hasDef(
		det.Detect(tree("guest", bss("00:00:00:00:00:02", "guest", "Open", "5 GHz", 36))),
		wifianomaly.DefPMFNotRequired,
	) {
		t.Error("Open network must not raise pmf-not-required")
	}

	// WPA2 with PMF → clean.
	ok := bss("00:00:00:00:00:03", "corp", "WPA2", "5 GHz", 36)
	ok.PMFRequired = true
	if hasDef(det.Detect(tree("corp", ok)), wifianomaly.DefPMFNotRequired) {
		t.Error("WPA2 with PMF required must be clean")
	}
}

func TestSecurityMismatchAcrossSSID(t *testing.T) {
	det := wifianomaly.NewDetector()
	mixed := tree("corp",
		bss("00:00:00:00:00:01", "corp", "WPA2", "5 GHz", 36),
		bss("00:00:00:00:00:02", "corp", "Open", "5 GHz", 40),
	)
	if !hasDef(det.Detect(mixed), wifianomaly.DefSecurityMismatch) {
		t.Error("WPA2 + Open under one SSID should be a security-mismatch")
	}

	// Consistent strong security → no mismatch.
	consistent := tree("corp",
		bss("00:00:00:00:00:03", "corp", "WPA2", "5 GHz", 36),
		bss("00:00:00:00:00:04", "corp", "WPA2/WPA3", "5 GHz", 40),
	)
	if hasDef(det.Detect(consistent), wifianomaly.DefSecurityMismatch) {
		t.Error("WPA2 + WPA2/WPA3 are both strong; not a mismatch")
	}
}

func TestEvilTwinVendorMismatch(t *testing.T) {
	det := wifianomaly.NewDetector()
	twin := []airspace.SSIDGroup{{
		SSID:    "corp",
		APCount: 2,
		APs: []airspace.APGroup{
			{
				Key:    "a",
				Vendor: "Cisco",
				BSSes:  []airspace.BSSView{bss("00:00:00:00:00:01", "corp", "WPA2", "5 GHz", 36)},
			},
			{
				Key:    "b",
				Vendor: "TP-Link",
				BSSes:  []airspace.BSSView{bss("de:ad:be:00:00:02", "corp", "WPA2", "5 GHz", 40)},
			},
		},
	}}
	if !hasDef(det.Detect(twin), wifianomaly.DefEvilTwin) {
		t.Error("same SSID via two vendors should raise evil-twin")
	}

	// Single vendor across both APs → no evil-twin.
	same := []airspace.SSIDGroup{{
		SSID:    "corp",
		APCount: 2,
		APs: []airspace.APGroup{
			{
				Key:    "a",
				Vendor: "Cisco",
				BSSes:  []airspace.BSSView{bss("00:00:00:00:00:01", "corp", "WPA2", "5 GHz", 36)},
			},
			{
				Key:    "b",
				Vendor: "Cisco",
				BSSes:  []airspace.BSSView{bss("00:00:00:00:00:02", "corp", "WPA2", "5 GHz", 40)},
			},
		},
	}}
	if hasDef(det.Detect(same), wifianomaly.DefEvilTwin) {
		t.Error("one vendor is a normal multi-AP WLAN, not evil-twin")
	}
}

func TestCoChannelContentionThreshold(t *testing.T) {
	mk := func(n int) []airspace.SSIDGroup {
		bsses := make([]airspace.BSSView, n)
		for i := range bsses {
			bsses[i] = bss("00:00:00:00:00:0"+string(rune('a'+i)), "n", "WPA2", "2.4 GHz", 6)
			bsses[i].PMFRequired = true
		}
		return tree("n", bsses...)
	}
	det := wifianomaly.NewDetector(wifianomaly.WithCoChannelThreshold(3))
	if hasDef(det.Detect(mk(2)), wifianomaly.DefCoChannelContention) {
		t.Error("2 BSSes is below threshold 3")
	}
	if !hasDef(det.Detect(mk(3)), wifianomaly.DefCoChannelContention) {
		t.Error("3 BSSes meets threshold 3")
	}
}

func TestAdjacentChannelOverlap(t *testing.T) {
	det := wifianomaly.NewDetector()
	for _, ch := range []int{2, 3, 4, 5, 7, 9} {
		b := bss("00:00:00:00:00:01", "n", "WPA2", "2.4 GHz", ch)
		b.PMFRequired = true
		if !hasDef(det.Detect(tree("n", b)), wifianomaly.DefAdjacentChannelOverlap) {
			t.Errorf("2.4 GHz channel %d should overlap (not 1/6/11)", ch)
		}
	}
	for _, ch := range []int{1, 6, 11} {
		b := bss("00:00:00:00:00:01", "n", "WPA2", "2.4 GHz", ch)
		b.PMFRequired = true
		if hasDef(det.Detect(tree("n", b)), wifianomaly.DefAdjacentChannelOverlap) {
			t.Errorf("2.4 GHz channel %d is non-overlapping", ch)
		}
	}
	// 5 GHz is exempt (all channels non-overlapping at 20 MHz).
	b := bss("00:00:00:00:00:01", "n", "WPA2", "5 GHz", 44)
	b.PMFRequired = true
	if hasDef(det.Detect(tree("n", b)), wifianomaly.DefAdjacentChannelOverlap) {
		t.Error("5 GHz must not raise adjacent-channel-overlap")
	}
}

func TestHiddenSSID(t *testing.T) {
	det := wifianomaly.NewDetector()
	b := airspace.BSSView{
		BSSID:       "00:00:00:00:00:01",
		SSID:        "",
		Hidden:      true,
		Security:    "WPA2",
		Band:        "5 GHz",
		Channel:     36,
		Standard:    "802.11ac (Wi-Fi 5)",
		PMFRequired: true,
	}
	hidden := []airspace.SSIDGroup{
		{SSID: "", Hidden: true, APs: []airspace.APGroup{{Key: "a", Vendor: "Cisco", BSSes: []airspace.BSSView{b}}}},
	}
	if !hasDef(det.Detect(hidden), wifianomaly.DefHiddenSSID) {
		t.Error("cloaked BSS should raise hidden-ssid")
	}
}

func TestCountryConflict(t *testing.T) {
	det := wifianomaly.NewDetector()
	conflict := []airspace.SSIDGroup{
		{SSID: "a", APCount: 1, APs: []airspace.APGroup{{Key: "a", Vendor: "Cisco", BSSes: []airspace.BSSView{
			{
				BSSID:       "00:00:00:00:00:01",
				SSID:        "a",
				Security:    "WPA2",
				Band:        "5 GHz",
				Channel:     36,
				Standard:    "802.11ac (Wi-Fi 5)",
				CountryCode: "US",
				PMFRequired: true,
			},
		}}}},
		{SSID: "b", APCount: 1, APs: []airspace.APGroup{{Key: "b", Vendor: "Cisco", BSSes: []airspace.BSSView{
			{
				BSSID:       "00:00:00:00:00:02",
				SSID:        "b",
				Security:    "WPA2",
				Band:        "5 GHz",
				Channel:     40,
				Standard:    "802.11ac (Wi-Fi 5)",
				CountryCode: "JP",
				PMFRequired: true,
			},
		}}}},
	}
	if !hasDef(det.Detect(conflict), wifianomaly.DefCountryConflict) {
		t.Error("two regulatory domains in one airspace should raise country-conflict")
	}

	// Single consistent country → clean.
	same := []airspace.SSIDGroup{
		{SSID: "a", APCount: 1, APs: []airspace.APGroup{{Key: "a", Vendor: "Cisco", BSSes: []airspace.BSSView{
			{
				BSSID:       "00:00:00:00:00:01",
				SSID:        "a",
				Security:    "WPA2",
				Band:        "5 GHz",
				Channel:     36,
				Standard:    "802.11ac (Wi-Fi 5)",
				CountryCode: "US",
				PMFRequired: true,
			},
		}}}},
	}
	if hasDef(det.Detect(same), wifianomaly.DefCountryConflict) {
		t.Error("one country is not a conflict")
	}
}

func TestStandardMismatch(t *testing.T) {
	det := wifianomaly.NewDetector()
	mixed := tree("corp",
		bssStd("00:00:00:00:00:01", "corp", "802.11n (Wi-Fi 4)", 36),
		bssStd("00:00:00:00:00:02", "corp", "802.11ax (Wi-Fi 6)", 40),
	)
	if !hasDef(det.Detect(mixed), wifianomaly.DefStandardMismatch) {
		t.Error("mixed 802.11 generations under one SSID should raise standard-mismatch")
	}
}

func TestCleanAirspaceHasNoAnomalies(t *testing.T) {
	det := wifianomaly.NewDetector()
	clean := []airspace.SSIDGroup{{
		SSID:    "corp",
		APCount: 1,
		APs: []airspace.APGroup{{Key: "a", Vendor: "Cisco", BSSes: []airspace.BSSView{
			{
				BSSID:       "00:00:00:00:00:01",
				SSID:        "corp",
				Security:    "WPA3",
				Band:        "5 GHz",
				Channel:     36,
				Standard:    "802.11ax (Wi-Fi 6)",
				CountryCode: "US",
				PMFRequired: true,
			},
			{
				BSSID:       "00:00:00:00:00:02",
				SSID:        "corp",
				Security:    "WPA3",
				Band:        "5 GHz",
				Channel:     149,
				Standard:    "802.11ax (Wi-Fi 6)",
				CountryCode: "US",
				PMFRequired: true,
			},
		}}},
	}}
	if dets := det.Detect(clean); len(dets) != 0 {
		t.Errorf("clean WPA3 airspace should have zero anomalies, got %v", defKeys(dets))
	}
}

// TestSecurityTierCoversEveryDot11Suite guards against drift between the
// airspace Tree's stringified Security and the detector's string mapping: every
// dot11.Security String() value must be classified (weak/strong/unknown), never
// silently dropped.
func TestSecurityTierCoversEveryDot11Suite(t *testing.T) {
	suites := []dot11.Security{
		dot11.SecurityOpen, dot11.SecurityWEP, dot11.SecurityWPA,
		dot11.SecurityWPA2, dot11.SecurityWPA3, dot11.SecurityWPA2WPA3,
	}
	for _, s := range suites {
		if !wifianomaly.SecurityRecognized(s.String()) {
			t.Errorf("detector does not classify security suite %q", s.String())
		}
	}
}

// TestDeterministicOrder verifies Detect returns a stable order so callers and
// tests see identical output for identical input.
func TestDeterministicOrder(t *testing.T) {
	det := wifianomaly.NewDetector()
	in := everyAnomalyTree()
	first := det.Detect(in)
	for i := range 5 {
		got := det.Detect(in)
		if len(got) != len(first) {
			t.Fatalf("run %d length %d != %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j].DefKey != first[j].DefKey || got[j].Subject != first[j].Subject {
				t.Fatalf("run %d index %d: %+v != %+v", i, j, got[j], first[j])
			}
		}
	}
}
