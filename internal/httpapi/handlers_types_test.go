package httpapi_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestErrorResponseStruct tests the ErrorResponse struct serialization.
func TestErrorResponseStruct(t *testing.T) {
	resp := api.ErrorResponse{
		Error:   "something went wrong",
		Code:    api.ErrCodeBadRequest,
		Details: "additional information",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorResponse: %v", err)
	}

	var decoded api.ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ErrorResponse: %v", err)
	}

	if decoded.Error != resp.Error {
		t.Errorf("Expected Error %q, got %q", resp.Error, decoded.Error)
	}
	if decoded.Code != resp.Code {
		t.Errorf("Expected Code %q, got %q", resp.Code, decoded.Code)
	}
	if decoded.Details != resp.Details {
		t.Errorf("Expected Details %q, got %q", resp.Details, decoded.Details)
	}
}

// TestErrorResponseWithoutDetails tests ErrorResponse serialization without details.
func TestErrorResponseWithoutDetails(t *testing.T) {
	resp := api.ErrorResponse{
		Error: "something went wrong",
		Code:  api.ErrCodeNotFound,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorResponse: %v", err)
	}

	// Verify "details" is omitted when empty
	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, exists := decoded["details"]; exists {
		t.Error("Expected 'details' to be omitted when empty")
	}
}

// TestErrorCodes tests that all error codes are defined correctly.
func TestErrorCodes(t *testing.T) {
	codes := map[string]string{
		"BAD_REQUEST":         api.ErrCodeBadRequest,
		"UNAUTHORIZED":        api.ErrCodeUnauthorized,
		"FORBIDDEN":           api.ErrCodeForbidden,
		"NOT_FOUND":           api.ErrCodeNotFound,
		"CONFLICT":            api.ErrCodeConflict,
		"METHOD_NOT_ALLOWED":  api.ErrCodeMethodNotAllowed,
		"INTERNAL_ERROR":      api.ErrCodeInternal,
		"SERVICE_UNAVAILABLE": api.ErrCodeServiceUnavail,
		"VALIDATION_ERROR":    api.ErrCodeValidation,
		"RATE_LIMIT_EXCEEDED": api.ErrCodeRateLimit,
	}

	for expected, actual := range codes {
		if actual != expected {
			t.Errorf("Expected error code %q, got %q", expected, actual)
		}
	}
}

// TestSendJSONResponse tests the sendJSONResponse helper.
func TestSendJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		data     any
		wantCode int
	}{
		{
			name:     "success response",
			status:   http.StatusOK,
			data:     map[string]string{"status": "ok"},
			wantCode: http.StatusOK,
		},
		{
			name:     "created response",
			status:   http.StatusCreated,
			data:     map[string]int{"id": 123},
			wantCode: http.StatusCreated,
		},
		{
			name:     "array response",
			status:   http.StatusOK,
			data:     []string{"item1", "item2"},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

			api.ExportSendJSONResponse(w, logger, tt.status, tt.data)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
			}
		})
	}
}

// TestSendErrorResponseWithDetails tests the standardized error response helper.
func TestSendErrorResponseWithDetails(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    string
		message string
		details string
	}{
		{
			name:    "bad request error",
			status:  http.StatusBadRequest,
			code:    api.ErrCodeBadRequest,
			message: "invalid input",
			details: "field X is required",
		},
		{
			name:    "not found error",
			status:  http.StatusNotFound,
			code:    api.ErrCodeNotFound,
			message: "resource not found",
			details: "",
		},
		{
			name:    "internal error",
			status:  http.StatusInternalServerError,
			code:    api.ErrCodeInternal,
			message: "internal server error",
			details: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

			api.ExportSendErrorResponseWithDetails(w, logger, tt.status, tt.code, tt.message, tt.details)

			if w.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}

			var resp api.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Error != tt.message {
				t.Errorf("Expected Error %q, got %q", tt.message, resp.Error)
			}
			if resp.Code != tt.code {
				t.Errorf("Expected Code %q, got %q", tt.code, resp.Code)
			}
			if resp.Details != tt.details {
				t.Errorf("Expected Details %q, got %q", tt.details, resp.Details)
			}
		})
	}
}

// TestReadLastLines tests reading the last N lines from a file.
func TestReadLastLines(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Write test content
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name        string
		maxBytes    int64
		maxLines    int
		expectedLen int
		firstLine   string
	}{
		{
			name:        "read all lines",
			maxBytes:    1024,
			maxLines:    20,
			expectedLen: 10,
			firstLine:   "line1",
		},
		{
			name:        "limit to 5 lines",
			maxBytes:    1024,
			maxLines:    5,
			expectedLen: 5,
			firstLine:   "line6",
		},
		{
			name:        "limit to 3 lines",
			maxBytes:    1024,
			maxLines:    3,
			expectedLen: 3,
			firstLine:   "line8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, err := api.ExportReadLastLines(testFile, tt.maxBytes, tt.maxLines)
			if err != nil {
				t.Fatalf("ReadLastLines failed: %v", err)
			}

			if len(lines) != tt.expectedLen {
				t.Errorf("Expected %d lines, got %d", tt.expectedLen, len(lines))
			}

			if len(lines) > 0 && lines[0] != tt.firstLine {
				t.Errorf("Expected first line %q, got %q", tt.firstLine, lines[0])
			}
		})
	}
}

// TestReadLastLinesEmptyFile tests reading from an empty file.
func TestReadLastLinesEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.log")

	if err := os.WriteFile(testFile, []byte(""), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	lines, err := api.ExportReadLastLines(testFile, 1024, 10)
	if err != nil {
		t.Fatalf("ReadLastLines failed: %v", err)
	}

	if len(lines) != 0 {
		t.Errorf("Expected 0 lines from empty file, got %d", len(lines))
	}
}

// TestReadLastLinesFileNotFound tests reading from a non-existent file.
func TestReadLastLinesFileNotFound(t *testing.T) {
	_, err := api.ExportReadLastLines("/nonexistent/path/file.log", 1024, 10)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestReadLastLinesLongLines tests reading files with very long lines.
func TestReadLastLinesLongLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "long.log")

	// Create a file with some long lines
	longLine := make([]byte, 2000)
	for i := range longLine {
		longLine[i] = 'x'
	}
	content := string(longLine) + "\nshort line\n"

	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	lines, err := api.ExportReadLastLines(testFile, 3000, 10)
	if err != nil {
		t.Fatalf("ReadLastLines failed: %v", err)
	}

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

// TestReadLastLinesByteLimit tests reading with byte limit.
func TestReadLastLinesByteLimit(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "byteslimit.log")

	// Write 1000 bytes of content
	content := ""
	for i := 0; i < 100; i++ {
		content += "line" + string(rune('0'+(i%10))) + "\n"
	}

	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Read with byte limit that only covers part of the file
	lines, err := api.ExportReadLastLines(testFile, 50, 100)
	if err != nil {
		t.Fatalf("ReadLastLines failed: %v", err)
	}

	// Should only get lines from the last 50 bytes
	if len(lines) > 10 {
		t.Errorf("Expected fewer lines due to byte limit, got %d", len(lines))
	}
}

// TestBufferSizeConstants tests that buffer size constants are defined.
func TestBufferSizeConstants(t *testing.T) {
	// These should be accessible and have reasonable values
	if api.MaxBodySizeAuth <= 0 {
		t.Error("MaxBodySizeAuth should be positive")
	}
	if api.MaxBodySizeConfig <= 0 {
		t.Error("MaxBodySizeConfig should be positive")
	}
	if api.MaxBodySizeJSON <= 0 {
		t.Error("MaxBodySizeJSON should be positive")
	}
	if api.MaxBodySizeFloorPlan <= 0 {
		t.Error("MaxBodySizeFloorPlan should be positive")
	}
	if api.MaxBodySizeAirMapper <= 0 {
		t.Error("MaxBodySizeAirMapper should be positive")
	}
	if api.MaxBodySizeDefault <= 0 {
		t.Error("MaxBodySizeDefault should be positive")
	}

	// Auth limit should be smaller than config limit
	if api.MaxBodySizeAuth >= api.MaxBodySizeConfig {
		t.Error("MaxBodySizeAuth should be smaller than MaxBodySizeConfig")
	}

	// Default limit should be reasonable
	if api.MaxBodySizeDefault > api.MaxBodySizeAirMapper {
		t.Error("MaxBodySizeDefault should not exceed MaxBodySizeAirMapper")
	}
}
