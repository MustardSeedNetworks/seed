#!/bin/bash
# Setup script to populate GitHub Wiki with user-facing documentation
#
# Prerequisites:
# 1. Go to https://github.com/krisarmstrong/netscope/wiki
# 2. Click "Create the first page"
# 3. Title: "Home"
# 4. Content: "Initializing wiki..."
# 5. Click "Save Page"
# 6. Then run this script
#
# Usage: ./scripts/setup-wiki.sh

set -e

echo "🌱 The Seed - GitHub Wiki Setup"
echo "================================"
echo ""

# Check if wiki repo exists
if [ ! -d "/tmp/netscope.wiki" ]; then
    echo "📥 Cloning wiki repository..."
    cd /tmp
    rm -rf netscope.wiki 2>/dev/null || true

    if ! git clone https://github.com/krisarmstrong/netscope.wiki.git; then
        echo ""
        echo "❌ ERROR: Wiki repository not found."
        echo ""
        echo "Please initialize the wiki first:"
        echo "1. Go to: https://github.com/krisarmstrong/netscope/wiki"
        echo "2. Click 'Create the first page'"
        echo "3. Save with any content"
        echo "4. Then run this script again"
        echo ""
        exit 1
    fi
fi

cd /tmp/netscope.wiki

echo "✅ Wiki repository cloned"
echo ""

# Create wiki pages
echo "📝 Creating wiki pages..."

# Home page
cat > Home.md << 'EOFHOME'
# Welcome to The Seed Wiki

**The Seed** is an AI-powered network diagnostic platform that combines WiFi planning, network monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

## Quick Links

📥 **[Installation (macOS)](Installation-macOS)** - Get started on macOS
📥 **[Installation (Linux)](Installation-Linux)** - Get started on Linux
🚀 **[Quick Start Guide](Quick-Start-Guide)** - Your first network scan
📚 **[Features](Network-Discovery)** - What The Seed can do
🔧 **[Hardware Compatibility](Hardware-Compatibility)** - Tested hardware
❓ **[FAQ](FAQ)** - Common questions answered

## What's New

**Latest Version:** v0.102.0 (December 2025)

- ✨ Intelligent interface auto-detection
- 🔒 Comprehensive security linting
- 🎭 Playwright E2E tests for all user flows
- 📖 Storybook component documentation
- 🧪 Multi-browser testing (Chrome, Firefox, WebKit, Edge)

[View Full Changelog →](https://github.com/krisarmstrong/netscope/releases)

## Getting Help

- 💬 **Community:** [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
- 🐛 **Bug Reports:** [GitHub Issues](https://github.com/krisarmstrong/netscope/issues)
- 📧 **Support:** support@mustardseednetworks.com

## Contributing

The Seed is proprietary software, but we welcome feedback and bug reports.

- [GitHub Issues](https://github.com/krisarmstrong/netscope/issues)
- [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)

---

*From a tiny seed, a mighty network grows.*

**Mustard Seed Networks**
EOFHOME

echo "  ✅ Home.md"

# Installation macOS
cat > Installation-macOS.md << 'EOFMACOS'
# Installation on macOS

## System Requirements

- macOS 13.0 (Ventura) or later
- Apple Silicon (M1/M2/M3) or Intel processor
- 4GB RAM minimum (8GB recommended)
- 500MB disk space

## Installation Methods

### Method 1: Download Binary (Recommended)

1. **Download** the latest release for macOS from [Releases](https://github.com/krisarmstrong/netscope/releases/latest)

2. **Extract** the archive:
   ```bash
   tar -xzf seed-darwin-arm64.tar.gz  # Apple Silicon
   # OR
   tar -xzf seed-darwin-amd64.tar.gz  # Intel
   ```

3. **Move** to your PATH:
   ```bash
   sudo mv seed /usr/local/bin/
   sudo chmod +x /usr/local/bin/seed
   ```

4. **Verify** installation:
   ```bash
   seed --version
   ```

### Method 2: Build from Source

See the main [README](https://github.com/krisarmstrong/netscope/blob/main/README.md) for build instructions.

## First Run

1. **Launch** The Seed:
   ```bash
   seed
   ```

2. **macOS Permission Prompts:**
   - Network packet capture requires admin privileges
   - You may be prompted for your password

3. **Setup Wizard** will guide you through:
   - Network interface selection
   - Admin account creation
   - Default configuration

4. **Open Web UI:**
   - Browser should open automatically to http://localhost:8080
   - If not, manually navigate to http://localhost:8080

## Network Permissions

The Seed requires elevated privileges for packet capture.

**Run with sudo:**
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

- [Quick Start Guide](Quick-Start-Guide)
- [Network Discovery](Network-Discovery)
- [Hardware Compatibility](Hardware-Compatibility)

## Troubleshooting

**Issue:** "seed: command not found"
- **Solution:** Ensure `/usr/local/bin` is in your PATH

**Issue:** "Operation not permitted" when capturing packets
- **Solution:** Run with `sudo seed`

**Issue:** Web UI doesn't load
- **Solution:** Check if port 8080 is in use: `lsof -i :8080`
EOFMACOS

echo "  ✅ Installation-macOS.md"

# Installation Linux
cat > Installation-Linux.md << 'EOFLINUX'
# Installation on Linux

## System Requirements

- Linux kernel 4.15+ (Ubuntu 18.04+, Debian 10+, RHEL 8+)
- x86_64 or ARM64 architecture
- 4GB RAM minimum (8GB recommended)
- 500MB disk space
- `libpcap` installed (for packet capture)

## Installation

### 1. Install Dependencies

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install libpcap0.8
```

**RHEL/AlmaLinux/Fedora:**
```bash
sudo dnf install libpcap
```

### 2. Download Binary

```bash
wget https://github.com/krisarmstrong/netscope/releases/latest/download/seed-linux-amd64.tar.gz
# OR for ARM64
wget https://github.com/krisarmstrong/netscope/releases/latest/download/seed-linux-arm64.tar.gz
```

### 3. Install

```bash
tar -xzf seed-linux-amd64.tar.gz
sudo mv seed /usr/local/bin/
sudo chmod +x /usr/local/bin/seed
```

### 4. Grant Network Capabilities

```bash
sudo setcap cap_net_raw=+ep /usr/local/bin/seed
```

### 5. Verify

```bash
seed --version
```

## Running as a Service (systemd)

See [deploy/systemd](https://github.com/krisarmstrong/netscope/tree/main/deploy/systemd) for systemd service installation.

## Firewall Configuration

Allow Web UI access (port 8080):

**UFW (Ubuntu):**
```bash
sudo ufw allow 8080/tcp
```

**firewalld (RHEL/Fedora):**
```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## Next Steps

- [Quick Start Guide](Quick-Start-Guide)
- [Network Discovery](Network-Discovery)

## Troubleshooting

**Issue:** "Operation not permitted"
- **Solution:** Grant network capability: `sudo setcap cap_net_raw=+ep /usr/local/bin/seed`

**Issue:** "Cannot bind to port 8080"
- **Solution:** Port in use. Check: `sudo lsof -i :8080`
EOFLINUX

echo "  ✅ Installation-Linux.md"

# Quick Start
cat > Quick-Start-Guide.md << 'EOFQUICKSTART'
# Quick Start Guide

Get from zero to your first network scan in **5 minutes**.

## Step 1: Install The Seed

Choose your platform:
- [macOS Installation](Installation-macOS)
- [Linux Installation](Installation-Linux)

## Step 2: First Launch

```bash
seed
```

The setup wizard will:
1. Create your admin account
2. Auto-detect network interfaces
3. Configure defaults

## Step 3: Login

1. Open http://localhost:8080 in your browser
2. Login with the credentials you created

## Step 4: Your First Discovery Scan

1. Navigate to **Dashboard** (default view)
2. Click **"Run Discovery"** in the Network Discovery card
3. Wait 10-60 seconds while The Seed scans your network

### What You'll See

- List of all discovered devices
- IP addresses, MAC addresses, hostnames
- Device types (computer, printer, phone, etc.)
- OS fingerprinting (Windows, macOS, Linux, etc.)

## Step 5: Explore Features

### WiFi Survey
1. Click **"WiFi"** in sidebar
2. View detected access points, signal strength, channels

### Speed Test
1. Click **"Speed Test"**
2. Click "Run Test"
3. View download/upload speeds, latency

### Vulnerability Scan
1. Click **"Vulnerabilities"**
2. Click "Start Scan"
3. View security findings and remediation steps

## Next Steps

- [Network Discovery Deep Dive](Network-Discovery)
- [Hardware Compatibility](Hardware-Compatibility)
- [FAQ](FAQ)

---

**Need help?**
- [Troubleshooting](https://github.com/krisarmstrong/netscope/wiki)
- [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
- Email: support@mustardseednetworks.com
EOFQUICKSTART

echo "  ✅ Quick-Start-Guide.md"

# Network Discovery
cat > Network-Discovery.md << 'EOFDISCOVERY'
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
- [Hardware Compatibility](Hardware-Compatibility)
EOFDISCOVERY

echo "  ✅ Network-Discovery.md"

# Hardware Compatibility
cat > Hardware-Compatibility.md << 'EOFHARDWARE'
# Hardware Compatibility

The Seed works with most network interfaces, but some hardware provides enhanced capabilities.

## Quick Reference

### WiFi Adapters

| Chipset | Monitor Mode | Channel Switch | Signal Quality | Recommendation |
|---------|--------------|----------------|----------------|----------------|
| Intel AX200/210 | ✅ Excellent | ✅ Fast (<1s) | ✅ Accurate | **Best Choice** |
| Atheros AR9271 | ✅ Excellent | ✅ Fast (<1s) | ✅ Accurate | **Budget Option** |
| Broadcom BCM43xx | ⚠️ Limited | ⚠️ Slow (2-5s) | ⚠️ Variable | Avoid if possible |
| Realtek RTL88xx | ⚠️ Partial | ⚠️ Slow | ⚠️ Inaccurate | Not recommended |
| Apple Silicon | ❌ No | ❌ No | ❌ No | **Not Supported** |

### Ethernet NICs

| NIC Model | TDR Support | Cable Length | Fault Detection | Recommendation |
|-----------|-------------|--------------|-----------------|----------------|
| Intel I350 | ✅ Full | ✅ Yes | ✅ Distance to fault | **Best Choice** |
| Intel I210/I225-V | ✅ Full | ✅ Yes | ✅ Distance to fault | **Budget Option** |
| Broadcom BCM5719/5720 | ✅ Full | ✅ Yes | ✅ Distance to fault | Server-grade |
| Realtek RTL8111 | ❌ No | ❌ No | ❌ No | Basic only |

## Detailed Compatibility

For detailed compatibility reports by chipset:

- [Intel WiFi Adapters](Intel-WiFi)
- [Qualcomm Atheros Adapters](Qualcomm-Atheros-WiFi)
- [Broadcom WiFi Adapters](Broadcom-WiFi)
- [Realtek WiFi Adapters](Realtek-WiFi)
- [Intel Ethernet NICs](Intel-Ethernet)
- [Broadcom Ethernet NICs](Broadcom-Ethernet)
- [Realtek Ethernet NICs](Realtek-Ethernet)

## Testing Your Hardware

Run the compatibility test script:

```bash
curl -O https://raw.githubusercontent.com/krisarmstrong/netscope/main/scripts/test-hardware-compatibility.sh
chmod +x test-hardware-compatibility.sh
sudo ./test-hardware-compatibility.sh wlan0  # WiFi
sudo ./test-hardware-compatibility.sh eth0   # Ethernet
```

## Contributing

Submit your hardware test results to help the community!

[Create a Hardware Report Issue](https://github.com/krisarmstrong/netscope/issues/new?template=hardware-report.yml)

---

*See also: [HARDWARE.md](https://github.com/krisarmstrong/netscope/blob/main/HARDWARE.md)*
EOFHARDWARE

echo "  ✅ Hardware-Compatibility.md"

# FAQ
cat > FAQ.md << 'EOFFAQ'
# Frequently Asked Questions (FAQ)

## General

### What is The Seed?

The Seed is an AI-powered network diagnostic platform that combines WiFi planning, network monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

### Who is The Seed for?

- **Healthcare IT:** Hospitals, clinics (HIPAA compliance built-in)
- **SMB Network Admins:** Small/medium businesses (50-500 employees)
- **WiFi Consultants:** Site surveys and planning
- **MSPs:** Managed service providers

### Is The Seed free?

Yes! Free tier supports up to 50 devices with community support.

**Paid tiers:**
- Starter: $299/year (200 devices, AI classification)
- Professional: $799/year (unlimited devices, full AI)
- Premium: $1,999/year (predictive WiFi planning)
- Enterprise: $4,999/year (multi-site fleet management)

### What platforms are supported?

- ✅ macOS (Ventura 13.0+, Apple Silicon + Intel)
- ✅ Linux (Ubuntu 22.04+, Debian 11+, RHEL 8+)
- ✅ Docker (cross-platform)
- ❌ Windows (planned for future release)

## Installation & Setup

### Do I need admin/root privileges?

Yes, packet capture requires elevated privileges.

**macOS:** Run with `sudo seed`

**Linux:** Grant network capability:
```bash
sudo setcap cap_net_raw=+ep /usr/local/bin/seed
```

### Can I run The Seed on a server without a GUI?

Yes! The Seed is a web application:
- Backend runs on server (headless)
- Access UI from any browser on your network

## Features

### Can The Seed plan WiFi BEFORE a site survey?

**Yes!** This is unique to The Seed.

Upload a floor plan, place virtual APs, get instant coverage prediction. No competitor offers this.

### How accurate is predictive WiFi planning?

**85-95%** compared to real-world surveys.

Good enough for:
- Estimating AP count
- Budget planning
- Initial design

Not a replacement for final validation in complex RF environments.

### Does The Seed replace SolarWinds/PRTG?

**For SMBs and healthcare:** Yes! Covers 90% of use cases.

**For large enterprises (10,000+ devices):** No. The Seed targets SMBs, not Fortune 500.

## Security & Privacy

### Does The Seed send data to the cloud?

**No.** The Seed is **self-hosted** - all data stays on your network.

- No telemetry
- No usage tracking
- No cloud dependencies

### Is The Seed HIPAA compliant?

Yes! Designed for healthcare:
- Self-hosted (data never leaves your network)
- AES-256 encryption at rest
- TLS 1.3 in transit
- Audit logs
- Compliance report generation

## Troubleshooting

### The Seed isn't finding all devices. Why?

1. **Firewall blocking ICMP** - Enable TCP port scan
2. **Wrong subnet** - Verify your subnet configuration
3. **Insufficient timeout** - Increase to 60 seconds
4. **Permissions** - Run with sudo or grant capabilities

### Web UI won't load

1. Check if The Seed is running: `ps aux | grep seed`
2. Check if port 8080 is in use: `lsof -i :8080`
3. Check firewall: `sudo ufw status` (Ubuntu)

### Speed test fails

1. Check internet connection: `ping 8.8.8.8`
2. Check firewall settings
3. Disconnect VPN (if active)

## Support

### How do I get help?

- [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
- [GitHub Issues](https://github.com/krisarmstrong/netscope/issues)
- Email: support@mustardseednetworks.com

### How do I report a bug?

1. Check [existing issues](https://github.com/krisarmstrong/netscope/issues)
2. [Create new issue](https://github.com/krisarmstrong/netscope/issues/new)
3. Include: The Seed version, OS, steps to reproduce, logs

---

**Still have questions?**

Ask in [GitHub Discussions](https://github.com/krisarmstrong/netscope/discussions)
EOFFAQ

echo "  ✅ FAQ.md"

# Commit and push
echo ""
echo "📤 Committing wiki pages..."
git add .
git commit -m "docs: populate wiki with user-facing documentation

Created comprehensive user documentation:
- Home: Welcome page with quick links
- Installation-macOS: macOS installation guide
- Installation-Linux: Linux installation guide
- Quick-Start-Guide: 5-minute getting started guide
- Network-Discovery: Feature documentation
- Hardware-Compatibility: Tested hardware reference
- FAQ: Common questions and troubleshooting

All pages use correct naming (The Seed / Mustard Seed Networks)
Ready for beta users and internal testing"

echo "📤 Pushing to GitHub..."
git push origin master

echo ""
echo "✅ Wiki setup complete!"
echo ""
echo "🌐 View at: https://github.com/krisarmstrong/netscope/wiki"
echo ""
