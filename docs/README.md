# The Seed - Documentation

**Organization:** Mustard Seed Networks
**Product:** The Seed
**Repository:** Private (proprietary commercial software)

---

## 📂 Documentation Structure

```
docs/
├── wiki/          # 👥 USER-FACING (external, will be published)
├── internal/      # 🔒 TEAM-ONLY (never published)
└── reference/     # 📚 META (documentation about documentation)
```

---

## 👥 `wiki/` - User-Facing Documentation (External)

**Purpose:** Help users install, configure, and use The Seed

**Audience:** Customers, beta testers, community

**Will be published to:**
- GitHub Wiki (when enabled)
- OR public docs site (docs.mustardseednetworks.com)
- Currently accessible to beta testers with repo access

**Contents:**
- Installation guides (macOS, Linux, Docker)
- Quick start tutorials
- Feature documentation
- Hardware compatibility
- API reference
- Troubleshooting
- FAQ

**Update frequency:** With each release, when bugs fixed, when hardware tested

---

## 🔒 `internal/` - Team Documentation (Never Published)

**Purpose:** Internal knowledge base for strategy, sales, engineering

**Audience:** Team members only (founder, employees, contractors with access)

**Contents:**

### Business & Strategy
- BUSINESS_PLAN.md - 3-year financial projections, market sizing
- MARKETING_STRATEGY.md - Go-to-market, budgets, ROI
- LICENSING_STRATEGY.md - Pricing tiers, enforcement
- HEALTHCARE_MARKET_STRATEGY.md - Healthcare vertical strategy
- COMPETITIVE_ANALYSIS.md - Detailed competitor analysis
- WIFI_COMPETITIVE_ANALYSIS.md - WiFi market deep dive
- PRODUCT_ROADMAP.md - Feature roadmap

### Sales & Support
- SALES_PLAYBOOK.md - Sales methodology, battle cards
- SUPPORT_STRATEGY.md - Support tiers, SLAs, TAM

### Product & Engineering
- AI_QA_STRATEGY.md - AI-powered testing strategy
- AI_TOOLS_STRATEGY.md - Which AI tools for development
- AI_INTEGRATION_PLAN.md - AI integration roadmap
- TESTING_REQUIREMENTS.md - Hardware/software for test lab
- HARDWARE_PHASE4_PLAN.md - Hardware roadmap
- HARDWARE_PHASE5_PLAN.md - Future hardware plans
- COMPLIANCE_MAPPINGS.md - Regulatory compliance details

### Brand & Design
- BRAND_GUIDELINES.md - Visual identity, voice & tone

### Project Management
- TODO.md - Task tracking
- VERIFICATION.md - Verification checklists
- AI_ISSUES_SUMMARY.md - AI-related issues summary

**Update frequency:** Quarterly review, before major milestones

---

## 📚 `reference/` - Meta Documentation

**Purpose:** Documentation about documentation (guides, templates, summaries)

**Audience:** Team members, documentation maintainers

**Contents:**
- DOCUMENTATION_STRUCTURE.md - Complete guide to doc organization
- WIKI_CONTENT.md - Source content for wiki population
- SETUP_COMPLETE.md - Summary of all documentation created
- AI_README.md - AI integration documentation guide

**Update frequency:** When documentation structure changes

---

## 🚀 Quick Start

### For Beta Testers / Users
→ Start here: [wiki/Home.md](wiki/Home.md)

### For Team Members
→ Start here: [reference/DOCUMENTATION_STRUCTURE.md](reference/DOCUMENTATION_STRUCTURE.md)

### For Sales Team
→ Start here: [internal/SALES_PLAYBOOK.md](internal/SALES_PLAYBOOK.md)

### For Support Team
→ Start here: [internal/SUPPORT_STRATEGY.md](internal/SUPPORT_STRATEGY.md)

---

## 📋 What Goes Where?

| Content | Location | Published? |
|---------|----------|------------|
| Installation guide | `wiki/` | ✅ Yes (when wiki enabled) |
| Feature documentation | `wiki/` | ✅ Yes |
| Hardware compatibility | `wiki/` | ✅ Yes |
| FAQ | `wiki/` | ✅ Yes |
| API reference | `wiki/` | ✅ Yes |
| Business plan | `internal/` | ❌ Never |
| Sales playbook | `internal/` | ❌ Never |
| Pricing strategy | `internal/` | ❌ Never |
| Competitive analysis | `internal/` | ❌ Never |
| Support procedures | `internal/` | ❌ Never |
| Documentation guides | `reference/` | ❌ Never |

---

## 🔄 Publishing Workflow

### Now (Private Repo)
- User docs: `wiki/` directory (beta testers with repo access)
- Team docs: `internal/` directory (team only)
- Reference: `reference/` directory (team only)

### Future: Public Docs Site
When The Seed launches publicly:

1. **Build docs site** (GitBook or Docusaurus)
2. **Publish `wiki/` content** to docs.mustardseednetworks.com
3. **Keep `internal/` private** (never published)
4. **Option:** Move `internal/` to separate private repo for access control

---

## 📞 Contact

**Questions about documentation:**
- Email: kris.armstrong@mustardseednetworks.com

---

*From a tiny seed, a mighty network grows.*

**Mustard Seed Networks**
