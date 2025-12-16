# Marvell Ethernet NICs

⚠️ **Limited TDR Support** - Marvell server NICs may support cable diagnostics, but consumer
chipsets typically do not. **Intel preferred for reliability**.

[← Back to Home](Home)

## 📊 Marvell Chipset Overview

### Why Marvell is Uncommon

**Market Position:**

- Less common than Intel, Broadcom, or Realtek
- Found in some older servers and specialty hardware
- Driver support varies significantly by chipset

**TDR Support:**

- **Server chipsets:** May support basic TDR
- **Consumer chipsets:** Usually no TDR support
- **Documentation:** Limited compared to Intel

**Recommendation:** If you need reliable TDR, **choose Intel I210/I350** instead.

---

## 🔍 Common Marvell Chipsets

### Marvell 88E1512 (PHY Chip)

**Often Found on Embedded Boards**

- **Type:** PHY (Physical Layer) chip
- **Driver:** `marvell` PHY driver + MAC driver (varies)
- **Speed:** 10/100/1000 Mbps
- **TDR Support:** ⚠️ **Varies** (depends on MAC driver)

**Found In:**

- Raspberry Pi Compute Module carrier boards
- Industrial Ethernet devices
- Embedded systems

**The Seed Compatibility:**

- TDR Cable Testing: ⚠️ **Unknown** (test required)
- Link Speed Detection: ✅ Accurate
- Basic Diagnostics: ✅ Works

**Testing:**

```bash
sudo ethtool --cable-test eth0
# May work, may not - depends on implementation
```

---

### Marvell Yukon 88E8056 (Consumer)

**Common in Older Motherboards/Laptops**

- **Chipset:** Marvell Yukon 88E8056
- **Driver:** `sky2` (in-kernel)
- **Speed:** 10/100/1000 Mbps
- **Found In:** 2005-2010 era motherboards

**The Seed Compatibility:**

- TDR Cable Testing: ❌ **Not supported**
- Link Speed Detection: ✅ Accurate
- Basic Diagnostics: ✅ Works

**Notes:**

- Older chipset, mostly replaced by Realtek/Intel
- No TDR support in Linux driver

---

### Marvell 88E1111 (Legacy PHY)

**Older Embedded/Server PHY**

- **Type:** PHY chip
- **Driver:** `marvell` PHY driver
- **Speed:** 10/100/1000 Mbps
- **TDR Support:** ❌ **Not typically supported**

**Notes:**

- Legacy chipset (pre-2010)
- Found in older embedded systems

---

## ✅ What Marvell NICs CAN Do

### Supported The Seed Features

**✅ Generally Working:**

- Link status detection
- Speed/duplex detection
- Auto-negotiation
- LLDP/CDP packet capture
- Basic network diagnostics

**❌ Usually Not Working:**

- Cable diagnostics (TDR)
- Cable length measurement
- Fault detection

**⚠️ Varies By Chipset:**

- Some Marvell server NICs may support TDR
- Most consumer/embedded chipsets do not

---

## 🔧 Testing Marvell Hardware

### Identify Marvell Chipset

```bash
lspci | grep -i marvell
# Example: Marvell Technology Group Ltd. 88E8056 PCI-E Gigabit Ethernet Controller

ethtool -i eth0
# driver: sky2 (for Yukon)
# or
# driver: mvneta (for embedded)
```

### Test for TDR Support

```bash
sudo ethtool --cable-test eth0

# If supported:
# Cable test started for eth0.
# (results will follow)

# If not supported:
# netlink error: Operation not supported
```

### Use for Basic Diagnostics

Even without TDR, Marvell NICs work for:

```bash
# Link status
ethtool eth0

# LLDP discovery
sudo tcpdump -i eth0 'ether proto 0x88cc'

# Run The Seed (basic features)
sudo ./seed --interface eth0
```

---

## 🛒 Should You Use Marvell?

### ❌ **Don't Purchase** - For New Builds

**Reasons:**

- Limited availability (not as common as Intel/Realtek)
- Uncertain TDR support
- Limited documentation
- **Intel I210 is similar price with guaranteed TDR**

### ✅ **Maybe** - If You Already Have It

**If your system has Marvell:**

1. Test for TDR support: `sudo ethtool --cable-test eth0`
2. If TDR works: Great! Use it.
3. If TDR fails: Works for basic diagnostics, add Intel card for TDR

---

## 🐛 Common Issues

### Issue: Unknown TDR Capability

**Problem:** No documentation on whether specific Marvell model supports TDR

**Solution:** Test empirically

```bash
sudo ethtool --cable-test eth0
# Either works or doesn't - only way to know
```

### Issue: Driver Not Loading

**Symptom:** Network interface not detected

**Solution:**

```bash
# Try loading driver manually
sudo modprobe sky2   # For Yukon chipsets
# or
sudo modprobe mvneta # For embedded chipsets
```

---

## 📝 Community Test Reports

### Marvell Models with TDR

_(Community will report which models support TDR)_

### Marvell Models WITHOUT TDR

_(Community will report confirmed no-TDR models)_

### Submit Your Report

[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

⚠️ **Especially needed for Marvell!** - Documentation is sparse, community reports are valuable.

---

## 🔗 Additional Resources

### Driver Documentation

- [sky2 Driver](https://www.kernel.org/doc/html/latest/networking/device_drivers/ethernet/marvell/sky2.html)
- [Marvell PHY Drivers](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/drivers/net/phy/marvell.c)

### Marvell Official

- [Marvell Ethernet Solutions](https://www.marvell.com/products/ethernet-solutions.html)

---

**Last Updated:** 2025-12-14 **Recommendation:** Test if you have it, but buy Intel I210/I350 for
new hardware.

[← Back to Home](Home) | [← Previous: Realtek Ethernet](Realtek-Ethernet)
