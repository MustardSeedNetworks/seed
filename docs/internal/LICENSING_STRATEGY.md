# LuminetIQ Licensing & Monetization Strategy

**Document Version:** 1.0 **Last Updated:** 2025-12-15 **Status:** Planning

---

## Overview

This document defines LuminetIQ's software licensing model, enforcement mechanisms, and monetization
strategy.

## Executive Summary

**Licensing Model:** Subscription-based (annual/monthly) with feature-based tiers **Enforcement:**
License key + device fingerprinting + cloud validation **Monetization:** Freemium → Paid tiers →
Enterprise upsell **Philosophy:** Fair use, protect revenue, enable trials, prevent abuse

---

## Licensing Tiers

### **1. Free Tier (Community Edition)**

**Purpose:** Lead generation, product-led growth, freemium model

**Features Included:**

- Basic network discovery (ARP, ICMP)
- WiFi scanning (basic)
- Speed testing
- Device inventory (no AI classification)
- Link status monitoring
- Basic DHCP/DNS monitoring
- Export to CSV

**Limitations:**

- Max 50 devices
- No AI features
- No compliance reporting
- No vulnerability scanning
- Community support only (GitHub issues)
- LuminetIQ branding in reports
- No API access

**License Type:** Perpetual, no expiration **Activation:** Email signup, no credit card **Use
Case:** Individual users, hobbyists, small home networks

---

### **2. Starter Tier - $299/year or $29/month**

**Purpose:** Entry point for small businesses and consultants

**Features Included:**

- Everything in Free, plus:
- ✅ Up to 200 devices
- ✅ AI device classification
- ✅ Network health scoring
- ✅ Basic vulnerability scanning
- ✅ Email support (48-hour response)
- ✅ Remove branding from exports

**Limitations:**

- No predictive WiFi survey
- No anomaly detection
- No compliance automation
- No API access

**License Type:** Subscription (annual or monthly) **Activation:** Credit card required,
auto-renewal **Use Case:** Small IT departments, individual consultants, clinics

---

### **3. Professional Tier - $799/year or $79/month**

**Purpose:** Primary offering for SMB, healthcare, MSPs

**Features Included:**

- Everything in Starter, plus:
- ✅ Unlimited devices
- ✅ AI root cause analysis
- ✅ Anomaly detection & alerting
- ✅ Vulnerability risk scoring (CVSS + EPSS + exposure)
- ✅ Natural language query
- ✅ WiFi coverage heatmaps
- ✅ Dead zone detection
- ✅ Guided troubleshooting
- ✅ Automated HIPAA/CIS/NIST compliance reports
- ✅ Priority email support (24-hour response)
- ✅ Basic API access (100 calls/day)

**Limitations:**

- No predictive WiFi survey
- No fleet management
- No white-label reports
- No SSO/SAML

**License Type:** Subscription (annual or monthly) **Activation:** Credit card, 30-day free trial
**Use Case:** Community hospitals, MSPs, network consultants, IT departments

---

### **4. Premium Tier - $1,999/year or $199/month** 🚀

**Purpose:** WiFi consultants, professional deployments, flagship features

**Features Included:**

- Everything in Professional, plus:
- ✅ **Predictive WiFi survey simulation** (FLAGSHIP)
- ✅ **AP placement optimization**
- ✅ **Channel interference analysis**
- ✅ **Roaming pattern optimization**
- ✅ **What-if scenario analysis**
- ✅ Predictive maintenance (failure prediction)
- ✅ Rogue device detection
- ✅ White-label reports (your logo/branding)
- ✅ Priority support (24-48 hour response)
- ✅ Full API access (1,000 calls/day)
- ✅ Dedicated Slack channel

**Limitations:**

- Single-site (for multi-site, upgrade to Enterprise)
- No SSO/SAML
- No dedicated account manager

**License Type:** Subscription (annual or monthly) **Activation:** Credit card, 30-day free trial
**Use Case:** WiFi consultants, hospitals with complex WiFi, professional deployments

---

### **5. Enterprise Tier - $4,999/year or $499/month**

**Purpose:** MSPs, large healthcare systems, multi-site deployments

**Features Included:**

- Everything in Premium, plus:
- ✅ **Multi-site fleet management** (unlimited sites)
- ✅ **Comparative site analysis**
- ✅ **Configuration drift detection**
- ✅ **Capacity planning & forecasting**
- ✅ **Full API access** (unlimited calls)
- ✅ **SSO/SAML authentication**
- ✅ **Dedicated account manager**
- ✅ **Priority support** (4-8 hour response SLA)
- ✅ **Custom integrations** (Slack, Teams, ServiceNow)
- ✅ **Quarterly business reviews**
- ✅ **Custom SLA agreements**
- ✅ **Volume licensing** (negotiable for 10+ sites)

**License Type:** Subscription (annual preferred) **Activation:** Sales-assisted, custom contract
**Use Case:** MSPs with 5+ healthcare clients, large health systems, enterprise IT

---

## License Enforcement Mechanisms

### **1. License Key System**

**Generation:**

```
Format: LLLL-TTTT-NNNN-CCCC-HHHH
- L = License tier (FREE, STRT, PROF, PREM, ENTP)
- T = Timestamp (Unix epoch, base36)
- N = Customer ID (numeric)
- C = Random component (entropy)
- H = HMAC signature (prevent tampering)

Example: PROF-A7K9M-12345-R8Q2-9F3D
```

**Storage:**

- Client-side: `~/.luminetiq/license.key`
- Server-side: License database (customer_id, license_key, tier, expiry, max_devices)

**Validation:**

1. Parse license key
2. Verify HMAC signature
3. Check expiration date
4. Validate tier matches features in use
5. Cloud validation (once per 24 hours)

---

### **2. Device Fingerprinting**

**Purpose:** Prevent license sharing across organizations

**Fingerprint Components:**

- MAC address of primary interface
- Hostname
- CPU ID (if available)
- Installation UUID (generated on first run)

**Enforcement:**

- Free tier: 1 device per license
- Starter: Up to 3 devices (laptop, desktop, server)
- Pro/Premium/Enterprise: Up to 5 devices
- Enterprise (custom): Negotiate for large deployments

**Behavior:**

- Soft limit: Warning after device limit exceeded
- Hard limit: Block activation after 2x device limit
- Device deactivation: Allow user to deactivate old devices via portal

---

### **3. Cloud License Validation**

**Frequency:**

- Every 24 hours (online check)
- Grace period: 7 days offline (after last successful validation)
- After grace period: Downgrade to Free tier features

**Validation Endpoint:**

```
POST https://license.luminetiq.com/api/v1/validate
{
  "license_key": "PROF-...",
  "device_fingerprint": "sha256(MAC+hostname+uuid)",
  "version": "0.110.0",
  "features_in_use": ["ai_analysis", "vuln_scan"]
}

Response:
{
  "valid": true,
  "tier": "professional",
  "expires": "2026-12-15T00:00:00Z",
  "features_enabled": [...],
  "max_devices": 200,
  "api_quota": 100
}
```

**Offline Mode:**

- Cache last validation response
- Allow offline use for 7 days
- After 7 days: Show warning, downgrade to Free features
- Healthcare exception: 30-day grace period (air-gapped environments)

---

### **4. Feature Gating**

**Backend Feature Flags:**

```go
type LicenseTier int

const (
    TierFree LicenseTier = iota
    TierStarter
    TierProfessional
    TierPremium
    TierEnterprise
)

func (l *License) CanUse(feature string) bool {
    switch feature {
    case "ai_classification":
        return l.Tier >= TierStarter
    case "ai_root_cause":
        return l.Tier >= TierProfessional
    case "predictive_survey":
        return l.Tier >= TierPremium
    case "fleet_management":
        return l.Tier >= TierEnterprise
    default:
        return false
    }
}
```

**Frontend Feature Gating:**

- Hide unavailable features in UI
- Show "Upgrade to Premium" CTA when clicking disabled feature
- Preview mode: Allow users to see predictive survey UI (read-only) to entice upgrade

---

## Trial & Demo Licenses

### **30-Day Free Trial**

**Tiers Available:**

- Professional: 30-day trial
- Premium: 30-day trial (recommended default)
- Enterprise: Custom evaluation period (60-90 days)

**Trial Activation:**

1. User signs up with email
2. Credit card required (soft charge, no payment until trial ends)
3. Full Premium tier features unlocked
4. Email reminders: Day 7, 14, 21, 28, 30
5. Auto-convert to paid on Day 31 OR downgrade to Free tier

**Trial Extensions:**

- Healthcare/education: Auto-approve 15-day extension
- Enterprise POC: Sales can approve 30-60 day extension
- Academic/non-profit: 90-day trial

---

### **Demo Licenses**

**Purpose:** Sales demos, trade shows, screenshots

**Features:**

- Full Enterprise tier features
- 7-day expiration (non-renewable)
- Watermark: "DEMO LICENSE" on all exports
- No credit card required

**Activation:**

- Sales team can generate via admin portal
- Pre-loaded demo data included

---

## Volume & Enterprise Licensing

### **Volume Discounts**

| Sites/Licenses | Discount | Effective Price (Premium) |
| -------------- | -------- | ------------------------- |
| 1-4 sites      | 0%       | $1,999/year each          |
| 5-9 sites      | 15%      | $1,699/year each          |
| 10-24 sites    | 25%      | $1,499/year each          |
| 25-49 sites    | 35%      | $1,299/year each          |
| 50+ sites      | 40%      | $1,199/year each          |

**Enterprise tier includes volume pricing in base price ($4,999 = unlimited sites)**

---

### **MSP Partner Program**

**Tiers:**

- **Registered Partner:** 10% discount, co-marketing
- **Silver Partner:** 20% discount, dedicated support
- **Gold Partner:** 30% discount, revenue share, white-label

**Requirements:**

- Manage 5+ healthcare clients (Silver)
- Manage 20+ healthcare clients (Gold)
- Complete LuminetIQ certification training
- Quarterly revenue targets

**Revenue Share:**

- Partner sells LuminetIQ to their clients
- Partner keeps 20-30% margin
- LuminetIQ bills partner, partner bills client (or pass-through)

---

## Compliance & Audit Licensing

### **Academic & Non-Profit**

**Discount:** 50% off all tiers (verify via .edu email or 501(c)(3) status)

**Use Case:**

- Universities, K-12 schools
- Hospitals (non-profit only, not for-profit healthcare)
- Research institutions

---

### **Government & SLED**

**Special Licensing:**

- No cloud validation required (air-gapped environments)
- 90-day offline grace period
- Custom procurement terms (NET 60, POs accepted)
- FedRAMP compliance roadmap provided

**Pricing:**

- GSA Schedule pricing (once approved)
- State contract pricing (once negotiated)

---

## License Migration & Upgrades

### **Tier Upgrades**

**Mid-contract upgrade:**

- Prorate current tier, apply credit to higher tier
- Example: 6 months into Starter ($299/year) → Upgrade to Premium ($1,999/year)
  - Credit: $150 (6 months unused Starter)
  - Charge: $1,999 - $150 = $1,849
  - New expiration: 12 months from upgrade date

**Downgrades:**

- Allowed at renewal only (not mid-contract)
- Grace period: Features remain active until renewal date

---

### **Legacy License Migration**

**Perpetual → Subscription:**

- If you offered perpetual licenses (one-time purchase) pre-2026
- Grandfather perpetual customers for 3 years
- Offer discounted subscription conversion (50% off Year 1)

---

## License Compliance & Auditing

### **Usage Telemetry**

**What We Track (Privacy-Preserving):**

- License tier and expiration
- Feature usage (which features are used, not network data)
- Device count (number of devices monitored)
- API call volume
- Error rates and crashes (anonymous)

**What We DON'T Track:**

- Network topology
- Device details (IPs, MACs, hostnames)
- Vulnerability data
- User data (PHI, PCI, PII)
- Geographic location (unless consented)

**Opt-Out:**

- Enterprise customers can disable telemetry (on-prem mode)
- Must validate license via manual process quarterly

---

### **Audit Rights**

**Enterprise Contracts:**

- LuminetIQ reserves right to audit license usage annually
- Advance notice: 30 days
- Customer provides device count, site count, user count
- If over-usage found: Pay true-up + 10% penalty OR terminate

**Grace Period:**

- 30 days to remediate over-usage (buy more licenses)
- No penalty if resolved within grace period

---

## License Violations & Enforcement

### **Violation Types**

1. **Key Sharing:** Same license key used across multiple organizations
2. **Device Limit Exceeded:** More devices than license allows
3. **Feature Hacking:** Attempting to enable premium features on lower tier
4. **Expired License:** Continuing use after expiration without renewal

### **Enforcement Actions**

**Tier 1 (Warning):**

- Email warning to license holder
- In-app banner: "Please upgrade or reduce devices"
- 14-day grace period to comply

**Tier 2 (Soft Block):**

- Disable premium features (downgrade to Free tier)
- Retain data, but limit functionality
- Display upgrade prompt on every launch

**Tier 3 (Hard Block):**

- Repeated violations after warnings
- Suspected piracy/key sharing
- Block license key (cannot validate)
- Contact sales for resolution

**Appeal Process:**

- Email: license@luminetiq.com
- Legitimate cases resolved within 48 hours (e.g., company acquisition, system migration)

---

## License Portal (Customer Self-Service)

### **Features:**

**Account Management:**

- View license details (tier, expiration, device count)
- Upgrade/downgrade tier
- Update payment method
- View invoices and payment history

**Device Management:**

- List activated devices
- Deactivate old devices
- See device fingerprints (last 4 chars of MAC)
- Transfer license to new device

**API Keys:**

- Generate API keys (Professional tier+)
- Revoke API keys
- View API usage statistics

**Billing:**

- Update credit card
- Switch annual ↔ monthly
- Download invoices (for expense reports)
- Request quote for Enterprise upgrade

**Portal URL:** https://portal.luminetiq.com (or https://account.luminetiq.com)

---

## Payment Processing

### **Payment Methods:**

**Accepted:**

- Credit/debit cards (Stripe)
- ACH/bank transfer (Enterprise, annual only)
- Purchase orders (Enterprise, government, NET 30-60)
- Wire transfer (international, Enterprise)

**Not Accepted:**

- Cryptocurrency (too volatile, compliance issues)
- PayPal (high fees, chargeback risk)
- Cash/check (operational complexity)

---

### **Billing Cycle:**

**Monthly:**

- Auto-renew on same day each month
- 3-day grace period if payment fails
- Retry payment 3 times (Day 1, 3, 7)
- Suspend account after 7 days non-payment

**Annual:**

- Auto-renew on anniversary
- 30-day renewal notice email
- 7-day grace period if payment fails
- Downgrade to Free tier if not resolved

---

### **Refund Policy:**

**30-Day Money-Back Guarantee:**

- Full refund if unsatisfied within 30 days
- No questions asked
- Applies to first purchase only (not renewals)

**Pro-rated Refunds:**

- Annual subscription canceled mid-year: Pro-rated refund
- Monthly: No refund (current month completes)

**No Refunds:**

- Renewals (cancel before renewal date instead)
- Enterprise contracts (negotiated separately)

---

## License Key Storage Security

### **Backend:**

**Storage:**

- Encrypted at rest (AES-256)
- Hashed for lookups (HMAC-SHA256)
- Access logged (audit trail)

**Database Schema:**

```sql
CREATE TABLE licenses (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    license_key_hash TEXT NOT NULL,  -- HMAC(license_key)
    tier TEXT NOT NULL,               -- 'free', 'starter', 'pro', 'premium', 'enterprise'
    status TEXT NOT NULL,             -- 'active', 'suspended', 'expired', 'canceled'
    max_devices INT,
    issued_at TIMESTAMP,
    expires_at TIMESTAMP,
    last_validated TIMESTAMP,
    device_fingerprints JSONB         -- Array of device fingerprints
);

CREATE INDEX idx_license_hash ON licenses(license_key_hash);
```

---

### **Client-Side:**

**Storage Location:**

- Linux: `~/.config/luminetiq/license.key`
- macOS: `~/Library/Application Support/LuminetIQ/license.key`
- Windows: `%APPDATA%\LuminetIQ\license.key`

**Permissions:**

- Owner read-only (chmod 400)
- Not world-readable
- Warn if permissions too open

---

## License Transfer & Ownership

### **Company Acquisition:**

- License transfers to acquiring company
- Must notify us within 30 days
- Update billing/contact info
- No fee for transfer

### **Device Replacement:**

- User can deactivate old device via portal
- Activate new device with same license
- Limit: 5 device swaps per year (prevent abuse)

### **Resale/Transfer (Not Allowed):**

- Licenses are non-transferable between unrelated entities
- Cannot sell license to another company
- Exception: Asset sale with written approval

---

## Open Source & Community

### **Open Core Model (Future Consideration):**

**Option:** Make core network monitoring open source, premium features proprietary

**Pros:**

- Community contributions
- Transparency builds trust
- Easier adoption
- Marketing value

**Cons:**

- Competitors can fork
- Harder to monetize
- Support burden

**Decision:** Postpone until after initial product-market fit (Year 2+)

---

## License Compliance Checklist

**Before Launch:**

- [ ] Implement license key generation
- [ ] Build license validation API
- [ ] Add feature gating to backend
- [ ] Create customer license portal
- [ ] Set up Stripe billing integration
- [ ] Write license agreement (EULA)
- [ ] Create trial signup flow
- [ ] Implement device fingerprinting
- [ ] Add offline grace period logic
- [ ] Write license enforcement docs

**Post-Launch:**

- [ ] Monitor license violations
- [ ] Track tier conversion rates
- [ ] Analyze feature usage by tier
- [ ] Optimize trial-to-paid conversion
- [ ] Build volume licensing portal for MSPs
- [ ] Add usage-based billing (future)

---

## Legal & Terms

### **EULA Highlights:**

- **Grant:** Subscription license to use software during term
- **Restrictions:** No reverse engineering, no redistribution, no key sharing
- **Data:** Telemetry collected (see Privacy Policy), no network data shared
- **Warranty:** Limited warranty, no guarantee of specific results
- **Liability:** Capped at amount paid in last 12 months
- **Termination:** Either party can terminate with 30 days notice (annual), immediate (monthly)

### **Privacy & Compliance:**

- **GDPR:** Right to export data, right to deletion, minimal data collection
- **HIPAA:** LuminetIQ is not a Business Associate (network tool, not PHI processor)
- **SOC 2:** Roadmap for 2027 (after product-market fit)

---

## Appendix: License Tier Comparison Matrix

| Feature                    | Free      | Starter     | Pro         | Premium           | Enterprise           |
| -------------------------- | --------- | ----------- | ----------- | ----------------- | -------------------- |
| **Price**                  | $0        | $299/yr     | $799/yr     | $1,999/yr         | $4,999/yr            |
| **Max Devices**            | 50        | 200         | Unlimited   | Unlimited         | Unlimited            |
| **AI Classification**      | ❌        | ✅          | ✅          | ✅                | ✅                   |
| **Vulnerability Scanning** | ❌        | ✅ Basic    | ✅ Full     | ✅ Full           | ✅ Full              |
| **Root Cause Analysis**    | ❌        | ❌          | ✅          | ✅                | ✅                   |
| **Anomaly Detection**      | ❌        | ❌          | ✅          | ✅                | ✅                   |
| **Natural Language Query** | ❌        | ❌          | ✅          | ✅                | ✅                   |
| **WiFi Heatmaps**          | ❌        | ❌          | ✅          | ✅                | ✅                   |
| **Predictive Survey**      | ❌        | ❌          | ❌          | ✅                | ✅                   |
| **AP Optimization**        | ❌        | ❌          | ❌          | ✅                | ✅                   |
| **Predictive Maintenance** | ❌        | ❌          | ❌          | ✅                | ✅                   |
| **Fleet Management**       | ❌        | ❌          | ❌          | ❌                | ✅                   |
| **API Access**             | ❌        | ❌          | ✅ Basic    | ✅ Full           | ✅ Unlimited         |
| **White-Label Reports**    | ❌        | ❌          | ❌          | ✅                | ✅                   |
| **SSO/SAML**               | ❌        | ❌          | ❌          | ❌                | ✅                   |
| **Support**                | Community | Email (48h) | Email (24h) | Priority (24-48h) | Dedicated (4-8h SLA) |

---

**Next Steps:**

1. Implement license key system (Sprint 1)
2. Build customer portal (Sprint 2)
3. Integrate Stripe billing (Sprint 2)
4. Create trial signup flow (Sprint 3)
5. Launch beta with Professional tier trial
