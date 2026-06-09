package app

// wifi.go wires the composition root to the Wi-Fi troubleshooting application
// (use-case) services (ADR-0020 clean-hexagonal). The adapters below implement
// the narrow ports declared in internal/wifi/troubleshooting over the concrete
// collaborators (the Wi-Fi manager/scanner, the network interface manager, the
// live config, the discovery bridge, and the visibility component), so the API
// handlers depend on use-cases instead of reaching into the service container or
// background components directly. Collaborators are resolved through lazy
// accessors so a later-set value (the api test harness) is honored.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// NewWiFiQueries builds the Wi-Fi visibility read use-case (ADR-0020) over a lazy
// accessor for the live visibility component. A nil component (no capture wired,
// e.g. the test harness) yields a use-case that degrades to empty-but-valid
// results rather than erroring.
func NewWiFiQueries(src func() *visibility.Service) *troubleshooting.Queries {
	return troubleshooting.NewQueries(wifiVisibilitySource(src))
}

// wifiVisibilitySource returns the visibility port, or a genuinely-nil interface
// (not a typed nil) when no component is wired, so the use-case sees a truly
// absent source instead of panicking on a typed nil.
func wifiVisibilitySource(src func() *visibility.Service) troubleshooting.VisibilitySource {
	if s := src(); s != nil {
		return s
	}
	return nil
}

// NewWiFiManagement builds the Wi-Fi settings/scan/status/connect use-case
// (ADR-0020), assembling the hardware, interface-lister, and interface-store
// adapters over the lazy manager/scanner/netif accessors and the live config.
func NewWiFiManagement(
	mgr func() *wifi.Manager,
	scanner func() *wifi.Scanner,
	net func() *netif.Manager,
	cfg *config.Config,
	path string,
) *troubleshooting.Management {
	return troubleshooting.NewManagement(
		wifiHardware{mgr: mgr, scanner: scanner},
		wifiInterfaceLister{net: net},
		wifiInterfaceStore{cfg: cfg, path: path},
	)
}

// NewWiFiDiscovery builds the enhanced Wi-Fi discovery use-case (ADR-0020) over a
// lazy accessor for the discovery bridge. A nil bridge makes every method return
// troubleshooting.ErrDiscoveryUnavailable.
func NewWiFiDiscovery(bridge func() *enumerate.WiFiBridge) *troubleshooting.Discovery {
	return troubleshooting.NewDiscovery(wifiDiscoverySource(bridge))
}

// wifiDiscoverySource returns the discovery use-case source for the current
// bridge, or a genuinely-nil interface when no bridge is wired so the use-case
// degrades to ErrDiscoveryUnavailable rather than panicking on a typed nil.
func wifiDiscoverySource(bridge func() *enumerate.WiFiBridge) troubleshooting.DiscoverySource {
	if b := bridge(); b != nil {
		return wifiBridgeSource{bridge: b}
	}
	return nil
}

// wifiHardware implements troubleshooting.Hardware over the Wi-Fi manager and
// scanner. The collaborators are resolved lazily; the Available reporters mirror
// the handlers' historic nil-guards so the use-case can degrade gracefully.
type wifiHardware struct {
	mgr     func() *wifi.Manager
	scanner func() *wifi.Scanner
}

func (a wifiHardware) ManagerAvailable() bool { return a.mgr() != nil }
func (a wifiHardware) ScannerAvailable() bool { return a.scanner() != nil }

func (a wifiHardware) IsWireless() bool {
	m := a.mgr()
	return m != nil && m.IsWireless()
}

func (a wifiHardware) SetInterface(name string) {
	if m := a.mgr(); m != nil {
		m.SetInterface(name)
	}
}

func (a wifiHardware) Scan() ([]*wifi.ScannedNetwork, error) { return a.scanner().Scan() }

func (a wifiHardware) Connect(ssid, password string) (*wifi.ConnectionResult, error) {
	return a.mgr().Connect(ssid, password)
}

func (a wifiHardware) Disconnect() (*wifi.ConnectionResult, error) {
	return a.mgr().Disconnect()
}

// wifiInterfaceLister implements troubleshooting.InterfaceLister over the network
// interface manager, returning a non-nil slice even when no manager is present.
type wifiInterfaceLister struct {
	net func() *netif.Manager
}

func (l wifiInterfaceLister) WirelessInterfaceNames() []string {
	names := []string{}
	net := l.net()
	if net == nil {
		return names
	}
	for _, iface := range net.GetInterfaces() {
		if net.IsWireless(iface.Name) {
			names = append(names, iface.Name)
		}
	}
	return names
}

// wifiInterfaceStore implements troubleshooting.InterfaceStore over the live
// config, owning the lock + on-disk save that the port abstracts away.
type wifiInterfaceStore struct {
	cfg  *config.Config
	path string
}

func (s wifiInterfaceStore) ResolvedWiFiInterface() string {
	iface := s.cfg.Interface.WiFi
	if iface == "" {
		iface = s.cfg.Interface.Default
	}
	return iface
}

func (s wifiInterfaceStore) SaveWiFiInterface(name string) error {
	// Lock for the in-memory mutation only; Save acquires its own RLock, so it
	// must run unlocked to avoid the historic deadlock.
	s.cfg.Lock()
	s.cfg.Interface.WiFi = name
	s.cfg.Unlock()
	return s.cfg.Save(s.path)
}

// wifiBridgeSource adapts *enumerate.WiFiBridge to troubleshooting.DiscoverySource.
type wifiBridgeSource struct {
	bridge *enumerate.WiFiBridge
}

func (b wifiBridgeSource) Scan(ctx context.Context) (*discovery.WiFiScanResult, error) {
	return b.bridge.Scan(ctx)
}
func (b wifiBridgeSource) Networks() []discovery.WiFiNetwork { return b.bridge.GetNetworks() }
func (b wifiBridgeSource) AccessPoints() []discovery.WiFiAccessPoint {
	return b.bridge.GetAccessPoints()
}
func (b wifiBridgeSource) Stats() *discovery.WiFiDiscoveryStats { return b.bridge.GetStats() }
