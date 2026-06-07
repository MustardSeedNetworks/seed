package app

import (
	wificapture "github.com/MustardSeedNetworks/seed/internal/wifi/capture"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// NewWiFiCapture builds the monitor-mode capture producer feeding the visibility
// service from iface. The libpcap-backed opener is selected by build tag
// (wificapture.DefaultOpener); on CGO-free builds it is a no-op that disables
// capture gracefully. An empty iface also disables capture.
//
// When autoEnable is set, the OS monitor-mode enabler (Linux iw/nl80211) switches
// the interface into monitor mode on start and restores it on stop; otherwise
// capture is bring-your-own monitor.
func NewWiFiCapture(sink *visibility.Service, iface string, autoEnable bool) *wificapture.Capture {
	var opts []wificapture.Option
	if autoEnable {
		opts = append(opts, wificapture.WithEnabler(wificapture.DefaultEnabler()))
	}
	return wificapture.New(wificapture.DefaultOpener(), sink, iface, opts...)
}
