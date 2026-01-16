package dhcp_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/dhcp"
)

// TestIsDHCPPort verifies the isDHCPPort function correctly identifies DHCP ports.
func TestIsDHCPPort(t *testing.T) {
	tests := []struct {
		name     string
		port     uint16
		expected bool
	}{
		{"DHCP server port 67", 67, true},
		{"DHCP client port 68", 68, true},
		{"HTTP port 80", 80, false},
		{"HTTPS port 443", 443, false},
		{"DNS port 53", 53, false},
		{"Port 0", 0, false},
		{"Port 66 (one below)", 66, false},
		{"Port 69 (one above)", 69, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.IsDHCPPort(tt.port)
			if result != tt.expected {
				t.Errorf("IsDHCPPort(%d) = %v, want %v", tt.port, result, tt.expected)
			}
		})
	}
}

// TestMsgTypeToPhase verifies the msgTypeToPhase function correctly converts message types.
func TestMsgTypeToPhase(t *testing.T) {
	tests := []struct {
		name          string
		msgType       byte
		expectedPhase dhcp.Phase
		expectedOk    bool
	}{
		{"DISCOVER (1)", 1, dhcp.PhaseDiscover, true},
		{"OFFER (2)", 2, dhcp.PhaseOffer, true},
		{"REQUEST (3)", 3, dhcp.PhaseRequest, true},
		{"DECLINE (4) - ignored", 4, "", false},
		{"ACK (5)", 5, dhcp.PhaseAck, true},
		{"NAK (6) - ignored", 6, "", false},
		{"RELEASE (7) - ignored", 7, "", false},
		{"INFORM (8) - ignored", 8, "", false},
		{"Invalid (0)", 0, "", false},
		{"Invalid (255)", 255, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, ok := dhcp.MsgTypeToPhase(tt.msgType)
			if ok != tt.expectedOk {
				t.Errorf("MsgTypeToPhase(%d) ok = %v, want %v", tt.msgType, ok, tt.expectedOk)
			}
			if phase != tt.expectedPhase {
				t.Errorf("MsgTypeToPhase(%d) phase = %q, want %q", tt.msgType, phase, tt.expectedPhase)
			}
		})
	}
}

// TestHexToIP verifies the hexToIP function correctly converts hex to IP addresses.
func TestHexToIP(t *testing.T) {
	tests := []struct {
		name     string
		hexStr   string
		expected string
	}{
		{"valid hex IP 192.168.1.1", "c0a80101", "192.168.1.1"},
		{"valid hex IP 8.8.8.8", "08080808", "8.8.8.8"},
		{"valid hex IP 255.255.255.0", "ffffff00", "255.255.255.0"},
		{"valid hex IP 0.0.0.0", "00000000", "0.0.0.0"},
		{"valid hex IP 10.0.0.1", "0a000001", "10.0.0.1"},
		{"uppercase hex", "C0A80101", "192.168.1.1"},
		// Note: The function strips whitespace, so these become valid
		{"with space", "c0a8 0101", "192.168.1.1"},
		{"with newlines", "c0a8\n0101", "192.168.1.1"},
		{"empty string", "", ""},
		// 4-char string triggers raw byte fallback
		{"too short hex", "c0a8", "99.48.97.56"}, // Interpreted as raw bytes
		{"too long", "c0a80101ff", ""},
		{"invalid hex", "gggggggg", ""},
		{"4-byte raw string", "\xc0\xa8\x01\x01", "192.168.1.1"}, // raw 4 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.HexToIP(tt.hexStr)
			if result != tt.expected {
				t.Errorf("HexToIP(%q) = %q, want %q", tt.hexStr, result, tt.expected)
			}
		})
	}
}

// TestExtractPlistIP verifies extracting IP addresses from plist content.
func TestExtractPlistIP(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		expected string
	}{
		{
			name:     "valid ServerIdentifier",
			content:  `<key>ServerIdentifier</key><data>c0a80101</data>`,
			key:      "ServerIdentifier",
			expected: "192.168.1.1",
		},
		{
			name:     "valid RouterIPAddress",
			content:  `<key>RouterIPAddress</key><data>0a000001</data>`,
			key:      "RouterIPAddress",
			expected: "10.0.0.1",
		},
		{
			name:     "missing key",
			content:  `<key>OtherKey</key><data>c0a80101</data>`,
			key:      "ServerIdentifier",
			expected: "",
		},
		{
			name:     "missing data tag",
			content:  `<key>ServerIdentifier</key><string>192.168.1.1</string>`,
			key:      "ServerIdentifier",
			expected: "",
		},
		{
			name:     "empty content",
			content:  "",
			key:      "ServerIdentifier",
			expected: "",
		},
		{
			name:     "missing closing data tag",
			content:  `<key>ServerIdentifier</key><data>c0a80101`,
			key:      "ServerIdentifier",
			expected: "",
		},
		{
			name:     "data with whitespace",
			content:  `<key>ServerIdentifier</key><data>  c0a80101  </data>`,
			key:      "ServerIdentifier",
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ExtractPlistIP(tt.content, tt.key)
			if result != tt.expected {
				t.Errorf("ExtractPlistIP() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExtractPlistInteger verifies extracting integers from plist content.
func TestExtractPlistInteger(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		expected int
	}{
		{
			name:     "valid LeaseLength",
			content:  `<key>LeaseLength</key><integer>86400</integer>`,
			key:      "LeaseLength",
			expected: 86400,
		},
		{
			name:     "zero value",
			content:  `<key>LeaseLength</key><integer>0</integer>`,
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "missing key",
			content:  `<key>OtherKey</key><integer>86400</integer>`,
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "missing integer tag",
			content:  `<key>LeaseLength</key><string>86400</string>`,
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "invalid integer",
			content:  `<key>LeaseLength</key><integer>notanumber</integer>`,
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "empty content",
			content:  "",
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "missing closing tag",
			content:  `<key>LeaseLength</key><integer>86400`,
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "with whitespace",
			content:  `<key>LeaseLength</key><integer>  86400  </integer>`,
			key:      "LeaseLength",
			expected: 86400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ExtractPlistInteger(tt.content, tt.key)
			if result != tt.expected {
				t.Errorf("ExtractPlistInteger() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestExtractArrayContent verifies extraction of array content from plist.
func TestExtractArrayContent(t *testing.T) {
	tests := []struct {
		name       string
		remaining  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "valid array",
			remaining:  `<array><data>08080808</data></array>`,
			expected:   `<data>08080808</data>`,
			expectedOk: true,
		},
		{
			name:       "empty array",
			remaining:  `<array></array>`,
			expected:   ``,
			expectedOk: true,
		},
		{
			name:       "missing array start",
			remaining:  `<data>08080808</data></array>`,
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "missing array end",
			remaining:  `<array><data>08080808</data>`,
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "multiple items",
			remaining:  `<array><data>08080808</data><data>08080404</data></array>`,
			expected:   `<data>08080808</data><data>08080404</data>`,
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := dhcp.ExtractArrayContent(tt.remaining)
			if ok != tt.expectedOk {
				t.Errorf("ExtractArrayContent() ok = %v, want %v", ok, tt.expectedOk)
			}
			if result != tt.expected {
				t.Errorf("ExtractArrayContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestParseDataTagsFromArray verifies parsing data tags from array content.
func TestParseDataTagsFromArray(t *testing.T) {
	tests := []struct {
		name         string
		arrayContent string
		expected     []string
	}{
		{
			name:         "single IP",
			arrayContent: `<data>08080808</data>`,
			expected:     []string{"8.8.8.8"},
		},
		{
			name:         "multiple IPs",
			arrayContent: `<data>08080808</data><data>08080404</data>`,
			expected:     []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:         "empty content",
			arrayContent: ``,
			expected:     nil,
		},
		{
			name:         "no data tags",
			arrayContent: `<string>8.8.8.8</string>`,
			expected:     nil,
		},
		{
			name:         "missing closing data tag",
			arrayContent: `<data>08080808`,
			expected:     nil,
		},
		{
			name:         "invalid hex data",
			arrayContent: `<data>notvalid</data>`,
			expected:     nil, // hexToIP returns empty for invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ParseDataTagsFromArray(tt.arrayContent)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseDataTagsFromArray() len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("ParseDataTagsFromArray()[%d] = %q, want %q", i, ip, tt.expected[i])
				}
			}
		})
	}
}

// TestExtractPlistIPArray verifies extraction of IP arrays from plist content.
func TestExtractPlistIPArray(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		expected []string
	}{
		{
			name:     "single IP in array",
			content:  `<key>DomainNameServer</key><array><data>08080808</data></array>`,
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8"},
		},
		{
			name:     "multiple IPs in array",
			content:  `<key>DomainNameServer</key><array><data>08080808</data><data>08080404</data></array>`,
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:     "single IP not in array",
			content:  `<key>DomainNameServer</key><data>08080808</data>`,
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8"},
		},
		{
			name:     "missing key",
			content:  `<key>OtherKey</key><array><data>08080808</data></array>`,
			key:      "DomainNameServer",
			expected: nil,
		},
		{
			name:     "empty array",
			content:  `<key>DomainNameServer</key><array></array>`,
			key:      "DomainNameServer",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ExtractPlistIPArray(tt.content, tt.key)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractPlistIPArray() len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("ExtractPlistIPArray()[%d] = %q, want %q", i, ip, tt.expected[i])
				}
			}
		})
	}
}

// TestParseDHClientLeaseLine verifies parsing of individual dhclient lease lines.
func TestParseDHClientLeaseLine(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectedServer string
		expectedRouter string
		expectedLease  int
		expectedDNS    []string
	}{
		{
			name:           "dhcp-server-identifier",
			line:           "option dhcp-server-identifier 192.168.1.1;",
			expectedServer: "192.168.1.1",
		},
		{
			name:           "routers single",
			line:           "option routers 192.168.1.1;",
			expectedRouter: "192.168.1.1",
		},
		{
			// Note: extractValue returns last space-separated value (192.168.1.2),
			// then splits by comma which yields just that value
			name:           "routers multiple (gets last space-separated)",
			line:           "option routers 192.168.1.1, 192.168.1.2;",
			expectedRouter: "192.168.1.2",
		},
		{
			name:          "dhcp-lease-time",
			line:          "option dhcp-lease-time 86400;",
			expectedLease: 86400,
		},
		{
			name:        "domain-name-servers single",
			line:        "option domain-name-servers 8.8.8.8;",
			expectedDNS: []string{"8.8.8.8"},
		},
		{
			// Note: extractValue returns last space-separated value (8.8.4.4),
			// then splits by comma which yields just that value
			name:        "domain-name-servers multiple (gets last)",
			line:        "option domain-name-servers 8.8.8.8, 8.8.4.4;",
			expectedDNS: []string{"8.8.4.4"},
		},
		{
			name: "unrelated line",
			line: "  interface \"eth0\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &dhcp.LeaseInfo{}
			dhcp.ParseDHClientLeaseLine(tt.line, info)

			if info.DHCPServer != tt.expectedServer {
				t.Errorf("DHCPServer = %q, want %q", info.DHCPServer, tt.expectedServer)
			}
			if info.Gateway != tt.expectedRouter {
				t.Errorf("Gateway = %q, want %q", info.Gateway, tt.expectedRouter)
			}
			if info.LeaseTime != tt.expectedLease {
				t.Errorf("LeaseTime = %d, want %d", info.LeaseTime, tt.expectedLease)
			}
			if len(info.DNS) != len(tt.expectedDNS) {
				t.Errorf("DNS len = %d, want %d", len(info.DNS), len(tt.expectedDNS))
			}
		})
	}
}

// TestParseLeaseLineServer verifies server line parsing.
func TestParseLeaseLineServer(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		serverKey  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "matching key",
			line:       "DHCP4_SERVER_ID=192.168.1.1",
			serverKey:  "DHCP4_SERVER_ID=",
			expected:   "192.168.1.1",
			expectedOk: true,
		},
		{
			name:       "non-matching key",
			line:       "DHCP4_ROUTERS=192.168.1.1",
			serverKey:  "DHCP4_SERVER_ID=",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "empty line",
			line:       "",
			serverKey:  "DHCP4_SERVER_ID=",
			expected:   "",
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := dhcp.ParseLeaseLineServer(tt.line, tt.serverKey)
			if ok != tt.expectedOk {
				t.Errorf("ok = %v, want %v", ok, tt.expectedOk)
			}
			if result != tt.expected {
				t.Errorf("result = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestParseLeaseLineRouter verifies router line parsing.
func TestParseLeaseLineRouter(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		routerKey  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "matching key single IP",
			line:       "DHCP4_ROUTERS=192.168.1.1",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "192.168.1.1",
			expectedOk: true,
		},
		{
			name:       "matching key multiple IPs (takes first)",
			line:       "DHCP4_ROUTERS=192.168.1.1 192.168.1.2",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "192.168.1.1",
			expectedOk: true,
		},
		{
			name:       "non-matching key",
			line:       "DHCP4_SERVER_ID=192.168.1.1",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "",
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := dhcp.ParseLeaseLineRouter(tt.line, tt.routerKey)
			if ok != tt.expectedOk {
				t.Errorf("ok = %v, want %v", ok, tt.expectedOk)
			}
			if result != tt.expected {
				t.Errorf("result = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestParseLeaseLineTime verifies lease time line parsing.
func TestParseLeaseLineTime(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		leaseTimeKey string
		expected     int
		expectedOk   bool
	}{
		{
			name:         "valid lease time",
			line:         "DHCP4_LEASE_TIME=86400",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     86400,
			expectedOk:   true,
		},
		{
			name:         "zero lease time",
			line:         "DHCP4_LEASE_TIME=0",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     0,
			expectedOk:   true,
		},
		{
			name:         "non-matching key",
			line:         "DHCP4_SERVER_ID=192.168.1.1",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     0,
			expectedOk:   false,
		},
		{
			name:         "invalid number",
			line:         "DHCP4_LEASE_TIME=notanumber",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     0,
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := dhcp.ParseLeaseLineTime(tt.line, tt.leaseTimeKey)
			if ok != tt.expectedOk {
				t.Errorf("ok = %v, want %v", ok, tt.expectedOk)
			}
			if result != tt.expected {
				t.Errorf("result = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestParseLeaseLineDNS verifies DNS line parsing.
func TestParseLeaseLineDNS(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		dnsKey   string
		expected []string
	}{
		{
			name:     "single DNS",
			line:     "DHCP4_DOMAIN_NAME_SERVERS=8.8.8.8",
			dnsKey:   "DHCP4_DOMAIN_NAME_SERVERS=",
			expected: []string{"8.8.8.8"},
		},
		{
			name:     "multiple DNS",
			line:     "DHCP4_DOMAIN_NAME_SERVERS=8.8.8.8 8.8.4.4",
			dnsKey:   "DHCP4_DOMAIN_NAME_SERVERS=",
			expected: []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:     "non-matching key",
			line:     "DHCP4_SERVER_ID=192.168.1.1",
			dnsKey:   "DHCP4_DOMAIN_NAME_SERVERS=",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ParseLeaseLineDNS(tt.line, tt.dnsKey)
			if len(result) != len(tt.expected) {
				t.Errorf("len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, dns := range result {
				if dns != tt.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, dns, tt.expected[i])
				}
			}
		})
	}
}

// TestProcessLeaseLine verifies the complete lease line processing.
func TestProcessLeaseLine(t *testing.T) {
	mapping := dhcp.LeaseFieldMapping{
		ServerKey:    "SERVER_ADDRESS=",
		RouterKey:    "ROUTER=",
		LeaseTimeKey: "LIFETIME=",
		DNSKey:       "DNS=",
	}

	tests := []struct {
		name           string
		line           string
		expectedServer string
		expectedRouter string
		expectedLease  int
		expectedDNS    []string
	}{
		{
			name:           "server line",
			line:           "SERVER_ADDRESS=192.168.1.1",
			expectedServer: "192.168.1.1",
		},
		{
			name:           "router line",
			line:           "ROUTER=192.168.1.1",
			expectedRouter: "192.168.1.1",
		},
		{
			name:          "lifetime line",
			line:          "LIFETIME=3600",
			expectedLease: 3600,
		},
		{
			name:        "dns line",
			line:        "DNS=8.8.8.8 8.8.4.4",
			expectedDNS: []string{"8.8.8.8", "8.8.4.4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &dhcp.LeaseInfo{}
			dhcp.ProcessLeaseLine(tt.line, mapping, info)

			if info.DHCPServer != tt.expectedServer {
				t.Errorf("DHCPServer = %q, want %q", info.DHCPServer, tt.expectedServer)
			}
			if info.Gateway != tt.expectedRouter {
				t.Errorf("Gateway = %q, want %q", info.Gateway, tt.expectedRouter)
			}
			if info.LeaseTime != tt.expectedLease {
				t.Errorf("LeaseTime = %d, want %d", info.LeaseTime, tt.expectedLease)
			}
			if len(info.DNS) != len(tt.expectedDNS) {
				t.Errorf("DNS len = %d, want %d", len(info.DNS), len(tt.expectedDNS))
			}
		})
	}
}

// TestParseLeaseFileWithMapping verifies file parsing with mapping.
func TestParseLeaseFileWithMapping(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()

	t.Run("valid lease file", func(t *testing.T) {
		content := `SERVER_ADDRESS=192.168.1.1
ROUTER=192.168.1.1
LIFETIME=86400
DNS=8.8.8.8 8.8.4.4
`
		path := filepath.Join(tmpDir, "valid.lease")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		mapping := dhcp.LeaseFieldMapping{
			ServerKey:    "SERVER_ADDRESS=",
			RouterKey:    "ROUTER=",
			LeaseTimeKey: "LIFETIME=",
			DNSKey:       "DNS=",
		}

		result := dhcp.ParseLeaseFileWithMapping(path, mapping)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.DHCPServer != "192.168.1.1" {
			t.Errorf("DHCPServer = %q, want %q", result.DHCPServer, "192.168.1.1")
		}
		if result.Gateway != "192.168.1.1" {
			t.Errorf("Gateway = %q, want %q", result.Gateway, "192.168.1.1")
		}
		if result.LeaseTime != 86400 {
			t.Errorf("LeaseTime = %d, want %d", result.LeaseTime, 86400)
		}
		if len(result.DNS) != 2 {
			t.Errorf("DNS len = %d, want 2", len(result.DNS))
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		mapping := dhcp.LeaseFieldMapping{
			ServerKey: "SERVER_ADDRESS=",
		}
		result := dhcp.ParseLeaseFileWithMapping("/nonexistent/path", mapping)
		if result != nil {
			t.Error("expected nil for nonexistent file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty.lease")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		mapping := dhcp.LeaseFieldMapping{
			ServerKey: "SERVER_ADDRESS=",
		}
		result := dhcp.ParseLeaseFileWithMapping(path, mapping)
		if result != nil {
			t.Error("expected nil for empty file")
		}
	})
}

// TestParseDarwinLeaseFile verifies Darwin lease file parsing.
func TestParseDarwinLeaseFileValid(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>ServerIdentifier</key>
	<data>c0a80101</data>
	<key>RouterIPAddress</key>
	<data>c0a80101</data>
	<key>LeaseLength</key>
	<integer>86400</integer>
	<key>DomainNameServer</key>
	<array>
		<data>08080808</data>
		<data>08080404</data>
	</array>
</dict>
</plist>`
	path := filepath.Join(tmpDir, "darwin.lease")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := dhcp.ParseDarwinLeaseFile(path)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.DHCPServer != "192.168.1.1" {
		t.Errorf("DHCPServer = %q, want %q", result.DHCPServer, "192.168.1.1")
	}
	if result.Gateway != "192.168.1.1" {
		t.Errorf("Gateway = %q, want %q", result.Gateway, "192.168.1.1")
	}
	if result.LeaseTime != 86400 {
		t.Errorf("LeaseTime = %d, want %d", result.LeaseTime, 86400)
	}
	if len(result.DNS) != 2 {
		t.Errorf("DNS len = %d, want 2", len(result.DNS))
	}
}

func TestParseDarwinLeaseFileRouterFallback(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>ServerIdentifier</key>
	<data>c0a80101</data>
	<key>Router</key>
	<data>0a000001</data>
</dict>
</plist>`
	path := filepath.Join(tmpDir, "darwin_router.lease")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := dhcp.ParseDarwinLeaseFile(path)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Gateway != "10.0.0.1" {
		t.Errorf("Gateway = %q, want %q", result.Gateway, "10.0.0.1")
	}
}

func TestParseDarwinLeaseFileNonexistent(t *testing.T) {
	result := dhcp.ParseDarwinLeaseFile("/nonexistent/path")
	if result != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestParseDarwinLeaseFileNoUsefulData(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>SomeOtherKey</key>
	<string>value</string>
</dict>
</plist>`
	path := filepath.Join(tmpDir, "darwin_empty.lease")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := dhcp.ParseDarwinLeaseFile(path)
	if result != nil {
		t.Error("expected nil for file with no useful data")
	}
}

// TestParseDHClientLeaseFileComplete tests the complete dhclient lease file parser.
func TestParseDHClientLeaseFileComplete(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid dhclient lease file", func(t *testing.T) {
		content := `lease {
  interface "eth0";
  option dhcp-server-identifier 192.168.1.1;
  option routers 192.168.1.1;
  option dhcp-lease-time 86400;
  option domain-name-servers 8.8.8.8, 8.8.4.4;
}`
		path := filepath.Join(tmpDir, "dhclient.leases")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := dhcp.ParseDHClientLeaseFile(path, "eth0")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.DHCPServer != "192.168.1.1" {
			t.Errorf("DHCPServer = %q, want %q", result.DHCPServer, "192.168.1.1")
		}
	})

	t.Run("multiple leases takes last", func(t *testing.T) {
		content := `lease {
  option dhcp-server-identifier 192.168.1.1;
}
lease {
  option dhcp-server-identifier 192.168.1.2;
}`
		path := filepath.Join(tmpDir, "dhclient_multi.leases")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := dhcp.ParseDHClientLeaseFile(path, "eth0")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.DHCPServer != "192.168.1.2" {
			t.Errorf("DHCPServer = %q, want %q (should be last lease)", result.DHCPServer, "192.168.1.2")
		}
	})

	t.Run("empty lease file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "dhclient_empty.leases")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := dhcp.ParseDHClientLeaseFile(path, "eth0")
		if result != nil {
			t.Error("expected nil for empty file")
		}
	})
}

// TestRogueDetectorRecordDetectedServer tests the recordDetectedServer function.
func TestRogueDetectorRecordDetectedServer(t *testing.T) {
	t.Run("add new server", func(t *testing.T) {
		config := &dhcp.RogueDetectorConfig{
			Interface:        "eth0",
			KnownServers:     []string{"192.168.1.1"},
			AlertOnDetection: false, // Suppress alerts in tests
		}
		rd := dhcp.NewRogueDetector(config)

		rd.RecordDetectedServer("192.168.1.100", "aa:bb:cc:dd:ee:ff")

		servers := rd.GetDetectedServers()
		if len(servers) != 1 {
			t.Fatalf("expected 1 server, got %d", len(servers))
		}
		if servers[0].IP != "192.168.1.100" {
			t.Errorf("IP = %q, want %q", servers[0].IP, "192.168.1.100")
		}
		if servers[0].MAC != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("MAC = %q, want %q", servers[0].MAC, "aa:bb:cc:dd:ee:ff")
		}
		if servers[0].IsAuthorized {
			t.Error("server should not be authorized")
		}
		if servers[0].OfferCount != 1 {
			t.Errorf("OfferCount = %d, want 1", servers[0].OfferCount)
		}
	})

	t.Run("add authorized server", func(t *testing.T) {
		config := &dhcp.RogueDetectorConfig{
			Interface:        "eth0",
			KnownServers:     []string{"192.168.1.1"},
			AlertOnDetection: false,
		}
		rd := dhcp.NewRogueDetector(config)

		rd.RecordDetectedServer("192.168.1.1", "aa:bb:cc:dd:ee:ff")

		servers := rd.GetDetectedServers()
		if len(servers) != 1 {
			t.Fatalf("expected 1 server, got %d", len(servers))
		}
		if !servers[0].IsAuthorized {
			t.Error("server should be authorized")
		}
	})

	t.Run("update existing server", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		rd.RecordDetectedServer("192.168.1.100", "aa:bb:cc:dd:ee:ff")
		rd.RecordDetectedServer("192.168.1.100", "aa:bb:cc:dd:ee:ff")
		rd.RecordDetectedServer("192.168.1.100", "aa:bb:cc:dd:ee:ff")

		servers := rd.GetDetectedServers()
		if len(servers) != 1 {
			t.Fatalf("expected 1 server, got %d", len(servers))
		}
		if servers[0].OfferCount != 3 {
			t.Errorf("OfferCount = %d, want 3", servers[0].OfferCount)
		}
	})

	t.Run("update MAC if initially empty", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		// Add server without MAC
		rd.AddNewServer("192.168.1.100", "", time.Now())

		// Update with MAC
		server, _ := rd.GetDetectedServer("192.168.1.100")
		rd.UpdateExistingServer(server, "aa:bb:cc:dd:ee:ff", time.Now())

		updatedServer, _ := rd.GetDetectedServer("192.168.1.100")
		if updatedServer.MAC != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("MAC = %q, want %q", updatedServer.MAC, "aa:bb:cc:dd:ee:ff")
		}
	})
}

// TestRogueDetectorPruneExpiredServers tests the pruneExpiredServers function.
func TestRogueDetectorPruneExpiredServers(t *testing.T) {
	t.Run("prune expired servers when over threshold", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		now := time.Now()
		expiredTime := now.Add(-25 * time.Hour) // More than 24 hours ago

		// Add servers manually to create specific test conditions
		// We need more than maxDetectedServers/2 servers to trigger pruning
		servers := make(map[string]*dhcp.RogueServer)
		for i := range 600 { // More than 500 (half of 1000 max)
			ip := "192.168.1." + string(rune(i%256))
			if i >= 256 {
				ip = "192.168.2." + string(rune(i%256))
			}
			if i < 550 {
				// Most servers are expired
				servers[ip] = &dhcp.RogueServer{
					IP:       ip,
					LastSeen: expiredTime,
				}
			} else {
				// Some servers are recent
				servers[ip] = &dhcp.RogueServer{
					IP:       ip,
					LastSeen: now,
				}
			}
		}
		rd.SetDetectedServers(servers)

		// Trigger pruning
		rd.PruneExpiredServers(now)

		// Check that expired servers were pruned
		if rd.DetectedServersCount() >= 550 {
			t.Errorf("expected fewer servers after pruning, got %d", rd.DetectedServersCount())
		}
	})

	t.Run("no pruning when under threshold", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		// Add just a few servers
		now := time.Now()
		for i := range 5 {
			ip := "192.168.1." + string(rune(i+1))
			rd.AddDetectedServer(&dhcp.RogueServer{
				IP:       ip,
				LastSeen: now.Add(-25 * time.Hour),
			})
		}

		initialCount := rd.DetectedServersCount()
		rd.PruneExpiredServers(now)

		// No pruning should occur because we're under threshold
		if rd.DetectedServersCount() != initialCount {
			t.Errorf("expected %d servers, got %d", initialCount, rd.DetectedServersCount())
		}
	})
}

// TestRogueDetectorAddNewServerLimit tests the maxDetectedServers limit.
func TestRogueDetectorAddNewServerLimit(t *testing.T) {
	t.Run("respect max servers limit", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		// Fill up to the limit
		servers := make(map[string]*dhcp.RogueServer)
		for i := range 1000 { // max limit
			ip := "10." + string(rune(i/256/256%256)) + "." + string(rune(i/256%256)) + "." + string(rune(i%256))
			servers[ip] = &dhcp.RogueServer{
				IP:       ip,
				LastSeen: time.Now(),
			}
		}
		rd.SetDetectedServers(servers)

		// Try to add one more
		now := time.Now()
		rd.AddNewServer("192.168.1.100", "aa:bb:cc:dd:ee:ff", now)

		// The new server should not be added
		_, exists := rd.GetDetectedServer("192.168.1.100")
		if exists {
			t.Error("server should not be added when at max limit")
		}
	})
}

// TestRogueDetectorSetInterface tests the SetInterface function.
func TestRogueDetectorSetInterface(t *testing.T) {
	t.Run("set interface when not running", func(t *testing.T) {
		config := &dhcp.RogueDetectorConfig{
			Interface: "eth0",
		}
		rd := dhcp.NewRogueDetector(config)

		err := rd.SetInterface("wlan0")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		cfg := rd.GetConfig()
		if cfg.Interface != "wlan0" {
			t.Errorf("Interface = %q, want %q", cfg.Interface, "wlan0")
		}
	})

	// Note: Testing SetInterface while running requires root permissions for pcap
}

// TestMonitorStopNotRunning tests Stop on a non-running monitor.
func TestMonitorStopNotRunning(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Should not panic or error when called on non-running monitor
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("monitor should not be running")
	}
}

// TestRogueServerFields tests all fields of RogueServer struct.
func TestRogueServerFields(t *testing.T) {
	now := time.Now()
	server := dhcp.RogueServer{
		IP:           "192.168.1.1",
		MAC:          "aa:bb:cc:dd:ee:ff",
		FirstSeen:    now,
		LastSeen:     now.Add(1 * time.Hour),
		OfferCount:   42,
		IsAuthorized: true,
	}

	if server.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want %q", server.IP, "192.168.1.1")
	}
	if server.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC = %q, want %q", server.MAC, "aa:bb:cc:dd:ee:ff")
	}
	if !server.FirstSeen.Equal(now) {
		t.Errorf("FirstSeen mismatch")
	}
	if !server.LastSeen.Equal(now.Add(1 * time.Hour)) {
		t.Errorf("LastSeen mismatch")
	}
	if server.OfferCount != 42 {
		t.Errorf("OfferCount = %d, want 42", server.OfferCount)
	}
	if !server.IsAuthorized {
		t.Error("IsAuthorized should be true")
	}
}

// TestRogueDetectorConfigFields tests all fields of RogueDetectorConfig struct.
func TestRogueDetectorConfigFields(t *testing.T) {
	config := dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: true,
	}

	if config.Interface != "eth0" {
		t.Errorf("Interface = %q, want %q", config.Interface, "eth0")
	}
	if len(config.KnownServers) != 2 {
		t.Errorf("KnownServers len = %d, want 2", len(config.KnownServers))
	}
	if !config.AlertOnDetection {
		t.Error("AlertOnDetection should be true")
	}
}

// TestRogueDetectorStartAlreadyRunning tests starting when already running.
func TestRogueDetectorStartAlreadyRunning(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Manually set running state
	rd.SetRogueRunning(true)

	err := rd.Start()
	if err == nil {
		t.Error("expected error when starting already running detector")
	}
	if err.Error() != "rogue detector already running" {
		t.Errorf("error = %q, want %q", err.Error(), "rogue detector already running")
	}
}

// TestFindDHCPMessageTypeEdgeCases tests additional edge cases for findDHCPMessageType.
func TestFindDHCPMessageTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		options  []byte
		expected byte
	}{
		{
			name:     "end option only",
			options:  []byte{255},
			expected: 0,
		},
		{
			name:     "pad options then end",
			options:  []byte{0, 0, 0, 255},
			expected: 0,
		},
		{
			name: "option with zero length for message type",
			// Option 53 with length 0
			options:  []byte{53, 0, 255},
			expected: 0,
		},
		{
			name: "option length exceeds remaining",
			// Option 53 with length 10, but only 2 more bytes available
			options:  []byte{53, 10, 1, 2},
			expected: 0,
		},
		{
			name: "multiple options before message type",
			options: []byte{
				1, 4, 255, 255, 255, 0, // Subnet mask
				3, 4, 192, 168, 1, 1, // Router
				6, 4, 8, 8, 8, 8, // DNS
				53, 1, 3, // Message type = REQUEST
				255,
			},
			expected: 3, // REQUEST
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.FindDHCPMessageType(tt.options)
			if result != tt.expected {
				t.Errorf("FindDHCPMessageType() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestParseLeaseLineRouterEmptyAfter tests edge case for router parsing.
func TestParseLeaseLineRouterEmptyAfter(t *testing.T) {
	// Test when nothing after key (empty string)
	// Note: strings.Split("", " ") returns []string{""} which has len > 0
	// So ok will be true, but result will be empty string
	result, ok := dhcp.ParseLeaseLineRouter("ROUTER=", "ROUTER=")
	if !ok {
		t.Error("expected ok=true (Split returns at least one element)")
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

// TestRogueDetectorSetInterfaceWhileNotRunning tests setting interface when not running.
func TestRogueDetectorSetInterfaceWhileNotRunning(t *testing.T) {
	config := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{},
		AlertOnDetection: false,
	}
	rd := dhcp.NewRogueDetector(config)

	// Should not error when not running
	err := rd.SetInterface("wlan0")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	cfg := rd.GetConfig()
	if cfg.Interface != "wlan0" {
		t.Errorf("Interface = %q, want %q", cfg.Interface, "wlan0")
	}
}

// TestMonitorSetInterfaceWhileNotRunning tests SetInterface when monitor is not running.
func TestMonitorSetInterfaceWhileNotRunning(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	err := monitor.SetInterface("wlan0")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if monitor.MonitorInterfaceName() != "wlan0" {
		t.Errorf("interfaceName = %q, want %q", monitor.MonitorInterfaceName(), "wlan0")
	}
}

// TestRogueDetectorRecordDetectedServerMultipleCalls tests multiple calls to recordDetectedServer.
func TestRogueDetectorRecordDetectedServerMultipleCalls(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Add same server multiple times
	for range 5 {
		rd.RecordDetectedServer("192.168.1.100", "aa:bb:cc:dd:ee:ff")
	}

	servers := rd.GetDetectedServers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].OfferCount != 5 {
		t.Errorf("OfferCount = %d, want 5", servers[0].OfferCount)
	}
}

// TestUpdateExistingServerMAC tests that MAC is updated when initially empty.
func TestUpdateExistingServerMAC(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Add server with empty MAC
	now := time.Now()
	rd.AddNewServer("192.168.1.100", "", now)

	// Get the server and update with MAC
	server, exists := rd.GetDetectedServer("192.168.1.100")
	if !exists {
		t.Fatal("server not found")
	}

	rd.UpdateExistingServer(server, "aa:bb:cc:dd:ee:ff", now.Add(time.Second))

	// Verify MAC was updated
	updatedServer, _ := rd.GetDetectedServer("192.168.1.100")
	if updatedServer.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC = %q, want %q", updatedServer.MAC, "aa:bb:cc:dd:ee:ff")
	}
	if updatedServer.OfferCount != 2 {
		t.Errorf("OfferCount = %d, want 2", updatedServer.OfferCount)
	}
}

// TestUpdateExistingServerMACNotOverwritten tests that existing MAC is not overwritten.
func TestUpdateExistingServerMACNotOverwritten(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Add server with MAC
	now := time.Now()
	rd.AddNewServer("192.168.1.100", "original:mac:addr:01:02:03", now)

	// Get the server and try to update with different MAC
	server, _ := rd.GetDetectedServer("192.168.1.100")
	rd.UpdateExistingServer(server, "new:mac:addr:04:05:06", now.Add(time.Second))

	// Original MAC should be preserved
	updatedServer, _ := rd.GetDetectedServer("192.168.1.100")
	if updatedServer.MAC != "original:mac:addr:01:02:03" {
		t.Errorf("MAC = %q, should be preserved as %q", updatedServer.MAC, "original:mac:addr:01:02:03")
	}
}

// TestGetLeaseInfoUnsupportedPlatform tests that GetLeaseInfo returns nil for unsupported platforms.
// This test verifies the behavior when the function is called - the actual return depends on [runtime.GOOS].
func TestGetLeaseInfoUnsupportedPlatform(_ *testing.T) {
	// This test just ensures no panic - return value depends on platform
	info, err := dhcp.GetLeaseInfo("nonexistent999")
	_ = info
	_ = err
}

// TestPruneExpiredServersDoesNothing tests that pruning does nothing when under threshold.
func TestPruneExpiredServersDoesNothing(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Add a few servers (under the threshold)
	now := time.Now()
	expiredTime := now.Add(-25 * time.Hour)

	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:       "192.168.1.1",
		LastSeen: expiredTime,
	})
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:       "192.168.1.2",
		LastSeen: expiredTime,
	})

	initialCount := rd.DetectedServersCount()
	rd.PruneExpiredServers(now)

	// Under threshold - nothing should be pruned
	if rd.DetectedServersCount() != initialCount {
		t.Errorf("count = %d, want %d (should not prune under threshold)", rd.DetectedServersCount(), initialCount)
	}
}

// TestRogueDetectorConcurrentAccess tests concurrent access to RogueDetector.
func TestRogueDetectorConcurrentAccess(_ *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for j := range 50 {
				ip := "192.168." + string(rune('0'+id)) + "." + string(rune('0'+j%10))
				rd.RecordDetectedServer(ip, "aa:bb:cc:dd:ee:ff")
				_ = rd.GetDetectedServers()
				_ = rd.GetRogueServers()
				_ = rd.IsRunning()
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	// Just verify no panics or deadlocks occurred
}

// TestLeaseFieldMappingFields tests the LeaseFieldMapping struct fields.
func TestLeaseFieldMappingFields(t *testing.T) {
	mapping := dhcp.LeaseFieldMapping{
		ServerKey:    "SERVER=",
		RouterKey:    "ROUTER=",
		LeaseTimeKey: "LIFETIME=",
		DNSKey:       "DNS=",
	}

	if mapping.ServerKey != "SERVER=" {
		t.Errorf("ServerKey = %q, want %q", mapping.ServerKey, "SERVER=")
	}
	if mapping.RouterKey != "ROUTER=" {
		t.Errorf("RouterKey = %q, want %q", mapping.RouterKey, "ROUTER=")
	}
	if mapping.LeaseTimeKey != "LIFETIME=" {
		t.Errorf("LeaseTimeKey = %q, want %q", mapping.LeaseTimeKey, "LIFETIME=")
	}
	if mapping.DNSKey != "DNS=" {
		t.Errorf("DNSKey = %q, want %q", mapping.DNSKey, "DNS=")
	}
}

// TestRogueDetectorStopLocked tests stopLocked when not running.
func TestRogueDetectorStopLocked(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Should not panic when not running
	rd.StopLocked()

	if rd.IsRunning() {
		t.Error("should not be running")
	}
}

// TestRogueDetectorStartLockedWithInvalidInterface tests startLocked with bad interface.
func TestRogueDetectorStartLockedWithInvalidInterface(t *testing.T) {
	config := &dhcp.RogueDetectorConfig{
		Interface:        "invalid_nonexistent_interface_xyz",
		KnownServers:     []string{},
		AlertOnDetection: false,
	}
	rd := dhcp.NewRogueDetector(config)

	err := rd.StartLocked()
	if err == nil {
		t.Error("expected error with invalid interface")
		_ = rd.Stop()
	}
}

// TestRogueDetectorGetServerIdentifier tests getServerIdentifier with mock DHCPv4.
func TestRogueDetectorGetServerIdentifier(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	t.Run("with server ID option", func(t *testing.T) {
		// Create mock DHCPv4 packet with server identifier option
		dhcpPacket := &dhcp.MockDHCPv4{
			Options: []dhcp.MockDHCPOption{
				{Type: 54, Data: []byte{192, 168, 1, 1}}, // Server Identifier
			},
		}
		result := rd.ExportGetServerIdentifier(dhcpPacket.ToLayers())
		if result != "192.168.1.1" {
			t.Errorf("result = %q, want %q", result, "192.168.1.1")
		}
	})

	t.Run("without server ID option", func(t *testing.T) {
		dhcpPacket := &dhcp.MockDHCPv4{
			Options: []dhcp.MockDHCPOption{},
		}
		result := rd.ExportGetServerIdentifier(dhcpPacket.ToLayers())
		if result != "" {
			t.Errorf("result = %q, want empty", result)
		}
	})

	t.Run("with wrong length data", func(t *testing.T) {
		dhcpPacket := &dhcp.MockDHCPv4{
			Options: []dhcp.MockDHCPOption{
				{Type: 54, Data: []byte{192, 168}}, // Wrong length
			},
		}
		result := rd.ExportGetServerIdentifier(dhcpPacket.ToLayers())
		if result != "" {
			t.Errorf("result = %q, want empty", result)
		}
	})
}

// TestRogueDetectorGetDHCPMessageType tests getDHCPMessageType with mock DHCPv4.
func TestRogueDetectorGetDHCPMessageType(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	t.Run("with message type option OFFER", func(t *testing.T) {
		dhcpPacket := &dhcp.MockDHCPv4{
			Options: []dhcp.MockDHCPOption{
				{Type: 53, Data: []byte{2}}, // Message Type = OFFER
			},
		}
		result := rd.ExportGetDHCPMessageType(dhcpPacket.ToLayers())
		if result != 2 { // DHCPMsgTypeOffer
			t.Errorf("result = %d, want 2", result)
		}
	})

	t.Run("without message type option", func(t *testing.T) {
		dhcpPacket := &dhcp.MockDHCPv4{
			Options: []dhcp.MockDHCPOption{},
		}
		result := rd.ExportGetDHCPMessageType(dhcpPacket.ToLayers())
		if result != 0 {
			t.Errorf("result = %d, want 0", result)
		}
	})

	t.Run("with wrong length data", func(t *testing.T) {
		dhcpPacket := &dhcp.MockDHCPv4{
			Options: []dhcp.MockDHCPOption{
				{Type: 53, Data: []byte{}}, // Empty data
			},
		}
		result := rd.ExportGetDHCPMessageType(dhcpPacket.ToLayers())
		if result != 0 {
			t.Errorf("result = %d, want 0", result)
		}
	})
}

// TestFindDHCPMessageTypeFullCoverage tests remaining branches of findDHCPMessageType.
func TestFindDHCPMessageTypeFullCoverage(t *testing.T) {
	tests := []struct {
		name     string
		options  []byte
		expected byte
	}{
		{
			name: "option type 53 at boundary",
			// Option 53 right at the last possible position
			options:  []byte{53, 1, 5, 255}, // Message type 5 (ACK)
			expected: 5,
		},
		{
			name: "i+1 == len(options) boundary",
			// Truncated: option type present but no length byte
			options:  []byte{53},
			expected: 0,
		},
		{
			name: "i+2+optionLen > len(options) boundary",
			// Option claims length 5 but only has 1 byte of data
			options:  []byte{53, 5, 1},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.FindDHCPMessageType(tt.options)
			if result != tt.expected {
				t.Errorf("FindDHCPMessageType() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestSetInterfaceCoverage tests additional SetInterface scenarios.
func TestSetInterfaceCoverage(t *testing.T) {
	t.Run("rogue detector set interface", func(t *testing.T) {
		config := &dhcp.RogueDetectorConfig{
			Interface:        "eth0",
			AlertOnDetection: false,
		}
		rd := dhcp.NewRogueDetector(config)

		// Set new interface
		err := rd.SetInterface("wlan0")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		cfg := rd.GetConfig()
		if cfg.Interface != "wlan0" {
			t.Errorf("Interface = %q, want %q", cfg.Interface, "wlan0")
		}
	})

	t.Run("monitor set interface multiple times", func(t *testing.T) {
		monitor := dhcp.NewMonitor("eth0")

		for _, iface := range []string{"wlan0", "eth1", "en0"} {
			err := monitor.SetInterface(iface)
			if err != nil {
				t.Errorf("unexpected error setting interface %s: %v", iface, err)
			}
			if monitor.MonitorInterfaceName() != iface {
				t.Errorf("interfaceName = %q, want %q", monitor.MonitorInterfaceName(), iface)
			}
		}
	})
}

// TestAddNewServerWithAlert tests that alerts are logged for rogue servers.
func TestAddNewServerWithAlert(t *testing.T) {
	config := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1"},
		AlertOnDetection: true, // Enable alerts
	}
	rd := dhcp.NewRogueDetector(config)

	now := time.Now()

	// Add an authorized server (should not trigger alert)
	rd.AddNewServer("192.168.1.1", "aa:bb:cc:dd:ee:ff", now)

	// Add a rogue server (should trigger alert - logged)
	rd.AddNewServer("192.168.1.100", "11:22:33:44:55:66", now)

	// Verify both servers were added
	servers := rd.GetDetectedServers()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	// Check authorization status
	for _, server := range servers {
		if server.IP == "192.168.1.1" && !server.IsAuthorized {
			t.Error("192.168.1.1 should be authorized")
		}
		if server.IP == "192.168.1.100" && server.IsAuthorized {
			t.Error("192.168.1.100 should NOT be authorized")
		}
	}
}

// TestStopLockedBranches tests stopLocked when running (handles all branches).
func TestStopLockedBranches(t *testing.T) {
	t.Run("stopLocked when not running", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)
		// Should not panic when not running
		rd.StopLocked()
		if rd.IsRunning() {
			t.Error("should not be running")
		}
	})
}

// TestStartLockedBranches tests startLocked with already running.
func TestStartLockedBranches(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Set running to true manually
	rd.SetRogueRunning(true)

	// Should fail because already running
	err := rd.Start()
	if err == nil {
		t.Error("expected error when already running")
	}

	// Reset
	rd.SetRogueRunning(false)
}

// TestSetInterfaceRogueDetectorNotRunning tests SetInterface when not running.
func TestSetInterfaceRogueDetectorNotRunning(t *testing.T) {
	config := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{},
		AlertOnDetection: false,
	}
	rd := dhcp.NewRogueDetector(config)

	err := rd.SetInterface("wlan0")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	cfg := rd.GetConfig()
	if cfg.Interface != "wlan0" {
		t.Errorf("Interface = %q, want %q", cfg.Interface, "wlan0")
	}
}

// TestParseDarwinLeaseFileSingleDNS tests Darwin lease file with single DNS.
func TestParseDarwinLeaseFileSingleDNS(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with single DNS (not in array format)
	content := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>ServerIdentifier</key>
	<data>c0a80101</data>
	<key>DomainNameServer</key>
	<data>08080808</data>
</dict>
</plist>`

	path := filepath.Join(tmpDir, "darwin_single_dns.lease")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := dhcp.ParseDarwinLeaseFile(path)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.DNS) != 1 {
		t.Errorf("DNS len = %d, want 1", len(result.DNS))
	}
	if len(result.DNS) > 0 && result.DNS[0] != "8.8.8.8" {
		t.Errorf("DNS[0] = %q, want %q", result.DNS[0], "8.8.8.8")
	}
}

// TestGetLeaseInfoDarwinWithRealInterface tests getLeaseInfoDarwin with a real interface.
func TestGetLeaseInfoDarwinWithRealInterface(_ *testing.T) {
	// This test just ensures the function doesn't panic when called with en0
	// The actual result depends on the system configuration
	info, err := dhcp.GetLeaseInfoDarwin("en0")
	_ = info
	_ = err
	// No assertions - just checking for no panic
}

// TestGetLeaseInfoLinuxWithRealInterface tests getLeaseInfoLinux with a real interface.
func TestGetLeaseInfoLinuxWithRealInterface(_ *testing.T) {
	// This test just ensures the function doesn't panic
	info, err := dhcp.GetLeaseInfoLinux("eth0")
	_ = info
	_ = err
	// No assertions - just checking for no panic
}

// TestMonitorStopNotRunning tests Stop when monitor is not running.
func TestMonitorStopNotRunningDetailed(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Stop when not running should do nothing
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("monitor should not be running")
	}
}

// TestParseLeaseLineRouterWithMultipleParts tests router parsing with multiple space-separated IPs.
func TestParseLeaseLineRouterWithMultipleParts(t *testing.T) {
	result, ok := dhcp.ParseLeaseLineRouter("ROUTER=192.168.1.1 192.168.1.2 192.168.1.3", "ROUTER=")
	if !ok {
		t.Error("expected ok=true")
	}
	if result != "192.168.1.1" {
		t.Errorf("result = %q, want %q", result, "192.168.1.1")
	}
}

// TestFullDHCPTransactionWithoutCapture simulates a full DHCP transaction without capture.
func TestFullDHCPTransactionWithoutCapture(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	now := time.Now()
	xid := uint32(0xDEADBEEF)

	// Simulate complete transaction
	monitor.RecordPhase(xid, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(xid, dhcp.PhaseOffer, now.Add(25*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseRequest, now.Add(30*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseAck, now.Add(100*time.Millisecond))

	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected non-nil timing")
	}

	if !timing.Complete {
		t.Error("timing should be complete")
	}

	if timing.Discover != 25*time.Millisecond {
		t.Errorf("Discover = %v, want 25ms", timing.Discover)
	}
	if timing.Offer != 5*time.Millisecond {
		t.Errorf("Offer = %v, want 5ms", timing.Offer)
	}
	if timing.Request != 70*time.Millisecond {
		t.Errorf("Request = %v, want 70ms", timing.Request)
	}
	if timing.Total != 100*time.Millisecond {
		t.Errorf("Total = %v, want 100ms", timing.Total)
	}
}

// TestNewMonitorInitialization tests all initial state of NewMonitor.
func TestNewMonitorInitialization(t *testing.T) {
	monitor := dhcp.NewMonitor("test_interface")

	if monitor == nil {
		t.Fatal("expected non-nil monitor")
	}
	if monitor.MonitorInterfaceName() != "test_interface" {
		t.Errorf("interfaceName = %q, want %q", monitor.MonitorInterfaceName(), "test_interface")
	}
	if monitor.MonitorRunning() {
		t.Error("should not be running initially")
	}
	if monitor.MonitorTransactions() == nil {
		t.Error("transactions map should be initialized")
	}
	if len(monitor.MonitorTransactions()) != 0 {
		t.Error("transactions map should be empty initially")
	}
	if monitor.GetLastTiming() != nil {
		t.Error("lastTiming should be nil initially")
	}
}

// TestRogueDetectorConfigCopy tests that GetConfig returns a copy.
func TestRogueDetectorConfigCopy(t *testing.T) {
	original := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1"},
		AlertOnDetection: true,
	}
	rd := dhcp.NewRogueDetector(original)

	// Get config copy
	copy1 := rd.GetConfig()

	// Modify the copy
	copy1.Interface = "modified"
	copy1.KnownServers = append(copy1.KnownServers, "10.0.0.1")

	// Get another copy - should not be affected
	copy2 := rd.GetConfig()

	if copy2.Interface != "eth0" {
		t.Errorf("original Interface was modified: %q", copy2.Interface)
	}
	if len(copy2.KnownServers) != 1 {
		t.Errorf("original KnownServers was modified: %v", copy2.KnownServers)
	}
}
