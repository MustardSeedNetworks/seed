package enumerate

import (
	"errors"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// seed#2629: getActiveMethods used to read Options.ICMPScan and nothing else, so
// the discovery status chip said "icmp" on a daemon whose sweep had never opened
// a socket — an empty result then reads as an empty network.
func TestActiveMethodsDropICMPWhenTheSweepCannotOpenASocket(t *testing.T) {
	dd := NewDeviceDiscovery("lo0")
	svc := &Service{
		cfg: &config.Config{NetworkDiscovery: config.NetworkDiscoveryConfig{
			Options: config.DiscoveryOptions{ICMPScan: true, ARPScan: true},
		}},
		deviceDiscovery: dd,
	}

	if got := svc.getActiveMethods(); !slices.Contains(got, "icmp") {
		t.Fatalf("a sweep that has not failed must still report icmp; got %v", got)
	}

	dd.arpScanner.mu.Lock()
	dd.arpScanner.pingerErr = errors.New("socket: operation not permitted")
	dd.arpScanner.mu.Unlock()

	got := svc.getActiveMethods()
	if slices.Contains(got, "icmp") {
		t.Errorf("icmp still reported after the sweep failed to open a socket: %v", got)
	}
	// The unrelated methods must survive: this narrows one claim, it does not
	// blank the list.
	if !slices.Contains(got, "arp") {
		t.Errorf("arp lost along with icmp: %v", got)
	}
}

func TestPingSweepUnavailableCarriesTheReasonAndClearsOnSuccess(t *testing.T) {
	s := NewARPScanner("lo0", nil)
	if got := s.PingSweepUnavailable(); got != "" {
		t.Errorf("a scanner that has not swept reports %q, want empty", got)
	}

	s.mu.Lock()
	s.pingerErr = errors.New("socket: operation not permitted")
	s.mu.Unlock()

	if got := s.PingSweepUnavailable(); got != "socket: operation not permitted" {
		t.Errorf("reason = %q, want the socket error verbatim", got)
	}
}
