# Qualcomm Atheros Wi-Fi Adapters

Qualcomm Atheros chipsets using the `ath9k` and `ath10k` drivers are **recommended** for The Seed,
especially for packet injection. The AR9271 is a popular choice for wireless diagnostics and
penetration testing.

[← Back to Home](Home)

## 🎯 Recommended Models

### Atheros AR9271 (Wi-Fi 4)

**Best Choice for Packet Injection & Budget-Friendly**

- **Chipset:** Atheros AR9271
- **Driver:** `ath9k_htc` (in-kernel)
- **Form Factor:** USB 2.0
- **Bands:** 2.4GHz only
- **Max Speed:** 150 Mbps
- **Price Range:** $12-18 USD

**The Seed Compatibility:**

- Monitor Mode: ✅ Excellent
- Channel Switching: ✅ Fast (<500ms)
- Signal Quality: ✅ Accurate
- Packet Injection: ✅ Excellent (best-in-class)
- 2.4GHz Channels: 1-14

**Why Choose AR9271:**

- Proven reliable for monitor mode
- Excellent packet injection (best for advanced diagnostics)
- Wide vendor support (many USB adapters use this chipset)
- No firmware blob required (open-source firmware)
- Budget-friendly

**Popular USB Adapters:**

- TP-Link TL-WN722N v1 (⚠️ v2/v3 use different chipset!)
- Alfa AWUS036NHA
- Panda PAU05

**Tested Configurations:** _(Community reports will be added here)_

---

### Qualcomm QCA6174 (Wi-Fi 5)

**Modern PCIe Option**

- **Chipset:** Qualcomm QCA6174
- **Driver:** `ath10k_pci`
- **Form Factor:** M.2 2230 (Key E)
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 867 Mbps
- **Price Range:** $15-25 USD

**The Seed Compatibility:**

- Monitor Mode: ✅ Good
- Channel Switching: ✅ Fast
- Signal Quality: ✅ Accurate
- Packet Injection: ⚠️ Limited

**Notes:**

- Requires firmware (linux-firmware package)
- Common in Killer Wi-Fi branded products
- Good for general diagnostics, limited injection

**Tested Configurations:** _(Community reports will be added here)_

---

### Atheros AR9280 (Wi-Fi 4)

**Legacy PCIe Option**

- **Chipset:** Atheros AR9280
- **Driver:** `ath9k`
- **Form Factor:** PCIe Mini Card
- **Bands:** 2.4GHz, 5GHz
- **Max Speed:** 300 Mbps

**The Seed Compatibility:**

- Monitor Mode: ✅ Excellent
- Channel Switching: ✅ Fast
- Signal Quality: ✅ Good
- Packet Injection: ✅ Excellent

**Notes:**

- Older generation, harder to find new
- Excellent for diagnostics and injection
- Mini PCIe (not M.2)

---

## 💻 Installation & Setup

### USB Installation (AR9271)

1. **Plug in USB adapter**

2. **Verify detection:**

   ```bash
   lsusb | grep -i atheros
   # Output: Bus 001 Device 003: ID 0cf3:9271 Qualcomm Atheros Communications AR9271 802.11n

   dmesg | tail -20
   # Should show ath9k_htc loading successfully
   ```

3. **Check interface created:**
   ```bash
   ip link show
   # Should show new wlan interface (wlan0, wlan1, etc.)
   ```

### PCIe Installation (QCA6174, AR9280)

1. **Install M.2 or Mini PCIe card**

2. **Verify detection:**

   ```bash
   lspci | grep -i qualcomm
   # or
   lspci | grep -i atheros

   dmesg | grep ath10k
   # Should show driver loading
   ```

3. **Install firmware if needed:**
   ```bash
   sudo apt install linux-firmware
   # Reboot may be required
   ```

---

## 🔧 Driver Configuration

### Verify Driver Loaded

```bash
# For AR9271
lsmod | grep ath9k_htc

# For QCA6174
lsmod | grep ath10k_pci

# For AR9280
lsmod | grep ath9k
```

### Enable Monitor Mode

```bash
# AR9271 example
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up

# Verify
iw dev wlan0 info
# Output: type monitor
```

### Test Packet Injection (AR9271)

```bash
# Inject test frame
sudo aireplay-ng --test wlan0

# Should show:
# Injection is working!
# Found X APs
```

### Channel Switching

```bash
# AR9271 - 2.4GHz only
for ch in 1 6 11; do
  sudo iw dev wlan0 set channel $ch
  echo "Channel $ch: OK"
  sleep 0.5
done

# QCA6174/AR9280 - Test 5GHz
for ch in 36 40 44 48; do
  sudo iw dev wlan0 set channel $ch
  echo "Channel $ch: OK"
  sleep 0.5
done
```

---

## 📊 Performance Characteristics

### AR9271 (USB)

- **Monitor Mode Stability:** Excellent (24+ hour uptime tested)
- **Channel Switching:** <500ms (2.4GHz)
- **Signal Quality:** ±3 dBm accuracy
- **Packet Injection Rate:** Up to 500 packets/sec
- **USB Limitation:** USB 2.0 (max 480 Mbps - not an issue for diagnostics)

### QCA6174 (PCIe)

- **Monitor Mode Stability:** Good
- **Channel Switching:** <200ms (2.4/5GHz)
- **Signal Quality:** ±2 dBm accuracy
- **Packet Injection:** Limited support

### AR9280 (PCIe)

- **Monitor Mode Stability:** Excellent
- **Channel Switching:** <100ms
- **Signal Quality:** ±2 dBm accuracy
- **Packet Injection Rate:** Up to 1000 packets/sec

---

## 🐛 Known Issues & Workarounds

### AR9271: Firmware Loading Issues

**Symptoms:**

```
ath9k_htc: Firmware htc_9271.fw requested
ath9k_htc: Failed to find firmware
```

**Solution:**

```bash
# Install firmware
sudo apt install linux-firmware

# Or download manually
sudo wget https://github.com/kvalo/ath10k-firmware/raw/master/ath9k/htc_9271-1.4.0.fw -O /lib/firmware/ath9k_htc/htc_9271-1.4.0.fw

# Replug USB adapter
```

### AR9271: Monitor Mode Conflicts with NetworkManager

**Solution:**

```bash
# Blacklist interface from NetworkManager
sudo nmcli device set wlan0 managed no
```

### QCA6174: Monitor Mode Not Working

**Cause:** Some QCA6174 firmware versions don't support monitor mode

**Solution:**

```bash
# Check firmware version
dmesg | grep ath10k | grep firmware

# Try updating linux-firmware
sudo apt update && sudo apt upgrade linux-firmware
```

### AR9271: TX Power Too High Warning

**Symptoms:**

```
ath9k_htc: TX power exceeds regulatory limit
```

**Solution:**

```bash
# Set correct regulatory domain
sudo iw reg set US  # or your country code
```

---

## 🛒 Where to Buy

### AR9271-based USB Adapters

**Recommended Vendors:**

- **Alfa AWUS036NHA** - $20-25 USD (highly recommended)
- **TP-Link TL-WN722N v1** - $15-18 USD (⚠️ v2/v3 are NOT AR9271!)
- **Panda PAU05** - $12-15 USD

**Where to Buy:**

- Amazon (check chipset in reviews!)
- AliExpress (cheaper, longer shipping)
- Specialized vendors: Hak5, Rokland, Alfa Network

### Verifying Chipset Before Purchase

**⚠️ Critical:** Many USB adapters changed chipsets between versions!

**How to verify:**

1. Check product reviews mentioning chipset
2. Contact seller to confirm AR9271
3. Look for "ath9k_htc compatible" in description
4. Avoid "v2" or "v3" versions of previously-good models

**Known Good Models:**

- TP-Link TL-WN722N **v1** only (v2/v3 use Realtek)
- Alfa AWUS036NHA (consistent across versions)
- Panda PAU05 (usually AR9271, verify first)

---

## 🔬 Advanced Usage

### Packet Injection with aireplay-ng

```bash
# Test injection
sudo aireplay-ng --test wlan0

# Deauth attack (for diagnostics/testing on YOUR network only)
sudo aireplay-ng --deauth 10 -a [AP_MAC] wlan0
```

### Monitor Mode with airodump-ng

```bash
# Start monitoring
sudo airodump-ng wlan0

# Monitor specific channel
sudo airodump-ng -c 6 wlan0

# Save capture
sudo airodump-ng -c 6 -w capture wlan0
```

### Use with The Seed

```bash
# Put adapter in monitor mode
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up

# Set channel for site survey
sudo iw dev wlan0 set channel 6

# Run The Seed with monitor interface
sudo ./seed --wifi-interface wlan0
```

---

## 📝 Community Test Reports

### Report Format

Include:

- Exact chipset (AR9271, QCA6174, AR9280)
- USB adapter model or PCIe card model
- Kernel version
- Driver version (`modinfo ath9k_htc`)
- Distribution
- Monitor mode test results
- Injection test results (if tested)

### Submit Your Report

[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Documentation

- [ath9k Driver Wiki](https://wireless.wiki.kernel.org/en/users/drivers/ath9k)
- [ath10k Driver Wiki](https://wireless.wiki.kernel.org/en/users/drivers/ath10k)

### Community Resources

- [Kali Linux AR9271 Guide](https://www.kali.org/docs/networking/wifi-setup/)
- [Aircrack-ng Compatibility](https://www.aircrack-ng.org/doku.php?id=compatibility_drivers)

### Firmware

- [ath9k_htc Firmware Repository](https://github.com/qca/open-ath9k-htc-firmware)

---

**Last Updated:** 2025-12-14 **Total Community Reports:** 0 _(submit yours!)_

[← Back to Home](Home) | [← Previous: Intel](Intel-WiFi) | [Next: Broadcom →](Broadcom-WiFi)
