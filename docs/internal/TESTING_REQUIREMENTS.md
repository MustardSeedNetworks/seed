# The Seed - Testing Requirements (Hardware & Software)

**Version:** 1.0
**Last Updated:** December 2025
**Status:** Pre-Launch Testing Lab Specification

---

## Executive Summary

This document defines the hardware, software, and network infrastructure required to comprehensively test **The Seed** across realistic environments before launch and throughout development.

**Testing Goals:**
1. **Functional Testing:** Does every feature work correctly?
2. **Compatibility Testing:** Works on different hardware, OSes, network equipment?
3. **Performance Testing:** How does it scale? (10 devices vs 1,000 devices)
4. **Real-World Validation:** Does it work in actual hospital/office environments?
5. **Regression Testing:** Do new features break old features?

**Budget:**
- **Minimum Viable Lab:** $2,500 (core equipment only)
- **Recommended Lab:** $7,500 (realistic testing coverage)
- **Complete Lab:** $15,000 (enterprise-grade validation)

**Timeline:**
- **Phase 1 (Pre-Launch):** Minimum viable lab - Month 1
- **Phase 2 (Year 1):** Recommended lab - Month 3-6
- **Phase 3 (Year 2+):** Complete lab - as budget allows

---

## Table of Contents

1. [Testing Philosophy](#testing-philosophy)
2. [Hardware Requirements](#hardware-requirements)
3. [Software Requirements](#software-requirements)
4. [Network Topology](#network-topology)
5. [Test Scenarios](#test-scenarios)
6. [Progressive Lab Build](#progressive-lab-build)
7. [Alternative Testing Strategies](#alternative-testing-strategies)
8. [Testing Workflow](#testing-workflow)

---

## Testing Philosophy

### Testing Pyramid

**Level 1: Automated Unit Tests (80% of tests)**
- Fast, cheap, run on every commit
- Test individual functions (discovery logic, parsing, calculations)
- No hardware required (mocked responses)

**Level 2: Automated Integration Tests (15% of tests)**
- Test components together (API + database, frontend + backend)
- Some require real hardware (network interface access)
- Run before releases

**Level 3: Manual Real-World Testing (5% of tests)**
- Test on actual networks (hospital, office, home)
- Expensive, slow, but essential for validation
- Run before major releases

**This document focuses on Level 2 & 3 (hardware/software requirements).**

---

## Hardware Requirements

### Core Network Equipment

#### 1. Managed Switches (2 required)

**Purpose:**
- Test VLAN discovery
- Test SNMP queries (device fingerprinting)
- Test network segmentation (healthcare compliance)
- Test spanning tree, link aggregation

**Recommended Options:**

**Option A: Ubiquiti UniFi (Budget-Friendly)**
- **Model:** UniFi Switch 24 (USW-24-POE)
- **Price:** $379
- **Features:**
  - 24 ports (16 PoE+, 8 PoE++)
  - Layer 2/Layer 3 (VLANs, routing)
  - SNMP v2/v3
  - Web UI + CLI
  - PoE (can power APs, cameras, phones)
- **Why:** Affordable, popular in SMB, good for testing typical environments

**Option B: Cisco Catalyst (Enterprise Testing)**
- **Model:** Cisco Catalyst 1000-24P
- **Price:** $1,200
- **Features:**
  - 24 ports (PoE+)
  - Layer 2/Layer 3
  - SNMP, CLI
  - Enterprise-grade (test compatibility with common hospital switches)
- **Why:** Industry standard, better reflects enterprise environments

**Recommendation:** Start with Ubiquiti (Year 1), add Cisco (Year 2) for broader testing.

**Quantity:** 2 switches
- Switch 1: "Production" VLAN (office/clinic network)
- Switch 2: "Guest" VLAN (isolated network for testing segmentation)

---

#### 2. WiFi Access Points (3 required)

**Purpose:**
- Test WiFi scanning (SSID discovery, channel analysis)
- Test WiFi survey (signal strength, coverage mapping)
- Test predictive WiFi planning validation (compare prediction vs reality)
- Test different vendors (compatibility)

**Recommended Options:**

**Option A: Ubiquiti UniFi AP (Budget-Friendly)**
- **Model:** UniFi 6 Long-Range (U6-LR)
- **Price:** $179 each × 3 = $537
- **Features:**
  - WiFi 6 (802.11ax)
  - Dual-band (2.4GHz + 5GHz)
  - PoE powered (no separate power adapter)
  - Management via UniFi Controller
- **Why:** Popular, affordable, representative of SMB deployments

**Option B: Aruba Instant On (Enterprise Testing)**
- **Model:** Aruba Instant On AP22
- **Price:** $299 each × 3 = $897
- **Features:**
  - WiFi 6
  - Cloud-managed
  - Enterprise-grade
- **Why:** Common in healthcare (test compatibility)

**Option C: TP-Link Omada (Budget)**
- **Model:** TP-Link EAP660 HD
- **Price:** $149 each × 3 = $447
- **Features:**
  - WiFi 6
  - Cloud or on-prem controller
  - Very affordable
- **Why:** Cheapest option for basic testing

**Recommendation:** 2x Ubiquiti + 1x TP-Link (test multi-vendor environments)

**Placement:**
- AP 1: Office area (main testing location)
- AP 2: 30-50 feet away (test coverage, roaming)
- AP 3: Different floor/far location (test survey walk-through)

---

#### 3. Router/Firewall (1 required)

**Purpose:**
- Test gateway detection
- Test public IP detection
- Test traceroute
- Test speed testing (WAN throughput)

**Recommended Options:**

**Option A: Ubiquiti Dream Machine Pro (All-in-One)**
- **Model:** UDM-Pro
- **Price:** $379
- **Features:**
  - Router + firewall + controller (for UniFi APs/switches)
  - 1.7 Gbps routing throughput
  - 8-port switch built-in
  - VPN, traffic analysis, DPI
- **Why:** Ecosystem integration (if using UniFi APs/switches), one device does it all

**Option B: pfSense Box (Open Source)**
- **Hardware:** Netgate SG-3100
- **Price:** $399
- **Features:**
  - Open source pfSense firewall
  - Gigabit routing
  - VPN, traffic shaping, VLAN support
- **Why:** More configurable, common in advanced setups, free software

**Option C: Consumer Router (Minimal Budget)**
- **Model:** TP-Link Archer AX3000
- **Price:** $99
- **Features:**
  - WiFi 6 router
  - Gigabit WAN/LAN
  - Basic firewall
- **Why:** Cheapest option, represents home/small office

**Recommendation:** UDM-Pro (if using UniFi ecosystem) OR pfSense (if need flexibility)

---

#### 4. Test Devices (Simulating Real Networks)

**Purpose:**
- Test discovery (find different device types)
- Test fingerprinting (identify OS, manufacturer)
- Test vulnerability scanning (find weaknesses)

**Device Mix (Minimum 10 devices):**

**Computers:**
- 1x macOS laptop (testing + development)
- 1x Windows laptop (test Windows agent if needed)
- 1x Linux desktop/laptop (test Linux deployment)

**Mobile Devices:**
- 2x iPhones/iPads (test DHCP, WiFi survey on mobile)
- 2x Android phones/tablets

**IoT Devices:**
- 1x Network printer (common in offices/hospitals)
- 1x IP camera (common in healthcare for security)
- 1x Smart TV or streaming device (test consumer IoT)
- 1x VoIP phone (test PoE, VLAN tagging)

**Servers/NAS:**
- 1x Raspberry Pi (cheap Linux server for testing SSH, SNMP)
- 1x NAS (Synology or QNAP - test SMB, NFS shares)

**Total Device Count:** ~15 devices (good for initial testing)

**Cost:** $0-$500 (assume some devices already owned, buy a few IoT devices)

---

#### 5. Test Servers (Virtual or Physical)

**Purpose:**
- Test iPerf3 client/server mode
- Test SNMP targets
- Test vulnerability scanning targets
- Simulate hospital servers (PACS, EMR, etc.)

**Option A: Physical Server (If Budget Allows)**
- **Model:** Dell PowerEdge T40 (entry-level tower server)
- **Price:** $600
- **Specs:** Intel Xeon, 16GB RAM, 1TB HDD
- **Why:** Dedicated hardware, realistic testing, can run VMs

**Option B: High-End Workstation (Repurpose)**
- **Model:** Mac mini M2 Pro or refurbished Dell Precision
- **Price:** $1,000-$1,500
- **Specs:** 32GB RAM, 1TB SSD
- **Why:** Run multiple VMs (ESXi, Proxmox, VirtualBox)

**Option C: Cloud VMs (Pay-As-You-Go)**
- **Providers:** DigitalOcean, Linode, AWS EC2
- **Price:** $10-$50/month
- **Why:** No upfront cost, scalable, test remote scenarios

**Recommendation:** Start with VMs on existing hardware (Year 1), add dedicated server (Year 2)

---

### Additional Hardware (Nice-to-Have)

#### 6. Spectrum Analyzer (WiFi Advanced Testing)

**Purpose:**
- Test WiFi interference detection
- Validate WiFi survey accuracy (compare to professional tools)
- Test channel recommendations

**Options:**

**Option A: Wi-Spy DBx (Budget)**
- **Price:** $499
- **Features:** USB spectrum analyzer, 2.4GHz + 5GHz
- **Software:** Chanalyzer (Windows/Mac)
- **Why:** Cheapest dedicated spectrum analyzer

**Option B: Ekahau Sidekick (Gold Standard)**
- **Price:** $2,495
- **Features:** Professional spectrum analyzer + WiFi adapter
- **Why:** Industry standard, compare The Seed's survey to Ekahau

**Recommendation:** Skip for Year 1 (expensive), add Year 2 if serious about WiFi competition.

---

#### 7. Network Cable Tester

**Purpose:**
- Test cable diagnostics feature
- Validate TDR (Time Domain Reflectometry) accuracy
- Find cable faults

**Options:**

**Option A: Klein Tools VDV Scout Pro 3 (Budget)**
- **Price:** $299
- **Features:** Cable mapping, length measurement, PoE detection
- **Why:** Affordable, validates basic cable testing

**Option B: Fluke LinkRunner G2 (Professional)**
- **Price:** $2,995
- **Features:** Advanced cable testing, switch info, PoE load testing
- **Why:** Gold standard for cable testing

**Recommendation:** Skip for Year 1 (cable diagnostics not critical feature), add if demand grows.

---

### Hardware Summary & Budget

| Item | Quantity | Unit Cost | Total | Priority |
|------|----------|-----------|-------|----------|
| **Managed Switch (Ubiquiti)** | 2 | $379 | $758 | **Essential** |
| **WiFi Access Points (Ubiquiti)** | 2 | $179 | $358 | **Essential** |
| **WiFi Access Point (TP-Link)** | 1 | $149 | $149 | **Essential** |
| **Router (UDM-Pro)** | 1 | $379 | $379 | **Essential** |
| **Test Devices (IoT, phones, etc.)** | ~5 | $100 | $500 | **Essential** |
| **Raspberry Pi (test server)** | 2 | $75 | $150 | Recommended |
| **NAS (Synology DS220+)** | 1 | $299 | $299 | Recommended |
| **Cisco Switch (enterprise testing)** | 1 | $1,200 | $1,200 | Nice-to-have |
| **Spectrum Analyzer (Wi-Spy DBx)** | 1 | $499 | $499 | Nice-to-have |
| **Network Cable Tester** | 1 | $299 | $299 | Nice-to-have |
| | | **Total (Essential):** | **$2,144** | |
| | | **Total (Recommended):** | **$2,593** | |
| | | **Total (Complete Lab):** | **$4,491** | |

**Notes:**
- Does not include devices already owned (laptops, phones, etc.)
- Add $1,000-$2,000 for server hardware (or use existing hardware + VMs)
- Add $2,495 for Ekahau Sidekick (if serious WiFi validation needed)

---

## Software Requirements

### Operating Systems (Test Compatibility)

**Desktop/Laptop OSes:**
- **macOS:** Ventura (13.x), Sonoma (14.x), Sequoia (15.x)
- **Linux:**
  - Ubuntu 22.04 LTS, 24.04 LTS
  - Debian 11, 12
  - RHEL/AlmaLinux 8, 9
  - Arch Linux (bleeding edge testing)
- **Windows:** 11, Server 2022 (for future Windows agent)

**Mobile OSes (for WiFi survey mobile app - future):**
- iOS 16, 17, 18
- Android 12, 13, 14

**Server OSes (for iPerf, SNMP targets):**
- Ubuntu Server 22.04 LTS
- Debian 12
- CentOS/AlmaLinux 9

---

### Network Simulation Software

#### 1. GNS3 (Graphical Network Simulator)

**Purpose:**
- Simulate complex network topologies (hundreds of devices)
- Test routing, VLANs, ACLs without buying physical equipment
- Test at scale (1,000+ devices)

**Cost:** Free (open source)

**Requirements:**
- 16GB RAM minimum (32GB recommended)
- CPU with virtualization support (Intel VT-x or AMD-V)

**Use Cases:**
- Test discovery on 500-device network
- Test routing protocols (OSPF, BGP)
- Test network segmentation (healthcare VLANs)

**Alternatives:**
- **Cisco Packet Tracer:** Free, simpler, less powerful (good for basic testing)
- **EVE-NG:** More powerful than GNS3, commercial ($0-$399/year)

---

#### 2. Virtualization (VMs for Test Targets)

**Options:**

**Option A: VirtualBox (Free)**
- **Cost:** Free
- **Platforms:** macOS, Linux, Windows
- **Use:** Run Linux/Windows VMs for testing
- **Cons:** Slower than native hypervisors

**Option B: VMware Fusion (macOS) / Workstation (Windows/Linux)**
- **Cost:** $199 (Fusion), $199 (Workstation) - or free for personal use
- **Use:** Run multiple test VMs (Ubuntu, Windows Server, etc.)
- **Pros:** Fast, stable, good networking options

**Option C: Proxmox (Dedicated Server)**
- **Cost:** Free (open source)
- **Use:** Turn physical server into VM host
- **Pros:** Enterprise-grade, web UI, supports VMs and containers

**Recommendation:** VirtualBox (Year 1), VMware Fusion (if budget allows), Proxmox (Year 2+ on dedicated server)

---

### Testing Tools

#### 1. Network Traffic Generators

**iPerf3 (Bandwidth Testing)**
- **Cost:** Free
- **Use:** Test speed test feature, validate throughput calculations

**hping3 (Packet Generation)**
- **Cost:** Free
- **Use:** Test discovery under load, stress testing

**Ostinato (Packet Crafting)**
- **Cost:** Free (open source)
- **Use:** Create custom network traffic for edge case testing

---

#### 2. Network Monitoring (Baseline Comparison)

**Wireshark (Packet Capture)**
- **Cost:** Free
- **Use:** Debug discovery issues, validate packet parsing

**tcpdump (CLI Packet Capture)**
- **Cost:** Free
- **Use:** Capture packets for analysis, troubleshooting

**nmap (Network Scanner)**
- **Cost:** Free
- **Use:** Baseline comparison (does The Seed find same devices as nmap?)

---

#### 3. Vulnerability Testing

**Nessus Essentials (Free Tier)**
- **Cost:** Free (up to 16 IPs)
- **Use:** Baseline comparison (does The Seed find same vulnerabilities as Nessus?)

**OpenVAS (Open Source Scanner)**
- **Cost:** Free
- **Use:** Generate vulnerable targets, validate vulnerability detection

**Metasploit (Penetration Testing)**
- **Cost:** Free (community edition)
- **Use:** Test vulnerability detection accuracy, exploit validation

---

#### 4. WiFi Analysis Tools

**NetSpot (WiFi Survey)**
- **Cost:** Free (limited), $49 (Home), $299 (Pro)
- **Use:** Baseline comparison for WiFi surveys

**WiFi Analyzer (Android)**
- **Cost:** Free
- **Use:** Quick WiFi scanning comparison

**inSSIDer (Windows/Mac)**
- **Cost:** Free (older version), $29 (Office)
- **Use:** WiFi channel analysis, interference detection

---

#### 5. Compliance Testing

**Lynis (Security Auditing)**
- **Cost:** Free
- **Use:** Test compliance scanning accuracy (compare to established tool)

**OpenSCAP (Security Compliance)**
- **Cost:** Free
- **Use:** HIPAA/PCI baseline comparison

---

### Development & CI Tools (Already in Use)

- **GitHub Actions:** CI/CD, automated testing
- **Playwright:** E2E browser testing
- **golangci-lint:** Go linting
- **gosec:** Go security scanning
- **gitleaks:** Secret detection
- **Sentry:** Error tracking

---

## Network Topology

### Test Lab Layout

```
Internet (WAN)
      |
      |
  [UDM-Pro Router]
      |
      |-------------------------------------
      |                                    |
 [UniFi Switch 1]                  [UniFi Switch 2]
 (VLAN 10: Office)                 (VLAN 20: Guest)
      |                                    |
      |                                    |
  [8 ports used]                      [4 ports used]
      |                                    |
      |--- AP1 (Office)                    |--- AP3 (Guest WiFi)
      |--- AP2 (Conference Room)           |--- IoT Devices (isolated)
      |--- Mac mini (Dev/Test)             |
      |--- Raspberry Pi 1 (iPerf server)   |
      |--- Raspberry Pi 2 (SNMP target)    |
      |--- NAS (file server)               |
      |--- IP Camera                       |
      |--- VoIP Phone                      |
```

**VLAN Segmentation:**
- **VLAN 10 (Office):** Workstations, servers, APs, printers
- **VLAN 20 (Guest):** Guest WiFi, IoT devices (isolated from office)
- **VLAN 30 (Management):** Switch/AP management interfaces (future)

**IP Addressing:**
- **VLAN 10:** 192.168.10.0/24
- **VLAN 20:** 192.168.20.0/24
- **VLAN 30:** 192.168.30.0/24

**Testing Capabilities:**
- ✅ VLAN discovery (across 2+ VLANs)
- ✅ Network segmentation compliance (healthcare requirement)
- ✅ Multi-vendor equipment (Ubiquiti + TP-Link + various IoT)
- ✅ WiFi coverage testing (3 APs in different locations)
- ✅ SNMP queries (switches, APs, NAS)
- ✅ iPerf throughput testing (client/server mode)
- ✅ Vulnerability scanning (various OS targets)

---

## Test Scenarios

### Scenario 1: Small Office (50 Devices)

**Environment:**
- 1 managed switch
- 2 WiFi APs
- 30 workstations (simulated via VMs)
- 5 printers
- 5 IP cameras
- 5 VoIP phones
- 5 mobile devices

**Tests:**
- Discovery time: <60 seconds
- Accuracy: 100% of devices found
- Fingerprinting: Correct OS/vendor for 80%+ devices
- WiFi survey: Generate heatmap, recommend AP placement
- Compliance: HIPAA report generated, no critical findings

---

### Scenario 2: Medium Hospital (200 Devices)

**Environment:**
- 2 VLANs (office + medical devices)
- 5 WiFi APs
- 100 workstations
- 20 medical devices (PACS, EMR terminals)
- 20 printers
- 30 IP cameras (hallways, rooms)
- 30 VoIP phones

**Tests:**
- Discovery time: <5 minutes
- VLAN segmentation detected: 2 VLANs found
- Rogue DHCP detection: Identify rogue server on guest VLAN
- Vulnerability scan: Find 20+ vulnerabilities (test VMs with known vulns)
- WiFi coverage: 95%+ coverage in patient areas (simulated floor plan)

---

### Scenario 3: Large Enterprise (1,000+ Devices)

**Environment:**
- Simulated via GNS3 (virtual routers, switches, devices)
- 10 VLANs
- 50 subnets
- 1,000 virtual devices

**Tests:**
- Discovery time: <30 minutes
- Memory usage: <2GB RAM
- CPU usage: <50% (during active scan)
- Accuracy: 95%+ devices discovered
- Performance: UI remains responsive during scan

---

### Scenario 4: Edge Cases & Stress Testing

**Tests:**
- **No network access:** Graceful error (not crash)
- **Offline mode:** Can view cached results
- **Invalid SNMP credentials:** Clear error message
- **Huge subnet (255.255.0.0):** Doesn't crash, warns about time
- **Network change during scan:** Handles interface switching
- **Concurrent scans:** Multiple users on same network
- **Malformed packets:** Doesn't crash on bad data

---

## Progressive Lab Build

### Phase 1: Minimum Viable Lab ($2,500 budget, Month 1)

**Goal:** Test core features before launch

**Equipment:**
- 2x Ubiquiti switches ($758)
- 2x Ubiquiti APs + 1x TP-Link AP ($507)
- 1x UDM-Pro router ($379)
- 2x Raspberry Pi ($150)
- IoT devices (printer, camera, phone) ($500)
- NAS ($299)

**Total:** ~$2,593

**Capabilities:**
- WiFi survey and planning
- Network discovery
- VLAN detection
- SNMP fingerprinting
- Basic vulnerability scanning

**What's Missing:**
- Large-scale testing (100+ devices)
- Enterprise hardware (Cisco, Aruba)
- Spectrum analysis
- Multiple physical locations

---

### Phase 2: Recommended Lab ($5,000 additional, Month 3-6)

**Add:**
- 1x Cisco switch ($1,200) - enterprise compatibility testing
- 1x Mac mini M2 Pro ($1,500) - VM host for 20+ virtual test devices
- 1x Synology NAS upgrade ($300) - larger storage for logs, test data
- Various test devices ($500) - more IoT, medical devices (eBay/used)
- GNS3 setup ($0 - free) - simulate large networks
- NetSpot Pro ($299) - WiFi baseline comparison
- Nessus Professional trial ($0 - 7 day trial) - vulnerability comparison

**Total Phase 1+2:** ~$7,593

**New Capabilities:**
- Enterprise equipment testing
- 100+ device testing (via VMs)
- WiFi accuracy validation (vs NetSpot)
- Vulnerability accuracy validation (vs Nessus)

---

### Phase 3: Complete Lab ($10,000 additional, Year 2)

**Add:**
- 1x Aruba AP ($900) - healthcare vendor compatibility
- 1x Dedicated server (Dell PowerEdge) ($1,500) - physical server for VMs
- 1x Ekahau Sidekick ($2,495) - WiFi gold standard comparison
- 1x Fluke LinkRunner G2 ($2,995) - cable testing validation
- 1x Wi-Spy DBx ($499) - spectrum analysis
- Various medical devices ($1,000) - used PACS terminals, nurse stations (eBay)

**Total Phase 1+2+3:** ~$17,593

**New Capabilities:**
- Direct Ekahau comparison (prove we're as accurate)
- Real medical device testing (compliance)
- Professional spectrum analysis
- Advanced cable diagnostics
- Full enterprise validation

---

## Alternative Testing Strategies

### Strategy 1: Customer Beta Testing (Free)

**How:**
- Recruit 10-20 beta customers (healthcare, SMB)
- They test in their real environments
- Provide feedback via Discord/GitHub

**Pros:**
- Free (no lab hardware cost)
- Real-world validation (actual hospitals, offices)
- Build relationships with early customers

**Cons:**
- Less control (can't reproduce bugs easily)
- Slower feedback loop
- Depends on customer willingness

**Recommendation:** Do this IN ADDITION to lab testing (not instead of)

---

### Strategy 2: Cloud-Based Testing (Pay-As-You-Go)

**How:**
- Spin up VMs on AWS, DigitalOcean, Linode
- Create virtual networks (VPCs, subnets)
- Simulate devices via Docker containers

**Pros:**
- No upfront hardware cost
- Scalable (test 10 devices or 10,000)
- Accessible from anywhere

**Cons:**
- Monthly costs ($100-$500/month)
- Can't test WiFi (no physical APs in cloud)
- Can't test physical devices (printers, cameras)

**Recommendation:** Use for scale testing (1,000+ devices), not primary lab

---

### Strategy 3: Partner with Testing Lab

**How:**
- Partner with network testing facility (e.g., university, enterprise IT lab)
- Pay for time or trade for free license

**Pros:**
- Access to enterprise equipment (Cisco, Aruba, Juniper)
- Large-scale testing (thousands of devices)
- Professional validation

**Cons:**
- Hard to find partners
- Scheduling challenges
- May require travel

**Recommendation:** Pursue for Year 2 (after product proven)

---

## Testing Workflow

### Pre-Release Testing Checklist

**1. Unit Tests (Automated)**
- [ ] All unit tests pass (`make test-backend`, `make test-frontend`)
- [ ] Code coverage >80% for new code
- [ ] No linting errors (`make lint`)

**2. Integration Tests (Automated + Manual)**
- [ ] Playwright E2E tests pass (`make test-e2e`)
- [ ] API endpoints tested (Postman/Insomnia collection)
- [ ] WebSocket connectivity tested (real-time updates work)

**3. Hardware Lab Tests (Manual)**
- [ ] Network discovery: Finds all 15 test devices
- [ ] VLAN discovery: Detects VLAN 10 and VLAN 20
- [ ] SNMP: Fingerprints switches and APs correctly
- [ ] WiFi survey: Generates heatmap for 3-AP office
- [ ] iPerf: Speed test runs successfully (client and server modes)
- [ ] Vulnerability scan: Finds known vulnerabilities on test VMs
- [ ] Compliance report: Generates HIPAA report with no critical issues

**4. Compatibility Tests (Manual)**
- [ ] macOS: Ventura, Sonoma, Sequoia
- [ ] Linux: Ubuntu 22.04, 24.04, Debian 12
- [ ] Browsers: Chrome, Firefox, Safari, Edge (for web UI)

**5. Performance Tests (Automated + Manual)**
- [ ] Discovery: 50 devices in <60 seconds
- [ ] Memory usage: <1GB during active scan
- [ ] CPU usage: <50% on average hardware
- [ ] UI responsiveness: No lag during scan

**6. Security Tests**
- [ ] gosec: No security warnings
- [ ] gitleaks: No secrets detected
- [ ] npm audit: No high/critical vulnerabilities
- [ ] Manual penetration test: Basic SQL injection, XSS, CSRF tested

**7. Real-World Validation (Beta Customers)**
- [ ] 3+ beta customers tested in real environments
- [ ] No critical bugs reported
- [ ] CSAT >4/5 for beta testers

---

### Continuous Testing (Post-Launch)

**Daily (Automated):**
- Unit tests on every commit (GitHub Actions)
- Linting on every PR (pre-commit hooks)

**Weekly (Manual):**
- Run full E2E test suite on lab hardware
- Test new features on 3 OSes (macOS, Ubuntu, Debian)
- Review error logs from Sentry (any crashes?)

**Monthly (Manual):**
- Full compatibility matrix (all supported OS versions)
- Performance benchmarks (has performance regressed?)
- Security scans (gosec, npm audit, manual pen-testing)

**Quarterly (Manual + Customer):**
- Large-scale testing (GNS3 with 500+ devices)
- Beta testing with 5-10 customers (new features)
- Competitive comparison (test vs Ekahau, SolarWinds - are we still better?)

---

## Appendix: Vendor Contact Info

### Network Equipment

**Ubiquiti:**
- Website: https://store.ui.com
- Resellers: Amazon, B&H Photo, Newegg

**Cisco:**
- Website: https://www.cisco.com
- Resellers: CDW, Insight, SHI (enterprise resellers)
- Used/Refurb: eBay (Cisco equipment retains value)

**TP-Link:**
- Website: https://www.tp-link.com
- Resellers: Amazon, Best Buy

**Aruba (HPE):**
- Website: https://www.arubanetworks.com
- Resellers: CDW, Insight

---

### Test Equipment

**Ekahau:**
- Website: https://www.ekahau.com
- Price: Contact sales (no online pricing)

**NetAlly:**
- Website: https://www.netally.com
- Resellers: Amazon, CDW

**Fluke Networks:**
- Website: https://www.flukenetworks.com
- Resellers: CDW, Graybar

---

### Software

**GNS3:**
- Website: https://www.gns3.com
- Cost: Free (download)

**Proxmox:**
- Website: https://www.proxmox.com
- Cost: Free (open source), $90/year (support subscription)

**Nessus:**
- Website: https://www.tenable.com/products/nessus
- Cost: Free (Essentials, 16 IPs), $4,620/year (Professional)

---

## Revision History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | Dec 2025 | Initial testing requirements | Kris Armstrong |

---

**Document Owner:** Kris Armstrong, Founder
**Last Reviewed:** December 2025
**Next Review:** After Phase 1 lab setup (Month 1), then quarterly

---

*Test early, test often, test real.*

**Mustard Seed Networks**
*From a tiny seed, a mighty network grows.*
