package gateway_test

import (
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

// The gateway shown on the Network page must belong to the interface the
// operator selected (#2690, #2759). It used to come from the system default
// route whatever the active interface was, so a probe pinned to a virtual link
// reported — and pinged — the Mac's Wi-Fi gateway at 100 % loss.

// The one gateway the routing table names, on en0, so the assertions name the
// same address the stub hands out.
const testGateway = "192.168.20.1"

// leaseGateway is the router feth0's DHCP lease names; the kernel has no
// default route over feth0.
const leaseGateway = "10.20.0.1"

func defaultRoute(family, dst, gw, iface string) gateway.RouteInfo {
	return gateway.RouteInfo{Destination: dst, Prefix: 0, Gateway: gw, Interface: iface, Family: family}
}

// hostReads is a host whose only IPv4 default route is over en0, and whose
// feth0 holds a DHCP lease naming a router. Both interfaces carry an IPv6
// default route, each through its own router, en0's preferred system-wide.
func hostReads() gateway.RoutingReads {
	return gateway.RoutingReads{
		Routes: []gateway.RouteInfo{
			defaultRoute("inet", "0.0.0.0", testGateway, "en0"),
			{Destination: "10.20.0.0", Prefix: 24, Interface: "feth0", Family: "inet"},
			defaultRoute("inet6", "::", "fe80::1", "en0"),
			defaultRoute("inet6", "::", "fe80::2", "feth0"),
		},
		SystemGateway:     testGateway,
		SystemGatewayIPv6: "fe80::1",
		LeaseRouters:      map[string]string{"feth0": leaseGateway},
	}
}

func TestGatewayForInterfaceScoping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		iface      string
		reads      func(r *gateway.RoutingReads)
		want       string
		wantSource gateway.Source
	}{
		{
			name:       "the interface carrying the default route gets its gateway",
			iface:      "en0",
			want:       testGateway,
			wantSource: gateway.SourceRoute,
		},
		{
			name:  "a routing-table default outranks the lease",
			iface: "en0",
			reads: func(r *gateway.RoutingReads) {
				r.LeaseRouters = map[string]string{"en0": leaseGateway}
			},
			want:       testGateway,
			wantSource: gateway.SourceRoute,
		},
		{
			name:       "an interface with no default route gets its lease's router",
			iface:      "feth0",
			want:       leaseGateway,
			wantSource: gateway.SourceLease,
		},
		{
			name:  "an interface with neither has no gateway of its own",
			iface: "feth0",
			reads: func(r *gateway.RoutingReads) { r.LeaseRouters = nil },
			want:  "",
		},
		{
			name:  "a second default route belongs to its own interface",
			iface: "feth0",
			reads: func(r *gateway.RoutingReads) {
				r.Routes = append(r.Routes, defaultRoute("inet", "0.0.0.0", "10.20.0.254", "feth0"))
			},
			want:       "10.20.0.254",
			wantSource: gateway.SourceRoute,
		},
		{
			name:       "no selection keeps the system-wide answer",
			iface:      "",
			want:       testGateway,
			wantSource: gateway.SourceRoute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reads := hostReads()
			if tt.reads != nil {
				tt.reads(&reads)
			}
			got, source, err := gateway.GatewayForInterfaceWithReads(tt.iface, reads)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if got != "" && source != tt.wantSource {
				t.Errorf("got source %q, want %q", source, tt.wantSource)
			}
		})
	}
}

func TestIPv6GatewayForInterfaceScoping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		iface  string
		routes []gateway.RouteInfo
		want   string
	}{
		{
			name:  "the selected interface gets its own router, not the system default",
			iface: "feth0",
			want:  "fe80::2",
		},
		{
			name:  "an interface with no IPv6 default route has no gateway",
			iface: "feth1",
			want:  "",
		},
		{
			name:  "a global next hop is preferred over a link-local one",
			iface: "en0",
			routes: []gateway.RouteInfo{
				defaultRoute("inet6", "::", "fe80::1", "en0"),
				defaultRoute("inet6", "::", "2001:db8::1", "en0"),
			},
			want: "2001:db8::1",
		},
		{
			name:  "an IPv4 default route is not an IPv6 gateway",
			iface: "en0",
			routes: []gateway.RouteInfo{
				defaultRoute("inet", "0.0.0.0", testGateway, "en0"),
			},
			want: "",
		},
		{
			name:  "a non-default route is not a gateway",
			iface: "en0",
			routes: []gateway.RouteInfo{
				{Destination: "2001:db8::", Prefix: 0, Gateway: "fe80::9", Interface: "en0", Family: "inet6"},
			},
			want: "",
		},
		{
			name:  "no selection keeps the system-wide answer",
			iface: "",
			want:  "fe80::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reads := hostReads()
			if tt.routes != nil {
				reads.Routes = tt.routes
			}
			got, err := gateway.IPv6GatewayForInterfaceWithReads(tt.iface, reads)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGatewayForInterfaceReportsReadErrors(t *testing.T) {
	t.Parallel()

	routesErr := errors.New("fetch RIB: permission denied")
	reads := hostReads()
	reads.RoutesErr = routesErr
	if _, _, err := gateway.GatewayForInterfaceWithReads("en0", reads); !errors.Is(err, routesErr) {
		t.Errorf("IPv4: got %v, want the routing-table error, not a silent fallback", err)
	}
	if _, err := gateway.IPv6GatewayForInterfaceWithReads("en0", reads); !errors.Is(err, routesErr) {
		t.Errorf("IPv6: got %v, want the routing-table error, not a silent fallback", err)
	}

	leaseErr := errors.New("ipconfig: exit status 70")
	reads = hostReads()
	reads.LeaseErr = leaseErr
	if _, _, err := gateway.GatewayForInterfaceWithReads("feth0", reads); !errors.Is(err, leaseErr) {
		t.Errorf("got %v, want the lease error, not an absence", err)
	}
}

func TestRouterOfLease(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	readErr := errors.New("read lease: input/output error")

	tests := []struct {
		name    string
		lease   *dhcp.LeaseInfo
		err     error
		want    string
		wantErr error
	}{
		{
			name:  "a current lease names its router",
			lease: &dhcp.LeaseInfo{Gateway: leaseGateway, Expiry: now.Add(time.Hour)},
			want:  leaseGateway,
		},
		{
			name:  "a lease with no expiry on record names its router",
			lease: &dhcp.LeaseInfo{Gateway: leaseGateway},
			want:  leaseGateway,
		},
		{
			name:  "an expired lease names no router",
			lease: &dhcp.LeaseInfo{Gateway: leaseGateway, Expiry: now.Add(-time.Minute)},
			want:  "",
		},
		{
			name: "an interface with no lease has no lease router",
			err:  &dhcp.InterfaceError{Message: "no lease file found"},
			want: "",
		},
		{
			name:    "any other read failure is an error",
			err:     readErr,
			wantErr: readErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := gateway.RouterOfLease(tt.lease, tt.err, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTestScopesDetectionToTheSelectedInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	reads := hostReads()
	reads.LeaseRouters = nil
	tester.SetRoutingForTesting(reads)
	tester.SetInterface("feth0")
	tester.TesterSetPingCount(1)

	stats := tester.Test()
	if stats.Gateway != "" {
		t.Errorf("got gateway %q, want none: the tester must not ping another interface's gateway", stats.Gateway)
	}
	if stats.Sent != 0 {
		t.Errorf("got %d packets sent, want 0: there is nothing on this interface to ping", stats.Sent)
	}
}

func TestTestPingsTheGatewayOfTheSelectedInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetRoutingForTesting(hostReads())
	tester.SetInterface("en0")
	tester.TesterSetPingCount(1)

	stats := tester.Test()
	if stats.Gateway != testGateway || stats.Source != gateway.SourceRoute {
		t.Errorf("got gateway %q from %q, want 192.168.20.1 from the route: en0 carries the default route",
			stats.Gateway, stats.Source)
	}
}

// The card labels a lease-sourced gateway, so the source must survive into
// the stats the handler serves — on the detecting run and on the next one,
// which pings the cached address without detecting again.
func TestTestLabelsALeaseGateway(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetRoutingForTesting(hostReads())
	tester.SetInterface("feth0")
	tester.TesterSetPingCount(1)
	tester.TesterSetPingTimeout(time.Millisecond)

	for run := range 2 {
		stats := tester.Test()
		if stats.Gateway != leaseGateway || stats.Source != gateway.SourceLease {
			t.Errorf("run %d: got gateway %q from %q, want %s from the lease",
				run, stats.Gateway, stats.Source, leaseGateway)
		}
	}
}

func TestSetInterfaceDropsAGatewayDetectedForTheOldInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway(testGateway)

	tester.SetInterface("feth0")

	if got := tester.GetGateway(); got != "" {
		t.Errorf("got %q, want the cached gateway cleared when the interface changes", got)
	}
}

func TestSetInterfaceKeepsTheGatewayWhenTheInterfaceIsUnchanged(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetInterface("en0")
	tester.SetGateway(testGateway)

	tester.SetInterface("en0")

	if got := tester.GetGateway(); got != testGateway {
		t.Errorf("got %q, want 192.168.20.1: a repeated selection is not a change", got)
	}
}

func TestSetInterfaceDropsStatsMeasuredForTheOldInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.TesterSetStats(&gateway.PingStats{
		Gateway:     testGateway,
		Sent:        3,
		Received:    0,
		LossPercent: 100,
		Status:      gateway.StatusError,
	})

	tester.SetInterface("feth0")

	stats := tester.GetStats()
	if stats.Gateway != "" {
		t.Errorf("got gateway %q in the cached stats, want them cleared with the interface", stats.Gateway)
	}
	if stats.LossPercent != 0 {
		t.Errorf("got %v %% loss carried over, want 0: it was measured against another link", stats.LossPercent)
	}
}
