//go:build darwin

package network

// macOS-specific interface configuration module uses networksetup command-line tool
// for interface configuration, static IP assignment, DHCP management, and DNS setup.

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Command timeout constants for macOS system utilities.
const (
	// networksetupTimeoutSeconds is the timeout for networksetup commands (static IP, DHCP).
	// 30 seconds allows for slow network reconfiguration operations.
	networksetupTimeoutSeconds = 30

	// ifconfigTimeoutSeconds is the timeout for ifconfig commands (MTU, service lookup).
	// 10 seconds is sufficient for quick interface queries.
	ifconfigTimeoutSeconds = 10
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
		if _, scanErr := fmt.Sscanf(netmask, "%d", &prefix); scanErr != nil {
			return fmt.Errorf("invalid netmask prefix %q: %w", netmask, scanErr)
		}
		netmask = cidrToNetmask(prefix)
	}

	ctx, cancel := context.WithTimeout(context.Background(), networksetupTimeoutSeconds*time.Second)
	defer cancel()

	// Set manual IP
	args := []string{"-setmanual", service, cfg.Address, netmask}
	if cfg.Gateway != "" {
		args = append(args, cfg.Gateway)
	}

	if runErr := exec.CommandContext(ctx, "networksetup", args...).Run(); runErr != nil {
		return fmt.Errorf("failed to set static IP: %w", runErr)
	}

	// Configure DNS if provided
	if len(cfg.DNS) > 0 {
		dnsArgs := append([]string{"-setdnsservers", service}, cfg.DNS...)

		if dnsErr := exec.CommandContext(ctx, "networksetup", dnsArgs...).Run(); dnsErr != nil {
			return fmt.Errorf("failed to configure DNS: %w", dnsErr)
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

	ctx, cancel := context.WithTimeout(context.Background(), networksetupTimeoutSeconds*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "networksetup", "-setdhcp", service).Run()
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
	ctx, cancel := context.WithTimeout(context.Background(), ifconfigTimeoutSeconds*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "networksetup", "-listnetworkserviceorder").Output()
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
			if _, after, found := strings.Cut(prev, ") "); found {
				return strings.TrimSpace(after), nil
			}
		}
	}

	return "", fmt.Errorf("network service not found for interface %s", iface)
}

// setMTUPlatform sets the MTU on macOS using ifconfig.
func setMTUPlatform(iface string, mtu int) error {
	// Validate interface name exists
	_, err := net.InterfaceByName(iface)
	if err != nil {
		return fmt.Errorf("interface not found: %w", err)
	}

	// Use ifconfig to set MTU
	ctx, cancel := context.WithTimeout(context.Background(), ifconfigTimeoutSeconds*time.Second)
	defer cancel()
	//nolint:gosec // G204: ifconfig is a known macOS system binary, args are validated
	if runErr := exec.CommandContext(ctx, "ifconfig", iface, "mtu", strconv.Itoa(mtu)).Run(); runErr != nil {
		return fmt.Errorf("failed to set MTU: %w", runErr)
	}

	return nil
}
