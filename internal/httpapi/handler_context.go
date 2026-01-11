package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Shared Handler Context (fixes #544 - consolidate duplicate error methods)
// ============================================================================

// HandlerContext provides common utilities for HTTP handlers.
// It bundles response writer, request, logger, and localizer for cleaner handler code.
type HandlerContext struct {
	W         http.ResponseWriter
	R         *http.Request
	Logger    *slog.Logger
	Localizer *i18n.Localizer
}

// NewHandlerContext creates a new handler context from an HTTP request.
// It automatically extracts the logger and localizer from the request context.
func NewHandlerContext(w http.ResponseWriter, r *http.Request) *HandlerContext {
	return &HandlerContext{
		W:         w,
		R:         r,
		Logger:    logging.FromContext(r.Context()),
		Localizer: i18n.FromRequest(r),
	}
}

// ============================================================================
// Error Response Methods
// ============================================================================

// SendValidationError sends a validation error response (400 Bad Request).
// Use for input validation failures.
func (c *HandlerContext) SendValidationError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusBadRequest,
		ErrCodeValidation,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendBadRequestError sends a bad request error response (400 Bad Request).
// Use for malformed requests or invalid parameters.
func (c *HandlerContext) SendBadRequestError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusBadRequest,
		ErrCodeBadRequest,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendInternalError sends an internal server error response (500 Internal Server Error).
// Use when an unexpected error occurs during processing.
func (c *HandlerContext) SendInternalError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusInternalServerError,
		ErrCodeInternal,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendNotFoundError sends a not found error response (404 Not Found).
// Use when a requested resource does not exist.
func (c *HandlerContext) SendNotFoundError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusNotFound,
		ErrCodeNotFound,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendUnauthorizedError sends an unauthorized error response (401 Unauthorized).
// Use when authentication is required but not provided or invalid.
func (c *HandlerContext) SendUnauthorizedError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusUnauthorized,
		ErrCodeUnauthorized,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendForbiddenError sends a forbidden error response (403 Forbidden).
// Use when the user is authenticated but lacks permission.
func (c *HandlerContext) SendForbiddenError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusForbidden,
		ErrCodeForbidden,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendConflictError sends a conflict error response (409 Conflict).
// Use when the request conflicts with the current state of the resource.
func (c *HandlerContext) SendConflictError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusConflict,
		ErrCodeConflict,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendMethodNotAllowedError sends a method not allowed error response (405 Method Not Allowed).
// Use when the HTTP method is not supported for the endpoint.
func (c *HandlerContext) SendMethodNotAllowedError() {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusMethodNotAllowed,
		ErrCodeMethodNotAllowed,
		c.Localizer.T("errors.api.methodNotAllowed"),
		"",
	)
}

// SendRateLimitError sends a rate limit exceeded error response (429 Too Many Requests).
// Returns true for convenience in early-return patterns.
func (c *HandlerContext) SendRateLimitError(msgKey string) bool {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusTooManyRequests,
		ErrCodeRateLimit,
		c.Localizer.T(msgKey),
		"",
	)
	return true
}

// SendServiceUnavailableError sends a service unavailable error response (503 Service Unavailable).
// Use when a required service or dependency is not available.
func (c *HandlerContext) SendServiceUnavailableError(msgKey string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		http.StatusServiceUnavailable,
		ErrCodeServiceUnavail,
		c.Localizer.T(msgKey),
		"",
	)
}

// SendErrorWithDetails sends a custom error response with all parameters.
// Use when you need full control over the error response.
func (c *HandlerContext) SendErrorWithDetails(status int, code, msgKey, details string) {
	sendErrorResponseWithDetails(
		c.W,
		c.Logger,
		status,
		code,
		c.Localizer.T(msgKey),
		details,
	)
}

// ============================================================================
// JSON Helpers
// ============================================================================

// DecodeJSON reads and decodes JSON from the request body.
// It applies a MaxBytesReader to prevent memory exhaustion attacks.
// Returns an error if the body exceeds maxSize or JSON is invalid.
func (c *HandlerContext) DecodeJSON(v any, maxSize int64) error {
	c.R.Body = http.MaxBytesReader(c.W, c.R.Body, maxSize)

	decoder := json.NewDecoder(c.R.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	// Ensure no additional data after the JSON object
	if decoder.More() {
		return io.ErrUnexpectedEOF
	}

	return nil
}

// SendJSON sends a JSON response with the given status code.
// Logs an error if encoding fails.
func (c *HandlerContext) SendJSON(status int, v any) {
	sendJSONResponse(c.W, c.Logger, status, v)
}

// SendOK sends a 200 OK JSON response.
func (c *HandlerContext) SendOK(v any) {
	sendJSONResponse(c.W, c.Logger, http.StatusOK, v)
}

// SendCreated sends a 201 Created JSON response.
func (c *HandlerContext) SendCreated(v any) {
	sendJSONResponse(c.W, c.Logger, http.StatusCreated, v)
}

// SendNoContent sends a 204 No Content response.
func (c *HandlerContext) SendNoContent() {
	c.W.WriteHeader(http.StatusNoContent)
}
