//go:build !dev
// +build !dev

// Package web provides the embedded frontend assets for production builds.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFS returns the embedded frontend filesystem.
// Returns the dist subdirectory as the root.
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// IsEmbedded returns true when assets are embedded (production build).
func IsEmbedded() bool {
	return true
}
