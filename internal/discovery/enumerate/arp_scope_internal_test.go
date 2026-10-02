package enumerate

import (
	"net"
	"testing"
)

// isInSubnet decides which discovered addresses survive a scan, so its
// fallback is load-bearing: an unconfigured scanner accepts everything. That
// was relied on while diagnosing #2272 and was untested at the time.
func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("parse %s: %v", cidr, err)
	}

	return network
}

func TestIsInSubnet_UnconfiguredScannerAcceptsEverything(t *testing.T) {
	scanner := &ARPScanner{}

	for _, ip := range []string{"10.44.20.5", "8.8.8.8", "192.168.5.9"} {
		if !scanner.isInSubnet(ip) {
			t.Errorf("isInSubnet(%q) = false; an unscoped scanner must accept all", ip)
		}
	}
}

func TestIsInSubnet_PrimarySubnetExcludesOutsiders(t *testing.T) {
	scanner := &ARPScanner{subnet: mustCIDR(t, "10.44.20.0/24")}

	if !scanner.isInSubnet("10.44.20.5") {
		t.Error("an address inside the primary subnet was rejected")
	}
	if scanner.isInSubnet("8.8.8.8") {
		t.Error("an address outside every configured subnet was accepted")
	}
}

func TestIsInSubnet_TargetNetworksWidenTheScope(t *testing.T) {
	scanner := &ARPScanner{
		subnet:         mustCIDR(t, "10.44.20.0/24"),
		targetNetworks: []*net.IPNet{mustCIDR(t, "192.168.5.0/24")},
	}

	if !scanner.isInSubnet("192.168.5.9") {
		t.Error("an address in a target network was rejected")
	}
	// The fallback must not fire once any subnet is configured.
	if scanner.isInSubnet("8.8.8.8") {
		t.Error("the accept-all fallback fired despite configured subnets")
	}
}

func TestIsInSubnet_TargetNetworksAloneDisableTheFallback(t *testing.T) {
	scanner := &ARPScanner{targetNetworks: []*net.IPNet{mustCIDR(t, "192.168.5.0/24")}}

	if !scanner.isInSubnet("192.168.5.9") {
		t.Error("an address in the only configured subnet was rejected")
	}
	if scanner.isInSubnet("8.8.8.8") {
		t.Error("accepted an outside address although a subnet is configured")
	}
}

func TestIsInSubnet_MalformedAddressIsNeverInScope(t *testing.T) {
	if (&ARPScanner{}).isInSubnet("not-an-ip") {
		t.Error("a malformed address was accepted by the accept-all fallback")
	}
}

// isInLocalSubnet is what separates "Local Network" from "Extended Networks"
// in the UI, so an address in a target network must not read as local.
func TestIsInLocalSubnet_ExcludesTargetNetworks(t *testing.T) {
	s := &ARPScanner{
		subnet:         mustCIDR(t, "10.44.20.0/24"),
		targetNetworks: []*net.IPNet{mustCIDR(t, "192.168.5.0/24")},
	}

	if !s.isInLocalSubnet("10.44.20.5") {
		t.Error("a primary-subnet address is local")
	}
	if s.isInLocalSubnet("192.168.5.9") {
		t.Error("an additional-subnet address must not be reported as local")
	}
	// Unlike isInSubnet, this has no accept-all fallback: without a primary
	// subnet nothing is local.
	if (&ARPScanner{}).isInLocalSubnet("10.44.20.5") {
		t.Error("nothing is local when no primary subnet is configured")
	}
}

func TestMaxHostsPerSubnet_NonPositiveMeansTheDefault(t *testing.T) {
	s := NewARPScanner("", nil)

	for _, in := range []int{0, -1, -999} {
		s.SetMaxHostsPerSubnet(in)
		if got := s.GetMaxHostsPerSubnet(); got != DefaultMaxHostsPerSubnet {
			t.Errorf("SetMaxHostsPerSubnet(%d) then Get = %d, want the default %d",
				in, got, DefaultMaxHostsPerSubnet)
		}
	}

	s.SetMaxHostsPerSubnet(32)
	if got := s.GetMaxHostsPerSubnet(); got != 32 {
		t.Errorf("a positive limit = %d, want 32", got)
	}
}

// A zero-valued scanner reports the default rather than zero, which is what
// stops an unconfigured scanner sweeping nothing at all.
func TestGetMaxHostsPerSubnet_ZeroValueReportsTheDefault(t *testing.T) {
	if got := (&ARPScanner{}).GetMaxHostsPerSubnet(); got != DefaultMaxHostsPerSubnet {
		t.Errorf("zero-value scanner = %d, want %d", got, DefaultMaxHostsPerSubnet)
	}
}

func TestDefaultBluetoothScanConfig(t *testing.T) {
	cfg := DefaultBluetoothScanConfig()

	if cfg == nil {
		t.Fatal("nil config")
	}
	// Both transports on by default: a scan that silently covered only one
	// would report fewer devices than exist without saying so.
	if !cfg.IncludeClassic || !cfg.IncludeBLE {
		t.Errorf("IncludeClassic=%v IncludeBLE=%v, want both true", cfg.IncludeClassic, cfg.IncludeBLE)
	}
	if cfg.ScanDurationSec <= 0 {
		t.Errorf("ScanDurationSec = %d, want positive", cfg.ScanDurationSec)
	}
	// Non-nil so a caller can append without a nil check.
	if cfg.AuthorizedAddresses == nil {
		t.Error("AuthorizedAddresses is nil, want an empty slice")
	}
}
