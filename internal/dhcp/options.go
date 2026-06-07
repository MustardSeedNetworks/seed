package dhcp

import (
	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/capture/nullcapture"
)

// Option configures optional dependencies of the dhcp constructors
// (NewMonitor, NewRogueDetector).
type Option func(*captureConfig)

// captureConfig collects optional construction settings.
type captureConfig struct {
	opener capture.Opener
}

// WithCapture injects the live packet-capture Opener used to sniff DHCP traffic.
// The composition root passes the libpcap-backed adapter (internal/capture/pcap)
// under CGO; absent an override the default is the CGO-free no-op (nullcapture),
// so the dhcp package and its tests never link libpcap. A nil opener is ignored.
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
