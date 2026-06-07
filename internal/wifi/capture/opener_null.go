//go:build !cgo && !windows

package wificapture

import (
	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/capture/nullcapture"
)

// DefaultOpener returns the CGO-free no-op capture adapter, compiled only for
// CGO_ENABLED=0 non-Windows builds where libpcap cannot be linked. OpenLive then
// fails with nullcapture.ErrUnavailable and Capture.Start degrades gracefully.
// Mirrors internal/api/wire_capture_null.go.
func DefaultOpener() capture.Opener { return nullcapture.New() }
