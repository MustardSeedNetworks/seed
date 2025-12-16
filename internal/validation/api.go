// Package validation provides input validation utilities and standardized API error responses.
// Contains helpers to write JSON errors with consistent structure and HTTP status mapping.
package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIError represents a standardized JSON error response.
type APIError struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error represents a collection of field validation errors.
type Error struct {
	Error  string       `json:"error"`
	Code   string       `json:"code"`
	Fields []FieldError `json:"fields"`
}

// WriteJSONError writes a standard JSON error response.
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	WriteJSONErrorWithCode(w, status, message, "")
}

// WriteJSONErrorWithCode writes a JSON error response with an error code.
func WriteJSONErrorWithCode(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := APIError{
		Error: message,
		Code:  code,
	}
	//nolint:errcheck // Response body encode errors are not actionable in HTTP handlers
	json.NewEncoder(w).Encode(resp)
}

// WriteValidationError writes a JSON validation error response with field-level errors.
func WriteValidationError(w http.ResponseWriter, fields []FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	resp := Error{
		Error:  "Validation failed",
		Code:   "VALIDATION_ERROR",
		Fields: fields,
	}
	//nolint:errcheck // Response body encode errors are not actionable in HTTP handlers
	json.NewEncoder(w).Encode(resp)
}

// LoginRequest represents the expected login payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ValidateLoginRequest validates a login request and returns field-level errors.
func ValidateLoginRequest(req *LoginRequest) []FieldError {
	var errors []FieldError

	if req.Username == "" {
		errors = append(errors, FieldError{Field: "username", Message: "username is required"})
	} else if len(req.Username) > 64 {
		errors = append(errors, FieldError{Field: "username", Message: "username too long (max 64 characters)"})
	}

	if req.Password == "" {
		errors = append(errors, FieldError{Field: "password", Message: "password is required"})
	} else if len(req.Password) > 128 {
		errors = append(errors, FieldError{Field: "password", Message: "password too long (max 128 characters)"})
	}

	return errors
}

// ThresholdSettings represents threshold configuration for validation.
type ThresholdSettings struct {
	Warning  time.Duration
	Critical time.Duration
}

// ValidateThreshold validates that warning and critical thresholds are sensible.
func ValidateThreshold(name string, warning, critical time.Duration) []FieldError {
	var errors []FieldError

	if warning < 0 {
		errors = append(errors, FieldError{
			Field:   name + ".warning",
			Message: "warning threshold must be non-negative",
		})
	}

	if critical < 0 {
		errors = append(errors, FieldError{
			Field:   name + ".critical",
			Message: "critical threshold must be non-negative",
		})
	}

	if warning > 0 && critical > 0 && warning >= critical {
		errors = append(errors, FieldError{
			Field:   name,
			Message: "warning threshold must be less than critical threshold",
		})
	}

	return errors
}

// HTTPEndpointRequest represents an HTTP endpoint configuration for validation.
type HTTPEndpointRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	Enabled        bool   `json:"enabled"`
}

// ValidateHTTPEndpoint validates an HTTP endpoint configuration.
func ValidateHTTPEndpoint(ep *HTTPEndpointRequest) []FieldError {
	var errors []FieldError

	if ep.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	} else if len(ep.Name) > 100 {
		errors = append(errors, FieldError{Field: "name", Message: "name too long (max 100 characters)"})
	}

	if ep.URL == "" {
		errors = append(errors, FieldError{Field: "url", Message: "URL is required"})
	} else if err := ValidateURL(ep.URL); err != nil {
		errors = append(errors, FieldError{Field: "url", Message: err.Error()})
	}

	if ep.ExpectedStatus < 100 || ep.ExpectedStatus > 599 {
		errors = append(errors, FieldError{
			Field:   "expected_status",
			Message: "expected_status must be a valid HTTP status code (100-599)",
		})
	}

	return errors
}

// PingTargetRequest represents a ping target configuration for validation.
type PingTargetRequest struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

// ValidatePingTarget validates a ping target configuration.
func ValidatePingTarget(pt *PingTargetRequest) []FieldError {
	var errors []FieldError

	if pt.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	} else if len(pt.Name) > 100 {
		errors = append(errors, FieldError{Field: "name", Message: "name too long (max 100 characters)"})
	}

	if pt.Host == "" {
		errors = append(errors, FieldError{Field: "host", Message: "host is required"})
	} else if !IsValidHostOrIP(pt.Host) {
		errors = append(errors, FieldError{Field: "host", Message: "invalid hostname or IP address"})
	}

	return errors
}

// TCPPortRequest represents a TCP port test configuration for validation.
type TCPPortRequest struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// ValidateTCPPort validates a TCP port test configuration.
func ValidateTCPPort(tp *TCPPortRequest) []FieldError {
	var errors []FieldError

	if tp.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	} else if len(tp.Name) > 100 {
		errors = append(errors, FieldError{Field: "name", Message: "name too long (max 100 characters)"})
	}

	if tp.Host == "" {
		errors = append(errors, FieldError{Field: "host", Message: "host is required"})
	} else if !IsValidHostOrIP(tp.Host) {
		errors = append(errors, FieldError{Field: "host", Message: "invalid hostname or IP address"})
	}

	if err := ValidatePort(tp.Port); err != nil {
		errors = append(errors, FieldError{Field: "port", Message: err.Error()})
	}

	return errors
}

// DNSServerRequest represents a DNS server configuration for validation.
type DNSServerRequest struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

// ValidateDNSServer validates a DNS server configuration.
func ValidateDNSServer(ds *DNSServerRequest) []FieldError {
	var errors []FieldError

	if ds.Address == "" {
		errors = append(errors, FieldError{Field: "address", Message: "address is required"})
	} else if !IsValidIP(ds.Address) {
		errors = append(errors, FieldError{Field: "address", Message: "invalid IP address"})
	}

	return errors
}

// InterfaceRequest represents network interface settings for validation.
type InterfaceRequest struct {
	Default   string   `json:"default"`
	Fallbacks []string `json:"fallbacks"`
}

// ValidateInterfaceSettings validates network interface configuration.
func ValidateInterfaceSettings(iface *InterfaceRequest) []FieldError {
	var errors []FieldError

	if iface.Default == "" {
		errors = append(errors, FieldError{Field: "default", Message: "default interface is required"})
	} else if !IsValidInterface(iface.Default) {
		errors = append(errors, FieldError{Field: "default", Message: "invalid interface name"})
	}

	for i, fb := range iface.Fallbacks {
		if fb != "" && !IsValidInterface(fb) {
			errors = append(errors, FieldError{
				Field:   fmt.Sprintf("fallbacks[%d]", i),
				Message: "invalid interface name",
			})
		}
	}

	return errors
}
