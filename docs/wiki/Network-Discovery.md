# Network Discovery

Discover every device on your network in seconds using ICMP, ARP, and TCP probes.

## How It Works

The Seed uses multiple discovery methods:

1. **ARP Scan** (Layer 2) - Fastest for local subnet
2. **ICMP Ping Sweep** (Layer 3) - Works across subnets
3. **TCP Port Scan** (Layer 4) - Finds devices that block ICMP
4. **SNMP Queries** (Application) - Detailed device info (requires credentials)

## Running a Scan

### Quick Scan

1. Click **"Run Discovery"** button
2. The Seed scans your current subnet automatically
3. Results appear in 10-60 seconds

### Custom Scan

1. Click **Settings** icon in Network Discovery card
2. Configure:
   - **Subnets:** 192.168.1.0/24 (or multiple: 192.168.1.0/24,10.0.0.0/24)
   - **Timeout:** 30 seconds (default)
   - **Methods:** ARP, ICMP, TCP
3. Click **"Run Custom Scan"**

## Discovery Results

Each discovered device shows:

- **IP Address:** 192.168.1.100
- **MAC Address:** AA:BB:CC:DD:EE:FF
- **Hostname:** device-name.local
- **Device Type:** 💻 Computer, 🖨️ Printer, 📱 Phone, etc.
- **Manufacturer:** Apple, Cisco, HP (from MAC lookup)
- **OS:** macOS, Windows, Linux (AI fingerprinting)
- **Last Seen:** 2 minutes ago
- **Status:** 🟢 Online / 🔴 Offline

## Export Options

- **CSV** - Spreadsheet import
- **JSON** - API integration
- **PDF** - Documentation, compliance reports

## Performance

**Typical scan times:**
- 50 devices (/24 subnet): 10-30 seconds
- 200 devices: 30-60 seconds
- 1,000 devices: 2-5 minutes

## Troubleshooting

**No devices found:**
- Check network interface (Settings → Network Interface)
- Verify subnet (192.168.1.0/24 vs 192.168.0.0/24)
- Try running with sudo (packet capture requires privileges)

**Some devices missing:**
- Enable all discovery methods (ARP, ICMP, TCP)
- Increase timeout (some devices respond slowly)
- Check firewall (may be blocking ICMP)

## Related

- [SNMP Configuration](https://github.com/krisarmstrong/netscope/blob/main/README.md)
- [Hardware Compatibility](Home.md)
