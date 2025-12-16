# Frequently Asked Questions (FAQ)

## General

### What is The Seed?

The Seed is an AI-powered network diagnostic platform that combines WiFi planning, network
monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

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
