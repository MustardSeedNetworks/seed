# Hardware Documentation Plan

> **Purpose:** Maintain comprehensive, accurate hardware compatibility documentation for LuminetIQ
> **Owner:** Technical Documentation Team
> **Review Cycle:** Quarterly

## Goals

1. **User Empowerment:** Help users select compatible hardware before purchase
2. **Community Knowledge:** Build crowdsourced compatibility database
3. **Support Reduction:** Reduce "why doesn't X work?" support tickets
4. **Quality Assurance:** Track regression/improvements in driver support

---

## Documentation Structure

### Primary Documents

1. **HARDWARE.md** (Root directory)
   - Consumer-facing hardware guide
   - Recommendations by use case
   - Quick reference tables
   - FAQ section
   - **Audience:** End users, technicians
   - **Update Frequency:** Quarterly or on major driver/hardware changes

2. **Wiki: Tested Hardware** (GitHub Wiki)
   - Crowdsourced compatibility reports
   - User-submitted test results
   - Detailed per-device notes
   - **Audience:** Community contributors
   - **Update Frequency:** Continuous (community-driven)

3. **HARDWARE_TESTING_GUIDE.md** (docs/)
   - How to test hardware compatibility
   - Automated testing procedures
   - Reporting format
   - **Audience:** Contributors, QA team
   - **Update Frequency:** As testing procedures evolve

---

## Maintenance Workflow

### Quarterly Review (Every 3 months)

**Checklist:**
- [ ] Check for new Wi-Fi chipset releases (Intel, Qualcomm, MediaTek)
- [ ] Review Linux kernel changelog for new driver support
- [ ] Check ethtool changelog for TDR improvements
- [ ] Update kernel version recommendations
- [ ] Scan GitHub issues for hardware-related problems
- [ ] Review wiki submissions for new tested devices
- [ ] Update pricing information (±20% fluctuation triggers update)
- [ ] Verify all external links still valid
- [ ] Check for deprecated hardware (EOL)

**Process:**
1. Create tracking issue: "Qx YYYY Hardware Documentation Review"
2. Assign to documentation lead
3. Use checklist template (see below)
4. Review community wiki submissions
5. Update HARDWARE.md with findings
6. Close issue with summary of changes

### Triggered Updates (As needed)

**Update immediately when:**
- New Intel/Qualcomm Wi-Fi generation released (e.g., Wi-Fi 7)
- Major kernel release changes driver support
- Critical hardware incompatibility discovered
- New product line (e.g., Raspberry Pi 5)
- User reports widespread issue with recommended hardware

---

## Testing Protocol

### Wi-Fi Adapter Testing

**Required Tests:**
1. **Basic Connectivity**
   ```bash
   # Verify interface detected
   ip link show wlan0
   iw dev wlan0 info
   ```

2. **nl80211 Support**
   ```bash
   # Check nl80211 capabilities
   iw list
   # Look for: "Supported interface modes: * monitor"
   ```

3. **Monitor Mode**
   ```bash
   sudo ip link set wlan0 down
   sudo iw dev wlan0 set type monitor
   sudo ip link set wlan0 up
   iw dev wlan0 info  # Should show "type monitor"
   ```

4. **Channel Switching**
   ```bash
   # Test channel hopping
   for ch in 1 6 11; do
     sudo iw dev wlan0 set channel $ch
     sleep 1
   done
   ```

5. **Signal Quality**
   ```bash
   # Scan networks
   sudo iw dev wlan0 scan | grep -E "signal|SSID"
   ```

**Pass Criteria:**
- ✅ Excellent: All tests pass, strong signal readings
- ⚠️ Limited: Basic connectivity only, no monitor mode
- ❌ Failed: Interface not detected or unstable

### Ethernet NIC TDR Testing

**Required Tests:**
1. **Driver Detection**
   ```bash
   ethtool -i eth0
   # Note driver name and version
   ```

2. **TDR Capability**
   ```bash
   sudo ethtool --cable-test eth0
   # Expected: "Cable test started" or "Operation not supported"
   ```

3. **TDR Results Validation**
   ```bash
   # With known-good cable
   sudo ethtool --cable-test eth0
   # Should show: OK status, approximate length

   # With disconnected cable
   sudo ethtool --cable-test eth0
   # Should show: Open/Short status, distance
   ```

**Pass Criteria:**
- ✅ Full TDR: Detects length, faults, distance to fault
- ⚠️ Basic TDR: Detects OK/fault only, no distance
- ❌ No TDR: "Operation not supported"

### Documentation Format

**Submission Template:**
```markdown
## [Chipset Model]

- **Tested By:** [GitHub username]
- **Date:** YYYY-MM-DD
- **Kernel:** X.XX.X
- **Driver:** [name] vX.XX
- **Distribution:** [e.g., Ubuntu 22.04]

### Capabilities
- Monitor Mode: ✅/⚠️/❌
- Injection: ✅/⚠️/❌
- Channel Switching: ✅/⚠️/❌
- Signal Quality: Excellent/Good/Fair/Poor
- TDR Support: ✅/⚠️/❌

### Notes
[Any additional observations, quirks, configuration needed]

### Recommendation
Best for: [use case]
Avoid for: [use case]
```

---

## Community Contribution Process

### GitHub Wiki: Tested Hardware

**Structure:**
```
Home
├── Wi-Fi Adapters
│   ├── Intel
│   ├── Qualcomm Atheros
│   ├── Broadcom
│   ├── Realtek
│   └── MediaTek
└── Ethernet NICs
    ├── Intel
    ├── Broadcom
    ├── Realtek
    └── Marvell
```

**Contribution Flow:**
1. User tests hardware with LuminetIQ
2. Fills out testing template
3. Submits via:
   - GitHub issue with `hardware-report` label
   - Direct wiki edit (if permissions)
   - Discussion post
4. Maintainer reviews and adds to wiki
5. Notable reports incorporated into HARDWARE.md quarterly

**Quality Control:**
- Require kernel version (driver support varies)
- Require distribution (Ubuntu vs Arch may differ)
- Flag conflicting reports for investigation
- Deprecate reports >2 years old (outdated drivers)

---

## Information Sources

### Primary Sources (Check quarterly)

**Wi-Fi Chipsets:**
- Intel ARK: https://ark.intel.com/content/www/us/en/ark/products/series/204836/intel-wi-fi-6e-products.html
- Qualcomm Product Selector: https://www.qualcomm.com/products/features/wi-fi
- Linux Wireless Wiki: https://wireless.wiki.kernel.org/
- ath9k driver status: https://wireless.wiki.kernel.org/en/users/drivers/ath9k

**Ethernet NICs:**
- Intel Ethernet Controllers: https://www.intel.com/content/www/us/en/products/details/ethernet/controllers.html
- ethtool Features: https://git.kernel.org/pub/scm/network/ethtool/ethtool.git
- Broadcom Server Adapters: https://www.broadcom.com/products/ethernet-connectivity/network-adapters

**Kernel/Driver:**
- Linux Kernel Changelog: https://kernelnewbies.org/
- nl80211 API Updates: https://wireless.wiki.kernel.org/en/developers/documentation/nl80211
- ethtool git log: https://git.kernel.org/pub/scm/network/ethtool/ethtool.git/log/

### Secondary Sources

- Reddit: r/homelab, r/networking (user experiences)
- Linux hardware forums
- Vendor documentation
- Academic research (Wi-Fi performance studies)

---

## Automation Opportunities

### Automated Data Collection (Future)

**Hardware Database API:**
```go
// Potential future implementation
type HardwareReport struct {
    ChipsetID      string
    Driver         string
    DriverVersion  string
    KernelVersion  string
    Capabilities   map[string]bool
    TestResults    []TestResult
    Timestamp      time.Time
}

// POST /api/v1/hardware/report
// Telemetry endpoint (opt-in) to collect hardware compatibility
```

**Benefits:**
- Automated compatibility matrix
- Real-time driver regression detection
- Statistical analysis of common configurations

**Privacy Considerations:**
- Opt-in only
- Anonymized reports
- No personal identifiers
- Clear data usage policy

### CI/CD Integration

**Automated Documentation Checks:**
```yaml
# .github/workflows/docs-check.yml
name: Documentation Check

on:
  pull_request:
    paths:
      - 'HARDWARE.md'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Check links
        run: |
          # Validate all external links
          markdown-link-check HARDWARE.md

      - name: Check formatting
        run: |
          # Ensure tables are properly formatted
          markdownlint HARDWARE.md

      - name: Check prices
        run: |
          # Flag if prices haven't been updated in >6 months
          ./scripts/check-pricing-staleness.sh
```

---

## Success Metrics

### Documentation Quality
- **Link Health:** <2% broken links
- **Freshness:** Updated within last 90 days
- **Completeness:** All major chipset families covered

### Community Engagement
- **Wiki Contributions:** Goal: 5+ unique contributors/quarter
- **Hardware Reports:** Goal: 10+ new reports/quarter
- **Issue Reduction:** Goal: 20% decrease in hardware-related support issues

### User Satisfaction
- **Pre-Purchase Clarity:** Survey users who bought hardware based on docs
- **Accuracy:** <5% reports of "recommended hardware doesn't work"
- **Usefulness:** Survey question: "Did HARDWARE.md help you select hardware?" >80% yes

---

## Roles & Responsibilities

### Documentation Lead
- Owns HARDWARE.md accuracy
- Performs quarterly reviews
- Merges community contributions
- Escalates driver regression issues

### QA Team
- Tests hardware per protocol
- Validates community reports
- Maintains test hardware inventory
- Reports findings to Documentation Lead

### Community Manager
- Encourages wiki contributions
- Highlights community findings
- Moderates hardware discussions
- Badges top contributors

### Development Team
- Provides technical input on driver capabilities
- Reviews hardware-related PRs
- Implements telemetry (if approved)
- Fixes compatibility bugs

---

## Risks & Mitigation

### Risk: Documentation Becomes Outdated
**Impact:** Users purchase incompatible hardware
**Mitigation:**
- Automated staleness checks
- Quarterly review mandate
- Community contribution system
- Version date prominently displayed

### Risk: Driver Support Regresses
**Impact:** Previously-working hardware breaks
**Mitigation:**
- CI testing on multiple kernel versions
- Monitor kernel mailing lists
- Community reports flag regressions quickly
- Document workarounds prominently

### Risk: Vendor EOL Without Notice
**Impact:** Recommend unavailable hardware
**Mitigation:**
- Check product availability during reviews
- Mark discontinued items as "(EOL)"
- Provide current alternatives
- Archive EOL hardware in "Legacy" section

### Risk: Community Spam/Low-Quality Reports
**Impact:** Unreliable compatibility data
**Mitigation:**
- Require GitHub account for submissions
- Template enforcement
- Maintainer review before merge
- Flag conflicting reports for investigation

---

## Timeline

### Phase 1: Foundation (Complete)
- ✅ Create HARDWARE.md
- ✅ Define testing protocols
- ✅ Establish documentation plan

### Phase 2: Community Launch (Month 1-2)
- [ ] Create GitHub Wiki structure
- [ ] Set up hardware-report issue template
- [ ] Announce in README and docs
- [ ] Recruit 3-5 initial testers

### Phase 3: Automation (Month 3-6)
- [ ] Implement link checking in CI
- [ ] Create staleness detection script
- [ ] Build hardware report aggregation tool
- [ ] Consider telemetry opt-in

### Phase 4: Expansion (Month 6-12)
- [ ] Add video demonstrations
- [ ] Create hardware vendor partnerships
- [ ] Publish academic comparison study
- [ ] Conference presentation on findings

---

## Appendix A: Testing Hardware Inventory

**Recommended Test Lab Setup:**

### Wi-Fi Adapters (Minimum)
- 1x Intel AX200/210 (current gen)
- 1x Atheros AR9271 (reference standard)
- 1x Realtek RTL8812AU (common budget option)
- 1x Broadcom BCM43xx (problematic, for testing edge cases)

### Ethernet NICs (Minimum)
- 1x Intel I350 or I210 (TDR reference)
- 1x Realtek RTL8111 (common, no TDR)
- 1x Broadcom BCM5720 (alternative TDR)

### Test Infrastructure
- 3x Cable samples: 1m good, 5m good, 10m with fault
- 1x Wi-Fi AP with WPA3 support
- 1x Managed switch with LLDP/CDP
- 1x Spectrum analyzer (optional, for RF validation)

**Total Budget:** ~$800-1000

---

## Appendix B: Quarterly Review Template

```markdown
# Qx YYYY Hardware Documentation Review

**Assignee:** [Name]
**Due Date:** [Last day of quarter]
**Related Issue:** #XXX

## Checklist

### Market Research
- [ ] Check Intel wireless product page for new releases
- [ ] Check Qualcomm wireless announcements
- [ ] Review Linux kernel X.XX release notes
- [ ] Check ethtool release notes

### Community Input
- [ ] Review 10 most recent wiki submissions
- [ ] Scan GitHub issues with `hardware` label
- [ ] Check r/homelab discussions
- [ ] Review support tickets for patterns

### Documentation Updates
- [ ] Update kernel version recommendations
- [ ] Adjust pricing (check 3 vendors)
- [ ] Add new tested hardware
- [ ] Mark EOL products
- [ ] Update driver version references
- [ ] Verify all external links

### Quality Assurance
- [ ] Run markdown linter
- [ ] Check for typos/inconsistencies
- [ ] Verify tables format correctly
- [ ] Test any changed procedures

## Findings

[Summarize what changed in this quarter]

## Action Items

- [ ] [Specific update needed]
- [ ] [Another update]

## Metrics

- Wiki contributions this quarter: XX
- New hardware reports: XX
- Hardware-related issues: XX (vs XX last quarter)
- Documentation age: XX days since last update

---
Approved by: [Lead]
Date: [YYYY-MM-DD]
```

---

**Document Version:** 1.0
**Created:** 2025-12-14
**Next Review:** 2026-03-14
