//go:build !cgo && !windows

package api

import (
	"github.com/krisarmstrong/seed/internal/capture"
	"github.com/krisarmstrong/seed/internal/capture/nullcapture"
)

// defaultCaptureOpener returns the CGO-free no-op capture adapter. This file is
// compiled only for CGO_ENABLED=0 builds on non-Windows platforms, where libpcap
// cannot be linked — live capture is unavailable and capture calls fail with
// nullcapture.ErrUnavailable. The libpcap-backed variant lives in
// wire_capture_real.go. See docs/architecture/CGO_BUILD_STRATEGY.md.
func defaultCaptureOpener() capture.Opener { return nullcapture.New() }
