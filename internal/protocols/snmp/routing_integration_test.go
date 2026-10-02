//go:build integration

package snmp_test

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// TestGetRoutesReadsAnAgentThatServesOnlyIPRouteTable runs against the
// net-snmp fixture in internal/polling/snmp/snmpclient/testdata, whose
// legacyroutes community sees the system group and ipRouteTable only. The
// IP-FORWARD-MIB tables come back empty there, so any route read is the
// RFC 1213 fallback's (seed#2587).
func TestGetRoutesReadsAnAgentThatServesOnlyIPRouteTable(t *testing.T) {
	addr := os.Getenv("SEED_SNMP_ADDR")
	if addr == "" {
		if os.Getenv("SEED_SNMP_REQUIRED") != "" {
			t.Fatal("SEED_SNMP_REQUIRED is set but SEED_SNMP_ADDR is empty")
		}
		t.Skip("SEED_SNMP_ADDR unset; see CONTRIBUTING.md for the local agent")
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SEED_SNMP_ADDR = %q: %v", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("SEED_SNMP_ADDR port %q: %v", portText, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	table, err := snmp.GetRoutes(ctx, host, &snmp.Session{
		Port:        port,
		Timeout:     3 * time.Second,
		Retries:     1,
		Communities: []snmp.Community{{String: "legacyroutes"}},
	})
	if err != nil {
		t.Fatalf("GetRoutes: %v", err)
	}
	if len(table.Routes) == 0 {
		t.Fatal("no routes read from an agent that serves only ipRouteTable")
	}
	for _, route := range table.Routes {
		if route.Prefix < 0 || route.NextHop == "" {
			t.Errorf("route %+v lacks its prefix or next hop", route)
		}
	}
}
