package validation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/validation"
)

func TestTranslateFieldErrors(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	tests := []struct {
		name   string
		errors []validation.FieldErrorWithKey
		count  int
	}{
		{
			name:   "empty errors",
			errors: []validation.FieldErrorWithKey{},
			count:  0,
		},
		{
			name: "single error without data",
			errors: []validation.FieldErrorWithKey{
				{Field: "username", MessageKey: "validation.login.usernameRequired"},
			},
			count: 1,
		},
		{
			name: "multiple errors",
			errors: []validation.FieldErrorWithKey{
				{Field: "username", MessageKey: "validation.login.usernameRequired"},
				{Field: "password", MessageKey: "validation.login.passwordRequired"},
			},
			count: 2,
		},
		{
			name: "error with template data",
			errors: []validation.FieldErrorWithKey{
				{
					Field:      "port",
					MessageKey: "validation.port.invalidRange",
					Data:       map[string]any{"value": 70000},
				},
			},
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validation.TranslateFieldErrors(localizer, tt.errors)
			if len(result) != tt.count {
				t.Errorf("TranslateFieldErrors() returned %d errors, want %d", len(result), tt.count)
			}
			// Verify that each result has the correct field name
			for i, err := range tt.errors {
				if i < len(result) && result[i].Field != err.Field {
					t.Errorf("TranslateFieldErrors() field = %q, want %q", result[i].Field, err.Field)
				}
			}
		})
	}
}

func TestWriteValidationErrorI18n(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	tests := []struct {
		name   string
		errors []validation.FieldErrorWithKey
	}{
		{
			name:   "empty errors",
			errors: []validation.FieldErrorWithKey{},
		},
		{
			name: "single error",
			errors: []validation.FieldErrorWithKey{
				{Field: "username", MessageKey: "validation.login.usernameRequired"},
			},
		},
		{
			name: "multiple errors",
			errors: []validation.FieldErrorWithKey{
				{Field: "username", MessageKey: "validation.login.usernameRequired"},
				{Field: "password", MessageKey: "validation.login.passwordRequired"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			validation.WriteValidationErrorI18n(w, localizer, tt.errors)

			if w.Code != http.StatusBadRequest {
				t.Errorf("WriteValidationErrorI18n() status = %d, want %d",
					w.Code, http.StatusBadRequest)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("WriteValidationErrorI18n() Content-Type = %q, want %q",
					ct, "application/json")
			}

			// Verify response body is valid JSON
			var resp validation.Error
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("WriteValidationErrorI18n() invalid JSON response: %v", err)
			}

			if resp.Code != "VALIDATION_ERROR" {
				t.Errorf("WriteValidationErrorI18n() code = %q, want %q",
					resp.Code, "VALIDATION_ERROR")
			}
		})
	}
}

func TestWriteJSONErrorI18n(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	tests := []struct {
		name       string
		status     int
		messageKey string
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			messageKey: "errors.validation.failed",
		},
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			messageKey: "errors.auth.invalidCredentials",
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			messageKey: "errors.notFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			validation.WriteJSONErrorI18n(w, localizer, tt.status, tt.messageKey)

			if w.Code != tt.status {
				t.Errorf("WriteJSONErrorI18n() status = %d, want %d", w.Code, tt.status)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("WriteJSONErrorI18n() Content-Type = %q, want %q",
					ct, "application/json")
			}

			// Verify response body is valid JSON
			var resp validation.APIError
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("WriteJSONErrorI18n() invalid JSON response: %v", err)
			}
		})
	}
}

func TestWriteJSONErrorI18nWithCode(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	tests := []struct {
		name       string
		status     int
		messageKey string
		code       string
	}{
		{
			name:       "auth error",
			status:     http.StatusUnauthorized,
			messageKey: "errors.auth.invalidCredentials",
			code:       "AUTH_ERROR",
		},
		{
			name:       "validation error",
			status:     http.StatusBadRequest,
			messageKey: "errors.validation.failed",
			code:       "VALIDATION_ERROR",
		},
		{
			name:       "empty code",
			status:     http.StatusInternalServerError,
			messageKey: "errors.internal",
			code:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			validation.WriteJSONErrorI18nWithCode(w, localizer, tt.status, tt.messageKey, tt.code)

			if w.Code != tt.status {
				t.Errorf("WriteJSONErrorI18nWithCode() status = %d, want %d", w.Code, tt.status)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("WriteJSONErrorI18nWithCode() Content-Type = %q, want %q",
					ct, "application/json")
			}

			// Verify response body is valid JSON
			var resp validation.APIError
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("WriteJSONErrorI18nWithCode() invalid JSON response: %v", err)
			}

			if resp.Code != tt.code {
				t.Errorf("WriteJSONErrorI18nWithCode() code = %q, want %q", resp.Code, tt.code)
			}
		})
	}
}

func TestWriteJSONErrorI18nWithData(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	tests := []struct {
		name       string
		status     int
		messageKey string
		data       map[string]any
	}{
		{
			name:       "with single data",
			status:     http.StatusBadRequest,
			messageKey: "validation.port.invalidRange",
			data:       map[string]any{"value": 70000},
		},
		{
			name:       "with multiple data",
			status:     http.StatusBadRequest,
			messageKey: "validation.stringLength",
			data:       map[string]any{"min": 1, "max": 100, "actual": 150},
		},
		{
			name:       "empty data",
			status:     http.StatusBadRequest,
			messageKey: "errors.general",
			data:       map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			validation.WriteJSONErrorI18nWithData(w, localizer, tt.status, tt.messageKey, tt.data)

			if w.Code != tt.status {
				t.Errorf("WriteJSONErrorI18nWithData() status = %d, want %d", w.Code, tt.status)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("WriteJSONErrorI18nWithData() Content-Type = %q, want %q",
					ct, "application/json")
			}

			// Verify response body is valid JSON
			var resp validation.APIError
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("WriteJSONErrorI18nWithData() invalid JSON response: %v", err)
			}
		})
	}
}

func TestValidateLoginRequestI18n(t *testing.T) {
	tests := []struct {
		name      string
		req       validation.LoginRequest
		wantCount int
	}{
		{
			name:      "valid request",
			req:       validation.LoginRequest{Username: "admin", Password: "password123"},
			wantCount: 0,
		},
		{
			name:      "empty username",
			req:       validation.LoginRequest{Username: "", Password: "password123"},
			wantCount: 1,
		},
		{
			name:      "empty password",
			req:       validation.LoginRequest{Username: "admin", Password: ""},
			wantCount: 1,
		},
		{
			name:      "both empty",
			req:       validation.LoginRequest{Username: "", Password: ""},
			wantCount: 2,
		},
		{
			name: "username too long",
			req: validation.LoginRequest{
				Username: strings.Repeat("a", 65),
				Password: "password",
			},
			wantCount: 1,
		},
		{
			name: "password too long",
			req: validation.LoginRequest{
				Username: "admin",
				Password: strings.Repeat("a", 129),
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidateLoginRequestI18n(&tt.req)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateLoginRequestI18n() returned %d errors, want %d",
					len(errors), tt.wantCount)
			}
		})
	}
}

func TestValidateThresholdI18n(t *testing.T) {
	tests := []struct {
		name      string
		thName    string
		warning   int64
		critical  int64
		wantCount int
	}{
		{
			name:      "valid thresholds",
			thName:    "latency",
			warning:   100,
			critical:  500,
			wantCount: 0,
		},
		{
			name:      "warning equals critical",
			thName:    "latency",
			warning:   100,
			critical:  100,
			wantCount: 1,
		},
		{
			name:      "warning greater than critical",
			thName:    "latency",
			warning:   500,
			critical:  100,
			wantCount: 1,
		},
		{
			name:      "negative warning",
			thName:    "latency",
			warning:   -100,
			critical:  500,
			wantCount: 1,
		},
		{
			name:      "negative critical",
			thName:    "latency",
			warning:   100,
			critical:  -500,
			wantCount: 1,
		},
		{
			name:      "both negative",
			thName:    "latency",
			warning:   -100,
			critical:  -500,
			wantCount: 2, // Two errors: warning and critical both negative
		},
		{
			name:      "both zero is valid",
			thName:    "latency",
			warning:   0,
			critical:  0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidateThresholdI18n(tt.thName, tt.warning, tt.critical)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateThresholdI18n() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateHTTPEndpointI18n(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  validation.HTTPEndpointRequest
		wantCount int
	}{
		{
			name: "valid endpoint",
			endpoint: validation.HTTPEndpointRequest{
				Name:           "Test",
				URL:            "https://example.com",
				ExpectedStatus: 200,
			},
			wantCount: 0,
		},
		{
			name: "empty name",
			endpoint: validation.HTTPEndpointRequest{
				Name:           "",
				URL:            "https://example.com",
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
		{
			name: "name too long",
			endpoint: validation.HTTPEndpointRequest{
				Name:           strings.Repeat("a", 101),
				URL:            "https://example.com",
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
		{
			name: "empty URL",
			endpoint: validation.HTTPEndpointRequest{
				Name:           "Test",
				URL:            "",
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
		{
			name: "invalid URL",
			endpoint: validation.HTTPEndpointRequest{
				Name:           "Test",
				URL:            "http://192.168.1.1", // private IP
				ExpectedStatus: 200,
			},
			wantCount: 1,
		},
		{
			name: "invalid status low",
			endpoint: validation.HTTPEndpointRequest{
				Name:           "Test",
				URL:            "https://example.com",
				ExpectedStatus: 99,
			},
			wantCount: 1,
		},
		{
			name: "invalid status high",
			endpoint: validation.HTTPEndpointRequest{
				Name:           "Test",
				URL:            "https://example.com",
				ExpectedStatus: 600,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidateHTTPEndpointI18n(&tt.endpoint)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateHTTPEndpointI18n() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidatePingTargetI18n(t *testing.T) {
	tests := []struct {
		name      string
		target    validation.PingTargetRequest
		wantCount int
	}{
		{
			name:      "valid with IP",
			target:    validation.PingTargetRequest{Name: "Google DNS", Host: "8.8.8.8"},
			wantCount: 0,
		},
		{
			name:      "valid with hostname",
			target:    validation.PingTargetRequest{Name: "Example", Host: "example.com"},
			wantCount: 0,
		},
		{
			name:      "empty name",
			target:    validation.PingTargetRequest{Name: "", Host: "8.8.8.8"},
			wantCount: 1,
		},
		{
			name:      "name too long",
			target:    validation.PingTargetRequest{Name: strings.Repeat("a", 101), Host: "8.8.8.8"},
			wantCount: 1,
		},
		{
			name:      "empty host",
			target:    validation.PingTargetRequest{Name: "Test", Host: ""},
			wantCount: 1,
		},
		{
			name:      "invalid host",
			target:    validation.PingTargetRequest{Name: "Test", Host: "!!!invalid!!!"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidatePingTargetI18n(&tt.target)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidatePingTargetI18n() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateTCPPortI18n(t *testing.T) {
	tests := []struct {
		name      string
		target    validation.TCPPortRequest
		wantCount int
	}{
		{
			name:      "valid",
			target:    validation.TCPPortRequest{Name: "HTTP", Host: "example.com", Port: 80},
			wantCount: 0,
		},
		{
			name:      "empty name",
			target:    validation.TCPPortRequest{Name: "", Host: "example.com", Port: 80},
			wantCount: 1,
		},
		{
			name:      "name too long",
			target:    validation.TCPPortRequest{Name: strings.Repeat("a", 101), Host: "example.com", Port: 80},
			wantCount: 1,
		},
		{
			name:      "empty host",
			target:    validation.TCPPortRequest{Name: "Test", Host: "", Port: 80},
			wantCount: 1,
		},
		{
			name:      "invalid host",
			target:    validation.TCPPortRequest{Name: "Test", Host: "!!!invalid!!!", Port: 80},
			wantCount: 1,
		},
		{
			name:      "port too low",
			target:    validation.TCPPortRequest{Name: "Test", Host: "example.com", Port: 0},
			wantCount: 1,
		},
		{
			name:      "port too high",
			target:    validation.TCPPortRequest{Name: "Test", Host: "example.com", Port: 65536},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidateTCPPortI18n(&tt.target)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateTCPPortI18n() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateDNSServerI18n(t *testing.T) {
	tests := []struct {
		name      string
		server    validation.DNSServerRequest
		wantCount int
	}{
		{
			name:      "valid IPv4",
			server:    validation.DNSServerRequest{Address: "8.8.8.8"},
			wantCount: 0,
		},
		{
			name:      "valid IPv6",
			server:    validation.DNSServerRequest{Address: "2001:4860:4860::8888"},
			wantCount: 0,
		},
		{
			name:      "empty address",
			server:    validation.DNSServerRequest{Address: ""},
			wantCount: 1,
		},
		{
			name:      "hostname not allowed",
			server:    validation.DNSServerRequest{Address: "dns.google.com"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidateDNSServerI18n(&tt.server)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateDNSServerI18n() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}

func TestValidateInterfaceSettingsI18n(t *testing.T) {
	tests := []struct {
		name      string
		iface     validation.InterfaceRequest
		wantCount int
	}{
		{
			name: "valid with fallbacks",
			iface: validation.InterfaceRequest{
				Default:   "eth0",
				Fallbacks: []string{"enp0s3", "wlan0"},
			},
			wantCount: 0,
		},
		{
			name: "valid without fallbacks",
			iface: validation.InterfaceRequest{
				Default:   "en0",
				Fallbacks: []string{},
			},
			wantCount: 0,
		},
		{
			name: "empty default",
			iface: validation.InterfaceRequest{
				Default:   "",
				Fallbacks: []string{},
			},
			wantCount: 1,
		},
		{
			name: "invalid default",
			iface: validation.InterfaceRequest{
				Default:   "0invalid",
				Fallbacks: []string{},
			},
			wantCount: 1,
		},
		{
			name: "invalid fallback",
			iface: validation.InterfaceRequest{
				Default:   "eth0",
				Fallbacks: []string{"valid0", "0invalid"},
			},
			wantCount: 1,
		},
		{
			name: "empty fallback is skipped",
			iface: validation.InterfaceRequest{
				Default:   "eth0",
				Fallbacks: []string{"", "wlan0"},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validation.ValidateInterfaceSettingsI18n(&tt.iface)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateInterfaceSettingsI18n() returned %d errors, want %d: %v",
					len(errors), tt.wantCount, errors)
			}
		})
	}
}
