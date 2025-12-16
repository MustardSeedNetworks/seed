# Intel Ethernet NICs

Intel Ethernet NICs are **highly recommended** for The Seed cable diagnostics. The `igb` and `e1000e` drivers provide excellent TDR (Time Domain Reflectometry) support via `ethtool`.

[← Back to Home](Home)

## 🎯 Recommended Models for TDR Cable Testing

### Intel I350 Gigabit (Best Overall)
**Professional-Grade with Full TDR Support**

- **Chipset:** Intel I350-T4 (quad-port) / I350-T2 (dual-port)
- **Driver:** `igb` (in-kernel)
- **Form Factor:** PCIe x4
- **Speed:** 10/100/1000 Mbps
- **Ports:** 2 or 4 (RJ45)
- **Price Range:** $40-80 USD (used), $150-250 USD (new)

**The Seed Compatibility:**
- TDR Cable Testing: ✅ Excellent (full support)
- Cable Length Detection: ✅ Yes (±5m accuracy)
- Fault Detection: ✅ Yes (open, short, impedance mismatch)
- Distance to Fault: ✅ Yes (accurate)
- Link Speed Detection: ✅ Accurate
- Packet Capture (LLDP/CDP): ✅ Excellent

**TDR Capabilities:**
- **All 4 pairs tested** (pair A, B, C, D)
- **Cable status:** OK, Open, Short, CrossTalk, Impedance Mismatch
- **Cable length:** 1m to 180m range
- **Fault distance:** ±2-5m accuracy
- **Test time:** ~10-30 seconds

**Why Choose I350:**
- **Proven reliability** - Industry standard for 10+ years
- **Widely available** - Easy to find used on eBay
- **Best TDR accuracy** - Reference standard
- **4 ports** - Test multiple cables without swapping

**Use Cases:**
- Network technician field diagnostics
- Cable plant troubleshooting
- Data center cabling verification
- Pre-termination testing

**Tested Configurations:**
*(Community reports will be added here)*

---

### Intel I210 Gigabit (Budget Option)
**Single-Port with Full TDR**

- **Chipset:** Intel I210-T1
- **Driver:** `igb`
- **Form Factor:** PCIe x1
- **Speed:** 10/100/1000 Mbps
- **Ports:** 1 (RJ45)
- **Price Range:** $20-35 USD (used), $50-80 USD (new)

**The Seed Compatibility:**
- TDR Cable Testing: ✅ Excellent
- Cable Length Detection: ✅ Yes
- Fault Detection: ✅ Yes
- Distance to Fault: ✅ Yes
- Link Speed Detection: ✅ Accurate

**Why Choose I210:**
- **Budget-friendly** - Cheapest Intel TDR option
- **PCIe x1** - Works in any PCIe slot
- **Same TDR as I350** - Full capabilities
- **Low profile** - Fits small form factor builds

**Limitations vs I350:**
- Only 1 port (vs 2 or 4)
- PCIe x1 (vs x4, but gigabit doesn't need x4)

**Tested Configurations:**
*(Community reports will be added here)*

---

### Intel I225-V 2.5 Gigabit (Modern Option)
**Latest Generation with 2.5GbE**

- **Chipset:** Intel I225-V
- **Driver:** `igc` (in-kernel since 5.6)
- **Form Factor:** PCIe x1, often built into motherboards
- **Speed:** 10/100/1000/2500 Mbps
- **Ports:** 1 (RJ45)
- **Price Range:** $25-40 USD (add-in card), Built-in (free)

**The Seed Compatibility:**
- TDR Cable Testing: ✅ Excellent
- Cable Length Detection: ✅ Yes
- Fault Detection: ✅ Yes
- Distance to Fault: ✅ Yes
- 2.5GbE Support: ✅ Yes

**Why Choose I225-V:**
- **Modern** - Found on recent motherboards (2020+)
- **2.5GbE** - Future-proof for multi-gig networks
- **Built-in** - May already have it
- **Same TDR capabilities** as I210/I350

**Known Issue (Rev A0/A1):**
- **Early revisions** had link drop bugs
- **Rev A2+** fixed the issues
- Check revision: `ethtool -i eth0` → shows "firmware-version"
- **Recommendation:** Ensure rev A2 or newer

**Tested Configurations:**
*(Community reports will be added here)*

---

### Intel 82579LM/V Gigabit (Legacy)
**Common in Older Laptops/Desktops**

- **Chipset:** Intel 82579LM (laptop) / 82579V (desktop)
- **Driver:** `e1000e`
- **Form Factor:** Built-in (LPC interface)
- **Speed:** 10/100/1000 Mbps

**The Seed Compatibility:**
- TDR Cable Testing: ⚠️ Basic (limited functionality)
- Cable Length Detection: ⚠️ Approximate only
- Fault Detection: ✅ Yes (OK/fault status)
- Distance to Fault: ❌ No
- Link Speed Detection: ✅ Accurate

**Notes:**
- Common in Intel-based PCs (2010-2015)
- TDR support is **basic** - not as detailed as I210/I350
- Detects cable faults but doesn't measure distance accurately

**Tested Configurations:**
*(Community reports will be added here)*

---

## 💻 Installation & Setup

### PCIe Card Installation (I210, I350)

1. **Check PCIe slot compatibility:**
   - I210: PCIe x1 (fits any x1, x4, x8, x16 slot)
   - I350: PCIe x4 (fits x4, x8, x16 slot)

2. **Install card:**
   ```bash
   # Power off system
   # Insert card into PCIe slot
   # Connect power if required (usually not)
   # Boot system
   ```

3. **Verify detection:**
   ```bash
   lspci | grep -i ethernet
   # Should show: Intel Corporation I210 Gigabit Network Connection
   # or: Intel Corporation I350 Gigabit Network Connection

   dmesg | grep igb
   # Should show driver loading successfully
   ```

4. **Check interface name:**
   ```bash
   ip link show
   # Look for new interface (eth0, eno1, enp3s0, etc.)
   ```

### Verify TDR Support

```bash
# Test TDR capability
sudo ethtool --cable-test eth0

# Expected output if supported:
# Cable test started for eth0.
# Cable test completed for eth0.
# Pair A code: OK
# Pair A length: 15m
# (etc.)

# If not supported:
# netlink error: Operation not supported
```

---

## 🔧 Using TDR Cable Testing

### Basic Cable Test

```bash
# Start cable test
sudo ethtool --cable-test eth0

# Wait 10-30 seconds for results
# Results will be printed automatically
```

### Example Output - Good Cable

```
Cable test started for eth0.
Cable test completed for eth0.
Pair A code: OK
Pair A length: 23m
Pair B code: OK
Pair B length: 23m
Pair C code: OK
Pair C length: 23m
Pair D code: OK
Pair D length: 23m
```

### Example Output - Open (Disconnected) Cable

```
Cable test started for eth0.
Cable test completed for eth0.
Pair A code: Open
Pair A length: 2m
Pair B code: Open
Pair B length: 2m
Pair C code: Open
Pair C length: 2m
Pair D code: Open
Pair D length: 2m
```

**Interpretation:** Cable is unplugged or broken at ~2m from NIC

### Example Output - Short Circuit

```
Cable test started for eth0.
Cable test completed for eth0.
Pair A code: Short within Pair
Pair A length: 15m
Pair B code: OK
Pair B length: 45m
Pair C code: OK
Pair C length: 45m
Pair D code: OK
Pair D length: 45m
```

**Interpretation:** Pair A is shorted at ~15m (damaged cable)

### Interpreting Results

**Status Codes:**
- **OK** - Pair is functioning normally
- **Open** - Pair is disconnected (no connection)
- **Short within Pair** - Wires within pair are touching
- **Short to Another Pair** - CrossTalk/interference
- **Impedance Mismatch** - Wrong cable type or damaged insulation

**Length Accuracy:**
- **±2-5m typical** - Acceptable for fault locating
- **Affected by:** Cable quality, temperature, installation
- **Not calibrated** - For diagnostics, not surveying

---

## 📊 Performance Characteristics

### TDR Accuracy (I210/I350/I225-V)

**Cable Length Detection:**
- 1-10m cables: ±2m accuracy
- 10-50m cables: ±3m accuracy
- 50-100m cables: ±5m accuracy
- >100m cables: ±10m accuracy

**Fault Distance:**
- Short circuits: ±2m accuracy
- Opens: ±3m accuracy
- Impedance mismatch: ±5m accuracy

**Test Time:**
- ~10 seconds for short cables (<10m)
- ~20 seconds for medium cables (10-50m)
- ~30 seconds for long cables (>50m)

### Link Detection Speed
- 10/100/1000 auto-negotiation: <2 seconds
- LLDP/CDP neighbor discovery: <30 seconds
- Accurate speed/duplex reporting: 100%

---

## 🐛 Known Issues & Workarounds

### Issue: "Operation not supported" for TDR

**Symptoms:**
```bash
sudo ethtool --cable-test eth0
netlink error: Operation not supported
```

**Cause:** NIC doesn't support TDR (wrong chipset)

**Solution:**
- Verify chipset: `lspci | grep -i ethernet`
- If not I210/I350/I225-V: TDR not supported
- **Recommendation:** Purchase Intel I210 or I350

### Issue: TDR Test Hangs

**Symptoms:** `ethtool --cable-test` never completes

**Solution:**
```bash
# Kill hung test
sudo pkill ethtool

# Unload and reload driver
sudo modprobe -r igb
sudo modprobe igb

# Try test again
sudo ethtool --cable-test eth0
```

### Issue: Inaccurate Length Readings

**Symptoms:** Length reported doesn't match known cable length

**Possible Causes:**
1. **Cable quality** - Low-quality cable affects TDR
2. **Temperature** - Cold cables test longer
3. **Solid vs stranded** - Different propagation velocities

**Mitigation:**
- Use known-good cables for calibration
- Understand ±5m accuracy limitation
- Compare multiple cables

### Issue: I225-V Link Drops (Rev A0/A1)

**Symptoms:** Link randomly disconnects and reconnects

**Cause:** Firmware bug in early I225-V revisions

**Solution:**
```bash
# Check revision
ethtool -i eth0 | grep firmware

# If firmware shows A0 or A1:
# Update motherboard BIOS (may include NIC firmware update)
# Or replace with rev A2+ card
```

---

## 🛒 Where to Buy

### New Cards

**Intel I210:**
- Amazon: $50-80 USD
- Newegg: $50-75 USD
- eBay (new): $45-70 USD

**Intel I350:**
- Amazon: $150-250 USD (quad-port)
- Newegg: $180-220 USD
- eBay (new): $120-200 USD

**Intel I225-V:**
- Amazon: $25-40 USD (add-in card)
- Often built into motherboards (2020+)

### Used/Surplus (Best Value)

**eBay:**
- Intel I210: $20-30 USD
- Intel I350 (pulled from servers): $40-80 USD

**Server Liquidators:**
- I350 quad-port: $30-60 USD
- Often "pulls" from decommissioned servers
- Test before buying (eBay money-back guarantee)

**Recommended Searches:**
- "Intel I210-T1 PCIe"
- "Intel I350-T4 NIC"
- "Intel I225-V Ethernet"

---

## 🔬 Advanced Usage

### Scripted Cable Testing

```bash
#!/bin/bash
# Test all Intel NICs

for iface in $(ls /sys/class/net/ | grep -E 'eth|eno|enp'); do
    driver=$(ethtool -i $iface 2>/dev/null | grep driver | awk '{print $2}')

    if [[ "$driver" == "igb" || "$driver" == "igc" || "$driver" == "e1000e" ]]; then
        echo "=== Testing $iface ($driver) ==="
        sudo ethtool --cable-test $iface
        echo ""
        sleep 2
    fi
done
```

### Continuous Monitoring

```bash
# Monitor link status
watch -n 2 'ethtool eth0 | grep -E "Link|Speed|Duplex"'

# Monitor for link flaps
while true; do
    ethtool eth0 | grep "Link detected"
    sleep 1
done
```

### Integration with The Seed

```bash
# Run The Seed with Intel NIC
sudo ./seed --interface eth0

# The Seed will automatically detect TDR support
# and enable cable diagnostics card in UI
```

---

## 📝 Community Test Reports

### Successful TDR Tests
*(Community will report working configurations here)*

### Cable Fault Examples
*(Community will share real-world fault detection here)*

### Submit Your Report
[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Documentation
- [Intel igb Driver](https://www.kernel.org/doc/html/latest/networking/device_drivers/ethernet/intel/igb.html)
- [Intel e1000e Driver](https://www.kernel.org/doc/html/latest/networking/device_drivers/ethernet/intel/e1000e.html)

### Ethtool Documentation
- [ethtool cable-test](https://man7.org/linux/man-pages/man8/ethtool.8.html)
- [TDR Implementation](https://git.kernel.org/pub/scm/network/ethtool/ethtool.git/tree/README)

### Intel Official
- [Intel Ethernet Controllers](https://www.intel.com/content/www/us/en/products/details/ethernet/controllers.html)
- [Intel ARK Database](https://ark.intel.com/)

---

**Last Updated:** 2025-12-14
**Recommendation:** **Intel I350 or I210 for professional cable diagnostics**

[← Back to Home](Home) | [← Previous: MediaTek Wi-Fi](MediaTek-WiFi) | [Next: Broadcom Ethernet →](Broadcom-Ethernet)
