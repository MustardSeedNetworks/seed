package httpapi_test

import (
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestExtractHostFromOrigin tests host extraction from origin URLs.
func TestExtractHostFromOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		wantHost string
		wantOK   bool
	}{
		{
			name:     "http scheme",
			origin:   "http://192.168.1.1",
			wantHost: "192.168.1.1",
			wantOK:   true,
		},
		{
			name:     "https scheme",
			origin:   "https://192.168.1.1",
			wantHost: "192.168.1.1",
			wantOK:   true,
		},
		{
			name:     "http with port",
			origin:   "http://192.168.1.1:8080",
			wantHost: "192.168.1.1",
			wantOK:   true,
		},
		{
			name:     "https with port",
			origin:   "https://192.168.1.1:8443",
			wantHost: "192.168.1.1",
			wantOK:   true,
		},
		{
			name:     "localhost",
			origin:   "http://localhost",
			wantHost: "localhost",
			wantOK:   true,
		},
		{
			name:     "localhost with port",
			origin:   "http://localhost:3000",
			wantHost: "localhost",
			wantOK:   true,
		},
		{
			name:     "no scheme",
			origin:   "192.168.1.1",
			wantHost: "",
			wantOK:   false,
		},
		{
			name:     "invalid scheme",
			origin:   "ftp://192.168.1.1",
			wantHost: "",
			wantOK:   false,
		},
		{
			name:     "empty origin",
			origin:   "",
			wantHost: "",
			wantOK:   false,
		},
		{
			name:     "with path",
			origin:   "http://192.168.1.1/path",
			wantHost: "192.168.1.1",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, ok := api.ExportExtractHostFromOrigin(tt.origin)
			if ok != tt.wantOK {
				t.Errorf("ExtractHostFromOrigin(%q) ok = %v, want %v", tt.origin, ok, tt.wantOK)
			}
			if host != tt.wantHost {
				t.Errorf("ExtractHostFromOrigin(%q) host = %q, want %q", tt.origin, host, tt.wantHost)
			}
		})
	}
}

// TestIsLocalhostAddress tests localhost address detection.
func TestIsLocalhostAddress(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"[::1]", true},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := api.ExportIsLocalhostAddress(tt.host)
			if result != tt.expected {
				t.Errorf("IsLocalhostAddress(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

// TestIsPrivateNetworkAddress tests RFC 1918 private network address detection.
func TestIsPrivateNetworkAddress(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		// Class A: 10.0.0.0/8
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		// Class B: 172.16.0.0/12
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		// Class C: 192.168.0.0/16
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		// Public IPs
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		// Invalid
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := api.ExportIsPrivateNetworkAddress(tt.host)
			if result != tt.expected {
				t.Errorf("IsPrivateNetworkAddress(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

// TestIsValidClassCAddress tests Class C (192.168.x.x) address validation.
func TestIsValidClassCAddress(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"192.168.0.1", true},
		{"192.168.1.1", true},
		{"192.168.255.255", true},
		{"192.168.1.1.evil.com", false}, // Subdomain attack prevention
		{"192.168.1", false},
		{"192.168.1.256", false},
		{"192.168.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := api.ExportIsValidClassCAddress(tt.host)
			if result != tt.expected {
				t.Errorf("IsValidClassCAddress(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

// TestIsValidClassAAddress tests Class A (10.x.x.x) address validation.
func TestIsValidClassAAddress(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"10.0.0.1.evil.com", false}, // Subdomain attack prevention
		{"10.0.0", false},
		{"10.0.0.256", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := api.ExportIsValidClassAAddress(tt.host)
			if result != tt.expected {
				t.Errorf("IsValidClassAAddress(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

// TestIsValidClassBAddress tests Class B (172.16-31.x.x) address validation.
func TestIsValidClassBAddress(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.20.100.50", true},
		{"172.15.0.1", false},          // Below range
		{"172.32.0.1", false},          // Above range
		{"172.16.0.1.evil.com", false}, // Subdomain attack
		{"172.16.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := api.ExportIsValidClassBAddress(tt.host)
			if result != tt.expected {
				t.Errorf("IsValidClassBAddress(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

// TestIsRFC1918Origin tests RFC 1918 origin validation.
func TestIsRFC1918Origin(t *testing.T) {
	tests := []struct {
		origin   string
		expected bool
	}{
		// Localhost
		{"http://localhost", true},
		{"https://localhost", true},
		{"http://localhost:3000", true},
		{"http://127.0.0.1", true},
		// Note: IPv6 [::1] origin is not currently supported by isRFC1918Origin
		// The function only checks IPv4 private addresses
		{"http://[::1]", false},
		// Class A
		{"http://10.0.0.1", true},
		{"https://10.0.0.1:8443", true},
		// Class B
		{"http://172.16.0.1", true},
		{"http://172.31.255.255", true},
		{"http://172.15.0.1", false},
		{"http://172.32.0.1", false},
		// Class C
		{"http://192.168.1.1", true},
		{"https://192.168.1.1:8443", true},
		// Public IPs (should be rejected)
		{"http://8.8.8.8", false},
		{"https://1.1.1.1", false},
		// Invalid
		{"null", false},
		{"", false},
		{"ftp://192.168.1.1", false},
		// Subdomain attack prevention
		{"http://192.168.1.1.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			result := api.ExportIsRFC1918Origin(tt.origin)
			if result != tt.expected {
				t.Errorf("IsRFC1918Origin(%q) = %v, want %v", tt.origin, result, tt.expected)
			}
		})
	}
}

// TestMatchesAllowedOrigin tests origin pattern matching.
func TestMatchesAllowedOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  string
		expected bool
	}{
		{
			name:     "exact match",
			origin:   "https://192.168.1.1:8443",
			allowed:  "https://192.168.1.1:8443",
			expected: true,
		},
		{
			name:     "wildcard",
			origin:   "https://example.com",
			allowed:  "*",
			expected: true,
		},
		{
			name:     "prefix match",
			origin:   "http://192.168.1.100",
			allowed:  "http://192.168.",
			expected: true,
		},
		{
			name:     "no match",
			origin:   "https://example.com",
			allowed:  "https://other.com",
			expected: false,
		},
		{
			name:     "empty allowed",
			origin:   "https://example.com",
			allowed:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.ExportMatchesAllowedOrigin(tt.origin, tt.allowed)
			if result != tt.expected {
				t.Errorf("MatchesAllowedOrigin(%q, %q) = %v, want %v",
					tt.origin, tt.allowed, result, tt.expected)
			}
		})
	}
}

// TestMatchesOriginPrefix tests prefix matching with boundary validation.
func TestMatchesOriginPrefix(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  string
		expected bool
	}{
		{
			name:     "IP prefix with valid octet",
			origin:   "http://192.168.1.100",
			allowed:  "http://192.168.",
			expected: true,
		},
		{
			name:     "IP prefix with port",
			origin:   "http://192.168.1.100:8080",
			allowed:  "http://192.168.",
			expected: true,
		},
		{
			name:     "partial domain match - should fail",
			origin:   "http://192.168.evil.com",
			allowed:  "http://192.168.",
			expected: false,
		},
		{
			name:     "too short origin",
			origin:   "http://",
			allowed:  "http://192.168.",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.ExportMatchesOriginPrefix(tt.origin, tt.allowed)
			if result != tt.expected {
				t.Errorf("MatchesOriginPrefix(%q, %q) = %v, want %v",
					tt.origin, tt.allowed, result, tt.expected)
			}
		})
	}
}

// TestFindOctetEnd tests finding the end of a numeric octet.
func TestFindOctetEnd(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123", 3},
		{"12", 2},
		{"1", 1},
		{"", 0},
		{"1.2", 1},
		{"12:8080", 2},
		{"256", 3},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := api.ExportFindOctetEnd(tt.input)
			if result != tt.expected {
				t.Errorf("FindOctetEnd(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsValidOctetBoundary tests octet boundary character validation.
func TestIsValidOctetBoundary(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'.', true},
		{':', true},
		{'/', true},
		{'a', false},
		{'1', false},
		{' ', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := api.ExportIsValidOctetBoundary(tt.char)
			if result != tt.expected {
				t.Errorf("IsValidOctetBoundary(%q) = %v, want %v", tt.char, result, tt.expected)
			}
		})
	}
}

// TestIsAllowedWSOriginWithConfigured tests origin validation with configured origins.
func TestIsAllowedWSOriginWithConfigured(t *testing.T) {
	// Save original state
	defer api.ClearAllowedOrigins()

	tests := []struct {
		name              string
		configuredOrigins []string
		origin            string
		expected          bool
	}{
		{
			name:              "default RFC 1918 - localhost",
			configuredOrigins: nil,
			origin:            "http://localhost:3000",
			expected:          true,
		},
		{
			name:              "default RFC 1918 - private IP",
			configuredOrigins: nil,
			origin:            "http://192.168.1.1",
			expected:          true,
		},
		{
			name:              "default RFC 1918 - public IP rejected",
			configuredOrigins: nil,
			origin:            "http://8.8.8.8",
			expected:          false,
		},
		{
			name:              "configured wildcard",
			configuredOrigins: []string{"*"},
			origin:            "https://any.example.com",
			expected:          true,
		},
		{
			name:              "configured exact match",
			configuredOrigins: []string{"https://example.com"},
			origin:            "https://example.com",
			expected:          true,
		},
		{
			name:              "configured prefix match",
			configuredOrigins: []string{"https://192.168."},
			origin:            "https://192.168.1.100",
			expected:          true,
		},
		{
			name:              "configured - no match",
			configuredOrigins: []string{"https://allowed.com"},
			origin:            "https://not-allowed.com",
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api.SetAllowedOrigins(tt.configuredOrigins)
			result := api.ExportIsAllowedWSOrigin(tt.origin)
			if result != tt.expected {
				t.Errorf("IsAllowedWSOrigin(%q) with config %v = %v, want %v",
					tt.origin, tt.configuredOrigins, result, tt.expected)
			}
		})
	}
}

// TestHubBroadcastCardUpdateForInterface tests interface-scoped card updates.
func TestHubBroadcastCardUpdateForInterface(t *testing.T) {
	t.Parallel()

	hub := api.NewHub()

	// Should not panic with no clients
	hub.BroadcastCardUpdateForInterface("link", map[string]string{"status": "up"}, "eth0")
}

// TestHubBroadcastLogEntry tests log entry broadcasting.
func TestHubBroadcastLogEntry(t *testing.T) {
	t.Parallel()

	hub := api.NewHub()

	// Should not panic with no clients
	hub.BroadcastLogEntry(map[string]string{"level": "info", "message": "test"})
}

// TestMessageStruct tests the Message struct fields.
func TestMessageStruct(t *testing.T) {
	msg := api.Message{
		Type:    "test_type",
		Payload: map[string]string{"key": "value"},
	}

	if msg.Type != "test_type" {
		t.Errorf("Expected Type 'test_type', got %q", msg.Type)
	}

	payload, ok := msg.Payload.(map[string]string)
	if !ok {
		t.Fatal("Expected payload to be map[string]string")
	}

	if payload["key"] != "value" {
		t.Errorf("Expected payload key 'value', got %q", payload["key"])
	}
}

// TestCardUpdateStruct tests the CardUpdate struct fields.
func TestCardUpdateStruct(t *testing.T) {
	update := api.CardUpdate{
		CardID:    "link",
		Data:      map[string]any{"status": "up"},
		Interface: "eth0",
	}

	if update.CardID != "link" {
		t.Errorf("Expected CardID 'link', got %q", update.CardID)
	}

	if update.Interface != "eth0" {
		t.Errorf("Expected Interface 'eth0', got %q", update.Interface)
	}
}

// TestIPOctetValidation tests IP octet validation.
func TestIPOctetValidation(t *testing.T) {
	tests := []struct {
		octet    string
		expected bool
	}{
		{"0", true},
		{"1", true},
		{"255", true},
		{"256", false},
		{"", false},
		{"abc", false},
		{"-1", false},
		{"1000", false},
		{"01", true}, // Leading zeros are valid octets
	}

	for _, tt := range tests {
		t.Run(tt.octet, func(t *testing.T) {
			result := api.ExportIsValidIPOctet(tt.octet)
			if result != tt.expected {
				t.Errorf("IsValidIPOctet(%q) = %v, want %v", tt.octet, result, tt.expected)
			}
		})
	}
}
