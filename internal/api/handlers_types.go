// Package api provides the HTTP/WebSocket server.
package api

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
)

// sendJSONResponse is a helper to send JSON responses and handle encoding errors.
// Used across all handler files (fixes #544 - shared utilities).
func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Error encoding JSON response", "error", err)
	}
}

// readLastLines reads the last N lines from a file, up to maxBytes from the end.
// Used by handleLogs and other log-reading handlers (fixes #544 - shared utilities).
func readLastLines(path string, maxBytes int64, maxLines int) ([]string, error) {
	//nolint:gosec // G304: path is from config for log file location
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	var start int64
	if info.Size() > maxBytes {
		start = info.Size() - maxBytes
	}
	if start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return nil, err
		}
	}

	scanner := bufio.NewScanner(f)
	// Allow long lines (up to 1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := make([]string, 0, maxLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
