# MediaTek Wi-Fi Adapters

⚠️ **Mixed Support** - MediaTek chipsets have improving Linux support but still lag behind Intel. Some work well, others have limitations. **Intel or Atheros preferred for reliability**.

[← Back to Home](Home)

## 📊 Compatibility Overview

### Why MediaTek is Improving

**Recent Progress:**
- `mt76` driver in mainline kernel (actively developed)
- Improving nl80211 support
- Some chipsets support monitor mode
- Better than Realtek, not as good as Intel

**Remaining Issues:**
- Monitor mode support varies by chipset
- Packet injection limited
- Signal quality accuracy inconsistent
- Some chipsets require firmware

**Recommendation:** If you already have MediaTek hardware, test it. For new purchases, **Intel AX200/210 still preferred** for reliability.

---

## 🎯 MediaTek Chipset Matrix

### MT7921 (Wi-Fi 6, Common in Laptops)
**Moderate Support**

- **Driver:** `mt76` (in-kernel since 5.12)
- **Form Factor:** M.2 2230, PCIe
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 2400 Mbps

**The Seed Compatibility:**
- Monitor Mode: ⚠️ Limited (varies by kernel version)
- Channel Switching: ✅ Good
- Signal Quality: ⚠️ Variable
- Packet Injection: ❌ Not supported

**Notes:**
- Common in budget laptops (HP, Lenovo)
- Works well for managed mode
- Monitor mode improved in kernel 5.15+
- **Test before relying on it for diagnostics**

**Tested Configurations:**
*(Community reports will be added here)*

---

### MT7922 (Wi-Fi 6E)
**Improving Support**

- **Driver:** `mt76` (in-kernel since 5.18)
- **Form Factor:** M.2 2230
- **Bands:** 2.4GHz, 5GHz, 6GHz
- **Max Speed:** 2400 Mbps

**The Seed Compatibility:**
- Monitor Mode: ⚠️ Limited (kernel 6.0+ recommended)
- Channel Switching: ✅ Good
- Signal Quality: ⚠️ Variable
- Packet Injection: ❌ Not supported
- 6GHz Support: ✅ Yes (basic, no monitor mode)

**Notes:**
- 6GHz support is recent (kernel 6.0+)
- Monitor mode on 6GHz not reliable
- **For 6GHz diagnostics, Intel AX210 strongly preferred**

**Tested Configurations:**
*(Community reports will be added here)*

---

### MT7612U (USB, Wi-Fi 5)
**Better USB Option**

- **Driver:** `mt76x2u` (in-kernel)
- **Form Factor:** USB 3.0
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 867 Mbps

**The Seed Compatibility:**
- Monitor Mode: ⚠️ Partial
- Channel Switching: ✅ Good
- Signal Quality: ✅ Good
- Packet Injection: ⚠️ Limited

**Notes:**
- Better than Realtek USB adapters
- Some monitor mode support
- **Still prefer AR9271 for injection**

**Tested Configurations:**
*(Community reports will be added here)*

---

### MT7615/MT7663 (Wi-Fi 5, Older)
**Basic Support**

- **Driver:** `mt76` (in-kernel)
- **Form Factor:** M.2, PCIe
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 1733 Mbps

**The Seed Compatibility:**
- Monitor Mode: ⚠️ Limited
- Channel Switching: ✅ Good
- Signal Quality: ⚠️ Variable
- Packet Injection: ❌ No

**Tested Configurations:**
*(Community reports will be added here)*

---

## 💻 Installation & Setup

### Verify MediaTek Chipset

```bash
lspci | grep -i mediatek
# Example: MediaTek Inc. MT7921 802.11ax

lsusb | grep -i mediatek
# For USB adapters
```

### Check Driver Status

```bash
lsmod | grep mt76
# Should show:
# mt7921e (or mt7922e, mt76x2u, etc.)
# mt76_connac_lib
# mt76
```

### Install Firmware (if needed)

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install linux-firmware

# Arch Linux
sudo pacman -S linux-firmware

# Verify firmware loaded
dmesg | grep -i mediatek
```

### Test Monitor Mode

```bash
# Check if supported
iw list | grep -A 10 "Supported interface modes"
# Look for: * monitor

# Attempt to enable
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up

# Verify
iw dev wlan0 info
```

**Expected Result:**
- MT7921: May work on kernel 5.15+, unstable on older kernels
- MT7922: Limited, especially on 6GHz
- MT7612U: Partial support

---

## 🔧 Improving Monitor Mode Support

### Update to Latest Kernel

MediaTek support improves with each kernel version:

```bash
# Check current kernel
uname -r

# Ubuntu: Use HWE kernel for latest version
sudo apt install linux-generic-hwe-22.04

# Or use mainline kernel (advanced)
# https://kernel.ubuntu.com/~kernel-ppa/mainline/
```

**Recommended Kernels:**
- MT7921: 5.15+ (better with 6.0+)
- MT7922: 6.0+ (6.1+ preferred)

### Update Firmware

```bash
# Get latest linux-firmware
cd /tmp
git clone git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git
sudo cp -r linux-firmware/mediatek/* /lib/firmware/mediatek/
sudo update-initramfs -u
```

---

## 📊 Performance Characteristics

### MT7921 (Wi-Fi 6)
- **Monitor Mode Stability:** ⚠️ Variable (depends on kernel)
- **Channel Switching:** Good (<500ms)
- **Signal Quality:** ±5 dBm accuracy (worse than Intel)
- **Managed Mode:** Excellent

### MT7922 (Wi-Fi 6E)
- **Monitor Mode Stability:** ⚠️ Poor (especially 6GHz)
- **Channel Switching:** Good
- **Signal Quality:** ±5 dBm accuracy
- **6GHz Managed Mode:** Good

### MT7612U (USB)
- **Monitor Mode Stability:** ⚠️ Moderate
- **Channel Switching:** Good
- **Signal Quality:** ±4 dBm accuracy
- **USB Bandwidth:** Sufficient for diagnostics

---

## 🐛 Known Issues

### Issue: Monitor Mode Fails on MT7921

**Error:**
```
command failed: Operation not supported (-95)
```

**Solution:**
```bash
# Update to kernel 5.15 or newer
uname -r  # Check current version

# If kernel <5.15, upgrade
sudo apt install linux-generic-hwe-22.04
sudo reboot
```

### Issue: Firmware Loading Failure

**Error:**
```
mt7921e: Failed to load firmware
mt7921e: Download BIN timeout
```

**Solution:**
```bash
# Reinstall firmware
sudo apt install --reinstall linux-firmware

# Or get latest from git (see above)
```

### Issue: Inconsistent Signal Quality

**Symptom:** Signal strength jumps ±10 dBm

**Cause:** Driver reporting inconsistency

**Workaround:** Average readings over time, or use external Intel adapter for accurate measurements

---

## 🛒 Should You Buy MediaTek Hardware?

### ⚠️ **Maybe** - For Budget Builds

**When MediaTek makes sense:**
- Budget constraint (often cheaper than Intel)
- Managed mode only (no diagnostics)
- Latest kernel available (6.0+)

**When to choose Intel instead:**
- Need reliable monitor mode
- Site surveys and diagnostics
- 6GHz with monitor mode
- Packet injection
- Professional/production use

**Price Comparison:**
- MediaTek MT7921: $12-18 USD
- Intel AX200: $15-25 USD
- **Difference:** $3-7 → **Choose Intel for $7 more**

### ✅ **Yes** - If You Already Own It

If your laptop has MediaTek Wi-Fi:
1. Update to latest kernel (6.0+)
2. Test monitor mode capability
3. Works? Great! Doesn't work? Buy external USB adapter

---

## 📝 Community Test Reports

### Working Configurations
*(Community will report successful setups here)*

### Known Issues
*(Community will report problems here)*

### Submit Your Report
[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Documentation
- [mt76 Driver Wiki](https://wireless.wiki.kernel.org/en/users/drivers/mediatek)
- [Linux Wireless mt76](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/drivers/net/wireless/mediatek/mt76)

### Firmware
- [linux-firmware MediaTek](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git/tree/mediatek)

### Community Guides
- [Arch Linux MediaTek](https://wiki.archlinux.org/title/Network_configuration/Wireless#mt76)

---

**Last Updated:** 2025-12-14
**Recommendation:** Intel AX200/210 preferred, but MediaTek acceptable for budget builds on recent kernels.

[← Back to Home](Home) | [← Previous: Realtek](Realtek-WiFi) | [Next: Intel Ethernet →](Intel-Ethernet)
