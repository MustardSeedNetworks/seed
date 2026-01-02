// Package validation_test tests the validation package.
package validation_test

import (
	"errors"
	"net"
	"testing"

	"github.com/krisarmstrong/seed/internal/validation"
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
			got := validation.IsPrivateIP(ip)
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
			err := validation.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestSafeTransport(t *testing.T) {
	// Test that SafeTransport can be created
	transport := validation.SafeTransport()
	if transport == nil {
		t.Fatal("SafeTransport() returned nil")
	}

	if transport.DialContext == nil {
		t.Fatal("SafeTransport should have custom DialContext")
	}

	if !transport.DisableKeepAlives {
		t.Error("SafeTransport should have DisableKeepAlives set")
	}
}

func TestSafeHTTPClient(t *testing.T) {
	// Test that SafeHTTPClient can be created
	client := validation.SafeHTTPClient(10)
	if client == nil {
		t.Fatal("SafeHTTPClient() returned nil")
	}

	if client.Transport == nil {
		t.Fatal("SafeHTTPClient should have Transport set")
	}
}

func TestErrPrivateIPBlocked(t *testing.T) {
	// Test that the error sentinel is defined correctly
	if validation.ErrPrivateIPBlocked == nil {
		t.Fatal("ErrPrivateIPBlocked should not be nil")
	}

	// Test that it can be used with errors.Is
	wrappedErr := errors.New("wrapped: " + validation.ErrPrivateIPBlocked.Error())
	if errors.Is(wrappedErr, validation.ErrPrivateIPBlocked) {
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
			got := validation.IsValidIP(tt.ip)
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
			got := validation.IsValidHostname(tt.hostname)
			if got != tt.expected {
				t.Errorf("IsValidHostname(%q) = %v, want %v", tt.hostname, got, tt.expected)
			}
		})
	}
}

func TestIsValidIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 zero", "0.0.0.0", true},
		{"valid IPv4 broadcast", "255.255.255.255", true},
		{"IPv6", "2001:db8::1", false},
		{"IPv6 loopback", "::1", false},
		{"invalid", "not-an-ip", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.IsValidIPv4(tt.ip)
			if got != tt.expected {
				t.Errorf("IsValidIPv4(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIsValidHostOrIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid hostname", "example.com", true},
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:db8::1", true},
		{"invalid", "!!invalid!!", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.IsValidHostOrIP(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidHostOrIP(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateServerAddress(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		wantErr bool
	}{
		{"valid IP", "8.8.8.8", false},
		{"valid hostname", "dns.google.com", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 254)), true},
		{"invalid chars", "!!bad!!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateServerAddress(tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerAddress(%q) error = %v, wantErr %v", tt.server, err, tt.wantErr)
			}
		})
	}
}

func TestIsValidInterface(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		expected bool
	}{
		{"eth0", "eth0", true},
		{"enp0s3", "enp0s3", true},
		{"wlan0", "wlan0", true},
		{"lo", "lo", true},
		{"en0", "en0", true},
		{"empty", "", false},
		{"too long", "verylonginterfacename", false},
		{"starts with number", "0eth", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.IsValidInterface(tt.iface)
			if got != tt.expected {
				t.Errorf("IsValidInterface(%q) = %v, want %v", tt.iface, got, tt.expected)
			}
		})
	}
}

func TestValidateInterface(t *testing.T) {
	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{"valid eth0", "eth0", false},
		{"valid en0", "en0", false},
		{"empty", "", true},
		{"too long", "verylonginterfacename", true},
		{"invalid", "0invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateInterface(tt.iface)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInterface(%q) error = %v, wantErr %v", tt.iface, err, tt.wantErr)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"https url", "https://example.com", true},
		{"http url", "http://example.com", true},
		{"with path", "https://example.com/path", true},
		{"with port", "https://example.com:8080", true},
		{"empty", "", false},
		{"no scheme", "example.com", false},
		{"ftp scheme", "ftp://example.com", false},
		{"no host", "https://", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.IsValidURL(tt.url)
			if got != tt.expected {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid 80", 80, false},
		{"valid 443", 443, false},
		{"valid 1", 1, false},
		{"valid 65535", 65535, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNetmask(t *testing.T) {
	tests := []struct {
		name    string
		netmask string
		wantErr bool
	}{
		{"valid /24", "255.255.255.0", false},
		{"valid /16", "255.255.0.0", false},
		{"valid /8", "255.0.0.0", false},
		{"invalid format", "not-a-netmask", true},
		{"IPv6", "::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateNetmask(tt.netmask)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetmask(%q) error = %v, wantErr %v", tt.netmask, err, tt.wantErr)
			}
		})
	}
}

func TestIsPrivateIPLinkLocal(t *testing.T) {
	// Test link-local addresses
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"link-local IPv4", "169.254.1.1", true},
		{"link-local IPv6", "fe80::1", true},
		{"multicast", "224.0.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := validation.IsPrivateIP(ip)
			if got != tt.expected {
				t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}
