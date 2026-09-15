package topology_test

import (
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// roleOf runs one sys_info observation through the real reconciler
// and returns the role the node came out with. The classifier is
// unexported on purpose — what matters is the role a poll produces,
// not the signature of the function in the middle.
func roleOf(t *testing.T, sysDescr string, sysServices uint32) string {
	t.Helper()
	nodes := &fakeNodes{}
	r, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: &fakeObservations{rows: []*observation.SNMPObservation{
			roleObs("t-1", sysDescr, sysServices),
		}},
		Nodes: nodes, Settings: newFakeSettings(), Logger: silentLogger(), Now: at,
	})
	if err != nil {
		t.Fatalf("reconciler: %v", err)
	}
	if err = r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if len(nodes.upserts) != 1 {
		t.Fatalf("upserts = %d, want 1", len(nodes.upserts))
	}
	return nodes.upserts[0].DeviceType
}

// The recorded corpus is eight switches, so the roles it cannot reach
// are pinned here from the scalars the devices in the S4-4 hospital
// pack report. Those are the cases the pack itself confirms once the
// lab is reachable.
func TestDeviceRole_ClassifiesByWhatTheDeviceDoes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		descr    string
		services uint32
		want     string
	}{
		// Prose first: these all report a switch's service bits.
		{
			"wireless controller", "Cisco IOS Software, C9800 Wireless Controller",
			0x4E, topology.RoleWirelessController,
		},
		{
			"wlan controller spelling", "ArubaOS WLAN Controller 7220",
			0x4E, topology.RoleWirelessController,
		},
		{
			"access point", "Cisco CW9178I Access Point, IOS-XE",
			0x4E, topology.RoleAccessPoint,
		},
		{
			"firewall", "Palo Alto Networks PA-3220 firewall, PAN-OS 11.1",
			0x4E, topology.RoleFirewall,
		},
		{"printer", "HP LaserJet Enterprise M507 printer", 0x40, topology.RolePrinter},
		// Prose beats bits where they disagree: a router that also
		// bridges says so in its own description.
		{"router in prose over datalink bit", "Cisco 8200 Series Router", 0x06, topology.RoleRouter},
		// Bits, where the prose says nothing about the role.
		{"bridges only", "Arista Networks EOS version 4.28.3M", 0x02, topology.RoleSwitch},
		{"bridges and routes", "Some Vendor OS 1.0", 0x06, topology.RoleSwitch},
		{"routes without bridging", "Some Vendor OS 1.0", 0x04, topology.RoleRouter},
		{"application layer only", "Linux pump-01 6.5.0", 0x48, topology.RoleServer},
		{"physical layer only", "Media converter", 0x01, topology.RoleUnknown},
		// Neither scalar answered: an absence, not a verdict.
		{"nothing answered", "", 0, ""},
		{"descr but no bits", "Some Vendor OS 1.0", 0, topology.RoleUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := roleOf(t, tt.descr, tt.services); got != tt.want {
				t.Errorf("role of (%q, %#x) = %q, want %q",
					tt.descr, tt.services, got, tt.want)
			}
		})
	}
}

// The role strings are the same ones internal/discovery's profiler
// produces, because the UI branches on them for devices either
// subsystem may have classified (ui/src/hooks/useDiscoveredDevices.ts).
// Sharing a constant would couple the two packages, so this reads
// discovery's own source instead: respell one and this fails, rather
// than the UI quietly losing an icon.
func TestDeviceRole_VocabularyMatchesDiscovery(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("../discovery/profiler_types.go")
	if err != nil {
		t.Fatalf("read discovery vocabulary: %v", err)
	}
	shared := map[string]string{
		"deviceTypePrinter":  topology.RolePrinter,
		"deviceTypeServer":   topology.RoleServer,
		"deviceTypeRouter":   topology.RoleRouter,
		"deviceTypeSwitch":   topology.RoleSwitch,
		"deviceTypeFirewall": topology.RoleFirewall,
	}
	for constName, ours := range shared {
		re := regexp.MustCompile(constName + `\s*=\s*"([^"]+)"`)
		m := re.FindSubmatch(src)
		if m == nil {
			t.Errorf("%s not found in discovery/profiler_types.go; "+
				"the two vocabularies can no longer be compared", constName)
			continue
		}
		if string(m[1]) != ours {
			t.Errorf("%s = %q in discovery, %q in topology", constName, m[1], ours)
		}
	}
}
