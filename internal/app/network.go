package app

import (
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/network/ipconfig"
)

// NewNetworkIP builds the IP-config + MTU use-case (ADR-0020), assembling the
// ipconfig ports over the netif manager and the live config. The manager is
// resolved through mgr on each call so a later-set manager (the api test
// harness) is honored; cfg/path are fixed for the process lifetime.
func NewNetworkIP(mgr func() *netif.Manager, cfg *config.Config, path string) *ipconfig.Service {
	return ipconfig.NewService(
		networkHardware{mgr: mgr},
		networkConfigStore{cfg: cfg, path: path},
	)
}

// networkHardware implements ipconfig.Hardware over the netif manager. The
// manager is resolved lazily so a later-set manager (tests) is honored.
type networkHardware struct {
	mgr func() *netif.Manager
}

func (a networkHardware) ConfigureStaticIP(iface string, ip ipconfig.StaticIP) error {
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

// networkConfigStore implements ipconfig.ConfigStore over the live config,
// owning the lock + on-disk save the port abstracts away.
type networkConfigStore struct {
	cfg  *config.Config
	path string
}

func (c networkConfigStore) IPSettings() ipconfig.Settings {
	c.cfg.RLock()
	defer c.cfg.RUnlock()

	s := ipconfig.Settings{Mode: c.cfg.IP.Mode}
	if c.cfg.IP.Static != nil {
		s.Address = c.cfg.IP.Static.Address
		s.Netmask = c.cfg.IP.Static.Netmask
		s.Gateway = c.cfg.IP.Static.Gateway
		s.DNS = c.cfg.IP.Static.DNS
	}
	return s
}

func (c networkConfigStore) PersistStatic(ip ipconfig.StaticIP) error {
	// Lock for the in-memory mutation only; Save acquires its own RLock and so
	// must run unlocked to avoid the historic deadlock (fixes #783).
	c.cfg.Lock()
	c.cfg.IP.Mode = ipconfig.ModeStatic
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
	c.cfg.IP.Mode = ipconfig.ModeDHCP
	c.cfg.IP.Static = nil
	c.cfg.Unlock()
	return c.cfg.Save(c.path)
}
