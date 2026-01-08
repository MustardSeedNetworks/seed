//go:build darwin

package dhcp_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/dhcp"
)

func TestParseIPConfigOutput(t *testing.T) {
	// Sample output from ipconfig getpacket
	output := `op = BOOTREPLY
htype = 1
flags = 0
hlen = 6
hops = 0
xid = 0x12345678
secs = 0
ciaddr = 0.0.0.0
yiaddr = 192.168.1.100
siaddr = 192.168.1.1
giaddr = 0.0.0.0
chaddr = a1:b2:c3:d4:e5:f6
sname =
file =
options:
Options count is 9
dhcp_message_type (uint8): 5
server_identifier (ip): 192.168.1.1
lease_time (uint32): 0x15180
subnet_mask (ip): 255.255.255.0
router (ip_mult): {192.168.1.1}
domain_name_server (ip_mult): {8.8.8.8, 8.8.4.4}
domain_name (string): local
end (none): `

	result := &dhcp.TestResult{}
	dhcp.ExportParseIPConfigOutput(output, result)

	if result.OfferedIP != "192.168.1.100" {
		t.Errorf("expected OfferedIP '192.168.1.100', got %q", result.OfferedIP)
	}
	if result.ServerIP != "192.168.1.1" {
		t.Errorf("expected ServerIP '192.168.1.1', got %q", result.ServerIP)
	}
	if result.SubnetMask != "255.255.255.0" {
		t.Errorf("expected SubnetMask '255.255.255.0', got %q", result.SubnetMask)
	}
	if result.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", result.Gateway)
	}
	if len(result.DNSServers) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(result.DNSServers))
	} else {
		if result.DNSServers[0] != "8.8.8.8" {
			t.Errorf("expected first DNS server '8.8.8.8', got %q", result.DNSServers[0])
		}
		if result.DNSServers[1] != "8.8.4.4" {
			t.Errorf("expected second DNS server '8.8.4.4', got %q", result.DNSServers[1])
		}
	}
	if result.DomainName != "local" {
		t.Errorf("expected DomainName 'local', got %q", result.DomainName)
	}
}

func TestParseLeaseTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"hex value", "0x15180", 86400, false},
		{"hex uppercase", "0X15180", 86400, false},
		{"decimal value", "86400", 86400, false},
		{"small hex", "0x3c", 60, false},
		{"zero", "0", 0, false},
		{"invalid", "not-a-number", 0, true},
		{"hex with spaces", " 0x15180 ", 86400, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dhcp.ExportParseLeaseTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExportParseLeaseTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ExportParseLeaseTime(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDHCPLine(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectOffered  string
		expectServer   string
		expectMask     string
		expectGateway  string
		expectDNS      []string
		expectDomain   string
		expectLeaseSec int
	}{
		{
			name:          "yiaddr line",
			line:          "yiaddr = 192.168.1.100",
			expectOffered: "192.168.1.100",
		},
		{
			name:         "siaddr line",
			line:         "siaddr = 192.168.1.1",
			expectServer: "192.168.1.1",
		},
		{
			name:       "subnet_mask line",
			line:       "subnet_mask (ip): 255.255.255.0",
			expectMask: "255.255.255.0",
		},
		{
			name:          "router line",
			line:          "router (ip_mult): {192.168.1.1}",
			expectGateway: "192.168.1.1",
		},
		{
			name:      "dns servers line",
			line:      "domain_name_server (ip_mult): {8.8.8.8, 8.8.4.4}",
			expectDNS: []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:         "domain name line",
			line:         "domain_name (string): local",
			expectDomain: "local",
		},
		{
			name:           "lease time hex",
			line:           "lease_time (uint32): 0x15180",
			expectLeaseSec: 86400,
		},
		{
			name:         "server identifier",
			line:         "server_identifier (ip): 192.168.1.1",
			expectServer: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &dhcp.TestResult{}
			dhcp.ExportParseDHCPLine(tt.line, result)

			if tt.expectOffered != "" && result.OfferedIP != tt.expectOffered {
				t.Errorf("expected OfferedIP %q, got %q", tt.expectOffered, result.OfferedIP)
			}
			if tt.expectServer != "" && result.ServerIP != tt.expectServer {
				t.Errorf("expected ServerIP %q, got %q", tt.expectServer, result.ServerIP)
			}
			if tt.expectMask != "" && result.SubnetMask != tt.expectMask {
				t.Errorf("expected SubnetMask %q, got %q", tt.expectMask, result.SubnetMask)
			}
			if tt.expectGateway != "" && result.Gateway != tt.expectGateway {
				t.Errorf("expected Gateway %q, got %q", tt.expectGateway, result.Gateway)
			}
			if tt.expectDNS != nil {
				if len(result.DNSServers) != len(tt.expectDNS) {
					t.Errorf(
						"expected %d DNS servers, got %d",
						len(tt.expectDNS),
						len(result.DNSServers),
					)
				} else {
					for i, dns := range tt.expectDNS {
						if result.DNSServers[i] != dns {
							t.Errorf(
								"expected DNS[%d] %q, got %q",
								i,
								dns,
								result.DNSServers[i],
							)
						}
					}
				}
			}
			if tt.expectDomain != "" && result.DomainName != tt.expectDomain {
				t.Errorf("expected DomainName %q, got %q", tt.expectDomain, result.DomainName)
			}
			if tt.expectLeaseSec != 0 {
				expectedDuration := time.Duration(tt.expectLeaseSec) * time.Second
				if result.LeaseTime != expectedDuration {
					t.Errorf(
						"expected LeaseTime %v, got %v",
						expectedDuration,
						result.LeaseTime,
					)
				}
			}
		})
	}
}

func TestParseIPConfigOutputEmpty(t *testing.T) {
	result := &dhcp.TestResult{}
	dhcp.ExportParseIPConfigOutput("", result)

	if result.OfferedIP != "" {
		t.Errorf("expected empty OfferedIP, got %q", result.OfferedIP)
	}
	if result.ServerIP != "" {
		t.Errorf("expected empty ServerIP, got %q", result.ServerIP)
	}
}

func TestParseIPConfigOutputPartial(t *testing.T) {
	output := `yiaddr = 10.0.0.50
subnet_mask (ip): 255.255.0.0`

	result := &dhcp.TestResult{}
	dhcp.ExportParseIPConfigOutput(output, result)

	if result.OfferedIP != "10.0.0.50" {
		t.Errorf("expected OfferedIP '10.0.0.50', got %q", result.OfferedIP)
	}
	if result.SubnetMask != "255.255.0.0" {
		t.Errorf("expected SubnetMask '255.255.0.0', got %q", result.SubnetMask)
	}
	if result.Gateway != "" {
		t.Errorf("expected empty Gateway, got %q", result.Gateway)
	}
}

func TestParseDHCPLineMalformed(t *testing.T) {
	// Test lines that don't match expected patterns
	lines := []string{
		"some random text",
		"yiaddr:",                          // missing value
		"router {}",                        // empty braces
		"domain_name_server (ip_mult): {}", // empty DNS list
	}

	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			result := &dhcp.TestResult{}
			// Should not panic
			dhcp.ExportParseDHCPLine(line, result)
		})
	}
}

func TestParseIPConfigOutputMultipleDNS(t *testing.T) {
	output := `domain_name_server (ip_mult): {1.1.1.1, 8.8.8.8, 8.8.4.4, 9.9.9.9}`

	result := &dhcp.TestResult{}
	dhcp.ExportParseIPConfigOutput(output, result)

	if len(result.DNSServers) != 4 {
		t.Errorf("expected 4 DNS servers, got %d", len(result.DNSServers))
	}

	expected := []string{"1.1.1.1", "8.8.8.8", "8.8.4.4", "9.9.9.9"}
	for i, dns := range expected {
		if i < len(result.DNSServers) && result.DNSServers[i] != dns {
			t.Errorf("expected DNS[%d] %q, got %q", i, dns, result.DNSServers[i])
		}
	}
}

func TestParseIPConfigOutputSingleDNS(t *testing.T) {
	output := `domain_name_server (ip_mult): {192.168.1.1}`

	result := &dhcp.TestResult{}
	dhcp.ExportParseIPConfigOutput(output, result)

	if len(result.DNSServers) != 1 {
		t.Errorf("expected 1 DNS server, got %d", len(result.DNSServers))
	}
	if len(result.DNSServers) > 0 && result.DNSServers[0] != "192.168.1.1" {
		t.Errorf("expected DNS[0] '192.168.1.1', got %q", result.DNSServers[0])
	}
}
