package enumerate

import (
	"context"
	"errors"
	"net"
	"slices"
	"testing"
	"time"

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

	// The clear matters as much as the record: a daemon that failed once — before
	// setcap ran, say — would otherwise hide icmp for the rest of its life even
	// though every later sweep worked.
	requireAPingSweepIsPossible(t)

	_, loopback, err := net.ParseCIDR("127.0.0.2/31")
	if err != nil {
		t.Fatalf("parsing the sweep range: %v", err)
	}
	if sweepErr := s.pingSweepChunk(context.Background(), loopback); sweepErr != nil {
		t.Fatalf("pingSweepChunk: %v", sweepErr)
	}
	if got := s.PingSweepUnavailable(); got != "" {
		t.Errorf("reason survived a sweep that opened a socket: %q", got)
	}
}

// requireAPingSweepIsPossible skips where the kernel allows this process neither
// ICMP socket, which is a stock unprivileged Linux (net.ipv4.ping_group_range
// defaults to `1 0`). There is no sweep to clear a failure on such a host.
func requireAPingSweepIsPossible(t *testing.T) {
	t.Helper()

	p, err := NewICMPPinger(time.Second)
	if err != nil {
		t.Skipf("this host allows no ICMP socket to an unprivileged process: %v", err)
	}
	_ = p.Close()
}
