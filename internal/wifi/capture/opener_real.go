//go:build cgo || windows

package wificapture

import (
	"github.com/krisarmstrong/seed/internal/capture"
	"github.com/krisarmstrong/seed/internal/capture/pcap"
)

// DefaultOpener returns the libpcap-backed capture adapter. Compiled when CGO is
// enabled (libpcap linkable on linux/darwin) or on Windows (gopacket/pcap is
// CGO-free via wpcap). Mirrors internal/api/wire_capture_real.go so monitor-mode
// capture works on every platform that supports live capture.
func DefaultOpener() capture.Opener { return pcap.New() }
