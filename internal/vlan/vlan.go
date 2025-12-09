// Package vlan provides VLAN detection and configuration functionality.
package vlan

import (
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Info contains VLAN information for an interface.
type Info struct {
	NativeVlan  *int  `json:"nativeVlan"`
	TaggedVlans []int `json:"taggedVlans"`
	VoiceVlan   *int  `json:"voiceVlan"`
	Configured  struct {
		Enabled bool `json:"enabled"`
		ID      int  `json:"id"`
	} `json:"configured"`
}

// Manager handles VLAN detection and configuration.
type Manager struct {
	interfaceName string
	configuredID  int
	enabled       bool
	mu            sync.RWMutex
}

// NewManager creates a new VLAN manager.
func NewManager(interfaceName string) *Manager {
	return &Manager{
		interfaceName: interfaceName,
	}
}

// SetInterface updates the interface to monitor.
func (m *Manager) SetInterface(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
}

// SetConfigured sets the configured VLAN tagging.
func (m *Manager) SetConfigured(enabled bool, id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
	m.configuredID = id
}

// GetInfo returns current VLAN information.
func (m *Manager) GetInfo() *Info {
	m.mu.RLock()
	iface := m.interfaceName
	enabled := m.enabled
	configuredID := m.configuredID
	m.mu.RUnlock()

	info := &Info{
		TaggedVlans: make([]int, 0),
	}
	info.Configured.Enabled = enabled
	info.Configured.ID = configuredID

	// Detect VLAN subinterfaces
	taggedVlans := m.detectVlanSubinterfaces(iface)
	info.TaggedVlans = taggedVlans

	return info
}

// GetInfoWithLLDP returns VLAN info enriched with LLDP/CDP data.
func (m *Manager) GetInfoWithLLDP(nativeVlan, voiceVlan *int) *Info {
	info := m.GetInfo()
	info.NativeVlan = nativeVlan
	info.VoiceVlan = voiceVlan
	return info
}

// detectVlanSubinterfaces finds 802.1Q VLAN subinterfaces.
func (m *Manager) detectVlanSubinterfaces(iface string) []int {
	vlans := make([]int, 0)

	switch runtime.GOOS {
	case "linux":
		vlans = detectVlanSubinterfacesLinux(iface)
	case "darwin":
		vlans = detectVlanSubinterfacesDarwin(iface)
	}

	return vlans
}

// detectVlanSubinterfacesLinux detects VLAN subinterfaces on Linux.
func detectVlanSubinterfacesLinux(iface string) []int {
	vlans := make([]int, 0)

	// Method 1: Check for interfaces named eth0.100, eth0.200, etc.
	cmd := exec.Command("ip", "-d", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return vlans
	}

	// Parse output looking for vlan protocol 802.1Q
	// Example: "eth0.100@eth0: ... vlan protocol 802.1Q id 100"
	lines := strings.Split(string(output), "\n")
	vlanRe := regexp.MustCompile(`vlan protocol 802\.1Q id (\d+)`)
	ifaceRe := regexp.MustCompile(`^\d+:\s+` + regexp.QuoteMeta(iface) + `\.(\d+)@`)

	for i, line := range lines {
		// Check if this is a subinterface of our interface
		if matches := ifaceRe.FindStringSubmatch(line); matches != nil {
			// Look for VLAN ID in next few lines
			for j := i; j < len(lines) && j < i+3; j++ {
				if vlanMatches := vlanRe.FindStringSubmatch(lines[j]); vlanMatches != nil {
					if vlanID, err := strconv.Atoi(vlanMatches[1]); err == nil {
						vlans = append(vlans, vlanID)
					}
					break
				}
			}
		}
	}

	// Method 2: Check /proc/net/vlan/config if available
	procVlans := detectVlansFromProc(iface)
	for _, v := range procVlans {
		if !contains(vlans, v) {
			vlans = append(vlans, v)
		}
	}

	return vlans
}

// detectVlansFromProc reads VLAN config from /proc/net/vlan/config.
func detectVlansFromProc(iface string) []int {
	vlans := make([]int, 0)

	cmd := exec.Command("cat", "/proc/net/vlan/config")
	output, err := cmd.Output()
	if err != nil {
		return vlans
	}

	// Format: "eth0.100 | 100 | eth0"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) >= 3 {
			parentIface := strings.TrimSpace(fields[2])
			if parentIface == iface {
				if vlanID, err := strconv.Atoi(strings.TrimSpace(fields[1])); err == nil {
					vlans = append(vlans, vlanID)
				}
			}
		}
	}

	return vlans
}

// detectVlanSubinterfacesDarwin detects VLAN subinterfaces on macOS.
func detectVlanSubinterfacesDarwin(iface string) []int {
	vlans := make([]int, 0)

	// On macOS, VLANs are created with "vlan" prefix
	// Check using ifconfig
	cmd := exec.Command("ifconfig", "-a")
	output, err := cmd.Output()
	if err != nil {
		return vlans
	}

	// Look for vlan interfaces that reference our parent interface
	// Example: "vlan0: ... vlan: 100 parent interface: en0"
	lines := strings.Split(string(output), "\n")
	vlanRe := regexp.MustCompile(`vlan:\s*(\d+)\s+parent interface:\s*(\S+)`)

	for _, line := range lines {
		if matches := vlanRe.FindStringSubmatch(line); matches != nil {
			parentIface := matches[2]
			if parentIface == iface {
				if vlanID, err := strconv.Atoi(matches[1]); err == nil {
					vlans = append(vlans, vlanID)
				}
			}
		}
	}

	return vlans
}

// CreateVlanInterface creates an 802.1Q VLAN subinterface.
func CreateVlanInterface(parentIface string, vlanID int) error {
	switch runtime.GOOS {
	case "linux":
		return createVlanInterfaceLinux(parentIface, vlanID)
	case "darwin":
		return createVlanInterfaceDarwin(parentIface, vlanID)
	default:
		return nil
	}
}

// createVlanInterfaceLinux creates a VLAN interface on Linux.
func createVlanInterfaceLinux(parentIface string, vlanID int) error {
	vlanIface := parentIface + "." + strconv.Itoa(vlanID)

	// ip link add link eth0 name eth0.100 type vlan id 100
	cmd := exec.Command("ip", "link", "add", "link", parentIface,
		"name", vlanIface, "type", "vlan", "id", strconv.Itoa(vlanID))
	if err := cmd.Run(); err != nil {
		return err
	}

	// Bring interface up
	cmd = exec.Command("ip", "link", "set", vlanIface, "up")
	return cmd.Run()
}

// createVlanInterfaceDarwin creates a VLAN interface on macOS.
func createVlanInterfaceDarwin(parentIface string, vlanID int) error {
	// On macOS, we need to create a vlan interface first
	// This typically requires networksetup or manual configuration
	// For now, return nil as this is advanced functionality
	return nil
}

// DeleteVlanInterface removes an 802.1Q VLAN subinterface.
func DeleteVlanInterface(parentIface string, vlanID int) error {
	switch runtime.GOOS {
	case "linux":
		vlanIface := parentIface + "." + strconv.Itoa(vlanID)
		cmd := exec.Command("ip", "link", "delete", vlanIface)
		return cmd.Run()
	default:
		return nil
	}
}

// contains checks if a slice contains a value.
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
