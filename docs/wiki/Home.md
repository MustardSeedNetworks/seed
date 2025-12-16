# Welcome to The Seed Wiki

**The Seed** is an AI-powered network diagnostic platform that combines WiFi planning, network monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

## 🚀 Getting Started

📥 **Installation**
- [Installation on macOS](Installation-macOS.md)
- [Installation on Linux](Installation-Linux.md)
- [Quick Start Guide](Quick-Start-Guide.md) - Your first scan in 5 minutes

📚 **Features**
- [Network Discovery](Network-Discovery.md) - Find all devices on your network
- WiFi Survey & Planning (coming soon)
- Vulnerability Scanning (coming soon)
- Compliance Reporting (coming soon)

❓ **Help**
- [FAQ](FAQ.md) - Common questions answered
- [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
- [GitHub Issues](https://github.com/krisarmstrong/netscope/issues)

---

## 🔌 Hardware Compatibility

This community-driven resource documents tested hardware for network diagnostics.

## 📋 Quick Navigation

### Wi-Fi Adapters
- [Intel Wi-Fi Adapters](Intel-WiFi)
- [Qualcomm Atheros Adapters](Qualcomm-Atheros-WiFi)
- [Broadcom Wi-Fi Adapters](Broadcom-WiFi)
- [Realtek Wi-Fi Adapters](Realtek-WiFi)
- [MediaTek Wi-Fi Adapters](MediaTek-WiFi)

### Ethernet NICs
- [Intel Ethernet NICs](Intel-Ethernet)
- [Broadcom Ethernet NICs](Broadcom-Ethernet)
- [Realtek Ethernet NICs](Realtek-Ethernet)
- [Marvell Ethernet NICs](Marvell-Ethernet)

### Testing Guides

- [DHCP Testing Environment](DHCP-Testing)

## 🎯 Purpose

This Wiki collects **real-world test results** from the The Seed community. Unlike vendor specifications, these reports show actual compatibility with The Seed's diagnostic features.

### What Makes Hardware "Compatible"?

**Wi-Fi Adapters:**
- ✅ **Monitor Mode** - Packet capture for site surveys
- ✅ **Channel Switching** - Fast hopping between channels
- ✅ **Signal Quality Reporting** - Accurate dBm readings
- ⚠️ **Packet Injection** - Advanced testing (optional)

**Ethernet NICs:**
- ✅ **TDR Cable Testing** - Detect faults, measure length
- ✅ **Link Speed Detection** - Accurate speed/duplex reporting
- ✅ **Packet Capture** - LLDP/CDP/EDP protocol support

## 📝 Contributing

### How to Submit a Report

1. **Test Your Hardware** using the automated script:
   ```bash
   sudo ./scripts/test-hardware-compatibility.sh wlan0
   # or
   sudo ./scripts/test-hardware-compatibility.sh eth0
   ```

2. **Submit Results** via GitHub:
   - [Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)
   - Fill out the automated form with your test results

3. **Maintainers will review** and add your results to the Wiki

### What to Test

**Minimum Requirements:**
- Kernel version (affects driver support)
- Driver version
- Basic functionality (connectivity, scanning)

**Wi-Fi Specific:**
- Monitor mode switching
- Channel hopping (2.4GHz: 1, 6, 11)
- Network scanning (SSID detection)
- Signal quality (dBm readings)

**Ethernet Specific:**
- TDR cable test support
- Test with connected cable (should show length)
- Test with disconnected cable (should show open/short)

## 📊 Compatibility Matrix

### Wi-Fi Quick Reference

| Chipset | Monitor Mode | Channel Switch | Signal Quality | Recommendation |
|---------|--------------|----------------|----------------|----------------|
| Intel AX200/210 | ✅ Excellent | ✅ Fast (<1s) | ✅ Accurate | **Best Choice** |
| Atheros AR9271 | ✅ Excellent | ✅ Fast (<1s) | ✅ Accurate | **Budget Option** |
| Broadcom BCM43xx | ⚠️ Limited | ⚠️ Slow (2-5s) | ⚠️ Variable | Avoid if possible |
| Realtek RTL88xx | ⚠️ Partial | ⚠️ Slow | ⚠️ Inaccurate | Not recommended |
| Apple Silicon | ❌ No | ❌ No | ❌ No | **Not Supported** |

### Ethernet Quick Reference

| NIC Model | TDR Support | Cable Length | Fault Detection | Recommendation |
|-----------|-------------|--------------|-----------------|----------------|
| Intel I350 | ✅ Full | ✅ Yes | ✅ Distance to fault | **Best Choice** |
| Intel I210/I225-V | ✅ Full | ✅ Yes | ✅ Distance to fault | **Budget Option** |
| Broadcom BCM5719/5720 | ✅ Full | ✅ Yes | ✅ Distance to fault | Server-grade |
| Realtek RTL8111 | ❌ No | ❌ No | ❌ No | Basic diagnostics only |
| Realtek RTL8125 | ❌ No | ❌ No | ❌ No | Basic diagnostics only |

## 🔍 Understanding Test Results

### Wi-Fi Capabilities

**Monitor Mode:**
- **✅ Excellent** - Switches instantly, stable operation
- **⚠️ Limited** - Works but unstable or requires workarounds
- **❌ No** - Not supported by driver/chipset

**Channel Switching:**
- **✅ Fast** - Sub-second switching, ideal for site surveys
- **⚠️ Slow** - 2-5 seconds per channel, usable but not ideal
- **❌ Unreliable** - Fails or hangs

**Signal Quality:**
- **✅ Accurate** - Consistent dBm readings matching spectrum analyzer
- **⚠️ Variable** - Readings fluctuate excessively
- **❌ Inaccurate** - Missing data or clearly wrong values

### Ethernet TDR Support

**Full TDR Support:**
- Cable OK/fault status
- Approximate cable length (±5m accuracy typical)
- Distance to fault location
- Per-pair diagnostics (4 pairs in Cat5e/6)

**Basic TDR Support:**
- Cable OK/fault status only
- No length measurement
- No fault distance

**No TDR Support:**
- `ethtool --cable-test` returns "Operation not supported"
- Driver/hardware limitation
- Cannot perform cable diagnostics

## 🛠️ Testing Tools

### Automated Script
Download and run the compatibility test script:
```bash
# Download
curl -O https://raw.githubusercontent.com/krisarmstrong/seed/main/scripts/test-hardware-compatibility.sh
chmod +x test-hardware-compatibility.sh

# Run test
sudo ./test-hardware-compatibility.sh wlan0  # Wi-Fi adapter
sudo ./test-hardware-compatibility.sh eth0   # Ethernet NIC
```

### Manual Testing

**Wi-Fi - Check nl80211 support:**
```bash
iw list | grep -A 10 "Supported interface modes"
# Should show "* monitor" for full support
```

**Wi-Fi - Test monitor mode:**
```bash
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up
iw dev wlan0 info  # Should show "type monitor"
```

**Ethernet - Test TDR:**
```bash
sudo ethtool --cable-test eth0
# Wait 10-30 seconds for results
# Should show cable status and length if supported
```

## 📚 Additional Resources

- [HARDWARE.md](https://github.com/krisarmstrong/seed/blob/main/HARDWARE.md) - Official hardware guide
- [HARDWARE_DOCUMENTATION_PLAN.md](https://github.com/krisarmstrong/seed/blob/main/docs/HARDWARE_DOCUMENTATION_PLAN.md) - Maintenance procedures
- [Submit Hardware Report](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml) - Report your test results
- [GitHub Issues](https://github.com/krisarmstrong/seed/issues?q=label%3Ahardware-report) - Browse community reports

## ⚠️ Important Notes

### Kernel Version Matters
Driver capabilities vary significantly between kernel versions. Always include your kernel version (`uname -r`) when reporting.

### Distribution Differences
Ubuntu, Arch, Fedora may have different driver packages. Specify your distribution when reporting.

### Firmware Requirements
Some adapters require specific firmware files. Missing firmware = missing features.

### USB vs PCIe
USB adapters may have reduced capabilities due to bus limitations. PCIe/M.2 preferred for professional use.

## 🏆 Community Contributors

Thank you to everyone who has tested hardware and submitted reports!

*(This section will be updated as reports are submitted)*

---

**Last Updated:** 2025-12-14
**Maintained By:** The Seed Community
**Report Issues:** [GitHub Issues](https://github.com/krisarmstrong/seed/issues)
