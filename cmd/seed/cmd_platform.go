package main

import (
	"io"
	"runtime"

	"github.com/spf13/cobra"
)

// initPlatformCmd adds the platform command to the CLI.
func initPlatformCmd(state *cliState) {
	platformCmd := &cobra.Command{
		Use:   "platform",
		Short: "Display platform-specific capabilities and limitations",
		Long: `Display platform-specific capabilities and limitations for The Seed.

Different operating systems have varying levels of support for network
diagnostics features. This command shows what's available on your platform
and provides guidance for features that require additional setup.`,
		Run: func(cmd *cobra.Command, _ []string) {
			runPlatform(cmd.OutOrStdout())
		},
	}

	state.rootCmd.AddCommand(platformCmd)
}

// runPlatform displays platform-specific information.
func runPlatform(w io.Writer) {
	writeStr(w, "The Seed - Platform Support ("+runtime.GOOS+"/"+runtime.GOARCH+")\n")
	writeStr(w, "==================================================\n\n")

	switch runtime.GOOS {
	case "linux":
		printLinuxCapabilities(w)
	case "darwin":
		printDarwinCapabilities(w)
	case "windows":
		printWindowsCapabilities(w)
	default:
		writeStr(w, "Platform "+runtime.GOOS+" is not fully supported.\n")
	}
}

func writeStr(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

func printLinuxCapabilities(w io.Writer) {
	writeStr(w, "Linux Platform - Full Support\n\n")
	writeStr(w, "✓ FULLY SUPPORTED:\n")
	writeStr(w, "  • Interface configuration (static IP, DHCP, MTU)\n")
	writeStr(w, "  • Link status monitoring with netlink\n")
	writeStr(w, "  • Speed/duplex detection via ethtool/sysfs\n")
	writeStr(w, "  • Wi-Fi scanning and connection (nl80211)\n")
	writeStr(w, "  • ARP/NDP neighbor discovery\n")
	writeStr(w, "  • Bluetooth scanning (BlueZ)\n")
	writeStr(w, "  • VLAN creation and management (ip link)\n")
	writeStr(w, "  • Gateway and DNS detection\n")
	writeStr(w, "  • DHCP lease information\n\n")
	writeStr(w, "✓ HARDWARE-DEPENDENT (requires compatible NIC):\n")
	writeStr(w, "  • Cable diagnostics (TDR) - Intel, Broadcom, Marvell NICs\n")
	writeStr(w, "  • PHY layer information - via ethtool\n")
	writeStr(w, "  • DOM (Digital Optical Monitoring) - SFP/SFP+ modules\n\n")
	writeStr(w, "RECOMMENDED HARDWARE:\n")
	writeStr(w, "  • Intel I210/I211/I225/I350 for cable diagnostics\n")
	writeStr(w, "  • Intel AX200/AX210 for Wi-Fi\n")
	writeStr(w, "  • Intel X520/X540 or Mellanox ConnectX for SFP+ DOM\n\n")
	writeStr(w, "See HARDWARE.md for detailed compatibility information.\n")
}

func printDarwinCapabilities(w io.Writer) {
	writeStr(w, "macOS Platform - Partial Support\n\n")
	writeStr(w, "✓ FULLY SUPPORTED:\n")
	writeStr(w, "  • Interface configuration (networksetup)\n")
	writeStr(w, "  • Wi-Fi scanning and connection (CoreWLAN)\n")
	writeStr(w, "  • ARP/NDP neighbor discovery\n")
	writeStr(w, "  • Gateway and DNS detection\n")
	writeStr(w, "  • DHCP lease information\n")
	writeStr(w, "  • Link status monitoring\n\n")
	writeStr(w, "⚠ LIMITED SUPPORT:\n")
	writeStr(w, "  • Speed/duplex detection - basic only\n")
	writeStr(w, "  • PHY layer information - limited access\n")
	writeStr(w, "  • Bluetooth scanning - system API restrictions\n")
	writeStr(w, "  • VLAN configuration - Network Preferences only\n\n")
	writeStr(w, "✗ NOT AVAILABLE:\n")
	writeStr(w, "  • Cable diagnostics (TDR) - no macOS API\n")
	writeStr(w, "  • DOM (Digital Optical Monitoring) - no macOS API\n")
	writeStr(w, "  • Direct ethtool-equivalent access\n\n")
	writeStr(w, "NOTES:\n")
	writeStr(w, "  • macOS security model restricts low-level network access\n")
	writeStr(w, "  • Some features require Full Disk Access or elevated privileges\n")
	writeStr(w, "  • Bluetooth scanning may require app permissions\n\n")
	writeStr(w, "See HARDWARE.md for detailed compatibility information.\n")
}

func printWindowsCapabilities(w io.Writer) {
	writeStr(w, "Windows Platform - Partial Support\n\n")
	writeStr(w, "✓ FULLY SUPPORTED:\n")
	writeStr(w, "  • Interface configuration (netsh)\n")
	writeStr(w, "  • Wi-Fi scanning and connection (netsh wlan)\n")
	writeStr(w, "  • ARP/NDP neighbor discovery (iphlpapi)\n")
	writeStr(w, "  • Gateway and DNS detection\n")
	writeStr(w, "  • DHCP lease information (ipconfig)\n")
	writeStr(w, "  • Link status monitoring (PowerShell/WMI)\n")
	writeStr(w, "  • Speed/duplex detection (Get-NetAdapter)\n\n")
	writeStr(w, "⚠ LIMITED SUPPORT:\n")
	writeStr(w, "  • PHY layer information - PowerShell only\n")
	writeStr(w, "  • VLAN detection - depends on NIC driver\n")
	writeStr(w, "  • Bluetooth scanning - requires vendor SDK\n\n")
	writeStr(w, "✗ NOT AVAILABLE (requires vendor tools):\n")
	writeStr(w, "  • Cable diagnostics (TDR)\n")
	writeStr(w, "    → Intel NICs: Install Intel PROSet\n")
	writeStr(w, "    → Broadcom NICs: Install BACS\n")
	writeStr(w, "  • DOM (Digital Optical Monitoring)\n")
	writeStr(w, "    → Intel NICs: Intel PROSet\n")
	writeStr(w, "    → Mellanox NICs: Mellanox WinOF\n")
	writeStr(w, "  • VLAN creation/deletion\n")
	writeStr(w, "    → Use vendor tools or Hyper-V Virtual Switch\n\n")
	writeStr(w, "RECOMMENDED SETUP:\n")
	writeStr(w, "  • Intel I210/I211/I225 with Intel PROSet for full features\n")
	writeStr(w, "  • Run as Administrator for network configuration\n")
	writeStr(w, "  • Enable PowerShell script execution for some features\n\n")
	writeStr(w, "VENDOR TOOL DOWNLOADS:\n")
	writeStr(w, "  • Intel: https://downloadcenter.intel.com/\n")
	writeStr(w, "  • Broadcom: Contact your server/workstation vendor\n")
	writeStr(w, "  • Mellanox: https://network.nvidia.com/\n\n")
	writeStr(w, "See HARDWARE.md for detailed compatibility information.\n")
}
