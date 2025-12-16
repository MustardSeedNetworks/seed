# Network Discovery

Discover every device on your network in seconds using ICMP, ARP, and TCP probes.

## How It Works

The Seed uses multiple discovery methods to find devices:

1. **ARP Scan** (Layer 2)
   - Fastest method for local subnet
   - Works on same network segment
   - Finds MAC addresses

2. **ICMP Ping Sweep** (Layer 3)
   - Sends ICMP echo requests to IP range
   - Works across subnets
   - Identifies responsive hosts

3. **TCP Port Scan** (Layer 4)
   - Probes common ports (22, 80, 443, 3389, etc.)
   - Finds devices that block ICMP
   - Identifies services

4. **SNMP Queries** (Application Layer)
   - Queries managed devices (switches, APs, printers)
   - Retrieves detailed device info
   - Requires SNMP credentials (optional)

## Running a Discovery Scan

### Quick Scan (Automatic)

1. Click **"Run Discovery"** button
2. The Seed scans your current subnet automatically
3. Results appear in 10-60 seconds

### Custom Scan

1. Click **Settings** icon in Network Discovery card
2. Configure:
   - **Subnets:** 192.168.1.0/24 (or multiple: 192.168.1.0/24,10.0.0.0/24)
   - **Timeout:** 30 seconds (default)
   - **Methods:** ARP, ICMP, TCP (select all for best results)
3. Click **"Run Custom Scan"**

## Discovery Results

### Device List

Each discovered device shows:

- **IP Address:** 192.168.1.100
- **MAC Address:** AA:BB:CC:DD:EE:FF
- **Hostname:** janes-macbook.local
- **Device Type:** Computer (icon + label)
- **Manufacturer:** Apple (from MAC OUI lookup)
- **OS:** macOS 14.2 Sonoma (fingerprinted)
- **Last Seen:** 2 minutes ago
- **Status:** Online / Offline

### Filtering and Sorting

**Filter by:**

- Device type (computers, printers, phones, IoT)
- Manufacturer (Apple, Cisco, HP, etc.)
- Status (online, offline)
- VLAN (if detected)

**Sort by:**

- IP address (ascending/descending)
- Hostname (alphabetical)
- Last seen (most recent first)

### Export

**Export to:**

- CSV (spreadsheet import)
- JSON (API integration)
- PDF (documentation, compliance reports)

**Example CSV:**

```csv
IP Address,MAC Address,Hostname,Type,Manufacturer,OS,Last Seen
192.168.1.1,AA:BB:CC:DD:EE:FF,router.local,Router,Ubiquiti,UniFi OS,2024-12-15 10:30:00
192.168.1.100,11:22:33:44:55:66,janes-macbook,Computer,Apple,macOS 14.2,2024-12-15 10:29:00
```

## Device Fingerprinting

The Seed uses **AI-powered fingerprinting** to identify:

- **Operating System:** Windows, macOS, Linux, iOS, Android
- **Device Type:** Computer, printer, phone, switch, AP, camera, etc.
- **Manufacturer:** Apple, HP, Cisco, etc. (from MAC OUI database)
- **Services:** HTTP, SSH, RDP, SMB, SNMP (from port scan)

**How it works:**

1. Collect fingerprint data (TTL, TCP window size, open ports, etc.)
2. AI model classifies device based on patterns
3. Confidence score: High (>90%), Medium (70-90%), Low (<70%)

## VLAN Discovery

If you have managed switches with SNMP configured:

1. **Configure SNMP:**
   - Settings then SNMP Settings
   - Add switch IP, SNMP community/credentials

2. **Run Discovery:**
   - The Seed queries switches via SNMP
   - Retrieves VLAN assignments for each port

3. **View VLANs:**
   - Devices grouped by VLAN ID
   - Color-coded for easy identification

**Example:**

- VLAN 10 (Office): 30 devices
- VLAN 20 (Guest): 10 devices
- VLAN 30 (IoT): 15 devices

## Advanced Configuration

### Subnet Notation

**Single subnet:**

```text
192.168.1.0/24
```

**Multiple subnets:**

```text
192.168.1.0/24,10.0.0.0/24,172.16.0.0/16
```

**CIDR ranges:**

- `/24` = 256 IPs (192.168.1.0 - 192.168.1.255)
- `/16` = 65,536 IPs (10.0.0.0 - 10.0.255.255)
- `/8` = 16,777,216 IPs (not recommended - very slow!)

### Discovery Profiles

Save common configurations as profiles:

1. **Create Profile:**
   - Settings then Discovery Profiles
   - Name: "Main Office"
   - Subnets: 192.168.1.0/24,192.168.2.0/24
   - Methods: All
   - SNMP: Enabled
   - Save

2. **Use Profile:**
   - Select "Main Office" from dropdown
   - Click "Run Discovery"

**Built-in profiles:**

- Default (current subnet, all methods)
- Quick (ARP only, fast but limited)
- Deep (all methods + port scan, slow but comprehensive)

## Performance

**Scan times (typical):**

- 50 devices (/24 subnet): 10-30 seconds
- 200 devices (/23 subnet): 30-60 seconds
- 1,000 devices (/22 subnet): 2-5 minutes

**Factors affecting speed:**

- Number of IPs to scan
- Discovery methods enabled (ARP fastest, TCP slowest)
- Network latency
- Device responsiveness

**Tips for faster scans:**

- Limit subnets to active ranges (not entire /16)
- Use ARP-only for local subnet (fastest)
- Schedule scans during off-hours (less network congestion)

## Troubleshooting

**Issue:** No devices found

- **Check:** Is your interface correct? (Settings then Network Interface)
- **Check:** Is subnet correct? (192.168.1.0/24 vs 192.168.0.0/24)
- **Try:** Run as sudo (packet capture requires elevated privileges)

**Issue:** Some devices missing

- **Try:** Enable all discovery methods (ARP, ICMP, TCP)
- **Try:** Increase timeout (some devices respond slowly)
- **Check:** Firewall blocking ICMP? (try TCP scan)

**Issue:** Wrong device types/OS

- **Report:** GitHub issue with device details (helps improve AI model)

[More Troubleshooting](Troubleshooting)

## Related

- [SNMP Configuration](SNMP-Settings)
- [Vulnerability Scanning](Vulnerability-Scanning)
- [API: Discovery Endpoints](API-Discovery)
