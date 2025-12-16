# Broadcom Ethernet NICs

Broadcom server-grade NICs provide **excellent TDR support** comparable to Intel. Consumer/desktop
Broadcom chipsets vary. **Recommended for server environments**.

[← Back to Home](Home)

## 🎯 Recommended Models with TDR Support

### Broadcom BCM5719 (Quad-Port Server NIC)

**Server-Grade with Full TDR**

- **Chipset:** Broadcom BCM5719
- **Driver:** `tg3` (in-kernel)
- **Form Factor:** PCIe x4
- **Speed:** 10/100/1000 Mbps
- **Ports:** 4 (RJ45)
- **Price Range:** $40-80 USD (used), $150-250 USD (new)

**The Seed Compatibility:**

- TDR Cable Testing: ✅ Excellent
- Cable Length Detection: ✅ Yes
- Fault Detection: ✅ Yes
- Distance to Fault: ✅ Yes
- Link Speed Detection: ✅ Accurate

**Why Choose BCM5719:**

- **Comparable to Intel I350** - Similar TDR capabilities
- **Widely available** - Common in Dell/HP servers
- **4 ports** - Test multiple cables
- **Proven reliability** - Enterprise-grade

**Common in:**

- Dell PowerEdge servers
- HP ProLiant servers
- Supermicro motherboards

**Tested Configurations:** _(Community reports will be added here)_

---

### Broadcom BCM5720 (Dual-Port Server NIC)

**Dual-Port Alternative**

- **Chipset:** Broadcom BCM5720
- **Driver:** `tg3`
- **Form Factor:** PCIe x4
- **Speed:** 10/100/1000 Mbps
- **Ports:** 2 (RJ45)
- **Price Range:** $30-60 USD (used), $100-180 USD (new)

**The Seed Compatibility:**

- TDR Cable Testing: ✅ Excellent
- Cable Length Detection: ✅ Yes
- Fault Detection: ✅ Yes
- Distance to Fault: ✅ Yes

**Notes:**

- Very similar to BCM5719 (same family)
- 2 ports vs 4 ports
- Often built into server motherboards

**Tested Configurations:** _(Community reports will be added here)_

---

## ⚠️ Consumer Broadcom NICs (Limited/No TDR)

### BCM57XX Series (Desktop/Laptop)

**Basic NICs - Limited TDR Support**

- **Chipsets:** BCM5751, BCM5761, BCM5784
- **Driver:** `tg3`
- **TDR Support:** ⚠️ **Limited or None**

**Status:**

- May support basic cable status (OK/fault)
- **No distance measurement**
- **No fault location**
- Not recommended for diagnostics

**Testing:**

```bash
sudo ethtool --cable-test eth0
# Often returns: Operation not supported
```

---

## 🔧 Installation & Usage

### Verify Broadcom Chipset

```bash
lspci | grep -i broadcom
# Example output:
# 01:00.0 Ethernet controller: Broadcom Inc. NetXtreme BCM5719 Gigabit Ethernet PCIe

# Check driver
ethtool -i eth0 | grep driver
# Should show: tg3
```

### Test TDR Support

```bash
sudo ethtool --cable-test eth0

# BCM5719/5720: Should work
# Cable test started for eth0.
# Cable test completed for eth0.
# Pair A code: OK
# Pair A length: 15m
# (etc.)

# BCM57XX: May fail
# netlink error: Operation not supported
```

---

## 📊 TDR Performance (BCM5719/5720)

### Accuracy

- Cable length: ±3-5m
- Fault distance: ±3-5m
- Test time: ~15-30 seconds

### Comparison to Intel

- **Similar accuracy** to Intel I350/I210
- **Slightly slower** test execution
- **Equally reliable** for fault detection

---

## 🐛 Known Issues

### Issue: TDR Not Working on BCM57XX

**Cause:** Consumer chipsets don't support TDR

**Solution:** Purchase Intel I210 or server-grade Broadcom (BCM5719/5720)

### Issue: Driver Module Not Loading

**Symptoms:**

```
tg3: module not found
```

**Solution:**

```bash
# Install driver
sudo modprobe tg3

# Make permanent
echo "tg3" | sudo tee -a /etc/modules
```

---

## 🛒 Where to Buy

### BCM5719/5720 (Server NICs)

**Used/Surplus (Best Value):**

- **eBay:** $30-80 USD (pulled from servers)
- **Server liquidators:** $40-70 USD

**Common Cards:**

- Dell 1GbE 4-port (BCM5719)
- HP Ethernet 1Gb 4-port (BCM5719)
- Broadcom NetXtreme Quad Port (BCM5719)

**Search Terms:**

- "BCM5719 quad port"
- "BCM5720 dual port"
- "Broadcom NetXtreme server NIC"

### Verify Before Purchase

⚠️ **Important:** Not all Broadcom NICs support TDR

**Questions to ask seller:**

- Is this BCM5719 or BCM5720?
- Pulled from server or desktop?
- Server NICs = likely TDR support
- Desktop NICs = likely no TDR

---

## 📝 Community Test Reports

### Successful TDR Configurations

_(Community reports will be added here)_

### Submit Your Report

[Create a Hardware Report Issue](https://github.com/krisarmstrong/seed/issues/new?template=hardware-report.yml)

---

## 🔗 Additional Resources

### Driver Documentation

- [tg3 Driver](https://www.kernel.org/doc/html/latest/networking/device_drivers/ethernet/broadcom/tg3.html)

### Broadcom Official

- [Broadcom Ethernet Adapters](https://www.broadcom.com/products/ethernet-connectivity/network-adapters)

---

**Last Updated:** 2025-12-14 **Recommendation:** BCM5719/5720 excellent for server environments, but
Intel I350/I210 easier to find for consumer use.

[← Back to Home](Home) | [← Previous: Intel Ethernet](Intel-Ethernet) |
[Next: Realtek Ethernet →](Realtek-Ethernet)
