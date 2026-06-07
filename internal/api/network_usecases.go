package api

// network_usecases.go wires the API layer to the network application (use-case)
// service (ADR-0016 strangle phase 3). The adapters implement the narrow
// networkapp ports over the netif manager and the live config, so the IP-config
// and MTU handlers depend on a use-case instead of reaching into s.netManager()
// and s.config directly.

import (
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	networkapp "github.com/MustardSeedNetworks/seed/internal/network/app"
)

// networkHardware implements networkapp.Hardware over the netif manager. The
// manager is resolved lazily so a later-set manager (tests) is honored.
type networkHardware struct {
	mgr func() *netif.Manager
}

func (a networkHardware) ConfigureStaticIP(iface string, ip networkapp.StaticIP) error {
	return a.mgr().ConfigureStaticIP(iface, &netif.StaticIPConfig{
		Address: ip.Address,
		Netmask: ip.Netmask,
		Gateway: ip.Gateway,
		DNS:     ip.DNS,
	})
}

func (a networkHardware) ConfigureDHCP(iface string) error { return a.mgr().ConfigureDHCP(iface) }
func (a networkHardware) SetMTU(iface string, mtu int) error {
	return a.mgr().SetMTU(iface, mtu)
}
func (a networkHardware) CurrentInterface() string { return a.mgr().GetCurrentInterface() }
func (a networkHardware) RefreshInterfaces() error { return a.mgr().RefreshInterfaces() }

// networkConfigStore implements networkapp.ConfigStore over the live config,
// owning the lock + on-disk save the port abstracts away.
type networkConfigStore struct {
	cfg  *config.Config
	path string
}

func (c networkConfigStore) IPSettings() networkapp.Settings {
	c.cfg.RLock()
	defer c.cfg.RUnlock()

	s := networkapp.Settings{Mode: c.cfg.IP.Mode}
	if c.cfg.IP.Static != nil {
		s.Address = c.cfg.IP.Static.Address
		s.Netmask = c.cfg.IP.Static.Netmask
		s.Gateway = c.cfg.IP.Static.Gateway
		s.DNS = c.cfg.IP.Static.DNS
	}
	return s
}

func (c networkConfigStore) PersistStatic(ip networkapp.StaticIP) error {
	// Lock for the in-memory mutation only; Save acquires its own RLock and so
	// must run unlocked to avoid the historic deadlock (fixes #783).
	c.cfg.Lock()
	c.cfg.IP.Mode = ipModeStatic
	c.cfg.IP.Static = &config.StaticIP{
		Address: ip.Address,
		Netmask: ip.Netmask,
		Gateway: ip.Gateway,
		DNS:     ip.DNS,
	}
	c.cfg.Unlock()
	return c.cfg.Save(c.path)
}

func (c networkConfigStore) PersistDHCP() error {
	c.cfg.Lock()
	c.cfg.IP.Mode = ipModeDHCP
	c.cfg.IP.Static = nil
	c.cfg.Unlock()
	return c.cfg.Save(c.path)
}

// initNetworkUseCase builds the IP-config + MTU use-case (ADR-0016 phase 3).
func (s *Server) initNetworkUseCase() {
	s.networkIP = networkapp.NewIPService(
		networkHardware{mgr: s.netManager},
		networkConfigStore{cfg: s.config, path: s.configPath},
	)
}
