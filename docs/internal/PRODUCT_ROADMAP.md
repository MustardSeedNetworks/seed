# LuminetIQ Product Roadmap (2026-2027)

**Document Version:** 1.0  
**Last Updated:** 2025-12-15  
**Planning Horizon:** 18 months

---

## Vision & Strategy

**Mission:** Make professional network diagnostics accessible to everyone through AI-powered
insights and predictive planning.

**Product Pillars:**

1. **Network Intelligence** - Comprehensive visibility + AI analysis
2. **WiFi Planning** - Predictive survey + optimization (flagship)
3. **Security & Compliance** - Vulnerability management + framework mapping
4. **Operational Efficiency** - Troubleshooting + maintenance automation

---

## Release Schedule

| Version      | Release Date | Focus                        | Status     |
| ------------ | ------------ | ---------------------------- | ---------- |
| **v0.110.0** | Mar 2026     | AI Foundation                | 🔵 Planned |
| **v0.120.0** | May 2026     | Intelligent Analysis         | 🔵 Planned |
| **v0.130.0** | Aug 2026     | WiFi Intelligence (Flagship) | 🔵 Planned |
| **v0.140.0** | Nov 2026     | Advanced AI Features         | 🔵 Planned |
| **v1.0.0**   | Jan 2027     | General Availability         | 🔵 Planned |

---

## v0.110.0 - AI Foundation (March 2026)

**Theme:** Establish AI infrastructure and deliver quick wins

**Milestone:** [GitHub Milestone #11](https://github.com/krisarmstrong/netscope/milestone/11)

### Features

**AI Infrastructure (#575)**

- AI service architecture (local + cloud providers)
- Feature flag system for AI capabilities
- Provider abstraction layer (local ML, Claude API)

**Time-Series Storage (#576)**

- SQLite-based metric storage
- 30-day retention with aggregation
- Query API for historical data

**Baseline Learning (#577)**

- Statistical baseline calculation (mean, stddev, percentiles)
- Online learning (continuous updates)
- Per-metric baselines (latency, DHCP, DNS, etc.)

**Device Classification (#578)**

- AI-powered device type detection (90%+ accuracy)
- Classification confidence scores
- Device type tagging (printer, camera, server, etc.)

**Network Health Scoring (#579)**

- 0-100 health score algorithm
- Component breakdown (physical, network, performance, security)
- Trend detection (improving/stable/degrading)

**Frontend Insight Cards (#580)**

- HealthScoreCard component
- InsightCard component
- DeviceClassificationBadge
- Storybook stories for all components

**Configuration (#581)**

- AI feature configuration (YAML)
- License tier integration
- Feature flag management

### Success Criteria

- ✅ Devices auto-classified with 90%+ accuracy
- ✅ Network health score visible and actionable
- ✅ Baseline learning operational for 3+ metrics
- ✅ 80%+ test coverage for AI modules

**Target Users:** Beta testers, early adopters

---

## v0.120.0 - Intelligent Analysis (May 2026)

**Theme:** Add intelligent analysis and automated recommendations

**Milestone:** [GitHub Milestone #12](https://github.com/krisarmstrong/netscope/milestone/12)

### Features

**Root Cause Analysis (#582)**

- DHCP timing analysis (which phase is slow?)
- Gateway latency diagnosis (local vs ISP)
- DNS failure root cause
- Multi-factor correlation

**Anomaly Detection (#583)**

- Z-score based detection (>3σ)
- Pattern anomalies (spikes, sustained degradation)
- Multi-metric correlation
- Real-time alerting (<30 seconds)

**Vulnerability Risk Assessment (#584)**

- Contextual risk scoring (CVSS + EPSS + exposure)
- Network exposure analysis (DMZ, internal, isolated)
- Prioritized remediation recommendations
- HIPAA/CIS compliance integration

**Natural Language Query (#585)**

- Claude API integration with tool calling
- Common queries: "Why is DHCP slow?", "Which devices are vulnerable?"
- Conversational interface
- Query history

**Guided Troubleshooting (#586)**

- Step-by-step diagnostic workflows
- Automated checks where possible
- Exportable troubleshooting logs

**Adaptive Thresholds (#587)**

- Network-aware threshold recommendations
- Baseline-driven suggestions
- One-click apply

### Success Criteria

- ✅ Root cause accuracy >80%
- ✅ Anomaly detection false positive rate <5%
- ✅ NLQ response time <3 seconds
- ✅ User satisfaction >4.0/5

**Target Users:** Professional tier customers, healthcare IT

---

## v0.130.0 - WiFi Intelligence (August 2026) 🚀 FLAGSHIP

**Theme:** Predictive WiFi planning and optimization

**Milestone:** [GitHub Milestone #13](https://github.com/krisarmstrong/netscope/milestone/13)

### Features

**Coverage Heatmap Generation (#588)**

- IDW/Kriging interpolation
- 10cm resolution grid
- Multi-floor support
- Professional visualization

**Dead Zone Detection (#589)**

- Contiguous low-coverage area identification
- Severity ranking
- Remediation recommendations

**AP Placement Optimization (#590)**

- Genetic algorithm for placement
- Cost optimization (minimize AP count)
- Channel assignment

**Predictive Survey Simulation (#591)** 🌟 **FLAGSHIP**

- Floor plan editor (wall drawing, materials)
- RF path loss modeling (FSPL + walls)
- What-if scenario analysis
- Professional PDF export with BOM
- **Target accuracy:** ±10 dB

**Channel Interference Analysis (#592)**

- Channel utilization stats
- Co-channel interference detection
- Optimal channel recommendations

**Roaming Pattern Analysis (#593)**

- Handoff analysis
- Ping-pong detection
- Power/threshold recommendations

### Success Criteria

- ✅ Predictive survey accuracy ±10 dB
- ✅ Heatmap generation <5 seconds
- ✅ User can complete floor plan in <10 minutes
- ✅ User satisfaction >4.5/5
- ✅ Competitive with Hamina on quality

**Target Users:** WiFi consultants, hospitals, premium tier

---

## v0.140.0 - Advanced AI Features (November 2026)

**Theme:** Predictive maintenance and enterprise features

**Milestone:** [GitHub Milestone #14](https://github.com/krisarmstrong/netscope/milestone/14)

### Features

**Predictive Maintenance (#594)**

- Link failure prediction (24-48h early warning)
- Time-series forecasting
- Device health trending
- Proactive alerts

**Automated Reporting (#595)**

- PDF/HTML report generation
- HIPAA/CIS/NIST compliance templates
- White-label customization
- Scheduled reports

**Rogue Device Detection (#596)**

- Behavior-based anomaly detection
- Port scan detection
- Unauthorized device alerting
- Quarantine recommendations

**Multi-Site Fleet Management (#597)**

- Cross-site comparative analytics
- Configuration drift detection
- Fleet-wide vulnerability rollup
- Centralized dashboard

**Capacity Planning (#598)**

- Growth forecasting
- Resource utilization trends
- Capacity recommendations

### Success Criteria

- ✅ Predict 50%+ of failures 24h+ early
- ✅ Reports suitable for compliance audits
- ✅ Rogue detection within 60 seconds
- ✅ Fleet management scales to 100+ sites

**Target Users:** Enterprise tier, MSPs

---

## v1.0.0 - General Availability (January 2027)

**Theme:** Production-ready, enterprise-grade platform

### Focus Areas

**Stability & Performance**

- 99.9% uptime SLA
- <100ms API response time (p95)
- Zero data loss guarantee

**Security & Compliance**

- SOC 2 Type 1 certification (in progress)
- GDPR compliance
- HIPAA Business Associate readiness

**Enterprise Features**

- SSO/SAML authentication
- Role-based access control (RBAC)
- Audit logging
- SLA agreements

**Documentation**

- Complete user guides
- API reference documentation
- Compliance implementation guides
- Video tutorials

**Support**

- 24/7 support for Enterprise tier
- Knowledge base (100+ articles)
- Community forum
- Partner enablement program

### Launch Criteria

- ✅ 100+ beta customers
- ✅ 50+ reference customers
- ✅ <1% critical bug rate
- ✅ NPS >40
- ✅ 90%+ feature adoption (core features)

---

## Beyond v1.0 (Future Vision)

### Q1 2027 - Q4 2027

**Advanced Analytics**

- Machine learning for device fingerprinting
- LSTM for time-series prediction
- Computer vision for cable/equipment recognition

**Integrations**

- Slack/Teams notifications
- ServiceNow/Jira ticketing
- Prometheus/Grafana metrics
- SIEM integrations (Splunk, QRadar)

**Hardware Partnerships**

- Spectrum analyzer integration
- Professional survey hardware bundles
- Raspberry Pi appliance

**New Verticals**

- Retail (guest WiFi + PCI compliance)
- Manufacturing (ICS/SCADA network monitoring)
- Education (K-12 specific features)

---

## Feature Prioritization Framework

### Scoring Criteria (1-10 scale)

1. **Customer Value** - How much does this help users?
2. **Revenue Impact** - Does this drive upgrades/retention?
3. **Competitive Advantage** - Unique vs competitors?
4. **Effort** - How long to build? (inverse score: easy=10, hard=1)
5. **Risk** - Technical/market risk (inverse: low risk=10)

**Formula:** Priority = (Customer Value × 3) + (Revenue × 2) + Competitive + Effort + Risk

**Threshold:** >60 = High Priority, 40-60 = Medium, <40 = Low

### Example Scores

| Feature               | Customer | Revenue | Competitive | Effort | Risk | **Total** | Priority       |
| --------------------- | -------- | ------- | ----------- | ------ | ---- | --------- | -------------- |
| Predictive Survey     | 10       | 10      | 10          | 4      | 6    | **76**    | 🔴 Critical    |
| Root Cause Analysis   | 9        | 8       | 8           | 7      | 8    | **72**    | 🔴 High        |
| Device Classification | 8        | 7       | 7           | 8      | 9    | **68**    | 🔴 High        |
| Fleet Management      | 7        | 9       | 6           | 5      | 7    | **63**    | 🟡 High        |
| Spectrum Analysis     | 5        | 3       | 4           | 2      | 3    | **35**    | 🟢 Low (defer) |

---

## Technical Debt Management

**Debt Allocation:** 20% of each sprint dedicated to tech debt

**Categories:**

1. **Critical Debt** - Blocks new features, security risk (fix immediately)
2. **Important Debt** - Slows development, impacts quality (fix within 2 sprints)
3. **Opportunistic Debt** - Refactor when touching code anyway

**Examples:**

- Critical: SQL injection vulnerability, memory leak
- Important: Slow test suite (>5 minutes), missing error handling
- Opportunistic: Variable naming, code duplication

---

## Sunset Policy

**When to Deprecate Features:**

- Usage <5% of user base for 6+ months
- Better replacement available
- High maintenance cost, low value
- Security risk cannot be mitigated

**Deprecation Process:**

1. Announce 6 months in advance
2. Mark as deprecated in docs/UI
3. Provide migration path
4. Remove after 12 months

**Never Deprecate Without Replacement:**

- Core network monitoring features
- Compliance-related features (HIPAA, CIS)
- Features in pricing tiers (downgrade tier instead)

---

## Roadmap Governance

**Review Cadence:**

- **Monthly:** Adjust sprint priorities
- **Quarterly:** Review milestone progress, adjust roadmap
- **Annually:** Revisit vision and multi-year strategy

**Stakeholder Input:**

- Product: Prioritizes based on customer feedback
- Engineering: Estimates effort, identifies technical dependencies
- Sales: Provides market/competitive intelligence
- Support: Highlights pain points from tickets

**Changes Require:**

- Major roadmap shift (>1 month delay): CEO approval
- Feature reprioritization: Product + Engineering consensus
- New feature requests: Submit via GitHub issue + voting

---

## Success Metrics by Release

### v0.110.0 (AI Foundation)

- 50+ beta users
- 80%+ use device classification
- Health score viewed daily by 60%+ users
- <5 critical bugs

### v0.120.0 (Intelligent Analysis)

- 150+ users
- 70%+ use root cause analysis
- NLQ queries: 100+ per day
- NPS >30

### v0.130.0 (WiFi Intelligence)

- 300+ users (50+ Premium tier)
- 80%+ Premium users use predictive survey
- Predictive survey accuracy validated ±10 dB
- 10+ case studies from WiFi consultants
- NPS >40

### v0.140.0 (Advanced Features)

- 500+ users (20+ Enterprise tier)
- 5+ MSP partners with 10+ clients each
- Predictive maintenance prevents 20+ failures
- NPS >45

### v1.0 (GA)

- 1,000+ paying customers
- $1.5M ARR
- <1% churn (monthly)
- NPS >50
- 90%+ uptime

---

## Appendix: Feature Backlog (Not Scheduled)

**Good Ideas, But Not Now:**

- Mobile app (iOS/Android)
- Network automation (auto-remediation)
- SD-WAN monitoring
- Cellular network testing
- Container/Kubernetes monitoring
- IoT device provisioning
- Network access control (NAC)

**Why Deferred:**

- Scope creep risk
- Niche use cases (<10% of users)
- High effort, uncertain ROI
- Better served by specialized tools

**Revisit:** After v1.0 GA, if customer demand emerges

---

**Document Owner:** Product Team  
**Next Review:** March 2026 (post v0.110.0 launch)
