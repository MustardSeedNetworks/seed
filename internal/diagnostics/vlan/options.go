package vlan

import (
	"github.com/krisarmstrong/seed/internal/capture"
	"github.com/krisarmstrong/seed/internal/capture/nullcapture"
)

// Option configures optional dependencies of NewTrafficMonitor.
type Option func(*captureConfig)

// captureConfig collects optional construction settings.
type captureConfig struct {
	opener capture.Opener
}

// WithCapture injects the live packet-capture Opener used to sniff 802.1Q
// VLAN-tagged traffic. The composition root passes the libpcap-backed adapter
// (internal/capture/pcap) under CGO; absent an override the default is the
// CGO-free no-op (nullcapture), so the vlan package and its tests never link
// libpcap. A nil opener is ignored.
func WithCapture(opener capture.Opener) Option {
	return func(c *captureConfig) {
		if opener != nil {
			c.opener = opener
		}
	}
}

// resolveCapture applies opts over the CGO-free default opener.
func resolveCapture(opts ...Option) capture.Opener {
	cfg := captureConfig{opener: nullcapture.New()}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg.opener
}
