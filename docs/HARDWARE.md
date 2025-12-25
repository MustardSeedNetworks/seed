# The Seed Hardware Compatibility Guide

> **Version:** 0.13.0 **Last Updated:** December 2025 **Status:** Living Document

## Overview

The Seed's advanced diagnostic capabilities depend heavily on hardware support. This guide helps you select compatible
network adapters for optimal functionality.

## Quick Reference

| Feature                     | Hardware Dependency        | Recommended                       |
| --------------------------- | -------------------------- | --------------------------------- |
| **Basic Diagnostics**       | Any NIC                    | Any Ethernet adapter              |
| **Wi-Fi Site Surveys**      | nl80211-compatible chipset | Intel AX200/AX210                 |
| **Cable Diagnostics (TDR)** | Driver with TDR support    | Intel I350/I210, Broadcom BCM5719 |
| **LLDP/CDP Capture**        | Raw packet capture (pcap)  | Any with libpcap support          |
| **SNMP Discovery**          | Network connectivity       | Any NIC                           |

---

## Wi-Fi Adapters & Chipsets

### 🎯 Recommended (Excellent Support)

#### Intel Wi-Fi 6/6E

- **Intel AX200** (Wi-Fi 6)
  - Driver: `iwlwifi`
  - nl80211: ✅ Full support
  - Monitor Mode: ✅ Yes
  - Injection: ✅ Yes (with patches)
  - Channels: 2.4GHz, 5GHz
  - Use Case: **Best for site surveys and diagnostics**
  - Available: M.2 2230 (Laptop), M.2 E-key

- **Intel AX210** (Wi-Fi 6E)
  - Driver: `iwlwifi`
  - nl80211: ✅ Full support
  - Monitor Mode: ✅ Yes
  - Channels: 2.4GHz, 5GHz, 6GHz
  - Use Case: **Future-proof, 6GHz support**
  - Available: M.2 2230

- **Intel AX211** (Wi-Fi 6E)
  - Similar to AX210, newer revision
  - Better power efficiency
  - Available: M.2 2230

#### Qualcomm Atheros

- **Qualcomm QCNFA765** (Wi-Fi 6E)
  - Driver: `ath11k`
  - nl80211: ✅ Good support
  - Monitor Mode: ✅ Yes
  - Channels: 2.4GHz, 5GHz, 6GHz
  - Use Case: Good alternative to Intel

- **Atheros AR9271** (Wi-Fi 4)
  - Driver: `ath9k_htc`
  - nl80211: ✅ Excellent support
  - Monitor Mode: ✅ Yes, native
  - Injection: ✅ Excellent
  - Use Case: **Best for packet injection/monitoring**
  - Available: USB dongles
  - Note: Older standard but rock-solid for diagnostics

### ⚠️ Limited Support

#### Broadcom

- **BCM43xx series**
  - Driver: `brcmfmac` (open-source) or `wl` (proprietary)
  - nl80211: ⚠️ Partial (driver-dependent)
  - Monitor Mode: ⚠️ Limited (wl driver doesn't support)
  - Use Case: Works for basic Wi-Fi status
  - **Recommendation:** Avoid for advanced diagnostics

#### Realtek

- **RTL8812AU/RTL8814AU**
  - Driver: `rtl88xxau` (out-of-tree)
  - nl80211: ⚠️ Depends on driver version
  - Monitor Mode: ✅ Yes (with correct driver)
  - Use Case: Budget option, requires manual driver installation
  - **Recommendation:** OK for basic use, not ideal

#### MediaTek

- **MT7921/MT7922** (Wi-Fi 6/6E)
  - Driver: `mt76`
  - nl80211: ✅ Good support (improving)
  - Monitor Mode: ⚠️ Limited
  - Use Case: Good for connectivity, limited for diagnostics
  - **Recommendation:** Acceptable, but Intel preferred

### ❌ Not Recommended

#### Apple Silicon WiFi

- **Built-in M1/M2/M3 Wi-Fi**
  - Driver: Proprietary (macOS only)
  - nl80211: ❌ No Linux support
  - Monitor Mode: ❌ macOS only (limited)
  - **Recommendation:** Use external adapter on Apple Silicon

### Wi-Fi Capabilities Matrix

| Chipset           | Monitor Mode  | Injection     | Channel Switching | Signal Quality | Price |
| ----------------- | ------------- | ------------- | ----------------- | -------------- | ----- |
| Intel AX200/210   | ✅ Yes        | ⚠️ Limited    | ✅ Fast           | ✅ Excellent   | $$$   |
| Atheros AR9271    | ✅ Native     | ✅ Excellent  | ✅ Fast           | ✅ Good        | $     |
| Qualcomm QCNFA765 | ✅ Yes        | ⚠️ Limited    | ✅ Good           | ✅ Excellent   | $$$   |
| Broadcom BCM43xx  | ❌ No\*       | ❌ No         | ⚠️ Slow           | ⚠️ Variable    | $$    |
| Realtek RTL88xx   | ⚠️ Driver-dep | ⚠️ Driver-dep | ⚠️ OK             | ⚠️ OK          | $     |
| MediaTek MT7921   | ⚠️ Limited    | ❌ No         | ✅ Good           | ✅ Good        | $$    |

\*Broadcom: Only with `brcmfmac` driver, not `wl` proprietary driver

---

## Ethernet Adapters for Cable Diagnostics (TDR)

### What is TDR?

Time Domain Reflectometry (TDR) tests cable quality by sending electrical signals and analyzing reflections. It can
detect:

- Cable length
- Opens (disconnected cables)
- Shorts
- Impedance mismatches
- Approximate distance to fault

### 🎯 Recommended (TDR Support)

#### Intel Server NICs

- **Intel I350 Gigabit** (Quad-port)
  - Driver: `igb`
  - TDR Support: ✅ Yes (via ethtool)
  - Interface: PCIe x4
  - Use Case: **Best overall, proven reliability**
  - Form Factor: Standard PCIe, OCP, M.2
  - Price: $$$$

- **Intel I210 Gigabit**
  - Driver: `igb`
  - TDR Support: ✅ Yes (via ethtool)
  - Interface: PCIe x1
  - Use Case: Single-port option
  - Form Factor: Standard PCIe, M.2
  - Price: $$

- **Intel I225-V 2.5 Gigabit**
  - Driver: `igc`
  - TDR Support: ✅ Yes (via ethtool)
  - Interface: PCIe x1
  - Use Case: Modern 2.5GbE, desktop motherboards
  - Price: $$$

#### Broadcom Server NICs

- **BCM5719 Gigabit** (Quad-port)
  - Driver: `tg3`
  - TDR Support: ✅ Yes (via ethtool)
  - Interface: PCIe x4
  - Use Case: Alternative to Intel
  - Price: $$$$

- **BCM5720 Gigabit** (Dual-port)
  - Driver: `tg3`
  - TDR Support: ✅ Yes
  - Use Case: Dell/HP servers
  - Price: $$$

#### Marvell

- **Marvell 88E1512** (PHY)
  - Driver: `marvell` PHY driver
  - TDR Support: ✅ Yes (limited)
  - Note: PHY-level support, implementation varies
  - Use Case: Embedded systems
  - Price: $$

### ⚠️ Limited/No TDR Support

#### Consumer NICs

- **Realtek RTL8111/RTL8125**
  - Driver: `r8169`
  - TDR Support: ❌ No
  - Use Case: Budget consumer motherboards
  - **Note:** Most common NIC, but no cable diagnostics

- **Intel I219/I225 (Consumer)**
  - Driver: `e1000e` / `igc`
  - TDR Support: ⚠️ Varies by SKU
  - Use Case: Consumer desktop motherboards
  - **Note:** Some variants support TDR, most don't

- **USB Ethernet Adapters**
  - Driver: Various (`asix`, `ax88179_178a`, `r8152`)
  - TDR Support: ❌ No
  - Use Case: Convenience only
  - **Note:** No TDR support via USB

### Testing TDR Support

To verify if your NIC supports TDR on Linux:

````bash
# Check if ethtool supports cable test
sudo ethtool --cable-test eth0

# If supported, you'll see:
# Cable test started for eth0.
# Cable test completed for eth0.

# If not supported:
# Operation not supported
```python

### TDR Capabilities Matrix

| NIC Model           | TDR Support | Length Detection | Fault Location | Driver  | Price |
| ------------------- | ----------- | ---------------- | -------------- | ------- | ----- |
| Intel I350          | ✅ Full     | ✅ Yes           | ✅ Yes         | igb     | $$$$  |
| Intel I210          | ✅ Full     | ✅ Yes           | ✅ Yes         | igb     | $$    |
| Intel I225-V        | ✅ Full     | ✅ Yes           | ✅ Yes         | igc     | $$$   |
| Broadcom BCM5719/20 | ✅ Full     | ✅ Yes           | ✅ Yes         | tg3     | $$$$  |
| Realtek RTL8111     | ❌ None     | ❌ No            | ❌ No          | r8169   | $     |
| Marvell 88E1512     | ⚠️ Limited  | ⚠️ Basic         | ❌ No          | marvell | $$    |

---

## Platform-Specific Notes

### Linux (Recommended Platform)

- **Wi-Fi:** Full nl80211 support for most chipsets
- **Cable:** TDR via ethtool (kernel 5.10+)
- **Required:**
  - `libpcap-dev` for packet capture
  - `iw` and `wireless-tools` for Wi-Fi
  - `ethtool` for cable diagnostics
- **Kernel:** 5.10+ recommended (TDR support improved)

### macOS (Limited)

- **Wi-Fi:** Uses native APIs, no nl80211
  - Built-in adapters: Basic status only
  - External adapters: Limited driver support
- **Cable:** No TDR support (no ethtool)
- **Recommendation:** Use external Intel/Atheros USB adapter for Wi-Fi diagnostics

### Windows (Not Supported)

- The Seed is Linux-only
- Consider running in WSL2 with USB passthrough (limited functionality)

---

## Recommended Hardware Bundles

### 🏆 Professional Kit (Best Overall)

- **NIC:** Intel I350-T4 (Quad Gigabit PCIe)
- **Wi-Fi:** Intel AX210 (M.2 2230)
- **Platform:** Ubuntu 22.04 LTS or newer
- **Use Case:** Full-featured diagnostics, site surveys, cable testing
- **Price:** ~$300-400

### 💼 Technician Kit (Balanced)

- **NIC:** Intel I210 (Single Gigabit PCIe)
- **Wi-Fi:** Intel AX200 (M.2 2230)
- **Platform:** Ubuntu 22.04 LTS
- **Use Case:** Most features, good value
- **Price:** ~$80-120

### 💰 Budget Kit (Basic)

- **NIC:** Onboard (Realtek - no TDR)
- **Wi-Fi:** Atheros AR9271 USB dongle
- **Platform:** Raspberry Pi 4 with Ubuntu
- **Use Case:** Basic diagnostics, no cable testing
- **Price:** ~$15-30

### 🔬 Research/Pentesting Kit

- **NIC:** Intel I350-T4
- **Wi-Fi:** Atheros AR9271 (injection) + Intel AX210 (connectivity)
- **Platform:** Kali Linux or Ubuntu
- **Use Case:** Packet injection, monitor mode, full diagnostics
- **Price:** ~$300-350

---

## Future Considerations

### Emerging Technologies

- **Wi-Fi 7 (802.11be):** Support expected in 2026+
  - Intel BE200 series (when available)
  - Monitor mode support TBD

- **10 Gigabit Ethernet:**
  - Intel X540/X550 series
  - TDR support varies by model
  - Currently expensive ($200+)

- **2.5/5 Gigabit:**
  - Intel I225/I226 (improving support)
  - Realtek RTL8125 (no TDR)

### Tested Configurations

We maintain a list of tested hardware at: **https://github.com/krisarmstrong/seed/wiki/Tested-Hardware**

Please contribute your test results!

---

## FAQ

### Q: Will my built-in laptop Wi-Fi work?

**A:** For basic connectivity status, yes. For advanced diagnostics (monitor mode, site surveys), it depends on the
chipset. Check `lspci | grep -i wireless` and compare to the compatibility table above.

### Q: Why doesn't my Realtek NIC support cable testing?

**A:** TDR requires specialized PHY hardware and driver support. Consumer Realtek chips prioritize cost over advanced
diagnostics. Use Intel server NICs for TDR.

### Q: Can I use USB adapters?

#### A

- **Wi-Fi USB:** Yes, Atheros AR9271 dongles work great
- **Ethernet USB:** Works for basic diagnostics, but no TDR support

### Q: Does Apple Silicon Mac support The Seed?

**A:** Not natively. The built-in Wi-Fi chip isn't accessible from Linux. Options:

1. Run in VM with external USB Wi-Fi adapter
2. Use Intel Mac instead
3. Run on separate Linux hardware

### Q: What about Bluetooth adapters for diagnostics?

**A:** Bluetooth diagnostics are planned for v2.0. Any Bluetooth 5.0+ adapter should work when implemented.

---

## Contributing Hardware Data

Found a chipset that works (or doesn't)? Help the community:

1. Test your hardware with The Seed
2. Note capabilities (monitor mode, TDR, etc.)
3. Submit to: https://github.com/krisarmstrong/seed/wiki/Tested-Hardware
4. Include:
   - Chipset model (`lspci` or `lsusb` output)
   - Driver version
   - Kernel version
   - What works / doesn't work

---

## References

- [Linux Wireless Drivers](https://wireless.wiki.kernel.org/en/users/drivers)
- [Intel Ethernet Support](https://www.intel.com/content/www/us/en/support/articles/000005584/ethernet-products.html)
- [ethtool Documentation](https://mirrors.edge.kernel.org/pub/software/network/ethtool/)
- [nl80211 API](https://wireless.wiki.kernel.org/en/developers/documentation/nl80211)

---

**Document Maintenance:** This document should be reviewed quarterly and updated when:

- New chipsets are released
- Driver support changes significantly
- User testing reveals new compatibility information
````
