package netif

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// AutoRedetect refreshes interfaces and auto-detects the best available interface.
// Returns the new interface name and whether it changed from the previous one.
// #756: Called when link state changes to automatically switch to a working interface.
func (m *Manager) AutoRedetect() (string, bool) {
	// Refresh the interface list first
	if err := m.RefreshInterfaces(); err != nil {
		logging.GetLogger().Warn("Failed to refresh interfaces during auto-redetect", "error", err)
	}

	m.mu.Lock()
	oldInterface := m.currentInterface

	// Find the best available interface (no preferred list - pure auto-detection)
	candidates := m.collectCandidates()
	newInterface := candidates.selectBest()

	if newInterface == "" {
		m.mu.Unlock()
		logging.GetLogger().Warn("Auto-redetect found no suitable interface")
		return oldInterface, false
	}

	changed := newInterface != oldInterface
	if changed {
		m.currentInterface = newInterface
		logging.GetLogger().Info("Auto-redetect switched interface (#756)",
			"old", oldInterface,
			"new", newInterface,
			"reason", "link state change or better interface available")
	}
	m.mu.Unlock()

	// Notify callbacks if interface changed
	if changed {
		m.notifyInterfaceChange(oldInterface, newInterface)
	}

	return newInterface, changed
}

// interfaceCandidates holds categorized interface names for selection.
type interfaceCandidates struct {
	ethernetWithIP, wifiWithIP, ethernetUp, wifiUp []string
}

// FindFirstAvailable finds the first available interface from a list.
// If no preferred interface is found, it auto-detects the best physical interface:
// Priority: Ethernet with IP > WiFi with IP > Ethernet up > WiFi up
// Virtual interfaces (docker, bridge, veth, etc.) are excluded from auto-detection.
func (m *Manager) FindFirstAvailable(preferred []string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, name := range preferred {
		if info, ok := m.interfaces[name]; ok && info.Up {
			return name
		}
	}

	candidates := m.collectCandidates()
	return candidates.selectBest()
}

// collectCandidates categorizes interfaces by type and connectivity.
func (m *Manager) collectCandidates() *interfaceCandidates {
	c := &interfaceCandidates{}
	for name, info := range m.interfaces {
		if info.Type == InterfaceTypeLoopback || info.Type == InterfaceTypeVirtual || !info.Up {
			continue
		}
		hasIP := hasRoutableAddress(info.Addresses)
		switch info.Type {
		case InterfaceTypeEthernet:
			if hasIP {
				c.ethernetWithIP = append(c.ethernetWithIP, name)
			} else {
				c.ethernetUp = append(c.ethernetUp, name)
			}
		case InterfaceTypeWiFi:
			if hasIP {
				c.wifiWithIP = append(c.wifiWithIP, name)
			} else {
				c.wifiUp = append(c.wifiUp, name)
			}
		case InterfaceTypeOther:
			if hasIP {
				c.ethernetWithIP = append(c.ethernetWithIP, name)
			}
		case InterfaceTypeLoopback, InterfaceTypeVirtual:
			// Already filtered
		}
	}
	return c
}

// selectBest returns the best interface in priority order.
func (c *interfaceCandidates) selectBest() string {
	if len(c.ethernetWithIP) > 0 {
		return c.ethernetWithIP[0]
	}
	if len(c.wifiWithIP) > 0 {
		return c.wifiWithIP[0]
	}
	if len(c.ethernetUp) > 0 {
		return c.ethernetUp[0]
	}
	if len(c.wifiUp) > 0 {
		return c.wifiUp[0]
	}
	return ""
}

// GetLinkStatus returns the link status for an interface.
func (m *Manager) GetLinkStatus(name string) (*LinkStatus, error) {
	m.mu.RLock()
	info, ok := m.interfaces[name]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("interface %s not found", name)
	}

	// Separate carrier (Layer 2) from IP assignment (Layer 3)
	carrier := info.Running                     // Physical link/carrier detected
	hasIP := hasRoutableAddress(info.Addresses) // Has routable IP address
	linkUp := carrier && hasIP                  // Legacy: both conditions met

	status := &LinkStatus{
		LinkUp:  linkUp,
		Carrier: carrier,
		HasIP:   hasIP,
	}

	// Try to read speed from sysfs (Linux only)
	speedPath := filepath.Join("sys", "class", "net", name, "speed")
	speedPath = string(os.PathSeparator) + speedPath

	if data, err := os.ReadFile(speedPath); err == nil {
		speed := strings.TrimSpace(string(data))
		if speed != "" && speed != "-1" {
			status.Speed = speed + "Mb/s"
		}
	}

	// Try to read duplex from sysfs (Linux only)
	duplexPath := filepath.Join("sys", "class", "net", name, "duplex")
	duplexPath = string(os.PathSeparator) + duplexPath

	if data, err := os.ReadFile(duplexPath); err == nil {
		status.Duplex = strings.TrimSpace(string(data))
	}

	// macOS: try to get link info from ifconfig
	if status.Speed == "" {
		speed, duplex := getLinkInfoFromIfconfig(name)
		if speed != "" {
			status.Speed = speed
		}
		if duplex != "" {
			status.Duplex = duplex
		}
	}

	// Get ethtool settings (autoneg, advertised modes) on Linux
	autoNeg, advertised := getEthtoolSettings(name)
	status.AutoNeg = autoNeg
	status.Advertised = advertised

	return status, nil
}

// hasRoutableAddress checks if any address is routable (not link-local).
func hasRoutableAddress(addresses []string) bool {
	for _, addr := range addresses {
		// Parse the address (remove CIDR suffix if present)
		ipStr, _, _ := strings.Cut(addr, "/")
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		// Skip loopback
		if ip.IsLoopback() {
			continue
		}
		// Skip link-local (169.254.x.x for IPv4, fe80:: for IPv6)
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		// Found a routable address
		return true
	}
	return false
}

// getLinkInfoFromIfconfig parses ifconfig output on macOS.
func getLinkInfoFromIfconfig(_ string) (string, string) {
	// This is a placeholder - actual implementation would exec ifconfig
	// and parse "media: autoselect (1000baseT <full-duplex>)"
	return "", ""
}
