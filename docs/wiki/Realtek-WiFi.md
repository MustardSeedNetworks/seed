# Realtek Wi-Fi Adapters

❌ **Not Recommended** - Realtek Wi-Fi adapters have poor Linux support for monitor mode and advanced diagnostics. **Strongly recommend Intel or Atheros instead**.

[← Back to Home](Home)

## ⚠️ Why Realtek is Problematic for Diagnostics

### Driver Issues
- **Fragmented drivers:** `rtl8xxxu`, `rtw88`, `rtw89`, vendor drivers
- **Out-of-tree drivers** often required (not in mainline kernel)
- **Monitor mode:** Rarely works reliably
- **Packet injection:** Usually not supported

### Common Problems
- Kernel version sensitivity (breaks between versions)
- DKMS compilation failures after kernel updates
- Poor or no nl80211 support
- Inaccurate signal quality reporting

### Recommendation
**If you own Realtek hardware:** It may work for basic managed mode (normal Wi-Fi), but expect **no monitor mode, no site surveys, no packet injection**.

**If purchasing new hardware:** **Choose Intel AX200/210 or Atheros AR9271 instead**.

---

## 📊 Common Realtek Chipsets

### RTL8812AU/RTL8814AU (USB)
- **Driver:** Out-of-tree (aircrack-ng driver or vendor)
- **Monitor Mode:** ⚠️ Limited (with specific driver version)
- **Packet Injection:** ⚠️ Limited (with aircrack-ng driver)
- **Recommendation:** Use only if you already own one

### RTL8821CE (PCIe, common in laptops)
- **Driver:** `rtw88` (in-kernel since 5.9)
- **Monitor Mode:** ❌ Not supported
- **Packet Injection:** ❌ Not supported
- **Recommendation:** Use external USB adapter for diagnostics

### RTL8852AE/RTL8852BE (Wi-Fi 6)
- **Driver:** `rtw89` (in-kernel since 5.16)
- **Monitor Mode:** ❌ Not supported
- **Packet Injection:** ❌ Not supported
- **Recommendation:** Managed mode only

### RTL88x2BU (USB, Wi-Fi 5)
- **Driver:** Out-of-tree vendor driver
- **Monitor Mode:** ⚠️ Very limited
- **Packet Injection:** ❌ No
- **Recommendation:** Avoid for diagnostics

---

## 🛠️ If You Must Use Realtek

### Identify Your Chipset

```bash
lspci | grep -i realtek
# Example: Realtek Semiconductor Co., Ltd. RTL8821CE 802.11ac

lsusb | grep -i realtek
# For USB adapters
```

### Option 1: In-Kernel Drivers (Managed Mode Only)

```bash
# Check if driver is already loaded
lsmod | grep rtw

# For rtw88 (RTL8821CE, RTL8822CE)
sudo modprobe rtw88_8821ce

# For rtw89 (RTL8852AE, RTL8852BE)
sudo modprobe rtw89_8852ae
```

**Result:** Basic Wi-Fi connectivity, **no monitor mode**.

### Option 2: aircrack-ng Drivers (RTL8812AU only)

⚠️ **Warning:** Out-of-tree, requires DKMS, breaks often

```bash
# Install prerequisites
sudo apt install build-essential dkms git

# Clone driver
git clone https://github.com/aircrack-ng/rtl8812au.git
cd rtl8812au

# Build and install
sudo make dkms_install

# Load module
sudo modprobe 88XXau

# Test monitor mode
sudo airmon-ng start wlan0
```

**Success Rate:** ~50% depending on kernel version, chipset revision, phase of moon.

---

## 🐛 Common Issues

### Issue: Driver Won't Compile

**Error:**
```
make: *** [Makefile:1234] Error 1
fatal error: linux/something.h: No such file or directory
```

**Cause:** Kernel headers missing or incompatible driver version

**Solution:**
```bash
# Install headers
sudo apt install linux-headers-$(uname -r)

# Try different driver version
cd rtl8812au
git checkout <older-commit>
sudo make clean && sudo make dkms_install
```

### Issue: Monitor Mode Fails

**Error:**
```
command failed: Operation not supported (-95)
```

**Cause:** Driver doesn't support monitor mode

**Solution:** **Buy different hardware.** Seriously.

### Issue: Kernel Update Breaks Driver

**Symptom:** Wi-Fi stops working after `apt upgrade`

**Cause:** DKMS driver didn't rebuild for new kernel

**Solution:**
```bash
# Reinstall driver
cd rtl8812au
sudo make dkms_remove
sudo make dkms_install

# Or use previous kernel
# (Select old kernel at GRUB boot menu)
```

---

## 🛒 Should You Buy Realtek Hardware?

### ❌ **Absolutely NO** - For LuminetIQ

**Instead, buy:**
- **Wi-Fi 6:** Intel AX200 ($15-25)
- **Wi-Fi 6E:** Intel AX210 ($20-35)
- **Wi-Fi 7:** Intel BE200 ($25-40)
- **Budget/Injection:** Atheros AR9271 ($12-18)

**You will save time, frustration, and money** by buying the right hardware upfront.

### ✅ **Maybe** - If You Already Own It

If you already have Realtek hardware:
1. Test for managed mode (normal Wi-Fi) - usually works
2. Don't expect monitor mode or diagnostics
3. Buy external USB adapter for LuminetIQ diagnostics

---

## 📝 Community Test Reports

### Successful Configurations (Managed Mode)
*(Community reports will be added here)*

### Monitor Mode Attempts (Usually Failed)
*(Community reports will be added here)*

### Submit Your Report
[Create a Hardware Report Issue](https://github.com/krisarmstrong/luminetiq/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Repositories
- [rtw88 (in-kernel)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/drivers/net/wireless/realtek/rtw88)
- [rtw89 (in-kernel)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/drivers/net/wireless/realtek/rtw89)
- [aircrack-ng RTL8812AU driver](https://github.com/aircrack-ng/rtl8812au)

### Community Guides
- [Arch Linux Realtek Guide](https://wiki.archlinux.org/title/Network_configuration/Wireless#rtw88)
- [Ubuntu Realtek Troubleshooting](https://askubuntu.com/questions/tagged/realtek)

---

**Last Updated:** 2025-12-14
**Recommendation:** **Don't buy Realtek for network diagnostics.**

[← Back to Home](Home) | [← Previous: Broadcom](Broadcom-WiFi) | [Next: MediaTek →](MediaTek-WiFi)
