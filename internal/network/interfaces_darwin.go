//go:build darwin

package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// configureStaticIPPlatform applies static IP on macOS using networksetup.
func configureStaticIPPlatform(iface string, cfg *StaticIPConfig) error {
	// Get the network service name for the interface
	service, err := getNetworkServiceName(iface)
	if err != nil {
		return err
	}

	// Convert CIDR to dotted netmask if needed
	netmask := cfg.Netmask
	if net.ParseIP(netmask) == nil {
		// It's a CIDR prefix, convert to dotted
		var prefix int
		if _, err := fmt.Sscanf(netmask, "%d", &prefix); err != nil {
			return fmt.Errorf("invalid netmask prefix %q: %w", netmask, err)
		}
		netmask = cidrToNetmask(prefix)
	}

	// Set manual IP
	args := []string{"-setmanual", service, cfg.Address, netmask}
	if cfg.Gateway != "" {
		args = append(args, cfg.Gateway)
	}
	//nolint:gosec // G204: networksetup is a known macOS system binary, args are validated
	if err := exec.Command("networksetup", args...).Run(); err != nil {
		return fmt.Errorf("failed to set static IP: %w", err)
	}

	// Configure DNS if provided
	if len(cfg.DNS) > 0 {
		dnsArgs := append([]string{"-setdnsservers", service}, cfg.DNS...)
		//nolint:gosec // G204: networksetup is a known macOS system binary, args are validated
		if err := exec.Command("networksetup", dnsArgs...).Run(); err != nil {
			return fmt.Errorf("failed to configure DNS: %w", err)
		}
	}

	return nil
}

// configureDHCPPlatform switches to DHCP on macOS.
func configureDHCPPlatform(iface string) error {
	service, err := getNetworkServiceName(iface)
	if err != nil {
		return err
	}

	//nolint:gosec // G204: networksetup is a known macOS system binary, service is from validated iface
	return exec.Command("networksetup", "-setdhcp", service).Run()
}

// getNetworkServiceName gets the macOS network service name for an interface.
func getNetworkServiceName(iface string) (string, error) {
	// Common mappings
	serviceNames := map[string]string{
		"en0": "Ethernet",
		"en1": "Wi-Fi",
	}

	if name, ok := serviceNames[iface]; ok {
		return name, nil
	}

	// Try to find the service by listing all
	output, err := exec.Command("networksetup", "-listnetworkserviceorder").Output()
	if err != nil {
		return "", fmt.Errorf("failed to list network services: %w", err)
	}

	// Parse output looking for our interface
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Device: "+iface) && i > 0 {
			// Service name is in the previous line
			prev := lines[i-1]
			// Format: "(1) Service Name"
			if idx := strings.Index(prev, ") "); idx != -1 {
				return strings.TrimSpace(prev[idx+2:]), nil
			}
		}
	}

	return "", fmt.Errorf("network service not found for interface %s", iface)
}
