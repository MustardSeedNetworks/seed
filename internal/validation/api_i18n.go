// Package validation provides input validation utilities with i18n support.
package validation

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// FieldErrorWithKey represents a validation error with a translation key.
type FieldErrorWithKey struct {
	Field      string         // Field name
	MessageKey string         // i18n translation key (e.g., "validation.login.usernameRequired")
	Data       map[string]any // Template data for interpolation
}

// TranslateFieldErrors converts field errors with keys to localized FieldError messages.
func TranslateFieldErrors(localizer *i18n.Localizer, errors []FieldErrorWithKey) []FieldError {
	result := make([]FieldError, 0, len(errors))
	for _, err := range errors {
		var message string
		if len(err.Data) > 0 {
			message = localizer.TWithData(err.MessageKey, err.Data)
		} else {
			message = localizer.T(err.MessageKey)
		}
		result = append(result, FieldError{
			Field:   err.Field,
			Message: message,
		})
	}
	return result
}

// WriteValidationErrorI18n writes a localized validation error response.
func WriteValidationErrorI18n(w http.ResponseWriter, localizer *i18n.Localizer, errors []FieldErrorWithKey) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	translatedErrors := TranslateFieldErrors(localizer, errors)
	resp := Error{
		Error:  localizer.T("errors.validation.failed"),
		Code:   "VALIDATION_ERROR",
		Fields: translatedErrors,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Warn("failed to encode validation error response", "error", err)
	}
}

// WriteJSONErrorI18n writes a localized JSON error response.
func WriteJSONErrorI18n(w http.ResponseWriter, localizer *i18n.Localizer, status int, messageKey string) {
	WriteJSONErrorI18nWithCode(w, localizer, status, messageKey, "")
}

// WriteJSONErrorI18nWithCode writes a localized JSON error response with an error code.
func WriteJSONErrorI18nWithCode(w http.ResponseWriter, localizer *i18n.Localizer, status int, messageKey, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := APIError{
		Error: localizer.T(messageKey),
		Code:  code,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Warn("failed to encode i18n error response", "error", err)
	}
}

// WriteJSONErrorI18nWithData writes a localized JSON error response with template data.
func WriteJSONErrorI18nWithData(
	w http.ResponseWriter,
	localizer *i18n.Localizer,
	status int,
	messageKey string,
	data map[string]any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := APIError{
		Error: localizer.TWithData(messageKey, data),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Warn("failed to encode i18n error response with data", "error", err)
	}
}

// ValidateLoginRequestI18n validates a login request and returns field-level errors with i18n keys.
//

func ValidateLoginRequestI18n(req *LoginRequest) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if req.Username == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "username",
			MessageKey: "validation.login.usernameRequired",
		})
	} else if len(req.Username) > 64 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "username",
			MessageKey: "validation.login.usernameTooLong",
		})
	}

	if req.Password == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "password",
			MessageKey: "validation.login.passwordRequired",
		})
	} else if len(req.Password) > 128 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "password",
			MessageKey: "validation.login.passwordTooLong",
		})
	}

	return errors
}

// ValidateThresholdI18n validates thresholds and returns field-level errors with i18n keys.
func ValidateThresholdI18n(name string, warning, critical int64) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if warning < 0 {
		errors = append(errors, FieldErrorWithKey{
			Field:      name + ".warning",
			MessageKey: "validation.threshold.warningNonNegative",
		})
	}

	if critical < 0 {
		errors = append(errors, FieldErrorWithKey{
			Field:      name + ".critical",
			MessageKey: "validation.threshold.criticalNonNegative",
		})
	}

	if warning > 0 && critical > 0 && warning >= critical {
		errors = append(errors, FieldErrorWithKey{
			Field:      name,
			MessageKey: "validation.threshold.warningLessThanCritical",
		})
	}

	return errors
}

// ValidateHTTPEndpointI18n validates an HTTP endpoint with i18n support.
func ValidateHTTPEndpointI18n(ep *HTTPEndpointRequest) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if ep.Name == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "name",
			MessageKey: "validation.endpoint.nameRequired",
		})
	} else if len(ep.Name) > 100 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "name",
			MessageKey: "validation.endpoint.nameTooLong",
		})
	}

	if ep.URL == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "url",
			MessageKey: "validation.endpoint.urlRequired",
		})
	} else if err := ValidateURL(ep.URL); err != nil {
		// URL validation errors are already localized in validation.go
		errors = append(errors, FieldErrorWithKey{
			Field:      "url",
			MessageKey: "validation.endpoint.invalidUrl",
		})
	}

	if ep.ExpectedStatus < 100 || ep.ExpectedStatus > 599 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "expected_status",
			MessageKey: "validation.endpoint.invalidStatus",
		})
	}

	return errors
}

// ValidatePingTargetI18n validates a ping target with i18n support.
//

func ValidatePingTargetI18n(pt *PingTargetRequest) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if pt.Name == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "name",
			MessageKey: "validation.endpoint.nameRequired",
		})
	} else if len(pt.Name) > 100 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "name",
			MessageKey: "validation.endpoint.nameTooLong",
		})
	}

	if pt.Host == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "host",
			MessageKey: "validation.host.required",
		})
	} else if !IsValidHostOrIP(pt.Host) {
		errors = append(errors, FieldErrorWithKey{
			Field:      "host",
			MessageKey: "validation.host.invalid",
		})
	}

	return errors
}

// ValidateTCPPortI18n validates a TCP port test with i18n support.
func ValidateTCPPortI18n(tp *TCPPortRequest) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if tp.Name == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "name",
			MessageKey: "validation.endpoint.nameRequired",
		})
	} else if len(tp.Name) > 100 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "name",
			MessageKey: "validation.endpoint.nameTooLong",
		})
	}

	if tp.Host == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "host",
			MessageKey: "validation.host.required",
		})
	} else if !IsValidHostOrIP(tp.Host) {
		errors = append(errors, FieldErrorWithKey{
			Field:      "host",
			MessageKey: "validation.host.invalid",
		})
	}

	if tp.Port < 1 || tp.Port > 65535 {
		errors = append(errors, FieldErrorWithKey{
			Field:      "port",
			MessageKey: "validation.port.invalidRange",
			Data:       map[string]any{"value": tp.Port},
		})
	}

	return errors
}

// ValidateDNSServerI18n validates a DNS server with i18n support.
func ValidateDNSServerI18n(ds *DNSServerRequest) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if ds.Address == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "address",
			MessageKey: "validation.address.required",
		})
	} else if !IsValidIP(ds.Address) {
		errors = append(errors, FieldErrorWithKey{
			Field:      "address",
			MessageKey: "validation.address.invalid",
		})
	}

	return errors
}

// ValidateInterfaceSettingsI18n validates interface settings with i18n support.
func ValidateInterfaceSettingsI18n(iface *InterfaceRequest) []FieldErrorWithKey {
	var errors []FieldErrorWithKey

	if iface.Default == "" {
		errors = append(errors, FieldErrorWithKey{
			Field:      "default",
			MessageKey: "validation.interface.defaultRequired",
		})
	} else if !IsValidInterface(iface.Default) {
		errors = append(errors, FieldErrorWithKey{
			Field:      "default",
			MessageKey: "validation.interface.invalid",
		})
	}

	for i, fb := range iface.Fallbacks {
		if fb != "" && !IsValidInterface(fb) {
			errors = append(errors, FieldErrorWithKey{
				Field:      "fallbacks[" + string(rune('0'+i)) + "]",
				MessageKey: "validation.interface.invalid",
			})
		}
	}

	return errors
}
