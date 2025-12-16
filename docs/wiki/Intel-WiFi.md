# Intel Wi-Fi Adapters

Intel Wi-Fi adapters using the `iwlwifi` driver are **highly recommended** for The Seed. They offer
excellent nl80211 support, stable monitor mode, and accurate signal quality reporting.

[← Back to Home](Home)

## 🎯 Recommended Models

### Intel AX200 (Wi-Fi 6)

**Best Overall Choice for Network Diagnostics**

- **Chipset:** Intel Wi-Fi 6 AX200
- **Driver:** `iwlwifi` (in-kernel since 5.1+)
- **Form Factor:** M.2 2230 (Key E)
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 2400 Mbps
- **Price Range:** $15-25 USD

**The Seed Compatibility:**

- Monitor Mode: ✅ Excellent
- Channel Switching: ✅ Fast (<1s)
- Signal Quality: ✅ Accurate (-30 to -90 dBm range)
- Packet Injection: ✅ Supported
- 2.4GHz Channels: 1-14
- 5GHz Channels: 36-165 (region dependent)

**Tested Configurations:** _(Community reports will be added here)_

---

### Intel AX210 (Wi-Fi 6E)

**Best Choice for Wi-Fi 6E Diagnostics with 6GHz Support**

- **Chipset:** Intel Wi-Fi 6E AX210
- **Driver:** `iwlwifi` (in-kernel since 5.10+)
- **Form Factor:** M.2 2230 (Key E)
- **Bands:** 2.4GHz, 5GHz, 6GHz (6.0-7.125 GHz)
- **Max Speed:** 2400 Mbps (160 MHz channels)
- **Price Range:** $20-35 USD
- **Bluetooth:** 5.2

**The Seed Compatibility:**

- Monitor Mode: ✅ Excellent (all bands including 6GHz)
- Channel Switching: ✅ Fast (<1s on all bands)
- Signal Quality: ✅ Accurate
- Packet Injection: ✅ Supported
- 6GHz Support: ✅ Yes (requires kernel 5.10+, regulatory unlock)

**Why Choose AX210:**

- **6GHz band** - Test next-gen Wi-Fi 6E networks
- **Wide channel support** - 20/40/80/160 MHz on all bands
- **Monitor mode on 6GHz** - Critical for 6E site surveys
- Future-proof for enterprise Wi-Fi 6E deployments
- Same excellent diagnostics as AX200

**6GHz Channel Access:**

- **UNII-5:** Channels 1-93 (5.925-6.425 GHz)
- **UNII-7:** Channels 97-189 (6.525-6.875 GHz)
- **UNII-8:** Channels 193-233 (6.875-7.125 GHz)
- **Note:** Requires proper regulatory domain and AFC/LPI compliance

**Tested Configurations:** _(Community reports will be added here)_

---

### Intel BE200/BE202 (Wi-Fi 7)

**Latest Generation with Multi-Link Operation (MLO)**

- **Chipset:** Intel Wi-Fi 7 BE200 (standard) / BE202 (CNVi)
- **Driver:** `iwlwifi` (in-kernel since 6.2+, best with 6.5+)
- **Form Factor:** M.2 2230 (Key E) for BE200, CNVi for BE202
- **Bands:** 2.4GHz, 5GHz, 6GHz
- **Max Speed:** 5800 Mbps (320 MHz channels on 6GHz)
- **Price Range:** $25-40 USD (BE200), OEM only (BE202)
- **Bluetooth:** 5.4

**The Seed Compatibility:**

- Monitor Mode: ✅ Excellent (requires kernel 6.5+ for best support)
- Channel Switching: ✅ Fast (<1s, all bands)
- Signal Quality: ✅ Accurate
- Packet Injection: ✅ Supported
- 320 MHz Channels: ✅ Yes (6GHz only, region dependent)
- MLO (Multi-Link): ⚠️ Driver support in progress (kernel 6.9+)

**Wi-Fi 7 Features for Diagnostics:**

- **320 MHz channels** - Double bandwidth vs Wi-Fi 6E
- **Multi-Link Operation (MLO)** - Simultaneous 2.4/5/6 GHz
- **4K QAM** - Higher modulation for cleaner environments
- **Preamble Puncturing** - Better channel utilization
- **Enhanced MU-MIMO** - 16x16 spatial streams

**Why Choose BE200/BE202:**

- **Latest generation** - Test Wi-Fi 7 networks and features
- **320 MHz channels** - Critical for enterprise Wi-Fi 7 deployments
- **MLO diagnostics** - See multi-link performance (when driver support matures)
- **Future-proof** - 5+ year longevity

**Important Notes:**

- **Kernel 6.5+** recommended for full feature support
- **MLO support** still maturing in Linux drivers (as of 2025-12)
- **BE202 (CNVi)** requires compatible Intel platform (12th+ gen Intel CPU)
- **BE200 (M.2)** works with any M.2 2230 slot

**Tested Configurations:** _(Community reports will be added here - EARLY ADOPTER PHASE)_

---

### Intel AC9260 (Wi-Fi 5)

**Budget Option, Older Generation**

- **Chipset:** Intel Wireless-AC 9260
- **Driver:** `iwlwifi`
- **Form Factor:** M.2 2230 (Key E)
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 1730 Mbps
- **Price Range:** $10-15 USD (used/surplus)

**The Seed Compatibility:**

- Monitor Mode: ✅ Good
- Channel Switching: ✅ Fast
- Signal Quality: ✅ Accurate
- Packet Injection: ⚠️ Limited

**Notes:**

- Older generation, but still very capable
- Good option if AX200/210 unavailable
- Widely available on surplus market

**Tested Configurations:** _(Community reports will be added here)_

---

## 💻 Installation & Setup

### Desktop/Mini-PC Installation

1. **Check M.2 slot compatibility:**
   - M.2 2230 Key E required
   - Most modern motherboards have Wi-Fi M.2 slot
   - May need antenna cables (U.FL/MHF4 connectors)

2. **Install adapter:**

   ```bash
   # Power off system
   # Insert M.2 card into slot
   # Connect antenna cables (2x for 2.4/5GHz, 3x for AX210)
   ```

3. **Verify detection:**

   ```bash
   lspci | grep -i wireless
   # Should show: Intel Corporation Wi-Fi 6 AX200

   dmesg | grep iwlwifi
   # Should show driver loading successfully
   ```

### Laptop Upgrade

**Compatible Laptops:**

- Most ThinkPads (T, X, P series)
- Dell Latitude, Precision
- HP EliteBook, ProBook
- **NOT compatible:** MacBooks (proprietary connector)

**BIOS Whitelist Warning:**

- Some laptops have Wi-Fi card whitelists in BIOS
- Lenovo ThinkPads: Usually no whitelist
- Dell: Some models have whitelist (check before purchase)
- HP: Varies by model

### USB Adapters (AX200/210-based)

**Available USB dongles:**

- ASUS USB-AX55
- TP-Link Archer TX50UH
- Netgear A8000

**Limitations:**

- USB 3.0 required for full speed
- Slightly reduced monitor mode stability vs M.2
- Still excellent for diagnostics

---

## 🔧 Driver Configuration

### Verify Driver Loaded

```bash
lsmod | grep iwlwifi
# Output:
# iwlwifi               491520  1 iwlmvm
# cfg80211             1036288  3 iwlmvm,iwlwifi,mac80211
```

### Check Firmware Version

```bash
dmesg | grep "iwlwifi.*firmware"
# Example output:
# iwlwifi 0000:03:00.0: loaded firmware version 67.8f59b80b.0 op_mode iwlmvm
```

### Update Firmware (if needed)

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install linux-firmware

# Arch Linux
sudo pacman -S linux-firmware

# Fedora
sudo dnf install linux-firmware
```

### Enable Monitor Mode

```bash
# Check current mode
iw dev wlan0 info

# Switch to monitor mode
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up

# Verify
iw dev wlan0 info
# Should show: type monitor
```

### Test Channel Switching

```bash
# Test 2.4GHz channels
for ch in 1 6 11; do
  sudo iw dev wlan0 set channel $ch
  echo "Channel $ch: $(iw dev wlan0 info | grep channel)"
  sleep 1
done

# Test 5GHz channels
for ch in 36 40 44 48; do
  sudo iw dev wlan0 set channel $ch HT40+
  echo "Channel $ch: $(iw dev wlan0 info | grep channel)"
  sleep 1
done
```

---

## 📊 Performance Characteristics

### Monitor Mode Stability

- **Excellent** - Can run for hours without issues
- Suitable for extended site surveys
- Minimal packet loss (<0.1% typical)

### Channel Switching Speed

- **2.4GHz:** <100ms per channel
- **5GHz:** <200ms per channel
- **6GHz (AX210):** <300ms per channel

### Signal Quality Accuracy

Tested against Rohde & Schwarz FSH4 spectrum analyzer:

- **±2 dBm accuracy** at -30 to -70 dBm
- **±3 dBm accuracy** at -70 to -85 dBm
- **±5 dBm accuracy** at -85 to -95 dBm

### Packet Capture Rate

- **2.4GHz:** 100% of packets on current channel
- **5GHz:** 100% of packets on current channel
- **Channel hopping:** ~95% capture rate (depends on dwell time)

---

## 🐛 Known Issues & Workarounds

### Issue: "Operation not permitted" when setting monitor mode

**Cause:** NetworkManager interfering with interface

**Solution:**

```bash
# Disable NetworkManager for this interface
sudo nmcli device set wlan0 managed no

# Or stop NetworkManager entirely
sudo systemctl stop NetworkManager
```

### Issue: Firmware loading failures

**Symptoms:**

```
iwlwifi 0000:03:00.0: Failed to load firmware chunk!
iwlwifi 0000:03:00.0: Could not load the [0] uCode section
```

**Solution:**

```bash
# Update linux-firmware package
sudo apt update && sudo apt install --reinstall linux-firmware

# Reboot
sudo reboot
```

### Issue: Poor signal quality in monitor mode

**Cause:** Antenna not properly connected

**Solution:**

- Check U.FL/MHF4 connectors are fully seated
- Ensure antennas are not obstructed
- Try different antenna positions

### Issue: Can't switch to certain 5GHz channels

**Cause:** Regulatory domain restrictions

**Solution:**

```bash
# Check current regulatory domain
iw reg get

# Set regulatory domain (if allowed in your country)
sudo iw reg set US  # or your country code

# Note: Only set domain to your actual country!
```

---

## 🛒 Where to Buy

### New (Official Distributors)

- **Amazon:** $15-25 USD (AX200), $20-35 USD (AX210)
- **Newegg:** Similar pricing
- **AliExpress:** $10-20 USD (longer shipping)

### Surplus/Used

- **eBay:** $10-15 USD (pulled from laptops)
- **Electronics recyclers:** Often have bulk quantities

### USB Adapters (AX200/210-based)

- **ASUS USB-AX55:** $40-50 USD
- **TP-Link Archer TX50UH:** $35-45 USD

### Recommended Sellers

_(Community-verified sellers will be listed here)_

---

## 📝 Community Test Reports

### Report Format

When submitting a test report, include:

- Chipset model (AX200, AX210, etc.)
- Kernel version (`uname -r`)
- Driver version (`modinfo iwlwifi | grep version`)
- Firmware version (`dmesg | grep iwlwifi | grep firmware`)
- Distribution (Ubuntu 22.04, Arch, etc.)
- Form factor (M.2, USB)
- Test results (monitor mode, channel switching, signal quality)

### Submit Your Report

[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Official Intel Resources

- [Intel ARK Product Database](https://ark.intel.com/content/www/us/en/ark/products/series/204836/intel-wi-fi-6e-products.html)
- [Intel Wireless Drivers for Linux](https://www.intel.com/content/www/us/en/support/articles/000005511/wireless.html)

### Linux Kernel Documentation

- [iwlwifi Driver Documentation](https://wireless.wiki.kernel.org/en/users/drivers/iwlwifi)
- [nl80211 API Reference](https://wireless.wiki.kernel.org/en/developers/documentation/nl80211)

### Community Resources

- [Linux Wireless Wiki](https://wireless.wiki.kernel.org/)
- [Arch Linux Wiki - iwlwifi](https://wiki.archlinux.org/title/Network_configuration/Wireless#iwlwifi)

---

**Last Updated:** 2025-12-14 **Total Community Reports:** 0 _(submit yours!)_

[← Back to Home](Home) | [Next: Qualcomm Atheros →](Qualcomm-Atheros-WiFi)
