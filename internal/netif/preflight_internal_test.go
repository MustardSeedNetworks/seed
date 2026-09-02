package netif

import (
	"strings"
	"testing"
)

// #50 asks for pre-flight validation of a network configuration. What was there
// checked that each field parses; these are the checks that stop a change from
// stranding the box, which is the risk the issue actually names.
func TestValidateIPConfigRejectsUnreachableConfigurations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		cfg     StaticIPConfig
		wantErr string
	}{
		{
			name: "gateway outside the address's subnet",
			// The classic way to strand a box: every field parses, and the
			// default route can never be installed because the next hop is not
			// on the wire.
			cfg:     StaticIPConfig{Address: "192.0.2.10", Netmask: "255.255.255.0", Gateway: "198.51.100.1"},
			wantErr: "not reachable",
		},
		{
			name:    "address is the network address of its subnet",
			cfg:     StaticIPConfig{Address: "192.0.2.0", Netmask: "255.255.255.0", Gateway: "192.0.2.1"},
			wantErr: "network address",
		},
		{
			name:    "address is the broadcast address of its subnet",
			cfg:     StaticIPConfig{Address: "192.0.2.255", Netmask: "255.255.255.0", Gateway: "192.0.2.1"},
			wantErr: "broadcast address",
		},
		{
			name:    "address and gateway are the same host",
			cfg:     StaticIPConfig{Address: "192.0.2.1", Netmask: "255.255.255.0", Gateway: "192.0.2.1"},
			wantErr: "same address",
		},
		{
			name:    "loopback address",
			cfg:     StaticIPConfig{Address: "127.0.0.1", Netmask: "255.0.0.0"},
			wantErr: "loopback",
		},
		{
			name:    "multicast address",
			cfg:     StaticIPConfig{Address: "224.0.0.1", Netmask: "255.255.255.0"},
			wantErr: "multicast",
		},
		{
			name:    "gateway is a broadcast address",
			cfg:     StaticIPConfig{Address: "192.0.2.10", Netmask: "255.255.255.0", Gateway: "192.0.2.255"},
			wantErr: "broadcast address",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateIPConfig(&tc.cfg)
			if err == nil {
				t.Fatalf("validateIPConfig(%+v) = nil, want an error mentioning %q", tc.cfg, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("validateIPConfig(%+v) = %q, want it to mention %q", tc.cfg, err, tc.wantErr)
			}
		})
	}
}

// The checks must not reject configurations that are ordinary, including the
// /31 point-to-point case where every address is a host address.
func TestValidateIPConfigAcceptsWorkableConfigurations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cfg  StaticIPConfig
	}{
		{
			name: "a plain /24 host",
			cfg:  StaticIPConfig{Address: "192.0.2.10", Netmask: "255.255.255.0", Gateway: "192.0.2.1", DNS: []string{"192.0.2.53"}},
		},
		{
			name: "no gateway at all — an interface may legitimately have none",
			cfg:  StaticIPConfig{Address: "192.0.2.10", Netmask: "255.255.255.0"},
		},
		{
			// RFC 3021: on a /31 both addresses are usable, so the
			// network/broadcast checks must not fire.
			name: "a /31 point-to-point link",
			cfg:  StaticIPConfig{Address: "192.0.2.0", Netmask: "31", Gateway: "192.0.2.1"},
		},
		{
			name: "a /32 host route with an off-subnet gateway",
			cfg:  StaticIPConfig{Address: "192.0.2.10", Netmask: "32", Gateway: "198.51.100.1"},
		},
		{
			name: "DNS servers may be anywhere, including off-subnet",
			cfg:  StaticIPConfig{Address: "192.0.2.10", Netmask: "24", Gateway: "192.0.2.1", DNS: []string{"1.1.1.1"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := validateIPConfig(&tc.cfg); err != nil {
				t.Errorf("validateIPConfig(%+v) = %v, want nil", tc.cfg, err)
			}
		})
	}
}
