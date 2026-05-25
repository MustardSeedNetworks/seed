package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/krisarmstrong/seed/internal/logging"
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
