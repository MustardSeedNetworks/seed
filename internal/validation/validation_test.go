package validation

import (
	"errors"
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Loopback addresses
		{"loopback IPv4", "127.0.0.1", true},
		{"loopback IPv4 range", "127.255.255.255", true},

		// Private IPv4 ranges
		{"private 10.x.x.x", "10.0.0.1", true},
		{"private 10.x.x.x max", "10.255.255.255", true},
		{"private 172.16.x.x", "172.16.0.1", true},
		{"private 172.31.x.x", "172.31.255.255", true},
		{"private 192.168.x.x", "192.168.0.1", true},
		{"private 192.168.x.x max", "192.168.255.255", true},
		{"link-local 169.254.x.x", "169.254.0.1", true},

		// Public addresses
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 1.1.1.1", "1.1.1.1", false},
		{"public 93.184.216.34", "93.184.216.34", false},
		{"public 172.15.255.255", "172.15.255.255", false}, // Just below 172.16.x.x
		{"public 172.32.0.0", "172.32.0.0", false},         // Just above 172.31.x.x
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := IsPrivateIP(ip)
			if got != tt.expected {
				t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid public URLs
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com", false},
		{"valid with path", "https://example.com/path", false},
		{"valid with port", "https://example.com:8080", false},

		// Invalid URLs
		{"empty url", "", true},
		{"private IP 192.168", "http://192.168.1.1", true},
		{"private IP 10.x", "http://10.0.0.1", true},
		{"private IP 172.16", "http://172.16.0.1", true},
		{"localhost", "http://127.0.0.1", true},
		{"localhost name", "http://localhost", false}, // This passes validation (hostname is valid)

		// URLs without scheme get https:// prefix
		{"no scheme", "example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestSafeTransport(t *testing.T) {
	// Test that SafeTransport can be created
	transport := SafeTransport()
	if transport == nil {
		t.Fatal("SafeTransport() returned nil")
	}

	if transport.DialContext == nil {
		t.Error("SafeTransport should have custom DialContext")
	}

	if !transport.DisableKeepAlives {
		t.Error("SafeTransport should have DisableKeepAlives set")
	}
}

func TestSafeHTTPClient(t *testing.T) {
	// Test that SafeHTTPClient can be created
	client := SafeHTTPClient(10)
	if client == nil {
		t.Fatal("SafeHTTPClient() returned nil")
	}

	if client.Transport == nil {
		t.Error("SafeHTTPClient should have Transport set")
	}
}

func TestErrPrivateIPBlocked(t *testing.T) {
	// Test that the error sentinel is defined correctly
	if ErrPrivateIPBlocked == nil {
		t.Fatal("ErrPrivateIPBlocked should not be nil")
	}

	// Test that it can be used with errors.Is
	wrappedErr := errors.New("wrapped: " + ErrPrivateIPBlocked.Error())
	if errors.Is(wrappedErr, ErrPrivateIPBlocked) {
		t.Error("non-wrapped error should not match")
	}
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:db8::1", true},
		{"invalid", "not-an-ip", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidIP(tt.ip)
			if got != tt.expected {
				t.Errorf("IsValidIP(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIsValidHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{"simple", "example", true},
		{"with domain", "example.com", true},
		{"with subdomain", "sub.example.com", true},
		{"with hyphen", "my-host.example.com", true},
		{"empty", "", false},
		{"starts with hyphen", "-example.com", false},
		{"too long", string(make([]byte, 254)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidHostname(tt.hostname)
			if got != tt.expected {
				t.Errorf("IsValidHostname(%q) = %v, want %v", tt.hostname, got, tt.expected)
			}
		})
	}
}
