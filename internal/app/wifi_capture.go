package app

import (
	wificapture "github.com/krisarmstrong/seed/internal/wifi/capture"
	"github.com/krisarmstrong/seed/internal/wifi/visibility"
)

// NewWiFiCapture builds the monitor-mode capture producer feeding the visibility
// service from iface. The libpcap-backed opener is selected by build tag
// (wificapture.DefaultOpener); on CGO-free builds it is a no-op that disables
// capture gracefully. An empty iface also disables capture.
func NewWiFiCapture(sink *visibility.Service, iface string) *wificapture.Capture {
	return wificapture.New(wificapture.DefaultOpener(), sink, iface)
}
