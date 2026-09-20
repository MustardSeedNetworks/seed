package promote_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/promote"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// TestPromotesEverySNMPAnsweringDeviceOnce — the row's headline: a sweep that
// found devices answering SNMP must leave polling targets behind, so the
// topology fills without anyone POSTing a target list by hand (seed#2692).
func TestPromotesEverySNMPAnsweringDeviceOnce(t *testing.T) {
	devices := []promote.Device{
		{
			IP: "10.44.10.1", Name: "edge-rtr",
			Credential: snmp.CredentialRef{ID: "cred-a", Version: snmp.VersionV2c},
		},
		{
			IP:         "10.44.10.2",
			Credential: snmp.CredentialRef{ID: "cred-b", Version: snmp.VersionV3},
		},
	}

	got := promote.Targets(devices)

	if len(got) != 2 {
		t.Fatalf("promoted %d targets; want 2: %+v", len(got), got)
	}
	if got[0].Name != "edge-rtr" || got[0].CredentialID != "cred-a" ||
		got[0].SNMPVersion != snmp.VersionV2c {
		t.Errorf("first target %+v", got[0])
	}
	// A device SNMP named but DNS did not still has to be pollable, and a
	// target with no name is rejected by the repository.
	if got[1].Name != "10.44.10.2" {
		t.Errorf("unnamed device promoted as %q; want its address", got[1].Name)
	}
	if got[1].SNMPVersion != snmp.VersionV3 {
		t.Errorf("v3 device promoted as %q; polling it as v2c would fail silently",
			got[1].SNMPVersion)
	}
}

// TestPromotesNothingWithoutACredential — a device that answered no SNMP
// credential has nothing to poll it with: promoting it would create a target
// that fails every cycle. Nothing new means no write at all.
func TestPromotesNothingWithoutACredential(t *testing.T) {
	devices := []promote.Device{
		{IP: "10.44.10.3", Name: "printer"},
		{IP: "10.44.10.4", Credential: snmp.CredentialRef{Version: snmp.VersionV2c}},
		{IP: "", Credential: snmp.CredentialRef{ID: "cred-a", Version: snmp.VersionV2c}},
	}

	if got := promote.Targets(devices); len(got) != 0 {
		t.Fatalf("promoted %+v; want nothing", got)
	}
}

// TestPromotesNothingWithoutAVersion — a credential reference that names a row
// but no SNMP version says nothing about how to talk to the device, and the
// repository would default the target to v2c. Polling a v3-only device as v2c
// fails every cycle with no operator-visible cause.
func TestPromotesNothingWithoutAVersion(t *testing.T) {
	devices := []promote.Device{
		{IP: "10.44.10.6", Credential: snmp.CredentialRef{ID: "cred-a"}},
	}

	if got := promote.Targets(devices); len(got) != 0 {
		t.Fatalf("promoted %+v; want nothing", got)
	}
}

// TestPromotesOneTargetPerAddress — the same device reached twice in one sweep
// (two discovery methods, or a second interface) must not become two targets:
// the table has no unique index on the address.
func TestPromotesOneTargetPerAddress(t *testing.T) {
	cred := snmp.CredentialRef{ID: "cred-a", Version: snmp.VersionV2c}
	devices := []promote.Device{
		{IP: "10.44.10.5", Credential: cred},
		{IP: "10.44.10.5", Name: "same-box", Credential: cred},
	}

	if got := promote.Targets(devices); len(got) != 1 {
		t.Fatalf("promoted %+v; want one target", got)
	}
}
