//go:build darwin

package gateway_test

import (
	"syscall"
	"testing"

	"golang.org/x/net/route"

	"github.com/krisarmstrong/seed/internal/sap/gateway"
)

func TestExtractRouteIPInet4(t *testing.T) {
	addr := &route.Inet4Addr{IP: [4]byte{192, 168, 1, 1}}
	ip, family := gateway.ExtractRouteIP(addr)

	if ip != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", ip)
	}
	if family != "inet" {
		t.Errorf("expected family 'inet', got %q", family)
	}
}

func TestExtractRouteIPInet6(t *testing.T) {
	addr := &route.Inet6Addr{IP: [16]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}
	ip, family := gateway.ExtractRouteIP(addr)

	if ip != "fe80::1" {
		t.Errorf("expected IP 'fe80::1', got %q", ip)
	}
	if family != "inet6" {
		t.Errorf("expected family 'inet6', got %q", family)
	}
}

func TestExtractRouteIPNil(t *testing.T) {
	ip, family := gateway.ExtractRouteIP(nil)

	if ip != "" {
		t.Errorf("expected empty IP, got %q", ip)
	}
	if family != "" {
		t.Errorf("expected empty family, got %q", family)
	}
}

func TestExtractRouteIPLinkAddr(t *testing.T) {
	// Test with a LinkAddr - should return empty since it's not handled.
	addr := &route.LinkAddr{Name: "en0"}
	ip, family := gateway.ExtractRouteIP(addr)

	if ip != "" {
		t.Errorf("expected empty IP for LinkAddr, got %q", ip)
	}
	if family != "" {
		t.Errorf("expected empty family for LinkAddr, got %q", family)
	}
}

func TestParseRouteMessageWithDestAndGateway(t *testing.T) {
	rm := &route.RouteMessage{
		Type:  syscall.RTM_GET,
		Index: 0,
		Addrs: []route.Addr{
			&route.Inet4Addr{IP: [4]byte{0, 0, 0, 0}},     // Destination (index 0)
			&route.Inet4Addr{IP: [4]byte{192, 168, 1, 1}}, // Gateway (index 1)
		},
	}

	ri := gateway.ParseRouteMessage(rm)

	if ri == nil {
		t.Fatal("expected non-nil RouteInfo")
	}
	if ri.Destination != "0.0.0.0" {
		t.Errorf("expected Destination '0.0.0.0', got %q", ri.Destination)
	}
	if ri.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", ri.Gateway)
	}
	if ri.Family != "inet" {
		t.Errorf("expected Family 'inet', got %q", ri.Family)
	}
}

func TestParseRouteMessageNoAddrs(t *testing.T) {
	rm := &route.RouteMessage{
		Type:  syscall.RTM_GET,
		Index: 0,
		Addrs: []route.Addr{},
	}

	ri := gateway.ParseRouteMessage(rm)

	// With no destination or gateway, should return nil.
	if ri != nil {
		t.Errorf("expected nil RouteInfo for empty Addrs, got %+v", ri)
	}
}

func TestParseRouteMessageOnlyDestination(t *testing.T) {
	rm := &route.RouteMessage{
		Type:  syscall.RTM_GET,
		Index: 0,
		Addrs: []route.Addr{
			&route.Inet4Addr{IP: [4]byte{10, 0, 0, 0}}, // Destination only.
		},
	}

	ri := gateway.ParseRouteMessage(rm)

	if ri == nil {
		t.Fatal("expected non-nil RouteInfo")
	}
	if ri.Destination != "10.0.0.0" {
		t.Errorf("expected Destination '10.0.0.0', got %q", ri.Destination)
	}
	if ri.Gateway != "" {
		t.Errorf("expected empty Gateway, got %q", ri.Gateway)
	}
}

func TestParseRouteMessageWithNilAddrs(t *testing.T) {
	rm := &route.RouteMessage{
		Type:  syscall.RTM_GET,
		Index: 0,
		Addrs: []route.Addr{
			nil, // Nil destination.
			nil, // Nil gateway.
		},
	}

	ri := gateway.ParseRouteMessage(rm)

	// With nil addresses, should return nil.
	if ri != nil {
		t.Errorf("expected nil RouteInfo for nil Addrs, got %+v", ri)
	}
}

func TestIsDefaultRouteMsgIPv4Default(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			&route.Inet4Addr{IP: [4]byte{0, 0, 0, 0}}, // 0.0.0.0 = default.
		},
	}

	if !gateway.IsDefaultRouteMsg(rm, syscall.AF_INET) {
		t.Error("expected IsDefaultRouteMsg to return true for 0.0.0.0")
	}
}

func TestIsDefaultRouteMsgIPv4NonDefault(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			&route.Inet4Addr{IP: [4]byte{192, 168, 1, 0}}, // Not default.
		},
	}

	if gateway.IsDefaultRouteMsg(rm, syscall.AF_INET) {
		t.Error("expected IsDefaultRouteMsg to return false for 192.168.1.0")
	}
}

func TestIsDefaultRouteMsgIPv6Default(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			&route.Inet6Addr{IP: [16]byte{}}, // :: = default.
		},
	}

	if !gateway.IsDefaultRouteMsg(rm, syscall.AF_INET6) {
		t.Error("expected IsDefaultRouteMsg to return true for ::")
	}
}

func TestIsDefaultRouteMsgIPv6NonDefault(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			&route.Inet6Addr{IP: [16]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		},
	}

	if gateway.IsDefaultRouteMsg(rm, syscall.AF_INET6) {
		t.Error("expected IsDefaultRouteMsg to return false for non-default IPv6")
	}
}

func TestIsDefaultRouteMsgNilDest(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			nil, // Nil destination means default.
		},
	}

	if !gateway.IsDefaultRouteMsg(rm, syscall.AF_INET) {
		t.Error("expected IsDefaultRouteMsg to return true for nil destination")
	}
}

func TestIsDefaultRouteMsgEmptyAddrs(t *testing.T) {
	rm := &route.RouteMessage{
		Type:  syscall.RTM_GET,
		Addrs: []route.Addr{},
	}

	if gateway.IsDefaultRouteMsg(rm, syscall.AF_INET) {
		t.Error("expected IsDefaultRouteMsg to return false for empty Addrs")
	}
}

func TestIsDefaultRouteMsgWrongFamily(t *testing.T) {
	// IPv4 address but checking for IPv6.
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			&route.Inet4Addr{IP: [4]byte{0, 0, 0, 0}},
		},
	}

	if gateway.IsDefaultRouteMsg(rm, syscall.AF_INET6) {
		t.Error("expected IsDefaultRouteMsg to return false for wrong family")
	}
}

func TestExtractIPv6GatewayValid(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			nil, // Destination.
			&route.Inet6Addr{IP: [16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		},
	}

	gw := gateway.ExtractIPv6Gateway(rm, false)
	if gw != "2001:db8::1" {
		t.Errorf("expected gateway '2001:db8::1', got %q", gw)
	}
}

func TestExtractIPv6GatewayLinkLocal(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			nil, // Destination.
			&route.Inet6Addr{IP: [16]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		},
	}

	// With skipLinkLocal=true, should return empty.
	gw := gateway.ExtractIPv6Gateway(rm, true)
	if gw != "" {
		t.Errorf("expected empty gateway when skipping link-local, got %q", gw)
	}

	// With skipLinkLocal=false, should return the address.
	gw = gateway.ExtractIPv6Gateway(rm, false)
	if gw != "fe80::1" {
		t.Errorf("expected gateway 'fe80::1', got %q", gw)
	}
}

func TestExtractIPv6GatewayNilGateway(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			nil, // Destination.
			nil, // Nil gateway.
		},
	}

	gw := gateway.ExtractIPv6Gateway(rm, false)
	if gw != "" {
		t.Errorf("expected empty gateway for nil, got %q", gw)
	}
}

func TestExtractIPv6GatewayNotEnoughAddrs(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			nil, // Only destination.
		},
	}

	gw := gateway.ExtractIPv6Gateway(rm, false)
	if gw != "" {
		t.Errorf("expected empty gateway for insufficient addrs, got %q", gw)
	}
}

func TestExtractIPv6GatewayWrongType(t *testing.T) {
	rm := &route.RouteMessage{
		Type: syscall.RTM_GET,
		Addrs: []route.Addr{
			nil, // Destination.
			&route.Inet4Addr{IP: [4]byte{192, 168, 1, 1}}, // IPv4 instead of IPv6.
		},
	}

	gw := gateway.ExtractIPv6Gateway(rm, false)
	if gw != "" {
		t.Errorf("expected empty gateway for IPv4 addr, got %q", gw)
	}
}

func TestParseRouteMessageIPv6(t *testing.T) {
	rm := &route.RouteMessage{
		Type:  syscall.RTM_GET,
		Index: 0,
		Addrs: []route.Addr{
			&route.Inet6Addr{IP: [16]byte{}}, // Destination ::.
			&route.Inet6Addr{IP: [16]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		},
	}

	ri := gateway.ParseRouteMessage(rm)

	if ri == nil {
		t.Fatal("expected non-nil RouteInfo")
	}
	if ri.Destination != "::" {
		t.Errorf("expected Destination '::', got %q", ri.Destination)
	}
	if ri.Gateway != "fe80::1" {
		t.Errorf("expected Gateway 'fe80::1', got %q", ri.Gateway)
	}
	if ri.Family != "inet6" {
		t.Errorf("expected Family 'inet6', got %q", ri.Family)
	}
}

func TestDetectGatewayPlatformDarwin(t *testing.T) {
	// Test the platform-specific detection function.
	gw, err := gateway.DetectGateway()
	if err != nil {
		t.Logf("DetectGateway returned error: %v", err)
	}
	// Just verify it doesn't panic and returns valid types.
	_ = gw
}

func TestDetectGatewayIPv6PlatformDarwin(t *testing.T) {
	// Test the platform-specific IPv6 detection function.
	gw, err := gateway.DetectGatewayIPv6()
	if err != nil {
		t.Logf("DetectGatewayIPv6 returned error: %v", err)
	}
	// Just verify it doesn't panic and returns valid types.
	_ = gw
}

func TestGetAllRoutesDarwin(t *testing.T) {
	routes, err := gateway.GetAllRoutes()
	if err != nil {
		t.Logf("GetAllRoutes returned error: %v", err)
		return
	}

	// Verify we got some routes (most systems have at least one).
	if len(routes) == 0 {
		t.Log("GetAllRoutes returned empty slice (may be expected on some systems)")
	}

	// Verify route families are valid.
	for _, r := range routes {
		if r.Family != "inet" && r.Family != "inet6" && r.Family != "" {
			t.Errorf("unexpected Family %q", r.Family)
		}
	}
}

func TestGetDefaultGatewayInterfaceDarwin(t *testing.T) {
	iface, err := gateway.GetDefaultGatewayInterface()
	if err != nil {
		t.Logf("GetDefaultGatewayInterface returned error: %v", err)
	}

	// If we got an interface, log it.
	if iface != "" {
		t.Logf("Default gateway interface: %s", iface)
	}
}
