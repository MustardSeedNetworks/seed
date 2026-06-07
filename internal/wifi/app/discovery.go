package wifiapp

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// ErrDiscoveryUnavailable is returned by the discovery use-case when the Wi-Fi
// discovery bridge is not wired. The handler maps it to a 503 Service
// Unavailable, preserving the pre-strangle behavior.
var ErrDiscoveryUnavailable = errors.New("wifi discovery bridge not available")

// DiscoverySource is the narrow capability the enhanced Wi-Fi discovery
// use-case needs from the discovery bridge. Defined here at the consumer
// (ADR-0016) and satisfied by an adapter over *discovery.WiFiBridge.
type DiscoverySource interface {
	Scan(ctx context.Context) (*discovery.WiFiScanResult, error)
	Networks() []discovery.WiFiNetwork
	AccessPoints() []discovery.WiFiAccessPoint
	Stats() *discovery.WiFiDiscoveryStats
}

// Discovery is the enhanced Wi-Fi discovery use-case (vendor lookup,
// authorization, channel utilization) over the discovery bridge. A nil source
// (bridge not wired) makes every method return ErrDiscoveryUnavailable.
type Discovery struct {
	src DiscoverySource
}

// NewDiscovery builds the discovery use-case over src.
func NewDiscovery(src DiscoverySource) *Discovery { return &Discovery{src: src} }

// Scan triggers an enhanced Wi-Fi scan.
func (d *Discovery) Scan(ctx context.Context) (*discovery.WiFiScanResult, error) {
	if d == nil || d.src == nil {
		return nil, ErrDiscoveryUnavailable
	}
	return d.src.Scan(ctx)
}

// Networks returns the Wi-Fi networks from the most recent enhanced scan.
func (d *Discovery) Networks() ([]discovery.WiFiNetwork, error) {
	if d == nil || d.src == nil {
		return nil, ErrDiscoveryUnavailable
	}
	return d.src.Networks(), nil
}

// AccessPoints returns the access points from the most recent enhanced scan.
func (d *Discovery) AccessPoints() ([]discovery.WiFiAccessPoint, error) {
	if d == nil || d.src == nil {
		return nil, ErrDiscoveryUnavailable
	}
	return d.src.AccessPoints(), nil
}

// Stats returns the aggregated Wi-Fi discovery statistics.
func (d *Discovery) Stats() (*discovery.WiFiDiscoveryStats, error) {
	if d == nil || d.src == nil {
		return nil, ErrDiscoveryUnavailable
	}
	return d.src.Stats(), nil
}
