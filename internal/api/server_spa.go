package api

// server_spa.go contains the static-file fallback for the embedded React SPA:
// path normalization, API-route detection, file open with fallback to
// index.html for client-side routes, and directory-request handling.

import (
	"io"
	"net/http"
	"os"
	"strings"
)

// spaFileResult holds the result of opening a file for SPA serving.
type spaFileResult struct {
	file http.File
	stat os.FileInfo
}

// normalizeSPAPath normalizes the request path for SPA handling.
func normalizeSPAPath(path string) string {
	if path == "/" || path == "" {
		return indexHTMLPath
	}
	return path
}

// isAPIRoute checks if the path is an API or SSE route.
func isAPIRoute(path string) bool {
	return strings.HasPrefix(path, APIVersionPrefix) ||
		strings.HasPrefix(path, APIBasePath+"/events") // SSE endpoint
}

// openSPAFile attempts to open a file from the filesystem, falling back to index.html for SPA routes.
func openSPAFile(fsys http.FileSystem, path string) (http.File, error) {
	f, err := fsys.Open(path)
	if err == nil {
		return f, nil
	}

	// File doesn't exist - check if it's an API route (shouldn't happen, but be safe)
	if isAPIRoute(path) {
		return nil, err
	}

	// Serve index.html for SPA routing (client-side routes)
	return fsys.Open(indexHTMLPath)
}

// handleDirectoryRequest handles requests for directories by serving their index.html.
func handleDirectoryRequest(fsys http.FileSystem, f http.File, path string) (*spaFileResult, error) {
	// Try to serve index.html from the directory
	indexPath := strings.TrimSuffix(path, "/") + indexHTMLPath
	f2, indexErr := fsys.Open(indexPath)
	if indexErr != nil {
		// No index.html in directory - serve root index.html
		f2, indexErr = fsys.Open(indexHTMLPath)
		if indexErr != nil {
			return nil, indexErr
		}
	}
	_ = f.Close()

	stat, err := f2.Stat()
	if err != nil {
		_ = f2.Close()
		return nil, err
	}
	return &spaFileResult{file: f2, stat: stat}, nil
}

// spaHandler wraps a file server to support SPA (Single Page Application) routing.
// It serves index.html for any path that doesn't match a static file, enabling
// client-side routing in React/Vue/Angular apps.
func spaHandler(fsys http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := normalizeSPAPath(r.URL.Path)

		f, err := openSPAFile(fsys, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }()

		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Handle directory requests
		if stat.IsDir() {
			result, dirErr := handleDirectoryRequest(fsys, f, path)
			if dirErr != nil {
				http.NotFound(w, r)
				return
			}
			defer func() { _ = result.file.Close() }()
			f = result.file
			stat = result.stat
		}

		rs, ok := f.(io.ReadSeeker)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), rs)
	})
}
