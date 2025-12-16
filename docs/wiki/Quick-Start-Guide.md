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

   ```text
   Username: admin
   Password: ******** (choose strong password)
   ```

4. **Select Network Interface:**
   - The Seed auto-detects interfaces
   - Choose your primary network interface (e.g., `en0`, `eth0`)
   - Click "Next"

## Step 3: Login (30 seconds)

1. **Open Web UI:** `http://localhost:8080`

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

1. Click **"Settings"** then **"Compliance"**
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

- [Pricing & Features](https://github.com/krisarmstrong/seed/blob/main/docs/LICENSING_STRATEGY.md)
- Free tier: 50 devices, no AI
- Starter ($299/year): 200 devices, AI classification
- Professional ($799/year): Unlimited devices, full AI
- Premium ($1,999/year): Predictive WiFi planning
- Enterprise ($4,999/year): Multi-site fleet management

---

**Stuck? Need help?**

- Check [Common Issues](Troubleshooting#common-issues)
- Ask in [GitHub Discussions](https://github.com/krisarmstrong/seed/discussions)
- Email support@mustardseednetworks.com
