# LuminetIQ Compliance Framework Mappings

**Document Version:** 1.0 **Last Updated:** 2025-12-15 **Status:** Validated Mappings

---

## Overview

This document provides **validated, non-marketing mappings** between LuminetIQ features and major
compliance frameworks. Each mapping includes:

- **Control requirement** (what the framework mandates)
- **LuminetIQ capability** (how the product addresses it)
- **Evidence/Audit value** (what auditors can verify)
- **Implementation notes** (how to configure for compliance)

**Frameworks Covered:**

- CIS Controls v8 (Center for Internet Security)
- NIST Cybersecurity Framework (CSF) v1.1
- NIST SP 800-53 Rev 5 (Federal/DoD)
- HIPAA Security Rule (Healthcare)
- CMMC Level 2 (Defense contractors)

---

## CIS Controls v8 Mappings

### CIS Control 1: Inventory and Control of Enterprise Assets

**Requirement:**

> "Actively manage (inventory, track, and correct) all enterprise assets (end-user devices including
> portable and mobile; network devices; non-computing/IoT devices; and servers) connected to the
> infrastructure physically, virtually, remotely, and those within cloud environments, to accurately
> know the totality of assets that need to be monitored and protected within the enterprise."

#### LuminetIQ Capabilities:

| Sub-Control                                                        | LuminetIQ Feature                                                                                                    | Evidence/Audit Value                                                                                      |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **1.1** Establish and maintain detailed enterprise asset inventory | • Device discovery (ARP, NDP, ICMP, mDNS)<br>• AI device classification<br>• Device inventory with MAC, IP, hostname | • Automated inventory export (CSV/JSON)<br>• Timestamp of discovery<br>• Classification confidence scores |
| **1.2** Address unauthorized assets                                | • Rogue device detection<br>• New device alerting<br>• Whitelist/quarantine actions                                  | • Rogue device alerts log<br>• Quarantine action audit trail<br>• Time-to-detection metrics               |
| **1.3** Utilize an active discovery tool                           | • Continuous network scanning<br>• Multi-protocol discovery<br>• Configurable scan intervals                         | • Discovery service logs<br>• Scan frequency configuration<br>• Coverage reports                          |
| **1.4** Use dynamic host configuration (DHCP) logging              | • DHCP monitoring and timing<br>• DHCP phase analysis<br>• Lease tracking                                            | • DHCP transaction logs<br>• IP allocation history<br>• DHCP server detection                             |

**Implementation Notes:**

- Configure discovery profile to "standard" or "full_scan" for compliance
- Enable rogue device detection and alerting
- Export device inventory weekly for compliance records
- Enable DHCP monitoring on all active interfaces

**Audit Evidence:**

```bash
# Generate CIS Control 1 compliance report
luminetiq export --format json --filter "devices,discovery,dhcp" > cis-control-1-$(date +%Y%m%d).json

# Export device inventory for auditor
luminetiq devices list --format csv --include "ip,mac,hostname,first_seen,device_type,classification_confidence"
```

---

### CIS Control 2: Inventory and Control of Software Assets

**Requirement:**

> "Actively manage (inventory, track, and correct) all software (operating systems and applications)
> on the network so that only authorized software is installed and can execute, and that
> unauthorized and unmanaged software is found and prevented from installation or execution."

#### LuminetIQ Capabilities:

| Sub-Control                                       | LuminetIQ Feature                                                                                            | Evidence/Audit Value                                                                      |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- |
| **2.1** Establish and maintain software inventory | • OS detection via fingerprinting<br>• Service/port discovery<br>• Version detection (HTTP headers, banners) | • Software inventory reports<br>• OS distribution statistics<br>• Service version matrix  |
| **2.3** Address unauthorized software             | • Unexpected service detection<br>• Port scan detection<br>• Behavior anomaly detection                      | • Unauthorized service alerts<br>• Anomaly detection logs<br>• Port scan incident reports |
| **2.6** Allowlist authorized software             | • Expected device profiles<br>• Behavioral baselines<br>• Service whitelisting                               | • Baseline configuration exports<br>• Profile deviation reports                           |

**Implementation Notes:**

- Enable device profiling and fingerprinting
- Configure expected service baselines per device type
- Review "unexpected services" alerts weekly

---

### CIS Control 3: Data Protection

**Requirement:**

> "Develop processes and technical controls to identify, classify, securely handle, retain, and
> dispose of data."

#### LuminetIQ Capabilities:

| Sub-Control                                 | LuminetIQ Feature                                                                                  | Evidence/Audit Value                                                                        |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **3.3** Configure data access control lists | • VLAN detection and verification<br>• Network segmentation mapping<br>• Subnet isolation analysis | • VLAN topology maps<br>• Segmentation compliance reports<br>• Traffic flow analysis        |
| **3.10** Encrypt sensitive data in transit  | • TLS certificate inspection<br>• Encryption protocol detection<br>• Unencrypted protocol alerts   | • TLS compliance reports<br>• Weak cipher detection logs<br>• Unencrypted service inventory |

**Implementation Notes:**

- Configure VLAN monitoring for PHI/PCI networks
- Enable TLS inspection and weak cipher alerts
- Document network segmentation in compliance reports

**Healthcare-Specific:** For HIPAA §164.312(e)(1) Transmission Security, use LuminetIQ to:

- Verify PHI networks are on isolated VLANs
- Detect unencrypted medical device traffic
- Monitor for unauthorized VLAN hopping

---

### CIS Control 4: Secure Configuration of Enterprise Assets and Software

**Requirement:**

> "Establish and maintain the secure configuration of enterprise assets (end-user devices, including
> portable and mobile; network devices; non-computing/IoT devices; and servers) and software
> (operating systems and applications)."

#### LuminetIQ Capabilities:

| Sub-Control                                                 | LuminetIQ Feature                                                                             | Evidence/Audit Value                                                                           |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| **4.1** Establish and maintain secure configuration process | • Baseline learning<br>• Configuration drift detection (fleet)<br>• Expected state monitoring | • Baseline configuration exports<br>• Drift detection reports<br>• Remediation recommendations |
| **4.7** Manage default accounts                             | • Default credential detection (SNMP)<br>• Common service fingerprinting                      | • Default credential alerts<br>• Insecure service inventory                                    |

**Implementation Notes:**

- Enable baseline learning for 30 days before enforcement
- Configure fleet management for multi-site drift detection
- Export configuration baselines for compliance documentation

---

### CIS Control 7: Continuous Vulnerability Management

**Requirement:**

> "Develop a plan to continuously assess and track vulnerabilities on all enterprise assets within
> the enterprise's infrastructure, to remediate, and minimize the window of opportunity for
> attackers."

#### LuminetIQ Capabilities:

| Sub-Control                                                     | LuminetIQ Feature                                                                                                   | Evidence/Audit Value                                                                               |
| --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| **7.1** Establish and maintain vulnerability management process | • Automated CVE scanning<br>• NVD integration<br>• Continuous vulnerability monitoring                              | • Vulnerability scan reports<br>• CVE detection timeline<br>• Scan frequency logs                  |
| **7.2** Establish and maintain remediation process              | • **Contextual risk scoring** (CVSS + EPSS + exposure)<br>• Prioritized remediation plans<br>• Remediation tracking | • Risk-scored vulnerability lists<br>• Remediation priority reports<br>• Patch compliance tracking |
| **7.3** Perform automated OS patch management                   | • OS version tracking<br>• Outdated system detection<br>• Patch availability alerts                                 | • OS inventory with versions<br>• Patch status reports                                             |
| **7.4** Perform automated application patch management          | • Service version detection<br>• Vulnerable service alerts<br>• Firmware version tracking                           | • Application version matrix<br>• Vulnerable service inventory                                     |
| **7.5** Perform automated vulnerability scans                   | • Continuous network scanning<br>• Scheduled vulnerability assessment<br>• Real-time CVE matching                   | • Scan schedule configuration<br>• Scan completion logs<br>• Time-to-detection metrics             |

**Implementation Notes:**

- Configure vulnerability scanning to run daily or weekly
- Enable EPSS (Exploitability Prediction) for prioritization
- Export risk-scored reports for vulnerability management meetings
- Set up alerts for critical vulnerabilities (CVSS ≥9.0)

**Key Differentiator:** LuminetIQ doesn't just list CVEs - it provides **contextual risk scoring**:

```
CVE-2024-1234 on 192.168.1.10 (IP Camera, DMZ):
- CVSS: 9.8 (Critical)
- EPSS: 0.89 (89% probability of exploitation)
- Exposure: Public-facing (DMZ)
- Risk Score: 94/100 (CRITICAL)
- Remediation: Update firmware to v2.3.1 OR block port 80
```

---

### CIS Control 12: Network Infrastructure Management

**Requirement:**

> "Establish, implement, and actively manage (track, report, correct) network devices, in order to
> prevent attackers from exploiting vulnerable network services and access points."

#### LuminetIQ Capabilities:

| Sub-Control                                                          | LuminetIQ Feature                                                                                     | Evidence/Audit Value                                                                        |
| -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **12.1** Ensure network infrastructure is up-to-date                 | • Network device discovery<br>• Firmware version detection<br>• Outdated infrastructure alerts        | • Network device inventory<br>• Firmware compliance reports                                 |
| **12.2** Establish and maintain secure configuration                 | • Switch/router discovery (LLDP, CDP)<br>• Network topology mapping<br>• Configuration baseline       | • Network topology diagrams<br>• Switch port mappings<br>• Uplink detection logs            |
| **12.4** Establish and maintain network segmentation                 | • **VLAN detection and verification**<br>• Subnet isolation analysis<br>• Voice VLAN detection        | • VLAN topology maps<br>• Segmentation compliance reports<br>• Cross-VLAN traffic detection |
| **12.6** Use of secure protocols                                     | • TLS/SSL inspection<br>• Insecure protocol detection (Telnet, FTP, HTTP)<br>• Weak cipher alerts     | • Secure protocol compliance<br>• Insecure service inventory                                |
| **12.7** Ensure remote access uses MFA                               | • Remote access service detection<br>• VPN endpoint discovery                                         | • Remote access inventory                                                                   |
| **12.8** Establish and maintain network infrastructure documentation | • **Automated network topology**<br>• Device relationship mapping<br>• Switch discovery documentation | • Auto-generated network diagrams<br>• Switch/port/VLAN documentation                       |

**Implementation Notes:**

- Enable LLDP/CDP monitoring for topology discovery
- Configure VLAN detection for segmentation compliance
- Export network topology monthly for documentation

**Healthcare-Specific:** For HIPAA network segmentation requirements:

- Document PHI VLAN isolation
- Verify medical device network segregation
- Monitor for unauthorized VLAN changes

---

### CIS Control 13: Network Monitoring and Defense

**Requirement:**

> "Operate processes and tooling to establish and maintain comprehensive network monitoring and
> defense against security threats across the enterprise's network infrastructure and user base."

#### LuminetIQ Capabilities:

| Sub-Control                                 | LuminetIQ Feature                                                                                 | Evidence/Audit Value                                                                              |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **13.1** Centralize security event alerting | • Real-time anomaly detection<br>• WebSocket alerting<br>• Webhook integration (future)           | • Security event logs<br>• Alert timeline<br>• MTTR metrics                                       |
| **13.2** Deploy host-based IDS              | • Behavior-based anomaly detection<br>• Baseline deviation alerts<br>• Per-device anomaly scoring | • Anomaly detection reports<br>• Behavioral baseline exports                                      |
| **13.3** Deploy network-based IDS           | • **Rogue device detection**<br>• **Port scan detection**<br>• Unusual traffic pattern alerts     | • Rogue device incident reports<br>• Port scan detection logs<br>• Attack timeline reconstruction |
| **13.4** Perform traffic filtering          | • VLAN enforcement monitoring<br>• Unexpected protocol detection                                  | • Traffic filtering compliance<br>• Protocol anomaly reports                                      |
| **13.6** Collect network traffic flow logs  | • Device discovery logs<br>• Connection tracking<br>• Service usage logs                          | • Traffic flow exports<br>• Connection metadata                                                   |
| **13.10** Deploy NIPS                       | • Port scan detection<br>• Malicious behavior identification<br>• Rapid alerting (<30 seconds)    | • Intrusion detection events<br>• Time-to-alert metrics                                           |

**Implementation Notes:**

- Enable anomaly detection and baseline learning
- Configure rogue device detection with automatic alerting
- Set up port scan detection thresholds
- Enable WebSocket real-time alerts

**Key Capabilities:**

```
Rogue Device Detection:
- New MAC address appears → Alert within 60 seconds
- Port scanning detected → Alert + quarantine recommendation
- Behavior mismatch → "Device claims to be printer but behaves like scanner"

Anomaly Detection:
- Gateway latency spike (12ms → 145ms) → Root cause analysis
- DHCP timing degradation → Server overload detection
- Unusual connection patterns → Potential compromise alert
```

---

## NIST Cybersecurity Framework (CSF) v1.1 Mappings

### IDENTIFY (ID)

| Category    | Subcategory                                   | LuminetIQ Capability                               | Evidence                        |
| ----------- | --------------------------------------------- | -------------------------------------------------- | ------------------------------- |
| **ID.AM-1** | Physical devices are inventoried              | Device discovery, classification, inventory        | Device inventory exports        |
| **ID.AM-2** | Software platforms are inventoried            | OS detection, service discovery, version tracking  | Software inventory reports      |
| **ID.AM-3** | Organizational communication flows are mapped | Network topology, VLAN detection, switch discovery | Network diagrams, topology maps |
| **ID.RA-1** | Asset vulnerabilities are identified          | CVE scanning, vulnerability assessment             | Vulnerability scan reports      |
| **ID.RA-3** | Threats are identified and documented         | Rogue device detection, port scan detection        | Threat detection logs           |
| **ID.RA-5** | Threats and vulnerabilities are prioritized   | Contextual risk scoring (CVSS + EPSS + exposure)   | Risk-scored vulnerability lists |

---

### PROTECT (PR)

| Category    | Subcategory                                                | LuminetIQ Capability                             | Evidence                       |
| ----------- | ---------------------------------------------------------- | ------------------------------------------------ | ------------------------------ |
| **PR.AC-5** | Network integrity is protected (e.g., network segregation) | VLAN detection, segmentation verification        | VLAN topology reports          |
| **PR.DS-2** | Data-in-transit is protected                               | TLS inspection, encryption protocol detection    | TLS compliance reports         |
| **PR.IP-1** | Baseline configuration is established                      | Baseline learning, configuration drift detection | Baseline configuration exports |
| **PR.PT-1** | Audit/log records are determined                           | All events logged, exportable                    | Audit log exports              |

---

### DETECT (DE)

| Category    | Subcategory                                   | LuminetIQ Capability                             | Evidence                          |
| ----------- | --------------------------------------------- | ------------------------------------------------ | --------------------------------- |
| **DE.AE-1** | Baseline of network operations is established | Baseline learning (latency, DHCP, DNS, etc.)     | Baseline metrics reports          |
| **DE.AE-2** | Detected events are analyzed                  | Root cause analysis, anomaly detection           | Analysis reports, recommendations |
| **DE.AE-3** | Event data are collected and correlated       | Time-series storage, multi-metric correlation    | Correlated event reports          |
| **DE.AE-5** | Incident alert thresholds are established     | Adaptive threshold recommendations               | Threshold configuration           |
| **DE.CM-1** | Network is monitored                          | Continuous network monitoring, real-time updates | Monitoring logs, uptime reports   |
| **DE.CM-7** | Unauthorized devices are detected             | Rogue device detection, new device alerting      | Rogue device incident reports     |
| **DE.DP-4** | Event detection information is communicated   | WebSocket alerts, real-time notifications        | Alert delivery logs               |

---

### RESPOND (RS)

| Category    | Subcategory                                    | LuminetIQ Capability                                      | Evidence                  |
| ----------- | ---------------------------------------------- | --------------------------------------------------------- | ------------------------- |
| **RS.AN-1** | Notifications are investigated                 | Root cause analysis, guided troubleshooting               | Investigation reports     |
| **RS.AN-2** | Impact of the incident is understood           | Anomaly severity scoring, affected device identification  | Impact assessment reports |
| **RS.MI-3** | Newly identified vulnerabilities are mitigated | Vulnerability remediation recommendations, prioritization | Remediation tracking      |

---

### RECOVER (RC)

| Category    | Subcategory               | LuminetIQ Capability                       | Evidence                                       |
| ----------- | ------------------------- | ------------------------------------------ | ---------------------------------------------- |
| **RC.RP-1** | Recovery plan is executed | Predictive maintenance, failure prediction | Predictive alerts, maintenance recommendations |

---

## NIST SP 800-53 Rev 5 Mappings (Federal/DoD)

### Access Control (AC)

| Control   | Requirement                  | LuminetIQ Capability                                 | Evidence                                     |
| --------- | ---------------------------- | ---------------------------------------------------- | -------------------------------------------- |
| **AC-4**  | Information Flow Enforcement | VLAN detection, network segmentation verification    | VLAN topology reports, traffic flow analysis |
| **AC-20** | Use of External Systems      | Rogue device detection, unauthorized device alerting | Rogue device incident logs                   |

---

### Configuration Management (CM)

| Control  | Requirement                  | LuminetIQ Capability                                          | Evidence                         |
| -------- | ---------------------------- | ------------------------------------------------------------- | -------------------------------- |
| **CM-2** | Baseline Configuration       | Baseline learning, configuration state tracking               | Baseline configuration exports   |
| **CM-3** | Configuration Change Control | Configuration drift detection (fleet management)              | Drift detection reports          |
| **CM-6** | Configuration Settings       | Secure configuration verification, insecure service detection | Configuration compliance reports |
| **CM-8** | System Component Inventory   | Device discovery, software inventory                          | Inventory exports (CSV/JSON)     |

---

### Risk Assessment (RA)

| Control  | Requirement                           | LuminetIQ Capability                     | Evidence                                               |
| -------- | ------------------------------------- | ---------------------------------------- | ------------------------------------------------------ |
| **RA-3** | Risk Assessment                       | Contextual vulnerability risk scoring    | Risk-scored vulnerability reports                      |
| **RA-5** | Vulnerability Monitoring and Scanning | Continuous CVE scanning, NVD integration | Vulnerability scan reports, continuous monitoring logs |

---

### System and Information Integrity (SI)

| Control  | Requirement                    | LuminetIQ Capability                                              | Evidence                                          |
| -------- | ------------------------------ | ----------------------------------------------------------------- | ------------------------------------------------- |
| **SI-3** | Malicious Code Protection      | Rogue device detection, behavior anomaly detection                | Malicious behavior detection logs                 |
| **SI-4** | System Monitoring              | **Comprehensive network monitoring**, anomaly detection, alerting | Monitoring logs, anomaly reports, alert timelines |
| **SI-5** | Security Alerts and Advisories | CVE scanning, vulnerability alerts                                | CVE alert logs, advisory tracking                 |

---

## HIPAA Security Rule Mappings (Healthcare)

### Administrative Safeguards (§164.308)

| Requirement               | Sub-Requirement                                  | LuminetIQ Capability                                                                         | Evidence                                                                             |
| ------------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| **§164.308(a)(1)(ii)(A)** | Risk Analysis (Required)                         | • Vulnerability scanning<br>• Contextual risk scoring<br>• Network health assessment         | • Vulnerability risk reports<br>• Network health scores<br>• Risk assessment exports |
| **§164.308(a)(1)(ii)(B)** | Risk Management (Required)                       | • Prioritized remediation plans<br>• Vulnerability tracking<br>• Remediation recommendations | • Remediation priority lists<br>• Patch management reports                           |
| **§164.308(a)(5)(ii)(B)** | Protection from Malicious Software (Addressable) | • Rogue device detection<br>• Behavior anomaly detection<br>• Port scan detection            | • Rogue device alerts<br>• Anomaly detection logs                                    |
| **§164.308(a)(5)(ii)(C)** | Log-in Monitoring (Addressable)                  | • Device discovery and tracking<br>• New device alerting<br>• Unauthorized access detection  | • Device access logs<br>• New device alerts                                          |
| **§164.308(a)(8)**        | Evaluation (Required)                            | • Network health scoring<br>• Compliance reporting<br>• Audit log exports                    | • Health score reports<br>• Compliance audit exports                                 |

---

### Physical Safeguards (§164.310)

| Requirement     | Sub-Requirement                 | LuminetIQ Capability                                                                                     | Evidence                                                          |
| --------------- | ------------------------------- | -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **§164.310(b)** | Workstation Use (Required)      | • Device classification<br>• Workstation inventory<br>• Unauthorized device detection                    | • Device inventory by type<br>• Workstation access logs           |
| **§164.310(c)** | Workstation Security (Required) | • Device security posture<br>• Vulnerability assessment per device<br>• Insecure configuration detection | • Device security reports<br>• Vulnerability per-device breakdown |

---

### Technical Safeguards (§164.312)

| Requirement        | Sub-Requirement                            | LuminetIQ Capability                                                                                                 | Evidence                                                                                 |
| ------------------ | ------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **§164.312(a)(1)** | Access Control (Required)                  | • **VLAN detection and verification**<br>• Network segmentation mapping<br>• PHI network isolation verification      | • VLAN topology maps<br>• Segmentation compliance<br>• Isolation verification reports    |
| **§164.312(b)**    | Audit Controls (Required)                  | • Comprehensive logging<br>• Audit trail exports<br>• Activity monitoring                                            | • Complete audit logs (JSON/CSV)<br>• Event timeline exports                             |
| **§164.312(d)**    | Person or Entity Authentication (Required) | • Device authentication tracking<br>• Unauthorized device detection                                                  | • Authenticated device inventory<br>• Rogue device alerts                                |
| **§164.312(e)(1)** | Transmission Security (Addressable)        | • **TLS/encryption detection**<br>• Unencrypted protocol alerts<br>• Weak cipher detection<br>• PHI traffic analysis | • TLS compliance reports<br>• Unencrypted service inventory<br>• Cipher strength reports |

---

### HIPAA Implementation Guide for Healthcare Organizations

#### Required Actions:

**1. Network Segmentation Verification (§164.312(a)(1)):**

```bash
# Verify PHI networks are isolated
luminetiq vlan --verify-isolation --critical-vlan 10 --report hipaa-segmentation.pdf

# Expected Output:
✓ VLAN 10 (PHI Network) isolated from guest network
✓ No cross-VLAN traffic detected
✓ Medical devices on dedicated VLAN 20
```

**2. Encryption Compliance (§164.312(e)(1)):**

```bash
# Detect unencrypted PHI transmission
luminetiq scan --protocol-audit --alert-unencrypted --vlan 10

# Alert if:
- HTTP (not HTTPS) detected on PHI network
- Telnet, FTP, SMTP (unencrypted) detected
- TLS < 1.2 detected
```

**3. Risk Analysis (§164.308(a)(1)(ii)(A)):**

```bash
# Generate HIPAA risk assessment
luminetiq vulnerabilities --risk-report --format pdf --hipaa-compliance

# Report includes:
- All devices on PHI networks
- Vulnerability risk scores (CVSS + context)
- Remediation priorities
- Compliance status
```

**4. Device Inventory (§164.308(a)(5)(ii)(C)):**

```bash
# Export complete device inventory for HIPAA audit
luminetiq devices export --format csv --include "ip,mac,hostname,device_type,os,services,first_seen,last_seen"

# Auditors need to see:
- Complete asset inventory
- Medical device identification
- Unauthorized device detection
```

---

## CMMC Level 2 Mappings (Defense Industrial Base)

### Asset Management (AM)

| Practice     | Requirement                                           | LuminetIQ Capability                         | Evidence                              |
| ------------ | ----------------------------------------------------- | -------------------------------------------- | ------------------------------------- |
| **AM.1.055** | Identify system components                            | Device discovery, inventory, classification  | Device inventory exports              |
| **AM.2.058** | Maintain inventory of authorized/unauthorized devices | Rogue device detection, whitelist management | Authorized device lists, rogue alerts |

---

### Configuration Management (CM)

| Practice     | Requirement                                    | LuminetIQ Capability                      | Evidence                       |
| ------------ | ---------------------------------------------- | ----------------------------------------- | ------------------------------ |
| **CM.2.061** | Establish and maintain baseline configurations | Baseline learning, configuration tracking | Baseline configuration exports |
| **CM.2.062** | Track, review, approve configuration changes   | Configuration drift detection             | Drift detection reports        |

---

### System and Communications Protection (SC)

| Practice     | Requirement                                                         | LuminetIQ Capability                             | Evidence                               |
| ------------ | ------------------------------------------------------------------- | ------------------------------------------------ | -------------------------------------- |
| **SC.1.175** | Monitor, control, and protect communications at external boundaries | Network monitoring, VLAN detection, segmentation | Network monitoring logs, topology maps |
| **SC.2.179** | Use encrypted sessions for managing network devices                 | TLS inspection, encrypted protocol detection     | TLS compliance reports                 |

---

### System and Information Integrity (SI)

| Practice     | Requirement                                | LuminetIQ Capability                                | Evidence                                    |
| ------------ | ------------------------------------------ | --------------------------------------------------- | ------------------------------------------- |
| **SI.1.210** | Identify, report, and correct system flaws | Vulnerability scanning, remediation recommendations | Vulnerability reports, remediation tracking |
| **SI.2.216** | Monitor system for anomalous behavior      | Anomaly detection, baseline deviation alerts        | Anomaly detection reports                   |
| **SI.2.217** | Identify unauthorized use                  | Rogue device detection, port scan detection         | Rogue device alerts, port scan logs         |

---

## State/Local/Education (SLED) Specific Considerations

### K-12 Education (CIPA, FERPA)

**Applicable Frameworks:**

- CIPA (Children's Internet Protection Act): Network filtering, monitoring
- FERPA (Student privacy): Network segmentation, access control

**LuminetIQ Capabilities:**

- **Student WiFi isolation**: Verify student networks are segmented from staff/administrative
- **Guest network verification**: Ensure guest WiFi has no access to student data networks
- **Device classification**: Identify student devices vs administrative devices
- **Rogue AP detection**: Detect unauthorized WiFi access points (student hotspots)

**Use Cases:**

```
1. BYOD Management:
   - Classify student personal devices
   - Verify isolation from administrative systems
   - Detect unauthorized device sharing

2. Network Segmentation:
   - Student VLAN: Restricted internet access
   - Staff VLAN: Administrative system access
   - Guest VLAN: Internet-only, fully isolated

3. Compliance Reporting:
   - Export network topology for CIPA compliance
   - Document filtering effectiveness
   - Prove network segmentation for FERPA
```

---

### Higher Education

**Applicable Frameworks:**

- FERPA (Student records)
- GLBA (Financial aid data)
- HIPAA (Campus health centers)
- Research compliance (ITAR, EAR for research universities)

**LuminetIQ Use Cases:**

- Multi-tenant network management (dorms, academic, research, healthcare)
- Research network isolation (ITAR-controlled data)
- Campus health center HIPAA compliance
- Guest lecture/conference WiFi planning (predictive survey)

---

### State/Local Government

**Applicable Frameworks:**

- NIST Cybersecurity Framework (widely adopted)
- State-specific frameworks (e.g., NY SHIELD Act, CalOPPA)
- CIS Controls (often required by cyber insurance)

**LuminetIQ Capabilities:**

- Multi-site government facility management (city hall, libraries, fire/police)
- Public WiFi compliance (guest network isolation)
- Vulnerability management for legacy systems
- Compliance reporting for state audits

---

### Federal Government (FedRAMP, FISMA)

**Applicable Frameworks:**

- NIST SP 800-53 (Required for federal systems)
- FedRAMP (Cloud service providers)
- FISMA (Federal information systems)

**LuminetIQ Capabilities:**

- NIST 800-53 control mapping (see section above)
- Continuous monitoring (FISMA requirement)
- Configuration management (CM controls)
- Audit logging and reporting

**Note:** Federal deployment would require FedRAMP authorization for cloud features. For
self-hosted/on-prem deployments, LuminetIQ meets many NIST 800-53 controls directly.

---

## Evidence Generation for Audits

### Automated Compliance Reports

LuminetIQ can generate compliance-ready reports for auditors:

```bash
# CIS Controls v8 Report
luminetiq compliance --framework cis-v8 --controls 1,2,3,7,12,13 --format pdf --output cis-compliance-report.pdf

# NIST CSF Report
luminetiq compliance --framework nist-csf --functions identify,protect,detect --format pdf

# HIPAA Security Rule Report
luminetiq compliance --framework hipaa --sections 164.308,164.312 --format pdf

# Custom Date Range (for quarterly audits)
luminetiq compliance --framework cis-v8 --start-date 2024-01-01 --end-date 2024-03-31 --format pdf
```

### Audit Evidence Exports

**Device Inventory (CIS 1, NIST ID.AM-1):**

```bash
luminetiq devices export --format csv --include-all --timestamp
# Generates: device-inventory-20250115.csv
```

**Vulnerability Assessment (CIS 7, NIST ID.RA-1):**

```bash
luminetiq vulnerabilities export --risk-scored --format json
# Generates: vulnerabilities-risk-scored-20250115.json
```

**Network Topology (CIS 12, NIST ID.AM-3):**

```bash
luminetiq topology export --include-vlans --format png
# Generates: network-topology-20250115.png
```

**Rogue Device Detection Log (CIS 13, NIST DE.CM-7):**

```bash
luminetiq logs export --category rogue-devices --last-30-days --format csv
# Generates: rogue-device-log-20250115.csv
```

**Anomaly Detection Timeline (NIST DE.AE-2):**

```bash
luminetiq anomalies export --last-quarter --format json
# Generates: anomalies-Q1-2024.json
```

---

## Healthcare-Specific Implementation Checklist

### HIPAA Compliance Quick Start

**Week 1: Discovery and Inventory**

- [ ] Enable device discovery on all network interfaces
- [ ] Configure VLAN detection for PHI network identification
- [ ] Enable device classification for medical device identification
- [ ] Export baseline device inventory

**Week 2: Segmentation Verification**

- [ ] Map all VLANs and document purpose (PHI, medical devices, guest, staff)
- [ ] Verify PHI network isolation (no cross-VLAN traffic)
- [ ] Identify medical IoT devices and verify segmentation
- [ ] Document network topology for §164.312(a)(1) compliance

**Week 3: Vulnerability Assessment**

- [ ] Enable CVE scanning for all devices
- [ ] Configure risk scoring (CVSS + EPSS + exposure)
- [ ] Prioritize medical device vulnerabilities
- [ ] Generate risk analysis report for §164.308(a)(1)(ii)(A)

**Week 4: Encryption and Monitoring**

- [ ] Enable TLS inspection for transmission security
- [ ] Configure alerts for unencrypted PHI traffic
- [ ] Enable rogue device detection
- [ ] Set up anomaly detection baselines

**Ongoing:**

- [ ] Weekly vulnerability scans
- [ ] Monthly compliance reports
- [ ] Quarterly risk assessments
- [ ] Annual HIPAA audit evidence export

---

## Summary: Why LuminetIQ Genuinely Maps to Compliance

### This is NOT Marketing Hype

LuminetIQ provides **real, auditable capabilities** that directly address compliance requirements:

**What We DON'T Claim:**

- ❌ "LuminetIQ makes you compliant" - Compliance requires process + tools
- ❌ "Automatic compliance" - Still requires configuration and policy
- ❌ "Replaces all security tools" - Complements existing security stack

**What We DO Provide:**

- ✅ **Asset visibility**: Comprehensive device discovery and classification
- ✅ **Vulnerability management**: Risk-scored CVE assessment with remediation guidance
- ✅ **Segmentation verification**: VLAN detection and isolation verification
- ✅ **Anomaly detection**: Behavioral baselines and deviation alerts
- ✅ **Audit evidence**: Exportable reports and logs for compliance
- ✅ **Configuration management**: Baseline learning and drift detection

### Audit-Ready Evidence

Every LuminetIQ feature that maps to a control provides:

1. **Logs**: Timestamped, exportable evidence
2. **Reports**: Compliance-formatted PDF/CSV exports
3. **Metrics**: Quantifiable measurements (time-to-detect, coverage, accuracy)
4. **API access**: Integrate with SIEM, GRC tools, ticketing systems

### Unique Value for Compliance

**Traditional approach:** Buy 5+ tools for compliance

- Asset discovery tool
- Vulnerability scanner
- Network monitoring tool
- Configuration management tool
- Log aggregation tool

**LuminetIQ approach:** Single platform addresses multiple controls

- Device discovery + classification → CIS 1, CIS 2, NIST ID.AM
- Vulnerability scanning + risk scoring → CIS 7, NIST ID.RA
- VLAN detection + segmentation → CIS 12, HIPAA §164.312(a)(1)
- Anomaly detection + alerts → CIS 13, NIST DE
- Baseline learning + drift → CIS 4, NIST PR.IP-1

**Cost:** $299-4,999/year vs $20K-50K for multiple tools

---

**Document Maintenance:**

- Review mappings quarterly as frameworks update
- Validate new features against compliance requirements
- Update evidence generation procedures as needed

**Questions?**

- Compliance inquiries: compliance@luminetiq.com (future)
- Technical implementation: See AI_INTEGRATION_PLAN.md
- Healthcare-specific questions: See HEALTHCARE_MARKET_STRATEGY.md (to be created)
