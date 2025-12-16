# Realtek Ethernet NICs

❌ **No TDR Support** - Realtek consumer NICs do **NOT support cable diagnostics** via ethtool.
**Not recommended for The Seed cable testing**.

[← Back to Home](Home)

## ⚠️ The Realtek TDR Problem

### Why Realtek Doesn't Work

**TDR Support:** ❌ **None** in Linux drivers

- `r8169` driver: No cable testing support
- `r8125` driver: No cable testing support
- Consumer chipsets: No TDR hardware capability

**Attempted Test:**

```bash
sudo ethtool --cable-test eth0
# Output: netlink error: Operation not supported
```

**Why is this?**

1. **Hardware limitation** - Consumer Realtek chips don't have TDR circuitry
2. **Driver limitation** - Even if hardware supported it, Linux driver doesn't expose it
3. **Windows only** - Some Realtek diagnostic tools exist for Windows, but not Linux-compatible

**Bottom Line:** If you need cable diagnostics with The Seed, **you must use Intel or Broadcom
server NICs**.

---

## 📊 Common Realtek Chipsets

### RTL8111/RTL8168 (Most Common)

**Consumer Gigabit - No TDR**

- **Chipset:** Realtek RTL8111 or RTL8168
- **Driver:** `r8169` (in-kernel)
- **Speed:** 10/100/1000 Mbps
- **Found In:** 90% of consumer motherboards, laptops
- **Price:** Usually built-in (free)

**The Seed Compatibility:**

- TDR Cable Testing: ❌ **Not supported**
- Link Speed Detection: ✅ Accurate
- LLDP/CDP Capture: ✅ Works
- Basic Diagnostics: ✅ Works (no cable testing)

**Use Cases:**

- Basic network diagnostics (link status, DHCP, DNS)
- LLDP/CDP neighbor discovery
- **NOT for cable testing**

---

### RTL8125 (2.5 Gigabit)

**Modern Consumer 2.5GbE - No TDR**

- **Chipset:** Realtek RTL8125
- **Driver:** `r8169` (in-kernel since 5.9) or `r8125` (out-of-tree)
- **Speed:** 10/100/1000/2500 Mbps
- **Found In:** Modern motherboards (2019+)

**The Seed Compatibility:**

- TDR Cable Testing: ❌ **Not supported**
- 2.5GbE Support: ✅ Yes
- Link Speed Detection: ✅ Accurate
- Basic Diagnostics: ✅ Works (no cable testing)

**Notes:**

- Common on AMD B550/X570 and Intel Z490+ motherboards
- Works great for basic diagnostics
- **Still no TDR support** even though newer

---

### RTL8153 (USB Ethernet)

**USB to Gigabit Adapter - No TDR**

- **Chipset:** Realtek RTL8153
- **Driver:** `r8152` (in-kernel)
- **Form Factor:** USB 3.0 to Gigabit RJ45
- **Speed:** 10/100/1000 Mbps

**The Seed Compatibility:**

- TDR Cable Testing: ❌ **Not supported**
- USB Adapter: ✅ Works
- Basic Diagnostics: ✅ Works

**Common Adapters:**

- Anker USB-C to Ethernet
- Cable Matters USB 3.0 to Gigabit
- Many Amazon/AliExpress generic adapters

---

## ✅ What Realtek NICs CAN Do

While Realtek doesn't support TDR cable testing, they work perfectly for:

### Supported The Seed Features

**✅ Working Features:**

- Link status detection (carrier, up/down)
- Speed detection (10/100/1000/2500 Mbps)
- Duplex detection (full/half)
- Auto-negotiation status
- Link partner advertisement
- LLDP/CDP/EDP neighbor discovery
- Packet capture (pcap)
- DHCP testing
- DNS testing
- Gateway ping testing
- Network device discovery

**❌ Not Working:**

- Cable diagnostics (TDR)
- Cable length measurement
- Fault detection and location

### Recommendation

**If you have Realtek NIC:**

- ✅ Use it for all diagnostics **except cable testing**
- ✅ Add Intel I210 PCIe card ($20-35) for cable diagnostics
- ✅ Best of both worlds: Use Realtek for general diagnostics, Intel for TDR

---

## 🔧 Using Realtek with The Seed

### Verify Realtek Chipset

```bash
lspci | grep -i realtek
# Example: Realtek Semiconductor Co., Ltd. RTL8111/8168/8411 PCI Express Gigabit Ethernet Controller

ethtool -i eth0
# driver: r8169
# version: 5.15.0-91-generic
# firmware-version: rtl8168h-2_0.0.2 02/26/15
```

### Confirm No TDR Support

```bash
sudo ethtool --cable-test eth0
# Output: netlink error: Operation not supported

# This is expected and normal for Realtek
```

### Use for Basic Diagnostics

```bash
# Link status - WORKS
ethtool eth0
# Output:
# Link detected: yes
# Speed: 1000Mb/s
# Duplex: Full

# LLDP discovery - WORKS
sudo tcpdump -i eth0 -e -n 'ether proto 0x88cc'

# Run The Seed - WORKS (minus cable card)
sudo ./seed --interface eth0
# All cards work except Cable Diagnostics
```

---

## 🛒 Should You Upgrade from Realtek?

### ❌ **Don't Replace** - If You Don't Need TDR

**Keep Realtek if:**

- You only need link status, LLDP, DHCP, DNS testing
- Motherboard has built-in Realtek (free)
- Budget constrained

**Realtek works perfectly fine** for 90% of The Seed features.

### ✅ **Add Intel Card** - If You Need TDR

**Best Approach:**

1. Keep using Realtek for general diagnostics
2. Add Intel I210 PCIe card ($20-35) for cable testing
3. Switch interfaces in The Seed when you need cable diagnostics

**Installation:**

```bash
# Install Intel I210 in PCIe slot
# System now has two NICs: Realtek (eth0) + Intel (eth1)

# Use Realtek for general work
sudo ./seed --interface eth0

# Switch to Intel for cable testing
sudo ./seed --interface eth1
# Cable Diagnostics card now available
```

---

## 🐛 Common Realtek Issues

### Issue: Slow Link Negotiation

**Symptom:** Takes 5-10 seconds to get link after plugging cable

**Cause:** Realtek chipset behavior

**Workaround:** Force speed/duplex (not recommended unless necessary)

```bash
sudo ethtool -s eth0 speed 1000 duplex full autoneg off
```

### Issue: Link Drops Under Load

**Symptom:** Link randomly disconnects during high traffic

**Solutions:**

1. Update kernel (newer `r8169` driver versions)
2. Disable power saving:
   ```bash
   sudo ethtool -s eth0 wol d
   ```
3. Disable offloading:
   ```bash
   sudo ethtool -K eth0 tso off gso off
   ```

### Issue: Driver Not Loading

**Symptom:** No network after boot

**Solution:**

```bash
# Load driver manually
sudo modprobe r8169

# Make permanent
echo "r8169" | sudo tee -a /etc/modules
```

---

## 📝 Community Test Reports

### Realtek Models Tested

_(Community will report which models work for basic diagnostics)_

### Workarounds and Tips

_(Community will share Realtek-specific tips)_

### Submit Your Report

[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Documentation

- [r8169 Driver](https://www.kernel.org/doc/html/latest/networking/device_drivers/ethernet/realtek/r8169.html)

### Realtek Official

- [Realtek Ethernet Controllers](https://www.realtek.com/en/products/communications-network-ics/item/rtl8111h)

---

**Last Updated:** 2025-12-14 **Recommendation:** Realtek works for 90% of diagnostics, but add Intel
I210 ($20-35) if you need cable testing.

[← Back to Home](Home) | [← Previous: Broadcom Ethernet](Broadcom-Ethernet) |
[Next: Marvell Ethernet →](Marvell-Ethernet)
