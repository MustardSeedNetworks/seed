package enumerate

import "testing"

// The authorized-BSSID set decides whether a scanned access point is reported
// as authorized. It is keyed on a normalized MAC, so an operator who enters
// 74-AC-B9-3B-AF-40 must still match a scanner that reports 74:ac:b9:3b:af:40.
// Without the normalization the device reads as unauthorized -- a wrong answer
// on a security surface, not a cosmetic one.
func TestSetAuthorizedBSSIDs_NormalizesSoFormatDoesNotDecideAuthorization(t *testing.T) {
	bridge := &WiFiBridge{}

	bridge.SetAuthorizedBSSIDs([]string{
		"74-AC-B9-3B-AF-40", // hyphenated, upper
		"bc:24:11:0b:21:b2", // colon, lower
	})

	for _, form := range []string{
		"74:AC:B9:3B:AF:40",
		"74-ac-b9-3b-af-40",
		"74:ac:b9:3b:af:40",
	} {
		if !bridge.authorizedMACs[normalizeMAC(form)] {
			t.Errorf("%q did not match the authorized entry", form)
		}
	}
	if !bridge.authorizedMACs[normalizeMAC("BC-24-11-0B-21-B2")] {
		t.Error("the lower-case colon entry did not match an upper-case hyphenated lookup")
	}
	if bridge.authorizedMACs[normalizeMAC("00:00:00:00:00:00")] {
		t.Error("an address that was never authorized matched")
	}
}

// Both setters replace rather than merge. A caller lowering an authorization
// list must not leave the previous entries authorized.
func TestSetAuthorizedBSSIDs_ReplacesRatherThanMerges(t *testing.T) {
	bridge := &WiFiBridge{}

	bridge.SetAuthorizedBSSIDs([]string{"74:ac:b9:3b:af:40"})
	bridge.SetAuthorizedBSSIDs([]string{"bc:24:11:0b:21:b2"})

	if bridge.authorizedMACs[normalizeMAC("74:ac:b9:3b:af:40")] {
		t.Error("a revoked BSSID is still authorized after the list was replaced")
	}
	if len(bridge.authorizedMACs) != 1 {
		t.Errorf("authorized set has %d entries, want 1", len(bridge.authorizedMACs))
	}
}

func TestSetAuthorizedSSIDs_ReplacesRatherThanMerges(t *testing.T) {
	bridge := &WiFiBridge{}

	bridge.SetAuthorizedSSIDs([]string{"corp-wifi", "guest"})
	bridge.SetAuthorizedSSIDs([]string{"corp-wifi"})

	if bridge.authorizedSSIDs["guest"] {
		t.Error("a revoked SSID is still authorized after the list was replaced")
	}
	if !bridge.authorizedSSIDs["corp-wifi"] {
		t.Error("a retained SSID was dropped")
	}
}

// An empty list authorizes nothing, rather than leaving the previous set or
// being treated as "allow all".
func TestSetAuthorized_EmptyListRevokesEverything(t *testing.T) {
	bridge := &WiFiBridge{}
	bridge.SetAuthorizedSSIDs([]string{"corp-wifi"})
	bridge.SetAuthorizedBSSIDs([]string{"74:ac:b9:3b:af:40"})

	bridge.SetAuthorizedSSIDs(nil)
	bridge.SetAuthorizedBSSIDs(nil)

	if len(bridge.authorizedSSIDs) != 0 || len(bridge.authorizedMACs) != 0 {
		t.Errorf("clearing left %d SSIDs and %d BSSIDs authorized",
			len(bridge.authorizedSSIDs), len(bridge.authorizedMACs))
	}
}

func TestDefaultWiFiBridgeConfig(t *testing.T) {
	cfg := DefaultWiFiBridgeConfig()

	if cfg == nil {
		t.Fatal("nil config")
	}
	// A positive floor would exclude every real reading: signal is negative dBm.
	if cfg.MinSignalDBm >= 0 {
		t.Errorf("MinSignalDBm = %d, want a negative dBm floor", cfg.MinSignalDBm)
	}
	// Non-nil so a caller can append without a nil check.
	if cfg.AuthorizedSSIDs == nil || cfg.AuthorizedBSSIDs == nil {
		t.Error("authorization lists are nil, want empty slices")
	}
}
