package api

// wifi_usecases.go wires the API layer to the Wi-Fi application (use-case)
// services (ADR-0016 strangle phase 2). The adapters below implement the narrow
// ports declared in internal/wifi/app over the concrete service-container
// collaborators, so the handlers depend on use-cases instead of reaching into
// ServiceContainer / Server.background directly.

import (
	"context"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/netif"
	"github.com/krisarmstrong/seed/internal/wifi"
	wifiapp "github.com/krisarmstrong/seed/internal/wifi/app"
)

// wifiHardwareAdapter implements wifiapp.Hardware over the Wi-Fi manager and
// scanner. Availability reporters mirror the handlers' historic nil-guards.
type wifiHardwareAdapter struct {
	manager *wifi.Manager
	scanner *wifi.Scanner
}

func (a wifiHardwareAdapter) ManagerAvailable() bool { return a.manager != nil }
func (a wifiHardwareAdapter) ScannerAvailable() bool { return a.scanner != nil }

func (a wifiHardwareAdapter) IsWireless() bool {
	return a.manager != nil && a.manager.IsWireless()
}

func (a wifiHardwareAdapter) SetInterface(name string) {
	if a.manager != nil {
		a.manager.SetInterface(name)
	}
}

func (a wifiHardwareAdapter) Scan() ([]*wifi.ScannedNetwork, error) { return a.scanner.Scan() }

func (a wifiHardwareAdapter) Connect(ssid, password string) (*wifi.ConnectionResult, error) {
	return a.manager.Connect(ssid, password)
}

func (a wifiHardwareAdapter) Disconnect() (*wifi.ConnectionResult, error) {
	return a.manager.Disconnect()
}

// wifiInterfaceLister implements wifiapp.InterfaceLister over the network
// interface manager, returning a non-nil slice even when no manager is present.
type wifiInterfaceLister struct {
	net *netif.Manager
}

func (l wifiInterfaceLister) WirelessInterfaceNames() []string {
	names := []string{}
	if l.net == nil {
		return names
	}
	for _, iface := range l.net.GetInterfaces() {
		if l.net.IsWireless(iface.Name) {
			names = append(names, iface.Name)
		}
	}
	return names
}

// wifiInterfaceStore implements wifiapp.InterfaceStore over the live config,
// owning the lock + on-disk save that the port abstracts away.
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

// wifiBridgeSource adapts *discovery.WiFiBridge to wifiapp.DiscoverySource.
type wifiBridgeSource struct {
	bridge *discovery.WiFiBridge
}

func (b wifiBridgeSource) Scan(ctx context.Context) (*discovery.WiFiScanResult, error) {
	return b.bridge.Scan(ctx)
}
func (b wifiBridgeSource) Networks() []discovery.WiFiNetwork { return b.bridge.GetNetworks() }
func (b wifiBridgeSource) AccessPoints() []discovery.WiFiAccessPoint {
	return b.bridge.GetAccessPoints()
}
func (b wifiBridgeSource) Stats() *discovery.WiFiDiscoveryStats { return b.bridge.GetStats() }

// wifiDiscoverySource returns the discovery use-case source for the current
// bridge, or a genuinely-nil interface when no bridge is wired so the use-case
// degrades to ErrDiscoveryUnavailable rather than panicking on a typed nil.
func wifiDiscoverySource(bridge *discovery.WiFiBridge) wifiapp.DiscoverySource {
	if bridge == nil {
		return nil
	}
	return wifiBridgeSource{bridge: bridge}
}

// initWiFiUseCases builds the Wi-Fi management + discovery use-cases from the
// fully-wired services. Called from NewServer after initDiscovery so the
// discovery bridge (assigned during initDiscovery) is captured.
func (s *Server) initWiFiUseCases() {
	s.wifiManagement = wifiapp.NewManagement(
		wifiHardwareAdapter{manager: s.wifiManager(), scanner: s.wifiScanner()},
		wifiInterfaceLister{net: s.netManager()},
		wifiInterfaceStore{cfg: s.config, path: s.configPath},
	)
	s.wifiDiscovery = wifiapp.NewDiscovery(wifiDiscoverySource(s.wifiBridge()))
}
