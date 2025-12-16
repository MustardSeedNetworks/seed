# Frequently Asked Questions (FAQ)

## General

### What is The Seed?

The Seed is an AI-powered network diagnostic platform that combines WiFi planning, network
monitoring, vulnerability scanning, and compliance reporting in one affordable tool.

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

[View Pricing Details](https://github.com/krisarmstrong/seed/blob/main/docs/LICENSING_STRATEGY.md)

### Is The Seed open source?

Yes! The Seed is open source under the AGPL-3.0 license.

- Source code: <https://github.com/krisarmstrong/seed>
- Contributions welcome: [Contributing Guide](Contributing)

---

## Installation and Setup

### What platforms does The Seed support?

**Supported:**

- macOS (Ventura 13.0+, Apple Silicon + Intel)
- Linux (Ubuntu 22.04+, Debian 11+, RHEL 8+)
- Docker (cross-platform)

**Not supported (yet):**

- Windows (planned for future release)
- Mobile (iOS/Android - planned for WiFi survey mobile app)

### Do I need to be root/admin to run The Seed?

**macOS:** Yes, packet capture requires elevated privileges. Run with `sudo seed` or grant network
access permissions.

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

**For large enterprises (10,000+ devices):** No. SolarWinds, Datadog, etc. are designed for massive
scale and deep integrations. The Seed targets SMBs, not Fortune 500.

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

## Security and Privacy

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

**Note:** You are responsible for your overall HIPAA compliance. The Seed is a tool to _help_ you
comply, not a complete solution.

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
   - **Fix:** Enable TCP port scan (Settings then Discovery then Methods then TCP)

2. **Wrong subnet:** Scanning 192.168.0.0/24 but devices are on 192.168.1.0/24
   - **Fix:** Check your subnet (`ip addr` on Linux, `ifconfig` on macOS)

3. **Insufficient timeout:** Some devices respond slowly.
   - **Fix:** Increase timeout (Settings then Discovery then Timeout then 60 seconds)

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
   - Local: `http://localhost:8080`
   - Remote: `http://<server-ip>:8080`

### Speed test fails / shows 0 Mbps. Why?

**Common causes:**

1. **No internet connection:** Speed test requires internet.
   - **Fix:** Check gateway: `ping 8.8.8.8`

2. **Firewall blocking:** ISP or firewall blocking speed test servers.
   - **Fix:** Temporarily disable firewall or allow speed test domains

3. **VPN active:** Speed test measures VPN speed, not actual internet.
   - **Fix:** Disconnect VPN and retest

### How do I reset the admin password?

#### Option 1: Reset via CLI (if you have shell access)

```bash
seed reset-password admin
# Enter new password when prompted
```

#### Option 2: Reset via config file

1. Stop The Seed: `sudo systemctl stop seed`
2. Delete user database: `rm ~/.config/seed/users.db`
3. Start The Seed: `sudo systemctl start seed`
4. Run setup wizard again (creates new admin user)

#### Option 3: Reset all data (nuclear option)

```bash
sudo systemctl stop seed
rm -rf ~/.config/seed
sudo systemctl start seed
```

**Warning:** This deletes all scan history, settings, and user accounts.

---

## Licensing and Pricing

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

1. Deactivate license on old server (Settings then License then Deactivate)
2. Activate license on new server (enter license key during setup)

**Limitations:**

- One active instance per license (can't run on 2 servers simultaneously)
- Enterprise tier allows multi-site (multiple active instances)

---

## Support

### How do I get help?

**Free tier:**

- Community forum (Discord, TBD)
- GitHub Discussions: <https://github.com/krisarmstrong/seed/discussions>
- Documentation: This wiki

**Paid tiers:**

- Email support: support@mustardseednetworks.com
- Priority support (Professional+)
- Phone support (Premium+)
- Dedicated TAM (Enterprise)

[View Support Tiers](https://github.com/krisarmstrong/seed/blob/main/docs/SUPPORT_STRATEGY.md)

### How do I report a bug?

1. **Check existing issues:** <https://github.com/krisarmstrong/seed/issues>
2. **Create new issue** (if not already reported):
   - Use
     [Bug Report template](https://github.com/krisarmstrong/seed/issues/new?template=bug_report.md)
   - Include: The Seed version, OS, steps to reproduce, logs
3. **Expected response time:**
   - Free tier: Best-effort (community)
   - Paid tiers: 4-24 hours (depending on tier)

### How do I request a feature?

1. **Check existing requests:** <https://github.com/krisarmstrong/seed/discussions/categories/ideas>
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

[Security Policy](https://github.com/krisarmstrong/seed/blob/main/SECURITY.md)

---

## Comparison to Competitors

### The Seed vs Ekahau

| Feature                 | The Seed     | Ekahau AI Pro       |
| ----------------------- | ------------ | ------------------- |
| **Price**               | $1,999/year  | $5,198 (year 1)     |
| **Predictive Planning** | Yes          | No                  |
| **WiFi Survey**         | Yes          | Yes (more accurate) |
| **Network Diagnostics** | Yes          | No                  |
| **Platforms**           | macOS, Linux | Windows only        |

**When to choose The Seed:** Predictive planning, budget-conscious, need network diagnostics too

**When to choose Ekahau:** Large consultant firm, need extreme accuracy, clients expect Ekahau
branding

### The Seed vs SolarWinds

| Feature            | The Seed              | SolarWinds NPM         |
| ------------------ | --------------------- | ---------------------- |
| **Price**          | $799/year (unlimited) | $10,000+ (100 devices) |
| **Target**         | SMB, healthcare       | Enterprise IT          |
| **Setup Time**     | 30 minutes            | Weeks                  |
| **AI Diagnostics** | Yes                   | No                     |

**When to choose The Seed:** SMB, healthcare, need simple + affordable

**When to choose SolarWinds:** Enterprise (10,000+ devices), need deep integrations

### The Seed vs Nessus

| Feature                    | The Seed              | Nessus Professional |
| -------------------------- | --------------------- | ------------------- |
| **Price**                  | $1,999/year (Premium) | $4,620/year         |
| **Vulnerability Scanning** | Yes (10K+ checks)     | Yes (170K+ checks)  |
| **Network Diagnostics**    | Yes                   | No                  |
| **WiFi Planning**          | Yes                   | No                  |

**When to choose The Seed:** Need all-in-one (network + WiFi + security)

**When to choose Nessus:** Need comprehensive vulnerability database (security-first)

[Full Competitive Analysis](https://github.com/krisarmstrong/seed/blob/main/docs/COMPETITIVE_ANALYSIS.md)

---

**Still have questions?**

- Ask in [GitHub Discussions](https://github.com/krisarmstrong/seed/discussions)
- Email support@mustardseednetworks.com
- Join our Discord (TBD)
