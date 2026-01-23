# Hardware Compatibility and Platform Support

This document describes hardware requirements and platform-specific limitations for The Seed network diagnostic tool.

## Platform Support Matrix

| Feature | Linux | macOS | Windows |
|---------|-------|-------|---------|
| Interface listing | Full | Full | Full |
| Static IP configuration | Full | Full | Full |
| DHCP configuration | Full | Full | Full |
| MTU configuration | Full | Full | Full |
| Link status monitoring | Full | Full | Full |
| Speed/duplex detection | Full | Partial | Full |
| Wi-Fi scanning | Full | Full | Full |
| Wi-Fi connect/disconnect | Full | Full | Full |
| ARP table reading | Full | Full | Full |
| IPv6 NDP discovery | Full | Full | Full |
| Bluetooth scanning | Full | Partial | Limited |
| Gateway detection | Full | Full | Full |
| DNS server detection | Full | Full | Full |
| DHCP lease info | Full | Full | Full |
| VLAN detection | Full | Partial | Limited |
| VLAN creation/deletion | Full | Limited | None |
| Cable diagnostics (TDR) | Full* | None | None |
| PHY layer info | Full | Partial | Partial |
| Digital Optical Monitoring | Full* | None | None |

**Legend:**
- **Full**: Complete feature support through standard OS APIs
- **Partial**: Limited functionality through available APIs
- **Limited**: Requires vendor-specific tools or drivers
- **None**: Not available through standard APIs

*Requires compatible hardware (see sections below)

---

## Linux

Linux provides the most comprehensive support through:
- **netlink**: Low-level kernel interface for network configuration
- **ethtool**: PHY layer and driver statistics
- **sysfs**: Direct access to network driver information
- **iw/nl80211**: Modern Wi-Fi configuration

### Recommended Hardware

**Network Interface Cards (NICs):**
- Intel I210/I211 - Best for cable diagnostics (TDR)
- Intel I350 - Server-grade, full ethtool support
- Intel I225-V/LM - 2.5GbE with full feature support
- Broadcom BCM5719/5720 - Enterprise features
- Mellanox ConnectX series - High performance

**Wi-Fi Adapters:**
- Intel AX200/AX210 - Full nl80211 support
- Atheros-based adapters - Good Linux driver support
- MediaTek MT7921 - Good modern support

**SFP/SFP+ for DOM (Digital Optical Monitoring):**
- Intel X520/X540 - Full DOM support
- Mellanox ConnectX-3/4/5 - Comprehensive diagnostics
- Broadcom 57810 - Enterprise DOM support

---

## macOS

macOS provides network functionality through:
- **networksetup**: Command-line network configuration
- **CoreWLAN**: Wi-Fi framework
- **System Configuration**: Network state monitoring

### Limitations

1. **No ethtool equivalent**: PHY layer access is limited
2. **No TDR cable testing**: Not exposed through any standard API
3. **No DOM support**: SFP diagnostics require vendor tools
4. **VLAN limitations**: Can only configure through Network Preferences
5. **Bluetooth limitations**: System Bluetooth API may restrict scanning

### Recommended Hardware

Standard Mac hardware with:
- Built-in Wi-Fi (Airport)
- Thunderbolt/USB Ethernet adapters (Intel-based preferred)

For advanced features, consider:
- Intel-based USB 3.0 Ethernet adapters
- Sonnet Thunderbolt adapters (for server-grade NICs)

---

## Windows

Windows provides network functionality through:
- **PowerShell Get-NetAdapter**: Modern network management
- **WMI (Win32_NetworkAdapter)**: Legacy interface information
- **netsh**: Command-line network configuration
- **WLAN API**: Wi-Fi management

### Limitations

1. **No standard TDR API**: Cable diagnostics require vendor tools
2. **No DOM API**: SFP diagnostics require vendor software
3. **VLAN limitations**: Requires vendor-specific tools or Hyper-V
4. **Bluetooth limitations**: Requires Windows.Devices.Bluetooth API or vendor SDK
5. **PHY access limited**: No equivalent to Linux ethtool

### Vendor-Specific Tools

For advanced features on Windows, use these vendor tools:

**Intel NICs:**
- [Intel PROSet/Wireless Software](https://www.intel.com/content/www/us/en/support/products/36773/ethernet-products.html)
- Provides: Cable diagnostics, VLAN configuration, advanced settings

**Broadcom NICs:**
- [Broadcom Advanced Control Suite (BACS)](https://www.broadcom.com/)
- Provides: Cable diagnostics, VLAN configuration, team/bond setup

**Marvell NICs:**
- Marvell Yukon Device Manager
- Provides: Cable diagnostics, power management

**Mellanox NICs:**
- [NVIDIA/Mellanox WinOF](https://network.nvidia.com/)
- Provides: RDMA, advanced configuration

### Recommended Hardware

**For full feature support:**
- Intel I210/I211/I225 with Intel PROSet
- Intel X520/X540 for 10GbE with SFP+
- Broadcom BCM5719/5720 with BACS

**For basic operation:**
- Any Windows-compatible Ethernet adapter
- Intel or Realtek Wi-Fi adapters

---

## Feature-Specific Requirements

### Cable Diagnostics (TDR)

Time Domain Reflectometry requires:

| Platform | Requirement |
|----------|-------------|
| Linux | ethtool-compatible NIC (Intel, Broadcom, Marvell) |
| macOS | Not available |
| Windows | Vendor tools (Intel PROSet, BACS) |

**Supported NICs for TDR:**
- Intel I210, I211, I350, I225
- Broadcom BCM5719, BCM5720, BCM57810
- Marvell Yukon 88E8056, 88E8053

### Digital Optical Monitoring (DOM)

SFP/SFP+ diagnostics require:

| Platform | Requirement |
|----------|-------------|
| Linux | ethtool + compatible SFP module |
| macOS | Not available |
| Windows | Vendor tools (Intel PROSet, WinOF) |

**DOM Parameters:**
- Temperature
- Voltage
- TX/RX Power (optical)
- Laser bias current
- Alarm/warning thresholds

### VLAN Configuration

802.1Q VLAN support:

| Platform | Detection | Creation |
|----------|-----------|----------|
| Linux | Full (ip link, bridge) | Full |
| macOS | Via networksetup | Network Preferences only |
| Windows | PowerShell (if supported) | Vendor tools only |

---

## Troubleshooting

### "Cable diagnostics not supported"

**Linux:** Ensure your NIC driver supports ethtool cable test:
```bash
ethtool --show-features eth0 | grep -i cable
```

**Windows:** Install vendor management software:
- Intel NICs: Download Intel PROSet from Intel Download Center
- Broadcom NICs: Install BACS from your server/workstation vendor

### "VLAN creation failed"

**Linux:** Ensure 8021q module is loaded:
```bash
sudo modprobe 8021q
```

**Windows:** Use vendor tools or Hyper-V Virtual Switch Manager for VLAN support.

### "Bluetooth scanning limited"

**Linux:** Ensure BlueZ is installed and bluetoothd is running:
```bash
sudo systemctl status bluetooth
```

**macOS:** Grant Bluetooth permissions to Terminal/application.

**Windows:** Bluetooth scanning requires elevated privileges and compatible Bluetooth adapter.

---

## Performance Considerations

### Link Monitoring Latency

| Platform | Typical Detection Time |
|----------|----------------------|
| Linux (netlink) | <100ms |
| macOS (SCDynamicStore) | 100-500ms |
| Windows (PowerShell) | 500-1000ms |

### Speed Detection Accuracy

| Platform | Method | Accuracy |
|----------|--------|----------|
| Linux | ethtool/sysfs | High |
| macOS | networksetup | Medium |
| Windows | PowerShell/WMI | High |

---

## Getting Help

For hardware compatibility questions:
1. Check your NIC vendor documentation
2. Verify driver version and updates
3. Test with `seed diagnose` command
4. Report issues at: https://github.com/mustardseednetworks/seed/issues

For platform-specific guidance, use:
```bash
seed help --platform
```
