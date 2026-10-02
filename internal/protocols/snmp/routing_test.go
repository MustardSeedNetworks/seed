package snmp_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

func TestParseRouteType(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"reject", "2", "reject"},
		{"local", "3", snmp.IDSubtypeLocal},
		{"remote", "4", "remote"},
		{"blackhole", "5", "blackhole"},
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high value", "99", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseRouteType(tt.value)
			if got != tt.want {
				t.Errorf("ParseRouteType(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseRouteProtocol(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"other", "1", snmp.MACTypeOther},
		{"local", "2", snmp.IDSubtypeLocal},
		{"netmgmt", "3", "netmgmt"},
		{"icmp", "4", "icmp"},
		{"egp", "5", "egp"},
		{"ggp", "6", "ggp"},
		{"hello", "7", "hello"},
		{"rip", "8", "rip"},
		{"is-is", "9", "is-is"},
		{"es-is", "10", "es-is"},
		{"ciscoIgrp", "11", "ciscoIgrp"},
		{"bbnSpfIgp", "12", "bbnSpfIgp"},
		{"ospf", "13", "ospf"},
		{"bgp", "14", "bgp"},
		{"idpr", "15", "idpr"},
		{"ciscoEigrp", "16", "ciscoEigrp"},
		// Note: dvmrp (17) is not in the switch, defaults to unknown
		{"dvmrp_not_defined", "17", snmp.StatusUnknown},
		{"empty", "", snmp.StatusUnknown},
		{"invalid", "invalid", snmp.StatusUnknown},
		{"negative", "-1", snmp.StatusUnknown},
		{"high value", "99", snmp.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmp.ExportParseRouteProtocol(tt.value)
			if got != tt.want {
				t.Errorf("ParseRouteProtocol(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseIPCidrRouteIndex(t *testing.T) {
	tests := []struct {
		name    string
		oid     string
		wantDst string
		wantMsk string
		wantNH  string
	}{
		{
			name:    "valid route",
			oid:     "1.3.6.1.2.1.4.24.4.1.1.10.0.0.0.255.0.0.0.0.192.168.1.1",
			wantDst: "10.0.0.0",
			wantMsk: "255.0.0.0",
			wantNH:  "192.168.1.1",
		},
		{
			name:    "default route",
			oid:     "1.3.6.1.2.1.4.24.4.1.1.0.0.0.0.0.0.0.0.0.192.168.1.1",
			wantDst: "0.0.0.0",
			wantMsk: "0.0.0.0",
			wantNH:  "192.168.1.1",
		},
		{
			name:    "short OID",
			oid:     "1.3.6.1",
			wantDst: "",
			wantMsk: "",
			wantNH:  "",
		},
		{
			name:    "empty OID",
			oid:     "",
			wantDst: "",
			wantMsk: "",
			wantNH:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDst, gotMsk, gotNH := snmp.ExportParseIPCidrRouteIndex(tt.oid)
			if gotDst != tt.wantDst {
				t.Errorf("ParseIPCidrRouteIndex(%v) dest = %v, want %v", tt.oid, gotDst, tt.wantDst)
			}
			if gotMsk != tt.wantMsk {
				t.Errorf("ParseIPCidrRouteIndex(%v) mask = %v, want %v", tt.oid, gotMsk, tt.wantMsk)
			}
			if gotNH != tt.wantNH {
				t.Errorf(
					"ParseIPCidrRouteIndex(%v) nexthop = %v, want %v",
					tt.oid,
					gotNH,
					tt.wantNH,
				)
			}
		})
	}
}

// TestParseInetCidrRouteIndex reads RFC 4292 indexes front to back. The next
// hop is only in the index, after a policy OID whose length agents differ on
// (seed#2587).
func TestParseInetCidrRouteIndex(t *testing.T) {
	tests := []struct {
		name       string
		oid        string
		wantDest   string
		wantPrefix int
		wantNext   string
	}{
		{
			// net-snmp's index: policy zeroDotZero as a two-part OID.
			name:       "IPv4 route, two-part policy",
			oid:        ".1.3.6.1.2.1.4.24.7.1.7.1.4.10.0.0.0.24.2.0.0.1.4.192.168.1.1",
			wantDest:   "10.0.0.0",
			wantPrefix: 24,
			wantNext:   "192.168.1.1",
		},
		{
			// NIAC's index: policy as five zero sub-identifiers.
			name:       "IPv4 route, five-part policy",
			oid:        "1.3.6.1.2.1.4.24.7.1.7.1.4.10.51.0.0.16.5.0.0.0.0.0.1.4.10.254.200.2",
			wantDest:   "10.51.0.0",
			wantPrefix: 16,
			wantNext:   "10.254.200.2",
		},
		{
			name:       "default route",
			oid:        "1.3.6.1.2.1.4.24.7.1.8.1.4.0.0.0.0.0.2.0.0.1.4.10.0.0.1",
			wantDest:   "0.0.0.0",
			wantPrefix: 0,
			wantNext:   "10.0.0.1",
		},
		{
			name:       "host route /32",
			oid:        "1.3.6.1.2.1.4.24.7.1.7.1.4.192.168.1.100.32.2.0.0.1.4.192.168.1.1",
			wantDest:   "192.168.1.100",
			wantPrefix: 32,
			wantNext:   "192.168.1.1",
		},
		{
			name:       "connected route has no next hop",
			oid:        "1.3.6.1.2.1.4.24.7.1.7.1.4.10.254.200.0.24.2.0.0.0.0",
			wantDest:   "10.254.200.0",
			wantPrefix: 24,
		},
		{
			name: "IPv6 destination is not returned",
			oid: "1.3.6.1.2.1.4.24.7.1.7.2.16.32.1.13.184.0.0.0.0.0.0.0.0.0.0.0.0.32.2.0.0." +
				"2.16.254.128.0.0.0.0.0.0.0.0.0.0.0.0.0.1",
		},
		{
			name: "index cut short in the next hop",
			oid:  "1.3.6.1.2.1.4.24.7.1.7.1.4.10.0.0.0.24.2.0.0.1.4.192.168",
		},
		{
			name: "trailing sub-identifiers after the next hop",
			oid:  "1.3.6.1.2.1.4.24.7.1.7.1.4.10.0.0.0.24.2.0.0.1.4.192.168.1.1.9",
		},
		{
			name: "policy length past the end",
			oid:  "1.3.6.1.2.1.4.24.7.1.7.1.4.10.0.0.0.24.40.0.0",
		},
		{
			name: "octet out of range",
			oid:  "1.3.6.1.2.1.4.24.7.1.7.1.4.10.0.0.300.24.2.0.0.0.0",
		},
		{
			name: "prefix longer than the address",
			oid:  "1.3.6.1.2.1.4.24.7.1.7.1.4.10.0.0.0.33.2.0.0.0.0",
		},
		{
			name: "another table's OID",
			oid:  "1.3.6.1.2.1.4.24.4.1.5.10.0.0.0.255.255.255.0.0.192.168.1.1",
		},
		{
			name:       "too short OID",
			oid:        "1.3.6.1.2.1.4.24",
			wantDest:   "",
			wantPrefix: 0,
			wantNext:   "",
		},
		{
			name:       "empty OID",
			oid:        "",
			wantDest:   "",
			wantPrefix: 0,
			wantNext:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDest, gotPrefix, gotNext := snmp.ExportParseInetCidrRouteIndex(tt.oid)
			if gotDest != tt.wantDest {
				t.Errorf("parseInetCidrRouteIndex(%v) dest = %v, want %v", tt.oid, gotDest, tt.wantDest)
			}
			if gotPrefix != tt.wantPrefix {
				t.Errorf("parseInetCidrRouteIndex(%v) prefix = %v, want %v", tt.oid, gotPrefix, tt.wantPrefix)
			}
			if gotNext != tt.wantNext {
				t.Errorf("parseInetCidrRouteIndex(%v) next = %v, want %v", tt.oid, gotNext, tt.wantNext)
			}
		})
	}
}

func TestRouteEntryStruct(t *testing.T) {
	route := snmp.RouteEntry{
		Destination: "10.0.0.0",
		Prefix:      8,
		NextHop:     "192.168.1.1",
		IfIndex:     1,
		Type:        "remote",
		Protocol:    "ospf",
		Metric:      100,
	}

	if route.Destination != "10.0.0.0" {
		t.Errorf("Destination = %v, want '10.0.0.0'", route.Destination)
	}
	if route.Prefix != 8 {
		t.Errorf("Prefix = %v, want 8", route.Prefix)
	}
	if route.NextHop != "192.168.1.1" {
		t.Errorf("NextHop = %v, want '192.168.1.1'", route.NextHop)
	}
	if route.IfIndex != 1 {
		t.Errorf("IfIndex = %v, want 1", route.IfIndex)
	}
	if route.Type != "remote" {
		t.Errorf("Type = %v, want 'remote'", route.Type)
	}
	if route.Protocol != "ospf" {
		t.Errorf("Protocol = %v, want 'ospf'", route.Protocol)
	}
	if route.Metric != 100 {
		t.Errorf("Metric = %v, want 100", route.Metric)
	}
}

func TestGetRoutes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		ip      string
		cfg     *snmp.Session
		wantErr bool
	}{
		{
			name:    "nil config",
			ip:      "192.168.1.1",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "unreachable host",
			ip:   "192.0.2.1",
			cfg: &snmp.Session{
				Communities: []snmp.Community{{String: "public"}},
				Port:        161,
				Timeout:     100 * time.Millisecond,
				Retries:     1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := snmp.GetRoutes(ctx, tt.ip, tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("GetRoutes() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetRoutes() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestRoutingOIDConstants(t *testing.T) {
	// Verify routing OID constants are defined
	oids := map[string]string{
		"OIDIpCidrRouteDest":      snmp.OIDIpCidrRouteDest,
		"OIDIpCidrRouteMask":      snmp.OIDIpCidrRouteMask,
		"OIDIpCidrRouteNextHop":   snmp.OIDIpCidrRouteNextHop,
		"OIDIpCidrRouteIfIndex":   snmp.OIDIpCidrRouteIfIndex,
		"OIDIpCidrRouteType":      snmp.OIDIpCidrRouteType,
		"OIDIpCidrRouteProto":     snmp.OIDIpCidrRouteProto,
		"OIDIpCidrRouteMetric1":   snmp.OIDIpCidrRouteMetric1,
		"OIDInetCidrRouteDest":    snmp.OIDInetCidrRouteDest,
		"OIDInetCidrRouteNextHop": snmp.OIDInetCidrRouteNextHop,
		"OIDInetCidrRouteIfIndex": snmp.OIDInetCidrRouteIfIndex,
		"OIDInetCidrRouteType":    snmp.OIDInetCidrRouteType,
		"OIDInetCidrRouteProto":   snmp.OIDInetCidrRouteProto,
		"OIDInetCidrRouteMetric1": snmp.OIDInetCidrRouteMetric1,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("%s is empty", name)
		}
	}
}
