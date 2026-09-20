package api_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// A device is promotable only when a stored credential answered it. Both SNMP
// exchanges can be the one that reached the device — the profiler's own probe
// fills Profile.SNMPInfo and the MIB collection fills SNMPData — so reading
// one of them alone promotes nothing on a sweep that took the other.
func TestPromotionViewsReadBothSNMPPathsAndSkipTheRest(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		nil,
		{IP: "10.44.40.9"}, // seen, answered no SNMP
		{IP: "", SNMPData: &discovery.SNMPFullData{
			Credential: snmp.CredentialRef{ID: "cred-a", Version: snmp.VersionV2c},
		}}, // no address to poll
		{IP: "10.44.40.8", SNMPData: &discovery.SNMPFullData{
			System: &snmp.SystemInfo{SysName: "core-sw"},
			Credential: snmp.CredentialRef{
				ID: "cred-a", Version: snmp.VersionV2c,
			},
		}},
		{IP: "10.44.40.7", DisplayName: "dhcp-name", Profile: &discovery.DeviceProfile{
			SNMPInfo: &discovery.SNMPInfo{
				SysName:    "edge-rtr",
				Credential: snmp.CredentialRef{ID: "cred-b", Version: snmp.VersionV3},
			},
		}},
	}

	got := api.ExportPromotionViews(devices)

	if len(got) != 2 {
		t.Fatalf("promotionViews() returned %d views, want 2: %+v", len(got), got)
	}
	if got[0].IP != "10.44.40.8" || got[0].Name != "core-sw" ||
		got[0].Credential.ID != "cred-a" || got[0].Credential.Version != snmp.VersionV2c {
		t.Errorf("collected view = %+v", got[0])
	}
	if got[1].IP != "10.44.40.7" || got[1].Credential.ID != "cred-b" ||
		got[1].Credential.Version != snmp.VersionV3 {
		t.Errorf("profiled view = %+v", got[1])
	}
	// sysName is the device's own answer and beats a name resolution gave it.
	if got[1].Name != "edge-rtr" {
		t.Errorf("name = %q, want the device's sysName", got[1].Name)
	}
}

// A device whose only name came from DNS or NetBIOS still has to reach the
// target list under that name; the repository rejects a nameless target.
func TestPromotionViewsFallBackToTheNameDiscoveryFound(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		{IP: "10.44.40.6", DisplayName: "printer.local", SNMPData: &discovery.SNMPFullData{
			Credential: snmp.CredentialRef{ID: "cred-a", Version: snmp.VersionV2c},
		}},
		{IP: "10.44.40.5", Hostname: "ups-1", SNMPData: &discovery.SNMPFullData{
			Credential: snmp.CredentialRef{ID: "cred-a", Version: snmp.VersionV2c},
		}},
	}

	got := api.ExportPromotionViews(devices)

	if len(got) != 2 {
		t.Fatalf("promotionViews() returned %d views, want 2", len(got))
	}
	if got[0].Name != "printer.local" || got[1].Name != "ups-1" {
		t.Errorf("names = %q, %q", got[0].Name, got[1].Name)
	}
}
