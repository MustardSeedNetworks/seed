# GitHub Wiki Content Guide for The Seed

**Purpose:** This document contains the complete content structure and pages for the GitHub Wiki at https://github.com/krisarmstrong/netscope/wiki

**Status:** Wiki is enabled but empty - content below ready to copy/paste

---

## Wiki Structure

```
Home
├── Getting Started
│   ├── Installation (macOS)
│   ├── Installation (Linux)
│   ├── Installation (Docker)
│   ├── First-Time Setup
│   └── Quick Start Guide
├── Features
│   ├── Network Discovery
│   ├── WiFi Survey & Planning
│   ├── Speed Testing
│   ├── Cable Diagnostics
│   ├── DHCP Rogue Detection
│   ├── Vulnerability Scanning
│   └── Compliance Reporting
├── Configuration
│   ├── Network Interfaces
│   ├── SNMP Settings
│   ├── User Management
│   └── API Configuration
├── Troubleshooting
│   ├── Common Issues
│   ├── Error Messages
│   └── Performance Tuning
├── API Reference
│   ├── Authentication
│   ├── REST Endpoints
│   └── WebSocket Events
├── Development
│   ├── Building from Source
│   ├── Contributing
│   └── Development Environment
└── FAQ
```

---

## Page 1: Home

**URL:** https://github.com/krisarmstrong/netscope/wiki/Home

**Content:**

```markdown
# Welcome to The Seed Wiki

**The Seed** is an AI-powered network diagnostic platform that combines WiFi planning, network monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

## Quick Links

📥 **[Installation](Installation-macOS)** - Get started in 30 minutes
🚀 **[Quick Start Guide](Quick-Start-Guide)** - Your first network scan
📚 **[Features](Features)** - What The Seed can do
🔧 **[Configuration](Configuration)** - Customize your setup
❓ **[FAQ](FAQ)** - Common questions answered
🛠️ **[Troubleshooting](Troubleshooting)** - Solve common issues

## What's New

**Latest Version:** v0.102.0 (December 2025)

- ✨ Intelligent interface auto-detection
- 🔒 Comprehensive security linting
- 🎭 Playwright E2E tests for all user flows
- 📖 Storybook component documentation
- 🧪 Multi-browser testing (Chrome, Firefox, WebKit, Edge)

[View Full Changelog →](https://github.com/krisarmstrong/netscope/releases)

## Getting Help

- 💬 **Community:** [Discord](https://discord.gg/mustardseed) (TBD)
- 🐛 **Bug Reports:** [GitHub Issues](https://github.com/krisarmstrong/netscope/issues)
- 📧 **Support:** support@mustardseednetworks.com
- 📖 **Documentation:** [Official Docs](https://docs.mustardseednetworks.com) (TBD)

## Contributing

The Seed is open source! Contributions are welcome.

- [Contributing Guide](Contributing)
- [Development Environment](Development-Environment)
- [Code of Conduct](https://github.com/krisarmstrong/netscope/blob/main/CODE_OF_CONDUCT.md)

---

*From a tiny seed, a mighty network grows.*

**Mustard Seed Networks**
```

---

## Page 2: Installation (macOS)

**URL:** https://github.com/krisarmstrong/netscope/wiki/Installation-macOS

**Content:**

```markdown
# Installation on macOS

## System Requirements

- macOS 13.0 (Ventura) or later
- Apple Silicon (M1/M2/M3) or Intel processor
- 4GB RAM minimum (8GB recommended)
- 500MB disk space

## Installation Methods

### Method 1: Homebrew (Recommended)

**Coming Soon:** The Seed will be available via Homebrew tap.

```bash
# Not yet available - planned for launch
brew install mustardseednetworks/tap/seed
```

### Method 2: Download Binary

1. **Download** the latest release for macOS:
   - [Download for Apple Silicon (M1/M2/M3)](https://github.com/krisarmstrong/netscope/releases/latest)
   - [Download for Intel](https://github.com/krisarmstrong/netscope/releases/latest)

2. **Extract** the archive:
   ```bash
   tar -xzf seed-darwin-arm64.tar.gz  # Apple Silicon
   # OR
   tar -xzf seed-darwin-amd64.tar.gz  # Intel
   ```

3. **Move** to your PATH:
   ```bash
   sudo mv seed /usr/local/bin/
   ```

4. **Verify** installation:
   ```bash
   seed --version
   ```

### Method 3: Build from Source

See [Building from Source](Building-from-Source) guide.

## First Run

1. **Launch** The Seed:
   ```bash
   seed
   ```

2. **macOS Permission Prompts:**
   - You'll see "seed wants to access files" → Click **OK**
   - Network packet capture requires admin privileges

3. **Setup Wizard** will guide you through:
   - Network interface selection
   - Admin password (for first-time user creation)
   - Default configuration

4. **Open Web UI:**
   - Browser should open automatically to http://localhost:8080
   - If not, manually navigate to http://localhost:8080

## Network Permissions

The Seed requires elevated privileges for packet capture.

**macOS Ventura 13+ (Recommended):**
```bash
# Grant network access without full sudo
sudo chmod +x /usr/local/bin/seed
```

**Alternative:** Run with sudo:
```bash
sudo seed
```

## Uninstallation

```bash
# Stop The Seed if running
pkill seed

# Remove binary
sudo rm /usr/local/bin/seed

# Remove config files (optional)
rm -rf ~/.config/seed
```

## Next Steps

- [First-Time Setup](First-Time-Setup)
- [Quick Start Guide](Quick-Start-Guide)
- [Configuration](Configuration)

## Troubleshooting

**Issue:** "seed: command not found"
- **Solution:** Add `/usr/local/bin` to your PATH or move `seed` to `/usr/bin`

**Issue:** "Operation not permitted" when capturing packets
- **Solution:** Run with `sudo` or grant network access permissions

**Issue:** Web UI doesn't load
- **Solution:** Check if port 8080 is already in use: `lsof -i :8080`

[More Troubleshooting →](Troubleshooting)
```

---

## Page 3: Installation (Linux)

**URL:** https://github.com/krisarmstrong/netscope/wiki/Installation-Linux

**Content:**

```markdown
# Installation on Linux

## System Requirements

- Linux kernel 4.15+ (Ubuntu 18.04+, Debian 10+, RHEL 8+)
- x86_64 or ARM64 architecture
- 4GB RAM minimum (8GB recommended)
- 500MB disk space
- `libpcap` installed (for packet capture)

## Supported Distributions

| Distribution | Version | Status |
|--------------|---------|--------|
| Ubuntu | 22.04 LTS, 24.04 LTS | ✅ Recommended |
| Debian | 11, 12 | ✅ Supported |
| RHEL / AlmaLinux | 8, 9 | ✅ Supported |
| Fedora | 38, 39 | ✅ Supported |
| Arch Linux | Latest | ⚠️ Community supported |

## Installation Methods

### Method 1: Download Binary (Recommended)

1. **Install dependencies:**

   **Ubuntu/Debian:**
   ```bash
   sudo apt update
   sudo apt install libpcap0.8
   ```

   **RHEL/AlmaLinux/Fedora:**
   ```bash
   sudo dnf install libpcap
   ```

2. **Download** the latest release:
   ```bash
   wget https://github.com/krisarmstrong/netscope/releases/latest/download/seed-linux-amd64.tar.gz
   # OR for ARM64
   wget https://github.com/krisarmstrong/netscope/releases/latest/download/seed-linux-arm64.tar.gz
   ```

3. **Extract** the archive:
   ```bash
   tar -xzf seed-linux-amd64.tar.gz
   ```

4. **Move** to your PATH:
   ```bash
   sudo mv seed /usr/local/bin/
   sudo chmod +x /usr/local/bin/seed
   ```

5. **Grant network capture capabilities:**
   ```bash
   sudo setcap cap_net_raw=+ep /usr/local/bin/seed
   ```

6. **Verify** installation:
   ```bash
   seed --version
   ```

### Method 2: Build from Source

See [Building from Source](Building-from-Source) guide.

## Running as a Service (systemd)

**Recommended for servers and production deployments.**

1. **Create systemd service file:**
   ```bash
   sudo nano /etc/systemd/system/seed.service
   ```

2. **Add configuration:**
   ```ini
   [Unit]
   Description=The Seed - AI-Powered Network Diagnostics
   After=network.target

   [Service]
   Type=simple
   User=seed
   Group=seed
   ExecStart=/usr/local/bin/seed
   Restart=on-failure
   RestartSec=10

   # Security hardening
   NoNewPrivileges=true
   PrivateTmp=true

   [Install]
   WantedBy=multi-user.target
   ```

3. **Create dedicated user:**
   ```bash
   sudo useradd -r -s /bin/false seed
   ```

4. **Grant capabilities:**
   ```bash
   sudo setcap cap_net_raw=+ep /usr/local/bin/seed
   ```

5. **Enable and start:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable seed
   sudo systemctl start seed
   ```

6. **Check status:**
   ```bash
   sudo systemctl status seed
   ```

7. **View logs:**
   ```bash
   sudo journalctl -u seed -f
   ```

## Firewall Configuration

**Allow Web UI access (port 8080):**

**UFW (Ubuntu):**
```bash
sudo ufw allow 8080/tcp
```

**firewalld (RHEL/Fedora):**
```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## Uninstallation

```bash
# Stop service
sudo systemctl stop seed
sudo systemctl disable seed

# Remove binary
sudo rm /usr/local/bin/seed

# Remove service file
sudo rm /etc/systemd/system/seed.service
sudo systemctl daemon-reload

# Remove config files (optional)
sudo rm -rf /etc/seed
sudo rm -rf ~/.config/seed
```

## Next Steps

- [First-Time Setup](First-Time-Setup)
- [Quick Start Guide](Quick-Start-Guide)
- [Running as a Service (systemd)](https://github.com/krisarmstrong/netscope/tree/main/deploy/systemd)

## Troubleshooting

**Issue:** "Operation not permitted" when running
- **Solution:** Grant network capture capability: `sudo setcap cap_net_raw=+ep /usr/local/bin/seed`

**Issue:** "Cannot bind to port 8080"
- **Solution:** Port already in use. Stop conflicting service or change port in config.

**Issue:** Service fails to start
- **Solution:** Check logs: `sudo journalctl -u seed -n 50`

[More Troubleshooting →](Troubleshooting)
```

---

## Page 4: Quick Start Guide

**URL:** https://github.com/krisarmstrong/netscope/wiki/Quick-Start-Guide

**Content:**

```markdown
# Quick Start Guide

Get from zero to your first network scan in **5 minutes**.

## Step 1: Install The Seed (5 minutes)

Choose your platform:
- [macOS Installation](Installation-macOS)
- [Linux Installation](Installation-Linux)
- [Docker Installation](Installation-Docker)

## Step 2: First-Time Setup (2 minutes)

1. **Launch The Seed:**
   ```bash
   seed
   ```

2. **Setup Wizard:**
   - Creates first admin user
   - Detects network interfaces
   - Sets default configuration

3. **Create Admin Account:**
   ```
   Username: admin
   Password: ******** (choose strong password)
   ```

4. **Select Network Interface:**
   - The Seed auto-detects interfaces
   - Choose your primary network interface (e.g., `en0`, `eth0`)
   - Click "Next"

## Step 3: Login (30 seconds)

1. **Open Web UI:** http://localhost:8080

2. **Login:**
   - Username: `admin`
   - Password: (from step 2)

## Step 4: Your First Discovery Scan (1 minute)

1. **Navigate to Dashboard** (default view)

2. **Click "Run Discovery"** in the Network Discovery card

3. **Wait 10-60 seconds** while The Seed scans your network

4. **View Results:**
   - List of all discovered devices
   - IP addresses, MAC addresses, hostnames
   - Device types (computer, printer, phone, etc.)
   - OS fingerprinting (Windows, macOS, Linux, etc.)

## Step 5: Explore Features (10 minutes)

### WiFi Survey
1. Click **"WiFi"** in sidebar
2. View detected access points, signal strength, channels
3. Upload floor plan (optional) for heatmap

### Speed Test
1. Click **"Speed Test"** in sidebar
2. Click "Run Test"
3. View download/upload speeds, latency, jitter

### Vulnerability Scan
1. Click **"Vulnerabilities"** in sidebar
2. Click "Start Scan"
3. View security findings and remediation steps

### Compliance Report
1. Click **"Settings"** → **"Compliance"**
2. Select report type (HIPAA, PCI-DSS)
3. Click "Generate Report"
4. Download PDF

## Next Steps

**Learn More:**
- [Network Discovery Deep Dive](Network-Discovery)
- [WiFi Survey Guide](WiFi-Survey-Planning)
- [Configuration Options](Configuration)

**Get Help:**
- [FAQ](FAQ)
- [Troubleshooting](Troubleshooting)
- [Community Discord](https://discord.gg/mustardseed) (TBD)

**Upgrade:**
- [Pricing & Features](https://github.com/krisarmstrong/netscope/blob/main/docs/LICENSING_STRATEGY.md)
- Free tier: 50 devices, no AI
- Starter ($299/year): 200 devices, AI classification
- Professional ($799/year): Unlimited devices, full AI
- Premium ($1,999/year): Predictive WiFi planning
- Enterprise ($4,999/year): Multi-site fleet management

---

**Stuck? Need help?**
- Check [Common Issues](Troubleshooting#common-issues)
- Ask in [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
- Email support@mustardseednetworks.com
```

---

## Page 5: Network Discovery

**URL:** https://github.com/krisarmstrong/netscope/wiki/Network-Discovery

**Content:**

```markdown
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
- **Device Type:** 💻 Computer (icon + label)
- **Manufacturer:** Apple (from MAC OUI lookup)
- **OS:** macOS 14.2 Sonoma (fingerprinted)
- **Last Seen:** 2 minutes ago
- **Status:** 🟢 Online / 🔴 Offline

### Filtering & Sorting

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
   - Settings → SNMP Settings
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
```
192.168.1.0/24
```

**Multiple subnets:**
```
192.168.1.0/24,10.0.0.0/24,172.16.0.0/16
```

**CIDR ranges:**
- `/24` = 256 IPs (192.168.1.0 - 192.168.1.255)
- `/16` = 65,536 IPs (10.0.0.0 - 10.0.255.255)
- `/8` = 16,777,216 IPs (not recommended - very slow!)

### Discovery Profiles

Save common configurations as profiles:

1. **Create Profile:**
   - Settings → Discovery Profiles
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
- **Check:** Is your interface correct? (Settings → Network Interface)
- **Check:** Is subnet correct? (192.168.1.0/24 vs 192.168.0.0/24)
- **Try:** Run as sudo (packet capture requires elevated privileges)

**Issue:** Some devices missing
- **Try:** Enable all discovery methods (ARP, ICMP, TCP)
- **Try:** Increase timeout (some devices respond slowly)
- **Check:** Firewall blocking ICMP? (try TCP scan)

**Issue:** Wrong device types/OS
- **Report:** GitHub issue with device details (helps improve AI model)

[More Troubleshooting →](Troubleshooting)

## Related

- [SNMP Configuration](SNMP-Settings)
- [Vulnerability Scanning](Vulnerability-Scanning)
- [API: Discovery Endpoints](API-Discovery)
```

---

## Page 6: FAQ

**URL:** https://github.com/krisarmstrong/netscope/wiki/FAQ

**Content:**

```markdown
# Frequently Asked Questions (FAQ)

## General

### What is The Seed?

The Seed is an AI-powered network diagnostic platform that combines WiFi planning, network monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

It tells you **what's wrong** with your network and **how to fix it** - not just data dumps.

### Who is The Seed for?

- **Healthcare IT:** Hospitals, clinics, medical practices (HIPAA compliance built-in)
- **SMB Network Admins:** Small/medium businesses (50-500 employees)
- **WiFi Consultants:** Budget-conscious consultants doing site surveys
- **MSPs:** Managed service providers managing client networks

### Is The Seed free?

**Free tier:** Yes! Up to 50 devices, community support, no AI features.

**Paid tiers:**
- Starter: $299/year (200 devices, AI classification)
- Professional: $799/year (unlimited devices, full AI)
- Premium: $1,999/year (predictive WiFi planning)
- Enterprise: $4,999/year (multi-site fleet management)

[View Pricing Details →](https://github.com/krisarmstrong/netscope/blob/main/docs/LICENSING_STRATEGY.md)

### Is The Seed open source?

Yes! The Seed is open source under the AGPL-3.0 license.

- Source code: https://github.com/krisarmstrong/netscope
- Contributions welcome: [Contributing Guide](Contributing)

---

## Installation & Setup

### What platforms does The Seed support?

**Supported:**
- ✅ macOS (Ventura 13.0+, Apple Silicon + Intel)
- ✅ Linux (Ubuntu 22.04+, Debian 11+, RHEL 8+)
- ✅ Docker (cross-platform)

**Not supported (yet):**
- ❌ Windows (planned for future release)
- ❌ Mobile (iOS/Android - planned for WiFi survey mobile app)

### Do I need to be root/admin to run The Seed?

**macOS:** Yes, packet capture requires elevated privileges. Run with `sudo seed` or grant network access permissions.

**Linux:** Use `setcap` to grant network capabilities without full sudo:
```bash
sudo setcap cap_net_raw=+ep /usr/local/bin/seed
```

### Can I run The Seed on a server without a GUI?

Yes! The Seed is a web application:
- Backend runs on server (headless)
- Web UI accessed from any browser (on network)
- Perfect for VM, cloud instance, Raspberry Pi

Example: Run on Ubuntu server, access UI from laptop browser.

---

## Features

### Can The Seed plan WiFi coverage BEFORE doing a site survey?

**Yes!** This is The Seed's unique feature: **Predictive WiFi Planning**.

1. Upload floor plan (PNG, PDF, CAD)
2. Mark walls, obstructions
3. Place virtual APs
4. Get instant heatmap (predicted coverage)
5. Optimize AP placement BEFORE buying hardware

**No competitor has this.** Ekahau, Hamina, etc. require an on-site survey first.

### How accurate is the WiFi predictive planning?

**Typical accuracy:** 85-95% compared to real-world survey.

**Good enough for:**
- Estimating AP count (how many do I need?)
- Budget planning (before proposal)
- Initial design (before site visit)

**Not a replacement for:**
- Final validation (always do on-site survey for critical deployments)
- Complex RF environments (stadiums, warehouses with metal/concrete)

### Does The Seed replace SolarWinds / PRTG / Nagios?

**For SMBs and healthcare:** Yes! The Seed covers 90% of use cases:
- Network discovery
- Device monitoring
- Alerting
- Performance metrics
- WiFi monitoring

**For large enterprises (10,000+ devices):** No. SolarWinds, Datadog, etc. are designed for massive scale and deep integrations. The Seed targets SMBs, not Fortune 500.

### Does The Seed replace Ekahau?

**For small projects and predictive planning:** Yes! The Seed offers:
- Predictive WiFi planning (Ekahau doesn't have this)
- 60% cheaper ($1,999 vs $5,198)
- Cross-platform (Ekahau is Windows-only)

**For large consultant firms:** Maybe not. Ekahau is the industry standard with:
- Extreme accuracy (critical for stadiums, warehouses)
- Professional reports (clients expect Ekahau branding)
- Deep AP vendor integrations

---

## Security & Privacy

### Does The Seed send data to the cloud?

**No.** The Seed is **self-hosted** - all data stays on your network.

- No telemetry sent to Mustard Seed Networks
- No usage tracking
- No cloud dependencies

**Exception:** License validation (checks license key status, but doesn't send scan data).

### Is The Seed HIPAA compliant?

Yes! The Seed is designed for healthcare:

- **Self-hosted:** Data never leaves your network (meets HIPAA data sovereignty)
- **Encryption:** AES-256 at rest, TLS 1.3 in transit
- **Audit logs:** Track all user actions (who did what, when)
- **Compliance reports:** Generate HIPAA security risk assessments

**Note:** You are responsible for your overall HIPAA compliance. The Seed is a tool to *help* you comply, not a complete solution.

### What data does The Seed collect?

**During network scans:**
- IP addresses, MAC addresses, hostnames
- Open ports, services
- OS fingerprints
- SNMP data (if configured)

**Stored locally:**
- Scan results (in SQLite database)
- Config files (YAML)
- Logs (application logs, audit logs)

**Never collected:**
- Network traffic content (no DPI, no packet inspection beyond headers)
- Passwords (except hashed admin passwords for web UI login)
- Personal health information (PHI)

---

## Troubleshooting

### The Seed isn't finding all my devices. Why?

**Common causes:**

1. **Firewall blocking ICMP:** Some devices block ping.
   - **Fix:** Enable TCP port scan (Settings → Discovery → Methods → TCP)

2. **Wrong subnet:** Scanning 192.168.0.0/24 but devices are on 192.168.1.0/24
   - **Fix:** Check your subnet (`ip addr` on Linux, `ifconfig` on macOS)

3. **Insufficient timeout:** Some devices respond slowly.
   - **Fix:** Increase timeout (Settings → Discovery → Timeout → 60 seconds)

4. **VLANs:** Devices on different VLANs not discovered.
   - **Fix:** Configure SNMP to query managed switches, or add multiple subnets

5. **Permissions:** Packet capture requires root/admin.
   - **Fix:** Run with `sudo` or grant capabilities (`setcap`)

### Web UI won't load. What's wrong?

**Check:**

1. **Is The Seed running?**
   ```bash
   ps aux | grep seed
   ```
   If not, start it: `seed` (or `sudo systemctl start seed` if using systemd)

2. **Is port 8080 in use?**
   ```bash
   lsof -i :8080  # macOS/Linux
   ```
   If another service is using 8080, stop it or configure The Seed to use different port.

3. **Firewall blocking?**
   - Check firewall rules: `sudo ufw status` (Ubuntu) or `sudo firewall-cmd --list-all` (RHEL)
   - Allow port 8080: `sudo ufw allow 8080/tcp`

4. **Wrong URL?**
   - Local: http://localhost:8080
   - Remote: http://<server-ip>:8080

### Speed test fails / shows 0 Mbps. Why?

**Common causes:**

1. **No internet connection:** Speed test requires internet.
   - **Fix:** Check gateway: `ping 8.8.8.8`

2. **Firewall blocking:** ISP or firewall blocking speed test servers.
   - **Fix:** Temporarily disable firewall or allow speed test domains

3. **VPN active:** Speed test measures VPN speed, not actual internet.
   - **Fix:** Disconnect VPN and retest

### How do I reset the admin password?

**Option 1: Reset via CLI (if you have shell access)**
```bash
seed reset-password admin
# Enter new password when prompted
```

**Option 2: Reset via config file**
1. Stop The Seed: `sudo systemctl stop seed`
2. Delete user database: `rm ~/.config/seed/users.db`
3. Start The Seed: `sudo systemctl start seed`
4. Run setup wizard again (creates new admin user)

**Option 3: Reset all data (nuclear option)**
```bash
sudo systemctl stop seed
rm -rf ~/.config/seed
sudo systemctl start seed
```
⚠️ **Warning:** This deletes all scan history, settings, and user accounts.

---

## Licensing & Pricing

### Can I use The Seed for free forever?

**Yes!** The free tier has no time limit.

**Free tier includes:**
- Up to 50 devices
- Network discovery
- Basic WiFi scanning
- Speed testing
- Community support (forum, GitHub)

**Free tier does NOT include:**
- AI features (device classification, root cause analysis)
- Vulnerability scanning
- Compliance reporting
- Priority support

### What happens if I exceed the device limit?

**Free tier (50 devices):**
- Discovery will find all devices
- UI will show all devices
- But a banner will prompt you to upgrade

**You can continue using for free, but some features will be limited.**

**Paid tiers:**
- Starter: 200 devices (hard limit, won't scan beyond 200)
- Professional: Unlimited devices
- Premium: Unlimited devices
- Enterprise: Unlimited devices

### Do I need a separate license for each site?

**No.** One license = all sites.

**Example:** Enterprise tier ($4,999/year) covers:
- Headquarters
- 5 branch offices
- 10 remote clinics
- Unlimited sites

### Can I transfer my license to a different server?

**Yes.** Licenses are tied to organization, not hardware.

**How to transfer:**
1. Deactivate license on old server (Settings → License → Deactivate)
2. Activate license on new server (enter license key during setup)

**Limitations:**
- One active instance per license (can't run on 2 servers simultaneously)
- Enterprise tier allows multi-site (multiple active instances)

---

## Support

### How do I get help?

**Free tier:**
- Community forum (Discord, TBD)
- GitHub Discussions: https://github.com/krisarmstrong/netscope/discussions
- Documentation: This wiki

**Paid tiers:**
- Email support: support@mustardseednetworks.com
- Priority support (Professional+)
- Phone support (Premium+)
- Dedicated TAM (Enterprise)

[View Support Tiers →](https://github.com/krisarmstrong/netscope/blob/main/docs/SUPPORT_STRATEGY.md)

### How do I report a bug?

1. **Check existing issues:** https://github.com/krisarmstrong/netscope/issues
2. **Create new issue** (if not already reported):
   - Use [Bug Report template](https://github.com/krisarmstrong/netscope/issues/new?template=bug_report.md)
   - Include: The Seed version, OS, steps to reproduce, logs
3. **Expected response time:**
   - Free tier: Best-effort (community)
   - Paid tiers: 4-24 hours (depending on tier)

### How do I request a feature?

1. **Check existing requests:** https://github.com/krisarmstrong/netscope/discussions/categories/ideas
2. **Create new discussion** (if not already requested)
3. **Upvote** existing requests (helps us prioritize)

**Note:** Feature requests from Enterprise customers get 2x priority.

---

## Contributing

### How can I contribute to The Seed?

**Ways to contribute:**

1. **Code:** Submit PRs for bug fixes, features
   - [Contributing Guide](Contributing)
   - [Development Environment](Development-Environment)

2. **Documentation:** Improve wiki, guides, tutorials

3. **Community:** Answer questions in Discord, GitHub Discussions

4. **Testing:** Report bugs, test beta features

5. **Donations:** Not accepted (keep it open source and free of conflicts)

### I found a security vulnerability. How do I report it?

**DO NOT create a public GitHub issue.**

**Instead:**
- Email: security@mustardseednetworks.com
- Subject: "Security Vulnerability Report"
- Include: Description, steps to reproduce, impact

**We commit to:**
- Respond within 24 hours
- Fix critical vulnerabilities within 7 days
- Acknowledge your contribution (if you want credit)

[Security Policy →](https://github.com/krisarmstrong/netscope/blob/main/SECURITY.md)

---

## Comparison to Competitors

### The Seed vs Ekahau

| Feature | The Seed | Ekahau AI Pro |
|---------|----------|---------------|
| **Price** | $1,999/year | $5,198 (year 1) |
| **Predictive Planning** | ✅ Yes | ❌ No |
| **WiFi Survey** | ✅ Yes | ✅ Yes (more accurate) |
| **Network Diagnostics** | ✅ Yes | ❌ No |
| **Platforms** | macOS, Linux | Windows only |

**When to choose The Seed:** Predictive planning, budget-conscious, need network diagnostics too

**When to choose Ekahau:** Large consultant firm, need extreme accuracy, clients expect Ekahau branding

### The Seed vs SolarWinds

| Feature | The Seed | SolarWinds NPM |
|---------|----------|----------------|
| **Price** | $799/year (unlimited) | $10,000+ (100 devices) |
| **Target** | SMB, healthcare | Enterprise IT |
| **Setup Time** | 30 minutes | Weeks |
| **AI Diagnostics** | ✅ Yes | ❌ No |

**When to choose The Seed:** SMB, healthcare, need simple + affordable

**When to choose SolarWinds:** Enterprise (10,000+ devices), need deep integrations

### The Seed vs Nessus

| Feature | The Seed | Nessus Professional |
|---------|----------|---------------------|
| **Price** | $1,999/year (Premium) | $4,620/year |
| **Vulnerability Scanning** | ✅ Yes (10K+ checks) | ✅ Yes (170K+ checks) |
| **Network Diagnostics** | ✅ Yes | ❌ No |
| **WiFi Planning** | ✅ Yes | ❌ No |

**When to choose The Seed:** Need all-in-one (network + WiFi + security)

**When to choose Nessus:** Need comprehensive vulnerability database (security-first)

[Full Competitive Analysis →](https://github.com/krisarmstrong/netscope/blob/main/docs/COMPETITIVE_ANALYSIS.md)

---

**Still have questions?**

- Ask in [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
- Email support@mustardseednetworks.com
- Join our Discord (TBD)
```

---

## How to Populate the Wiki

### Step 1: Enable Wiki (Already Done)

The wiki is enabled. Now you need to add content.

### Step 2: Create Pages via GitHub Web Interface

1. **Go to:** https://github.com/krisarmstrong/netscope/wiki

2. **Click "Create the first page"** (or "New Page" if one exists)

3. **Copy/paste content** from this document into the wiki editor

4. **Create pages in this order:**
   1. Home (required - first page)
   2. Installation-macOS
   3. Installation-Linux
   4. Quick-Start-Guide
   5. Network-Discovery
   6. FAQ
   7. (Continue with remaining pages as time permits)

### Step 3: Link Pages

Wiki pages auto-link via [[Page-Name]] syntax.

**Example:**
```markdown
See the [Installation Guide](Installation-macOS) for setup instructions.
```

or

```markdown
See the [[Installation-macOS|Installation Guide]] for setup instructions.
```

### Step 4: Add Sidebar (Optional)

Create a page named `_Sidebar.md` with navigation links:

```markdown
**The Seed Wiki**

📥 **Getting Started**
- [Installation (macOS)](Installation-macOS)
- [Installation (Linux)](Installation-Linux)
- [Quick Start](Quick-Start-Guide)

🚀 **Features**
- [Network Discovery](Network-Discovery)
- [WiFi Survey](WiFi-Survey-Planning)
- [Vulnerability Scanning](Vulnerability-Scanning)

🔧 **Configuration**
- [Network Interfaces](Configuration)
- [SNMP Settings](SNMP-Settings)

❓ **Help**
- [FAQ](FAQ)
- [Troubleshooting](Troubleshooting)
```

---

## Next Steps

1. **Create wiki pages** using content above
2. **Add screenshots** (wiki supports image uploads)
3. **Link to wiki** from README.md
4. **Update as product evolves**

---

**Document Owner:** Kris Armstrong
**Last Updated:** December 2025
