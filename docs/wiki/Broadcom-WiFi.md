# Broadcom Wi-Fi Adapters

⚠️ **Limited Support** - Broadcom adapters have varying levels of Linux support. Some models work well, others have significant limitations. **Not recommended** as a first choice for The Seed.

[← Back to Home](Home)

## ⚠️ Compatibility Overview

### Why Broadcom is Problematic

**Driver Fragmentation:**
- Multiple competing drivers: `bcm43xx`, `b43`, `brcmfmac`, `wl` (proprietary)
- Different chipsets require different drivers
- Firmware often required (non-free)

**Monitor Mode Support:**
- ⚠️ **Limited** - Varies significantly by chipset
- Some chipsets: No monitor mode at all
- Others: Works but unstable

**Packet Injection:**
- ❌ **Not supported** on most chipsets
- Some community patches exist but unreliable

**Recommendation:** If you already have Broadcom hardware, test it. If purchasing new hardware for diagnostics, **choose Intel or Atheros instead**.

---

## 📊 Chipset Compatibility Matrix

| Chipset | Driver | Monitor Mode | Signal Quality | Recommendation |
|---------|--------|--------------|----------------|----------------|
| BCM4360 | brcmfmac | ⚠️ Limited | ⚠️ Variable | Avoid |
| BCM43142 | wl (proprietary) | ❌ No | ✅ Good | Managed mode only |
| BCM4352 | brcmfmac | ⚠️ Limited | ⚠️ Variable | Avoid |
| BCM43602 | brcmfmac | ⚠️ Limited | ⚠️ Variable | Avoid |

---

## 🛠️ Driver Installation

### Identify Your Chipset

```bash
lspci | grep -i broadcom
# Example output:
# 03:00.0 Network controller: Broadcom Inc. BCM4360 802.11ac

lsusb | grep -i broadcom
# For USB adapters
```

### Install brcmfmac (Open-Source Driver)

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install firmware-b43-installer

# Arch Linux
sudo pacman -S broadcom-wl

# Verify
lsmod | grep brcmfmac
```

### Install wl (Proprietary Driver)

⚠️ **Warning:** Proprietary driver does NOT support monitor mode

```bash
# Ubuntu/Debian
sudo apt install broadcom-sta-dkms

# Arch Linux
yay -S broadcom-wl-dkms

# Blacklist conflicting drivers
sudo tee /etc/modprobe.d/blacklist-broadcom.conf <<EOF
blacklist b43
blacklist b43legacy
blacklist ssb
blacklist bcm43xx
blacklist brcmfmac
blacklist brcmsmac
EOF

# Reboot
sudo reboot
```

---

## 🔧 Testing Monitor Mode

### Check if Monitor Mode is Supported

```bash
iw list | grep -A 10 "Supported interface modes"
# Look for: * monitor
```

### Attempt to Enable Monitor Mode

```bash
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up

# Verify
iw dev wlan0 info
```

**Expected Results:**
- BCM4360, BCM4352: May work but unstable
- BCM43142 (wl driver): Will fail with "Operation not supported"

---

## 🐛 Common Issues

### Issue: Monitor Mode Fails

**Error:**
```
command failed: Operation not supported (-95)
```

**Cause:** Chipset/driver combination doesn't support monitor mode

**Solution:**
- Try switching to `brcmfmac` driver if using `wl`
- If still fails, Broadcom chipset doesn't support monitor mode
- **Recommendation:** Use external USB adapter (Intel/Atheros)

### Issue: Firmware Missing

**Error:**
```
brcmfmac: brcmf_c_preinit_dcmds: Firmware version = wl0: Nov 10 2015 06:38:10 version 7.35.180.80 FWID 01-abcdef12
brcmfmac: brcmf_c_preinit_dcmds: Failed to get revision info: -110
```

**Solution:**
```bash
sudo apt install firmware-brcm80211
sudo modprobe -r brcmfmac
sudo modprobe brcmfmac
```

### Issue: Slow Channel Switching

**Symptom:** 2-5 seconds per channel change

**Cause:** Driver limitation

**Workaround:** None - inherent to Broadcom drivers

---

## 💻 Broadcom in MacBooks

### Apple-Broadcom Chipsets

Many MacBooks use Broadcom Wi-Fi with custom Apple firmware:
- BCM43602 (MacBook Pro 2015-2017)
- BCM4364 (MacBook Air 2018-2019)

**Linux Support:**
- ⚠️ **Varies significantly**
- Usually works for managed mode (normal Wi-Fi)
- **Monitor mode:** Often doesn't work
- Requires `brcmfmac` driver

**Recommendation for MacBook Users:**
- Don't expect full diagnostics capabilities
- Use external USB adapter (AR9271, Intel AX200-based USB)

---

## 🛒 Should You Buy Broadcom Hardware?

### ❌ **NO** - For New Purchases

If buying hardware specifically for The Seed:
- **Choose:** Intel AX200/210 or Atheros AR9271
- **Avoid:** Broadcom-based adapters

### ✅ **Maybe** - If You Already Own It

If you already have Broadcom hardware:
1. Test with the compatibility script:
   ```bash
   sudo ./scripts/test-hardware-compatibility.sh wlan0
   ```

2. If monitor mode works: Use it for basic diagnostics
3. If monitor mode fails: Use for managed mode only, get external adapter for advanced features

---

## 📝 Community Test Reports

### Known Working Configurations
*(Community will add successful configs here)*

### Known Problematic Configurations
*(Community will add failed attempts here)*

### Submit Your Report
[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Documentation
- [brcm80211 Wireless Wiki](https://wireless.wiki.kernel.org/en/users/drivers/brcm80211)
- [Broadcom Linux Support](https://wiki.debian.org/wl)

### Firmware
- [linux-firmware Repository](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git/tree/brcm)

### Community Guides
- [Arch Linux Broadcom Guide](https://wiki.archlinux.org/title/Broadcom_wireless)
- [Ubuntu Broadcom Guide](https://help.ubuntu.com/community/WifiDocs/Driver/bcm43xx)

---

**Last Updated:** 2025-12-14
**Total Community Reports:** 0 *(submit yours!)*

[← Back to Home](Home) | [← Previous: Qualcomm Atheros](Qualcomm-Atheros-WiFi) | [Next: Realtek →](Realtek-WiFi)
