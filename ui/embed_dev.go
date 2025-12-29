//go:build dev
// +build dev

// Package ui provides the frontend assets for development builds.
// In dev mode, assets are read from disk to support hot-reload.
package ui

import (
	"io/fs"
	"os"
)

// GetFS returns the filesystem for frontend assets.
// In dev mode, this reads from disk for hot-reload support.
func GetFS() (fs.FS, error) {
	return os.DirFS("ui/dist"), nil
}

// IsEmbedded returns false in dev mode.
func IsEmbedded() bool {
	return false
}
