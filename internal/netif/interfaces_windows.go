//go:build windows

package netif

// Windows-specific interface configuration module uses netsh command-line tool and
// Windows IP Helper API for interface configuration, static IP assignment, DHCP management,
// and DNS setup on Windows systems.

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// Command timeout constants for Windows system utilities.
const (
	// netshTimeoutSeconds is the timeout for netsh commands (static IP, DHCP, DNS).
	// 30 seconds allows for slow network reconfiguration operations.
	netshTimeoutSeconds = 30

	// ipconfigTimeoutSeconds is the timeout for ipconfig commands (DHCP release/renew).
	// 60 seconds required as DHCP renewal can be slow.
	ipconfigTimeoutSeconds = 60
)

// configureStaticIPPlatform applies static IP on Windows using netsh.
func configureStaticIPPlatform(iface string, cfg *StaticIPConfig) error {
	// Validate interface name to prevent command injection
	if err := validation.ValidateInterface(iface); err != nil {
		return fmt.Errorf("invalid interface: %w", err)
	}

	// Validate IP address
	if !validation.IsValidIPv4(cfg.Address) {
		return fmt.Errorf("invalid IP address: %s", cfg.Address)
	}

	// Validate gateway if provided
	if cfg.Gateway != "" && !validation.IsValidIPv4(cfg.Gateway) {
		return fmt.Errorf("invalid gateway address: %s", cfg.Gateway)
	}

	// netsh wants dotted decimal, so normalise whichever form was given.
	prefix, ok := parseNetmask(cfg.Netmask)
	if !ok {
		return fmt.Errorf("invalid netmask: %s", cfg.Netmask)
	}
	netmask := cidrToNetmask(prefix)

	ctx, cancel := context.WithTimeout(context.Background(), netshTimeoutSeconds*time.Second)
	defer cancel()

	// Set static IP using netsh
	// Example: netsh interface ip set address "Ethernet" static 192.168.1.100 255.255.255.0 192.168.1.1
	args := []string{
		"interface", "ip", "set", "address",
		iface, "static", cfg.Address, netmask,
	}
	if cfg.Gateway != "" {
		args = append(args, cfg.Gateway)
	}

	// netsh is a known Windows system binary, args are validated
	if err := exec.CommandContext(ctx, "netsh", args...).Run(); err != nil {
		return fmt.Errorf("failed to set static IP: %w", err)
	}

	// Configure DNS if provided
	if len(cfg.DNS) > 0 {
		if err := configureWindowsDNS(ctx, iface, cfg.DNS); err != nil {
			return fmt.Errorf("failed to configure DNS: %w", err)
		}
	}

	return nil
}

// configureWindowsDNS sets DNS servers on Windows using netsh.
func configureWindowsDNS(ctx context.Context, iface string, dnsServers []string) error {
	// Validate DNS servers
	if err := validation.ValidateDNSServers(dnsServers); err != nil {
		return err
	}

	// Set primary DNS
	// Example: netsh interface ip set dns "Ethernet" static 8.8.8.8
	args := []string{
		"interface", "ip", "set", "dns",
		iface, "static", dnsServers[0],
	}

	// netsh is a known Windows system binary, args are validated
	if err := exec.CommandContext(ctx, "netsh", args...).Run(); err != nil {
		return fmt.Errorf("failed to set primary DNS: %w", err)
	}

	// Add additional DNS servers
	for i := 1; i < len(dnsServers); i++ {
		// Example: netsh interface ip add dns "Ethernet" 8.8.4.4 index=2
		addArgs := []string{
			"interface", "ip", "add", "dns",
			iface, dnsServers[i], fmt.Sprintf("index=%d", i+1),
		}

		// netsh is a known Windows system binary, args are validated
		if err := exec.CommandContext(ctx, "netsh", addArgs...).Run(); err != nil {
			return fmt.Errorf("failed to add DNS server %s: %w", dnsServers[i], err)
		}
	}

	return nil
}

// configureDHCPPlatform switches to DHCP on Windows.
func configureDHCPPlatform(iface string) error {
	// Validate interface name to prevent command injection
	if err := validation.ValidateInterface(iface); err != nil {
		return fmt.Errorf("invalid interface: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ipconfigTimeoutSeconds*time.Second)
	defer cancel()

	// Enable DHCP for IP address
	// Example: netsh interface ip set address "Ethernet" dhcp
	// netsh is a known Windows system binary, args are validated
	if err := exec.CommandContext(ctx, "netsh", "interface", "ip", "set", "address", iface, "dhcp").Run(); err != nil {
		return fmt.Errorf("failed to enable DHCP for IP: %w", err)
	}

	// Enable DHCP for DNS
	// Example: netsh interface ip set dns "Ethernet" dhcp
	// netsh is a known Windows system binary, args are validated
	if err := exec.CommandContext(ctx, "netsh", "interface", "ip", "set", "dns", iface, "dhcp").Run(); err != nil {
		return fmt.Errorf("failed to enable DHCP for DNS: %w", err)
	}

	// Force DHCP renewal using ipconfig
	// ipconfig is a known Windows system binary
	_ = exec.CommandContext(ctx, "ipconfig", "/release", iface).Run()

	// ipconfig is a known Windows system binary
	if err := exec.CommandContext(ctx, "ipconfig", "/renew", iface).Run(); err != nil {
		// Non-fatal: DHCP renewal may fail if network is not connected
		// The interface is already configured for DHCP
		return nil
	}

	return nil
}

// setMTUPlatform sets the MTU on Windows using netsh.
func setMTUPlatform(iface string, mtu int) error {
	// Validate interface name to prevent command injection
	if err := validation.ValidateInterface(iface); err != nil {
		return fmt.Errorf("invalid interface: %w", err)
	}

	// Validate interface exists
	if _, err := net.InterfaceByName(iface); err != nil {
		return fmt.Errorf("interface not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), netshTimeoutSeconds*time.Second)
	defer cancel()

	// Set MTU using netsh
	// Example: netsh interface ipv4 set subinterface "Ethernet" mtu=1500 store=persistent
	// netsh is a known Windows system binary, args are validated
	if err := exec.CommandContext(ctx, "netsh", "interface", "ipv4", "set", "subinterface",
		iface, fmt.Sprintf("mtu=%d", mtu), "store=persistent").Run(); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}
