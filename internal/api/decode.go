package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// decodeJSONStrict reads and validates JSON from r.Body into dst. It applies
// [http.MaxBytesReader] to cap memory, DisallowUnknownFields to reject typoed
// keys, and rejects trailing data after the JSON object.
//
// On any decode failure it writes a structured error response (413 for
// oversized bodies, 400 for everything else) and returns false; the caller
// should return immediately. On success it returns true and dst is
// populated.
//
// Use this for handler call sites that do not already hold a [HandlerContext];
// for HandlerContext-based handlers, prefer [HandlerContext.DecodeJSONOrFail]
// which is a thin wrapper.
//
// Pick maxSize from the constants in limits.go ([MaxBodySizeAuth],
// [MaxBodySizeConfig], [MaxBodySizeJSON], …) — never pass a magic number.
func decodeJSONStrict(w http.ResponseWriter, r *http.Request, dst any, maxSize int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		logger := logging.FromContext(r.Context())
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			sendErrorResponseWithDetails(
				w, logger,
				http.StatusRequestEntityTooLarge,
				ErrCodeValidation,
				"Request body too large",
				"",
			)
			return false
		}
		// Don't leak json parser internals to clients.
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			"Invalid JSON in request body",
			"",
		)
		return false
	}

	// Reject "smuggled" data after the top-level JSON object — multiple
	// concatenated payloads, trailing garbage, etc.
	if decoder.More() {
		sendErrorResponseWithDetails(
			w, logging.FromContext(r.Context()),
			http.StatusBadRequest,
			ErrCodeValidation,
			"Unexpected trailing data after JSON object",
			"",
		)
		return false
	}

	return true
}

// DecodeJSONOrFail is the HandlerContext-flavoured wrapper around
// decodeJSONStrict. Same contract: returns false (and has already written
// a 400/413) if the body was unreadable or malformed; the caller should
// return immediately.
//
// Prefer this over the existing (*HandlerContext).DecodeJSON when the
// handler has no use for a typed error and just wants to bail.
func (c *HandlerContext) DecodeJSONOrFail(dst any, maxSize int64) bool {
	return decodeJSONStrict(c.W, c.R, dst, maxSize)
}

// invalidRequestBodyKey is the i18n message key used by
// decodeJSONStrictLocalized for the user-facing "Invalid request body"
// message. Every converted call site uses the same key today; the helper
// exposes it as a constant so future callers can override the formatting
// path (e.g., by calling localizer.T(...) themselves around a typed error)
// without breaking the standard contract.
const invalidRequestBodyKey = "errors.api.invalidRequestBody"

// decodeJSONStrictLocalized is the i18n-preserving variant of
// decodeJSONStrict. Same strictness (MaxBytesReader, DisallowUnknownFields,
// trailing-data guard) but on failure writes a localized 4xx response
// using localizer.T(invalidRequestBodyKey) and the ErrCodeBadRequest code,
// and logs a WARN with "Invalid request body" — matching the contract of
// the scattered bare-decode call sites this helper is meant to replace.
//
// Use this for handlers that already construct localized error responses
// and want to keep the BAD_REQUEST code instead of switching to
// VALIDATION_ERROR. Most existing seed handlers fit that profile.
func decodeJSONStrictLocalized(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	maxSize int64,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) bool {
	return decodeJSONStrictLocalizedWith(w, r, dst, maxSize, logger, localizer)
}

// decodeJSONStrictLocalizedWith is the variant of decodeJSONStrictLocalized
// that carries extra structured log fields (e.g., client_ip on auth flows,
// username on MFA verifications) into the WARN line on decode failure.
//
// Same contract as decodeJSONStrictLocalized otherwise. Use this for auth,
// MFA, and recovery endpoints where preserving the audit-log breadcrumb
// matters — the security team relies on those fields to spot brute-force
// patterns.
func decodeJSONStrictLocalizedWith(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	maxSize int64,
	logger *slog.Logger,
	localizer *i18n.Localizer,
	extraAttrs ...any,
) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		status := http.StatusBadRequest
		if errors.As(err, &maxBytesErr) {
			status = http.StatusRequestEntityTooLarge
		}
		attrs := append([]any{"error", err}, extraAttrs...)
		logger.WarnContext(r.Context(), "Invalid request body", attrs...)
		sendErrorResponseWithDetails(
			w, logger, status, ErrCodeBadRequest, localizer.T(invalidRequestBodyKey), "",
		)
		return false
	}

	if decoder.More() {
		logger.WarnContext(r.Context(), "Unexpected trailing data after JSON object", extraAttrs...)
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T(invalidRequestBodyKey), "",
		)
		return false
	}

	return true
}
