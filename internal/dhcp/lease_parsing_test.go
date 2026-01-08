package dhcp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/dhcp"
)

// TestParseDarwinLeaseFileWithFixtures tests Darwin lease file parsing using test fixtures.
func TestParseDarwinLeaseFileWithFixtures(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		wantNil        bool
		expectedServer string
		expectedRouter string
		expectedLease  int
		expectedDNS    int
	}{
		{
			name:           "full darwin lease with all fields",
			fixture:        "testdata/darwin_full.plist",
			wantNil:        false,
			expectedServer: "192.168.1.1",
			expectedRouter: "192.168.1.1",
			expectedLease:  86400,
			expectedDNS:    2,
		},
		{
			name:           "darwin lease with Router fallback",
			fixture:        "testdata/darwin_router_fallback.plist",
			wantNil:        false,
			expectedServer: "10.0.0.1",
			expectedRouter: "10.0.0.1",
			expectedLease:  0,
			expectedDNS:    0,
		},
		{
			name:           "darwin lease with single DNS",
			fixture:        "testdata/darwin_single_dns.plist",
			wantNil:        false,
			expectedServer: "192.168.1.1",
			expectedRouter: "",
			expectedLease:  0,
			expectedDNS:    1,
		},
		{
			name:    "darwin lease with no useful data",
			fixture: "testdata/darwin_empty.plist",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(".", tt.fixture)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("fixture not found: %s", path)
			}

			result := dhcp.ParseDarwinLeaseFile(path)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.DHCPServer != tt.expectedServer {
				t.Errorf("DHCPServer = %q, want %q", result.DHCPServer, tt.expectedServer)
			}
			if result.Gateway != tt.expectedRouter {
				t.Errorf("Gateway = %q, want %q", result.Gateway, tt.expectedRouter)
			}
			if result.LeaseTime != tt.expectedLease {
				t.Errorf("LeaseTime = %d, want %d", result.LeaseTime, tt.expectedLease)
			}
			if len(result.DNS) != tt.expectedDNS {
				t.Errorf("DNS count = %d, want %d", len(result.DNS), tt.expectedDNS)
			}
		})
	}
}

// TestParseDHClientLeaseFileWithFixtures tests dhclient lease file parsing using test fixtures.
func TestParseDHClientLeaseFileWithFixtures(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		wantNil        bool
		expectedServer string
		expectedRouter string
		expectedLease  int
	}{
		{
			name:           "single lease block",
			fixture:        "testdata/dhclient_single.lease",
			wantNil:        false,
			expectedServer: "192.168.1.1",
			expectedRouter: "192.168.1.1",
			expectedLease:  86400,
		},
		{
			name:           "multiple lease blocks (takes last)",
			fixture:        "testdata/dhclient_multiple.lease",
			wantNil:        false,
			expectedServer: "192.168.1.2",
			expectedRouter: "192.168.1.254",
			expectedLease:  43200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(".", tt.fixture)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("fixture not found: %s", path)
			}

			result := dhcp.ParseDHClientLeaseFile(path, "eth0")

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.DHCPServer != tt.expectedServer {
				t.Errorf("DHCPServer = %q, want %q", result.DHCPServer, tt.expectedServer)
			}
			if result.Gateway != tt.expectedRouter {
				t.Errorf("Gateway = %q, want %q", result.Gateway, tt.expectedRouter)
			}
			if result.LeaseTime != tt.expectedLease {
				t.Errorf("LeaseTime = %d, want %d", result.LeaseTime, tt.expectedLease)
			}
		})
	}
}

// TestParseNMLeaseFileWithFixture tests NetworkManager lease file parsing using test fixture.
func TestParseNMLeaseFileWithFixture(t *testing.T) {
	path := filepath.Join(".", "testdata/networkmanager.lease")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("fixture not found: %s", path)
	}

	result := dhcp.ParseNMLeaseFile(path)
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
		t.Errorf("DNS count = %d, want 2", len(result.DNS))
	}
}

// TestParseNetworkdLeaseFileWithFixture tests systemd-networkd lease file parsing using test fixture.
func TestParseNetworkdLeaseFileWithFixture(t *testing.T) {
	path := filepath.Join(".", "testdata/networkd.lease")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("fixture not found: %s", path)
	}

	result := dhcp.ParseNetworkdLeaseFile(path)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.DHCPServer != "10.0.0.1" {
		t.Errorf("DHCPServer = %q, want %q", result.DHCPServer, "10.0.0.1")
	}
	if result.Gateway != "10.0.0.1" {
		t.Errorf("Gateway = %q, want %q", result.Gateway, "10.0.0.1")
	}
	if result.LeaseTime != 7200 {
		t.Errorf("LeaseTime = %d, want %d", result.LeaseTime, 7200)
	}
	if len(result.DNS) != 2 {
		t.Errorf("DNS count = %d, want 2", len(result.DNS))
	}
}

// TestHexToIPComprehensive tests hexToIP with comprehensive test cases.
func TestHexToIPComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Valid hex strings
		{"localhost", "7f000001", "127.0.0.1"},
		{"class A", "0a000001", "10.0.0.1"},
		{"class B", "ac100001", "172.16.0.1"},
		{"class C", "c0a80001", "192.168.0.1"},
		{"broadcast", "ffffffff", "255.255.255.255"},
		{"all zeros", "00000000", "0.0.0.0"},
		{"google DNS", "08080808", "8.8.8.8"},
		{"cloudflare DNS", "01010101", "1.1.1.1"},

		// Uppercase hex
		{"uppercase", "C0A80101", "192.168.1.1"},
		{"mixed case", "C0a80101", "192.168.1.1"},

		// With whitespace (gets stripped)
		{"with spaces", "c0 a8 01 01", "192.168.1.1"},
		{"with tabs", "c0\ta8\t01\t01", "192.168.1.1"},
		{"with newlines", "c0\na8\n01\n01", "192.168.1.1"},

		// Raw 4-byte strings (fallback path)
		{"raw bytes", "\xc0\xa8\x01\x01", "192.168.1.1"},
		{"raw bytes localhost", "\x7f\x00\x00\x01", "127.0.0.1"},

		// Invalid cases
		{"empty string", "", ""},
		{"too short", "c0a8", "99.48.97.56"}, // 4 chars interpreted as raw bytes
		{"too long", "c0a8010100", ""},
		{"invalid hex chars", "zzzzzzzz", ""},
		{"odd length", "c0a801", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.HexToIP(tt.input)
			if result != tt.expected {
				t.Errorf("HexToIP(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtractValueComprehensive tests extractValue with comprehensive test cases.
func TestExtractValueComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Standard dhclient option lines
		{"server identifier", "option dhcp-server-identifier 192.168.1.1;", "192.168.1.1"},
		{"routers single", "option routers 192.168.1.1;", "192.168.1.1"},
		{"lease time", "option dhcp-lease-time 86400;", "86400"},
		{"domain name servers", "option domain-name-servers 8.8.8.8, 8.8.4.4;", "8.8.4.4"},

		// Without semicolon
		{"without semicolon", "option routers 192.168.1.1", "192.168.1.1"},

		// Multiple spaces
		{"multiple spaces", "option    routers    192.168.1.1;", "192.168.1.1"},

		// Edge cases
		{"empty string", "", ""},
		{"single value", "value", "value"},
		{"semicolon only", ";", ""},
		{"whitespace only", "   ", ""},
		{"many values", "a b c d e f", "f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ExtractValue(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtractPlistIPComprehensive tests extractPlistIP with comprehensive test cases.
func TestExtractPlistIPComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		expected string
	}{
		{
			name:     "basic server identifier",
			content:  "<key>ServerIdentifier</key><data>c0a80101</data>",
			key:      "ServerIdentifier",
			expected: "192.168.1.1",
		},
		{
			name:     "router address",
			content:  "<key>RouterIPAddress</key><data>0a000001</data>",
			key:      "RouterIPAddress",
			expected: "10.0.0.1",
		},
		{
			name:     "data with whitespace",
			content:  "<key>ServerIdentifier</key><data>  c0a80101  </data>",
			key:      "ServerIdentifier",
			expected: "192.168.1.1",
		},
		{
			name: "multiline content",
			content: `<dict>
				<key>ServerIdentifier</key>
				<data>c0a80101</data>
			</dict>`,
			key:      "ServerIdentifier",
			expected: "192.168.1.1",
		},
		{
			name:     "missing key",
			content:  "<key>OtherKey</key><data>c0a80101</data>",
			key:      "ServerIdentifier",
			expected: "",
		},
		{
			name:     "missing data tag",
			content:  "<key>ServerIdentifier</key><string>192.168.1.1</string>",
			key:      "ServerIdentifier",
			expected: "",
		},
		{
			name:     "unclosed data tag",
			content:  "<key>ServerIdentifier</key><data>c0a80101",
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
			name:     "key only, no data",
			content:  "<key>ServerIdentifier</key>",
			key:      "ServerIdentifier",
			expected: "",
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

// TestExtractPlistIntegerComprehensive tests extractPlistInteger with comprehensive test cases.
func TestExtractPlistIntegerComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		expected int
	}{
		{
			name:     "standard lease length",
			content:  "<key>LeaseLength</key><integer>86400</integer>",
			key:      "LeaseLength",
			expected: 86400,
		},
		{
			name:     "zero value",
			content:  "<key>LeaseLength</key><integer>0</integer>",
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "large value",
			content:  "<key>LeaseLength</key><integer>604800</integer>",
			key:      "LeaseLength",
			expected: 604800,
		},
		{
			name:     "with whitespace",
			content:  "<key>LeaseLength</key><integer>  86400  </integer>",
			key:      "LeaseLength",
			expected: 86400,
		},
		{
			name:     "missing key",
			content:  "<key>OtherKey</key><integer>86400</integer>",
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "missing integer tag",
			content:  "<key>LeaseLength</key><string>86400</string>",
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "invalid integer",
			content:  "<key>LeaseLength</key><integer>notanumber</integer>",
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "unclosed integer tag",
			content:  "<key>LeaseLength</key><integer>86400",
			key:      "LeaseLength",
			expected: 0,
		},
		{
			name:     "negative value",
			content:  "<key>LeaseLength</key><integer>-1</integer>",
			key:      "LeaseLength",
			expected: -1,
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

// TestExtractPlistIPArrayComprehensive tests extractPlistIPArray with comprehensive test cases.
func TestExtractPlistIPArrayComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		expected []string
	}{
		{
			name:     "array with single IP",
			content:  "<key>DomainNameServer</key><array><data>08080808</data></array>",
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8"},
		},
		{
			name:     "array with multiple IPs",
			content:  "<key>DomainNameServer</key><array><data>08080808</data><data>08080404</data></array>",
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:     "array with three IPs",
			content:  "<key>DomainNameServer</key><array><data>08080808</data><data>08080404</data><data>01010101</data></array>",
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},
		},
		{
			name:     "single IP not in array (fallback)",
			content:  "<key>DomainNameServer</key><data>08080808</data>",
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8"},
		},
		{
			name:     "empty array",
			content:  "<key>DomainNameServer</key><array></array>",
			key:      "DomainNameServer",
			expected: nil,
		},
		{
			name:     "missing key",
			content:  "<key>OtherKey</key><array><data>08080808</data></array>",
			key:      "DomainNameServer",
			expected: nil,
		},
		{
			name: "multiline array",
			content: `<key>DomainNameServer</key>
			<array>
				<data>08080808</data>
				<data>08080404</data>
			</array>`,
			key:      "DomainNameServer",
			expected: []string{"8.8.8.8", "8.8.4.4"},
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

// TestParseDHClientLeaseLineComprehensive tests parseDHClientLeaseLine with comprehensive cases.
func TestParseDHClientLeaseLineComprehensive(t *testing.T) {
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
			name:           "routers single IP",
			line:           "option routers 192.168.1.1;",
			expectedRouter: "192.168.1.1",
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
			name: "interface line (ignored)",
			line: "interface \"eth0\";",
		},
		{
			name: "fixed-address (ignored)",
			line: "fixed-address 192.168.1.100;",
		},
		{
			name: "renew line (ignored)",
			line: "renew 1 2024/01/01 12:00:00;",
		},
		{
			name: "empty line",
			line: "",
		},
		{
			name: "whitespace only",
			line: "   ",
		},
		{
			// Note: parseDHClientLeaseLine uses HasPrefix, so extra spaces after "option" won't match
			name: "server with multiple spaces (doesn't match prefix)",
			line: "option   dhcp-server-identifier   10.0.0.1;",
			// expectedServer: "", // Won't match due to extra spaces
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

// TestParseLeaseLineServerComprehensive tests parseLeaseLineServer comprehensively.
func TestParseLeaseLineServerComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		serverKey  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "NetworkManager style",
			line:       "DHCP4_SERVER_ID=192.168.1.1",
			serverKey:  "DHCP4_SERVER_ID=",
			expected:   "192.168.1.1",
			expectedOk: true,
		},
		{
			name:       "systemd-networkd style",
			line:       "SERVER_ADDRESS=10.0.0.1",
			serverKey:  "SERVER_ADDRESS=",
			expected:   "10.0.0.1",
			expectedOk: true,
		},
		{
			name:       "key mismatch",
			line:       "DHCP4_ROUTERS=192.168.1.1",
			serverKey:  "DHCP4_SERVER_ID=",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "empty value",
			line:       "DHCP4_SERVER_ID=",
			serverKey:  "DHCP4_SERVER_ID=",
			expected:   "",
			expectedOk: true,
		},
		{
			name:       "partial key match",
			line:       "DHCP4_SERVER_ID_NEW=192.168.1.1",
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

// TestParseLeaseLineRouterComprehensive tests parseLeaseLineRouter comprehensively.
func TestParseLeaseLineRouterComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		routerKey  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "single router",
			line:       "DHCP4_ROUTERS=192.168.1.1",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "192.168.1.1",
			expectedOk: true,
		},
		{
			name:       "multiple routers (takes first)",
			line:       "DHCP4_ROUTERS=192.168.1.1 192.168.1.2",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "192.168.1.1",
			expectedOk: true,
		},
		{
			name:       "multiple routers with three IPs",
			line:       "DHCP4_ROUTERS=10.0.0.1 10.0.0.2 10.0.0.3",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "10.0.0.1",
			expectedOk: true,
		},
		{
			name:       "systemd-networkd ROUTER",
			line:       "ROUTER=10.0.0.1",
			routerKey:  "ROUTER=",
			expected:   "10.0.0.1",
			expectedOk: true,
		},
		{
			name:       "empty value",
			line:       "DHCP4_ROUTERS=",
			routerKey:  "DHCP4_ROUTERS=",
			expected:   "",
			expectedOk: true,
		},
		{
			name:       "key mismatch",
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

// TestParseLeaseLineTimeComprehensive tests parseLeaseLineTime comprehensively.
func TestParseLeaseLineTimeComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		leaseTimeKey string
		expected     int
		expectedOk   bool
	}{
		{
			name:         "standard lease time (1 day)",
			line:         "DHCP4_LEASE_TIME=86400",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     86400,
			expectedOk:   true,
		},
		{
			name:         "short lease time (1 hour)",
			line:         "DHCP4_LEASE_TIME=3600",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     3600,
			expectedOk:   true,
		},
		{
			name:         "long lease time (1 week)",
			line:         "DHCP4_LEASE_TIME=604800",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     604800,
			expectedOk:   true,
		},
		{
			name:         "systemd-networkd LIFETIME",
			line:         "LIFETIME=7200",
			leaseTimeKey: "LIFETIME=",
			expected:     7200,
			expectedOk:   true,
		},
		{
			name:         "zero",
			line:         "DHCP4_LEASE_TIME=0",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     0,
			expectedOk:   true,
		},
		{
			name:         "not a number",
			line:         "DHCP4_LEASE_TIME=infinite",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     0,
			expectedOk:   false,
		},
		{
			name:         "key mismatch",
			line:         "DHCP4_SERVER_ID=192.168.1.1",
			leaseTimeKey: "DHCP4_LEASE_TIME=",
			expected:     0,
			expectedOk:   false,
		},
		{
			name:         "empty value",
			line:         "DHCP4_LEASE_TIME=",
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

// TestParseLeaseLineDNSComprehensive tests parseLeaseLineDNS comprehensively.
func TestParseLeaseLineDNSComprehensive(t *testing.T) {
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
			name:     "two DNS servers",
			line:     "DHCP4_DOMAIN_NAME_SERVERS=8.8.8.8 8.8.4.4",
			dnsKey:   "DHCP4_DOMAIN_NAME_SERVERS=",
			expected: []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:     "three DNS servers",
			line:     "DHCP4_DOMAIN_NAME_SERVERS=8.8.8.8 8.8.4.4 1.1.1.1",
			dnsKey:   "DHCP4_DOMAIN_NAME_SERVERS=",
			expected: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},
		},
		{
			name:     "systemd-networkd DNS",
			line:     "DNS=10.0.0.1 10.0.0.2",
			dnsKey:   "DNS=",
			expected: []string{"10.0.0.1", "10.0.0.2"},
		},
		{
			name:     "key mismatch",
			line:     "DHCP4_SERVER_ID=192.168.1.1",
			dnsKey:   "DHCP4_DOMAIN_NAME_SERVERS=",
			expected: nil,
		},
		{
			name:     "empty value",
			line:     "DHCP4_DOMAIN_NAME_SERVERS=",
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

// TestProcessLeaseLineComprehensive tests processLeaseLine comprehensively.
func TestProcessLeaseLineComprehensive(t *testing.T) {
	nmMapping := dhcp.LeaseFieldMapping{
		ServerKey:    "DHCP4_SERVER_ID=",
		RouterKey:    "DHCP4_ROUTERS=",
		LeaseTimeKey: "DHCP4_LEASE_TIME=",
		DNSKey:       "DHCP4_DOMAIN_NAME_SERVERS=",
	}

	networkdMapping := dhcp.LeaseFieldMapping{
		ServerKey:    "SERVER_ADDRESS=",
		RouterKey:    "ROUTER=",
		LeaseTimeKey: "LIFETIME=",
		DNSKey:       "DNS=",
	}

	tests := []struct {
		name           string
		line           string
		mapping        dhcp.LeaseFieldMapping
		expectedServer string
		expectedRouter string
		expectedLease  int
		expectedDNS    []string
	}{
		{
			name:           "NM server line",
			line:           "DHCP4_SERVER_ID=192.168.1.1",
			mapping:        nmMapping,
			expectedServer: "192.168.1.1",
		},
		{
			name:           "NM router line",
			line:           "DHCP4_ROUTERS=192.168.1.1",
			mapping:        nmMapping,
			expectedRouter: "192.168.1.1",
		},
		{
			name:          "NM lease time line",
			line:          "DHCP4_LEASE_TIME=86400",
			mapping:       nmMapping,
			expectedLease: 86400,
		},
		{
			name:        "NM DNS line",
			line:        "DHCP4_DOMAIN_NAME_SERVERS=8.8.8.8 8.8.4.4",
			mapping:     nmMapping,
			expectedDNS: []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:           "networkd server line",
			line:           "SERVER_ADDRESS=10.0.0.1",
			mapping:        networkdMapping,
			expectedServer: "10.0.0.1",
		},
		{
			name:           "networkd router line",
			line:           "ROUTER=10.0.0.1",
			mapping:        networkdMapping,
			expectedRouter: "10.0.0.1",
		},
		{
			name:          "networkd lifetime line",
			line:          "LIFETIME=7200",
			mapping:       networkdMapping,
			expectedLease: 7200,
		},
		{
			name:        "networkd DNS line",
			line:        "DNS=10.0.0.1 10.0.0.2",
			mapping:     networkdMapping,
			expectedDNS: []string{"10.0.0.1", "10.0.0.2"},
		},
		{
			name:    "unrelated line",
			line:    "HOSTNAME=myserver",
			mapping: nmMapping,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &dhcp.LeaseInfo{}
			dhcp.ProcessLeaseLine(tt.line, tt.mapping, info)

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

// TestExtractArrayContentComprehensive tests extractArrayContent comprehensively.
func TestExtractArrayContentComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		remaining  string
		expected   string
		expectedOk bool
	}{
		{
			name:       "basic array with data",
			remaining:  "<array><data>08080808</data></array>",
			expected:   "<data>08080808</data>",
			expectedOk: true,
		},
		{
			name:       "empty array",
			remaining:  "<array></array>",
			expected:   "",
			expectedOk: true,
		},
		{
			name:       "array with multiple data",
			remaining:  "<array><data>08080808</data><data>08080404</data></array>",
			expected:   "<data>08080808</data><data>08080404</data>",
			expectedOk: true,
		},
		{
			name:       "nested elements",
			remaining:  "<array><dict><key>k</key><data>v</data></dict></array>",
			expected:   "<dict><key>k</key><data>v</data></dict>",
			expectedOk: true,
		},
		{
			name:       "array with whitespace",
			remaining:  "<array>  <data>08080808</data>  </array>",
			expected:   "  <data>08080808</data>  ",
			expectedOk: true,
		},
		{
			name:       "no array tag",
			remaining:  "<data>08080808</data>",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "unclosed array",
			remaining:  "<array><data>08080808</data>",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "extra content after array",
			remaining:  "<array><data>08080808</data></array><more>stuff</more>",
			expected:   "<data>08080808</data>",
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := dhcp.ExtractArrayContent(tt.remaining)
			if ok != tt.expectedOk {
				t.Errorf("ok = %v, want %v", ok, tt.expectedOk)
			}
			if result != tt.expected {
				t.Errorf("result = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestParseDataTagsFromArrayComprehensive tests parseDataTagsFromArray comprehensively.
func TestParseDataTagsFromArrayComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		arrayContent string
		expected     []string
	}{
		{
			name:         "single data tag",
			arrayContent: "<data>08080808</data>",
			expected:     []string{"8.8.8.8"},
		},
		{
			name:         "multiple data tags",
			arrayContent: "<data>08080808</data><data>08080404</data>",
			expected:     []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:         "three data tags",
			arrayContent: "<data>08080808</data><data>08080404</data><data>01010101</data>",
			expected:     []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},
		},
		{
			name:         "with whitespace",
			arrayContent: "  <data>08080808</data>  <data>08080404</data>  ",
			expected:     []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:         "empty content",
			arrayContent: "",
			expected:     nil,
		},
		{
			name:         "no data tags",
			arrayContent: "<string>8.8.8.8</string>",
			expected:     nil,
		},
		{
			name:         "unclosed data tag",
			arrayContent: "<data>08080808",
			expected:     nil,
		},
		{
			name:         "invalid hex in data",
			arrayContent: "<data>zzzzzzzz</data>",
			expected:     nil, // hexToIP returns empty for invalid
		},
		{
			name:         "mixed valid and invalid",
			arrayContent: "<data>08080808</data><data>zzzzzzzz</data><data>01010101</data>",
			expected:     []string{"8.8.8.8", "1.1.1.1"}, // Invalid is skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ParseDataTagsFromArray(tt.arrayContent)
			if len(result) != len(tt.expected) {
				t.Errorf("len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, ip, tt.expected[i])
				}
			}
		})
	}
}
