package api

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// dtoValidator is the package-level validator instance used to enforce
// struct-tag rules on HTTP request DTOs. Registered with a json-tag-name
// function so error namespaces match what clients sent on the wire.
//
//nolint:gochecknoglobals // process-wide validator; matches existing pattern in handlers_mfa.go
var dtoValidator = newDTOValidator()

func newDTOValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonFieldName)
	return v
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// validateStruct runs the struct-tag validator against dto and, on failure,
// writes a localized 400 response and returns false. The caller should
// return immediately when this returns false.
//
// Pair this with decodeJSONStrictLocalized: decode first, then validate
// the populated struct. The two helpers together close the loop on
// boundary input — strict shape via the decoder, semantic constraints
// via the validator.
//
// The error response uses ErrCodeValidation and includes a `details` field
// listing the failing field paths so clients can surface per-field hints.
func validateStruct(
	w http.ResponseWriter,
	r *http.Request,
	dto any,
	localizer *i18n.Localizer,
) bool {
	if err := dtoValidator.Struct(dto); err != nil {
		logger := logging.FromContext(r.Context())
		var verrs validator.ValidationErrors
		details := ""
		if errors.As(err, &verrs) {
			details = formatValidationErrors(verrs)
		}
		logger.WarnContext(
			r.Context(), "Request validation failed",
			"path", r.URL.Path, "error", err,
		)
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.api.validationFailed"),
			details,
		)
		return false
	}
	return true
}

// formatValidationErrors collapses a ValidationErrors slice into a
// human-readable string suitable for the `details` field of the error
// envelope. Each line is one failing field rendered as
// `<json-path>: <rule>` (e.g., `username: required`).
func formatValidationErrors(verrs validator.ValidationErrors) string {
	parts := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		// Strip the leading struct-name prefix added by validator/v10 so
		// the path the client sees matches the JSON they sent. Example:
		// "LoginRequest.username" → "username".
		ns := fe.Namespace()
		if idx := strings.IndexByte(ns, '.'); idx >= 0 {
			ns = ns[idx+1:]
		}
		parts = append(parts, fmt.Sprintf("%s: %s", ns, fe.Tag()))
	}
	return strings.Join(parts, "; ")
}
