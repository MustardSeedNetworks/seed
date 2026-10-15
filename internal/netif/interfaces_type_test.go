package netif_test

import (
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

// typingDetector reports one interface with a fixed type, the way the platform
// detector reports macOS's en0 once CoreWLAN names it a Wi-Fi radio.
type typingDetector struct {
	name, ifType string
}

func (d typingDetector) DetectAll() ([]detection.InterfaceScore, error) {
	return []detection.InterfaceScore{{Name: d.name, Type: d.ifType}}, nil
}

func firstNonLoopbackName(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces() error = %v", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 {
			return iface.Name
		}
	}
	t.Skip("no non-loopback interface on this host")
	return ""
}

// TestRefreshInterfacesTypesByDetection pins #3466: the interface listing and
// the Wi-Fi adapter list typed interfaces by name while the detection pass
// asked the platform, so macOS's en0 was "Wi-Fi (en0)" in its description and
// "ethernet" in its type, and the Wi-Fi status found no adapter.
func TestRefreshInterfacesTypesByDetection(t *testing.T) {
	tests := []struct {
		detected string
		want     netif.InterfaceType
	}{
		{detection.TypeWifi, netif.InterfaceTypeWiFi},
		{detection.TypeEthernet, netif.InterfaceTypeEthernet},
		{detection.TypeFiber, netif.InterfaceTypeEthernet},
		{detection.TypeVirtual, netif.InterfaceTypeVirtual},
		{detection.TypeOther, netif.InterfaceTypeOther},
	}

	name := firstNonLoopbackName(t)
	for _, tt := range tests {
		t.Run(tt.detected, func(t *testing.T) {
			mgr := netif.NewManagerForDetector(typingDetector{name: name, ifType: tt.detected}, time.Now)
			if err := mgr.RefreshInterfaces(); err != nil {
				t.Fatalf("RefreshInterfaces() error = %v", err)
			}
			info, err := mgr.GetInterface(name)
			if err != nil {
				t.Fatalf("GetInterface(%q) error = %v", name, err)
			}
			if info.Type != tt.want {
				t.Errorf("%s detected as %q has type %q, want %q", name, tt.detected, info.Type, tt.want)
			}
			if got, want := mgr.IsWireless(name), tt.want == netif.InterfaceTypeWiFi; got != want {
				t.Errorf("IsWireless(%q) = %v, want %v", name, got, want)
			}
		})
	}
}

// TestRefreshInterfacesTypesUnscoredInterfaces covers what the detection pass
// does not score: loopbacks, which it skips, and every interface after a failed
// first pass.
func TestRefreshInterfacesTypesUnscoredInterfaces(t *testing.T) {
	mgr := netif.NewManagerForDetector(typingDetector{name: "no-such-interface"}, time.Now)
	if err := mgr.RefreshInterfaces(); err != nil {
		t.Fatalf("RefreshInterfaces() error = %v", err)
	}
	ifaces, listErr := net.Interfaces()
	if listErr != nil {
		t.Fatalf("net.Interfaces() error = %v", listErr)
	}
	for _, iface := range ifaces {
		info, err := mgr.GetInterface(iface.Name)
		if err != nil {
			t.Fatalf("GetInterface(%q) error = %v", iface.Name, err)
		}
		want := netif.InterfaceType(detection.TypeByName(iface.Name))
		if want == detection.TypeFiber {
			want = netif.InterfaceTypeEthernet
		}
		if iface.Flags&net.FlagLoopback != 0 {
			want = netif.InterfaceTypeLoopback
		}
		if info.Type != want {
			t.Errorf("%s has type %q, want %q", iface.Name, info.Type, want)
		}
	}
}
