package validation

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestWriteJSONErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONErrorWithCode(w, http.StatusUnauthorized, "unauthorized", "AUTH_ERROR")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	fields := []FieldError{
		{Field: "username", Message: "required"},
		{Field: "password", Message: "too short"},
	}
	WriteValidationError(w, fields)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestValidateLoginRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       LoginRequest
		wantCount int
	}{
		{
			name:      "valid request",
			req:       LoginRequest{Username: "admin", Password: "password123"},
			wantCount: 0,
		},
		{
			name:      "empty username",
			req:       LoginRequest{Username: "", Password: "password123"},
			wantCount: 1,
		},
		{
			name:      "empty password",
			req:       LoginRequest{Username: "admin", Password: ""},
			wantCount: 1,
		},
		{
			name:      "both empty",
			req:       LoginRequest{Username: "", Password: ""},
			wantCount: 2,
		},
		{
			name:      "username too long",
			req:       LoginRequest{Username: string(make([]byte, 65)), Password: "password"},
			wantCount: 1,
		},
		{
			name:      "password too long",
			req:       LoginRequest{Username: "admin", Password: string(make([]byte, 129))},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateLoginRequest(&tt.req)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateLoginRequest() returned %d errors, want %d", len(errors), tt.wantCount)
			}
		})
	}
}

func TestValidateThreshold(t *testing.T) {
	tests := []struct {
		name      string
		warning   time.Duration
		critical  time.Duration
		wantCount int
	}{
		{
			name:      "valid thresholds",
			warning:   100 * time.Millisecond,
			critical:  500 * time.Millisecond,
			wantCount: 0,
		},
		{
			name:      "warning equals critical",
			warning:   100 * time.Millisecond,
			critical:  100 * time.Millisecond,
			wantCount: 1,
		},
		{
			name:      "warning greater than critical",
			warning:   500 * time.Millisecond,
			critical:  100 * time.Millisecond,
			wantCount: 1,
		},
		{
			name:      "negative warning",
			warning:   -100 * time.Millisecond,
			critical:  500 * time.Millisecond,
			wantCount: 1,
		},
		{
			name:      "negative critical",
			warning:   100 * time.Millisecond,
			critical:  -500 * time.Millisecond,
			wantCount: 1,
		},
		{
			name:      "both zero is valid",
			warning:   0,
			critical:  0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateThreshold("test", tt.warning, tt.critical)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateThreshold() returned %d errors, want %d: %v", len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateHTTPEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  HTTPEndpointRequest
		wantCount int
	}{
		{
			name: "valid endpoint",
			endpoint: HTTPEndpointRequest{
				Name:           "Test",
				URL:            "https://example.com",
				ExpectedStatus: 200,
				Enabled:        true,
			},
			wantCount: 0,
		},
		{
			name: "empty name",
			endpoint: HTTPEndpointRequest{
				Name:           "",
				URL:            "https://example.com",
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
		{
			name: "empty URL",
			endpoint: HTTPEndpointRequest{
				Name:           "Test",
				URL:            "",
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
		{
			name: "invalid status code",
			endpoint: HTTPEndpointRequest{
				Name:           "Test",
				URL:            "https://example.com",
				ExpectedStatus: 999,
			},
			wantCount: 1,
		},
		{
			name: "private IP URL",
			endpoint: HTTPEndpointRequest{
				Name:           "Test",
				URL:            "http://192.168.1.1",
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateHTTPEndpoint(&tt.endpoint)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateHTTPEndpoint() returned %d errors, want %d: %v", len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidatePingTarget(t *testing.T) {
	tests := []struct {
		name      string
		target    PingTargetRequest
		wantCount int
	}{
		{
			name:      "valid hostname",
			target:    PingTargetRequest{Name: "Google DNS", Host: "8.8.8.8"},
			wantCount: 0,
		},
		{
			name:      "valid domain",
			target:    PingTargetRequest{Name: "Example", Host: "example.com"},
			wantCount: 0,
		},
		{
			name:      "empty name",
			target:    PingTargetRequest{Name: "", Host: "8.8.8.8"},
			wantCount: 1,
		},
		{
			name:      "empty host",
			target:    PingTargetRequest{Name: "Test", Host: ""},
			wantCount: 1,
		},
		{
			name:      "invalid host",
			target:    PingTargetRequest{Name: "Test", Host: "not a valid host!"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidatePingTarget(&tt.target)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidatePingTarget() returned %d errors, want %d: %v", len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateTCPPort(t *testing.T) {
	tests := []struct {
		name      string
		target    TCPPortRequest
		wantCount int
	}{
		{
			name:      "valid",
			target:    TCPPortRequest{Name: "HTTP", Host: "example.com", Port: 80},
			wantCount: 0,
		},
		{
			name:      "invalid port low",
			target:    TCPPortRequest{Name: "Test", Host: "example.com", Port: 0},
			wantCount: 1,
		},
		{
			name:      "invalid port high",
			target:    TCPPortRequest{Name: "Test", Host: "example.com", Port: 70000},
			wantCount: 1,
		},
		{
			name:      "invalid host",
			target:    TCPPortRequest{Name: "Test", Host: "invalid host!", Port: 80},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateTCPPort(&tt.target)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateTCPPort() returned %d errors, want %d: %v", len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateDNSServer(t *testing.T) {
	tests := []struct {
		name      string
		server    DNSServerRequest
		wantCount int
	}{
		{
			name:      "valid IPv4",
			server:    DNSServerRequest{Address: "8.8.8.8"},
			wantCount: 0,
		},
		{
			name:      "valid IPv6",
			server:    DNSServerRequest{Address: "2001:4860:4860::8888"},
			wantCount: 0,
		},
		{
			name:      "empty address",
			server:    DNSServerRequest{Address: ""},
			wantCount: 1,
		},
		{
			name:      "hostname not allowed",
			server:    DNSServerRequest{Address: "dns.google.com"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateDNSServer(&tt.server)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateDNSServer() returned %d errors, want %d: %v", len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateInterfaceSettings(t *testing.T) {
	tests := []struct {
		name      string
		iface     InterfaceRequest
		wantCount int
	}{
		{
			name:      "valid",
			iface:     InterfaceRequest{Default: "eth0", Fallbacks: []string{"enp0s3", "wlan0"}},
			wantCount: 0,
		},
		{
			name:      "empty default",
			iface:     InterfaceRequest{Default: ""},
			wantCount: 1,
		},
		{
			name:      "invalid default",
			iface:     InterfaceRequest{Default: "invalid interface name!"},
			wantCount: 1,
		},
		{
			name:      "invalid fallback",
			iface:     InterfaceRequest{Default: "eth0", Fallbacks: []string{"valid0", "inv@lid!"}},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateInterfaceSettings(&tt.iface)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateInterfaceSettings() returned %d errors, want %d: %v", len(errors), tt.wantCount, errors)
			}
		})
	}
}
