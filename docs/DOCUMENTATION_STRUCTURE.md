# The Seed - Documentation Structure

**Version:** 1.0 **Last Updated:** December 2025 **Purpose:** Complete guide to documentation
organization

---

## Overview

The Seed maintains **two types of documentation**:

1. **User-Facing:** Installation, features, troubleshooting (GitHub Wiki)
2. **Team/Internal:** Strategy, sales, engineering (docs/ folder in repo)

Since the repository is **private and proprietary**, both are currently in the same repo but serve
different purposes.

---

## Current Structure

```
netscope/ (private repo)
├── docs/
│   ├── wiki/                           # Hardware compatibility wiki
│   │   ├── Home.md
│   │   ├── Intel-WiFi.md
│   │   ├── Broadcom-Ethernet.md
│   │   └── ... (11 hardware pages)
│   │
│   ├── BUSINESS_PLAN.md                # 3-year business strategy (TEAM)
│   ├── SALES_PLAYBOOK.md               # Sales methodology (TEAM)
│   ├── MARKETING_STRATEGY.md           # Marketing plan (TEAM)
│   ├── SUPPORT_STRATEGY.md             # Support tiers & SLAs (TEAM)
│   ├── COMPETITIVE_ANALYSIS.md         # Market analysis (TEAM)
│   ├── BRAND_GUIDELINES.md             # Brand identity (TEAM)
│   ├── TESTING_REQUIREMENTS.md         # Test lab specs (TEAM)
│   ├── AI_QA_STRATEGY.md               # AI testing (TEAM)
│   ├── AI_TOOLS_STRATEGY.md            # Which AI for what (TEAM)
│   ├── LICENSING_STRATEGY.md           # Pricing tiers (TEAM)
│   ├── WIFI_COMPETITIVE_ANALYSIS.md    # WiFi market deep dive (TEAM)
│   └── WIKI_CONTENT.md                 # Source for GitHub Wiki (REFERENCE)
│
├── scripts/
│   └── setup-wiki.sh                   # Automated wiki population
│
└── README.md                            # Project overview
```

---

## User-Facing Documentation (GitHub Wiki)

**Purpose:** Help users install, configure, and use The Seed

**Audience:**

- Beta testers (with repo access)
- Customers
- Community contributors

**Location:** https://github.com/krisarmstrong/netscope/wiki

**Content:**

- Installation guides (macOS, Linux, Docker)
- Quick start tutorial
- Feature documentation (Network Discovery, WiFi Survey, etc.)
- Hardware compatibility (tested adapters/NICs)
- Troubleshooting (common issues, error messages)
- FAQ (general questions)
- API reference (for developers integrating with The Seed)

**Format:** Markdown (GitHub Wiki)

**Update Frequency:**

- With each release (new features, breaking changes)
- As bugs/issues are discovered and fixed
- When hardware is tested

---

## Team Documentation (docs/ folder)

**Purpose:** Internal knowledge base for team planning and execution

**Audience:**

- Kris Armstrong (founder)
- Future employees (sales, support, engineering)
- Contractors (with repo access)

**Location:** `docs/` folder in private repo

**Content:**

### Business & Strategy

- `BUSINESS_PLAN.md` - 3-year financial projections, SBA loan analysis, market sizing
- `MARKETING_STRATEGY.md` - Go-to-market plan, budgets, ROI analysis
- `LICENSING_STRATEGY.md` - Pricing tiers, enforcement, upgrade paths
- `COMPETITIVE_ANALYSIS.md` - Detailed competitor analysis with battle cards

### Sales & Support

- `SALES_PLAYBOOK.md` - Sales methodology, objection handling, demo scripts
- `SUPPORT_STRATEGY.md` - Support tiers, SLAs, escalation procedures, TAM

### Product & Engineering

- `AI_QA_STRATEGY.md` - AI-powered testing strategy (unit, E2E, security)
- `AI_TOOLS_STRATEGY.md` - Which AI tools to use for development
- `TESTING_REQUIREMENTS.md` - Hardware/software for test lab
- `BRAND_GUIDELINES.md` - Visual identity, voice & tone, messaging

### Reference

- `WIKI_CONTENT.md` - Source content for populating GitHub Wiki

**Format:** Markdown

**Update Frequency:**

- Quarterly review (business plan, competitive analysis)
- As needed (when strategy changes)
- Before major milestones (launch, funding, hiring)

---

## Populating the GitHub Wiki

The GitHub Wiki is currently **empty** and needs to be initialized.

### One-Time Setup

**Step 1: Initialize Wiki (Manual)**

1. Go to https://github.com/krisarmstrong/netscope/wiki
2. Click **"Create the first page"**
3. Title: `Home`
4. Content: `Initializing wiki...`
5. Click **"Save Page"**

**Step 2: Run Setup Script (Automated)**

```bash
./scripts/setup-wiki.sh
```

This will:

- Clone the wiki repository
- Create 6 wiki pages with full content:
  - Home (welcome page)
  - Installation-macOS
  - Installation-Linux
  - Quick-Start-Guide
  - Network-Discovery
  - Hardware-Compatibility
  - FAQ
- Commit and push to GitHub

**Step 3: Verify**

Visit https://github.com/krisarmstrong/netscope/wiki and confirm all pages are live.

---

## Future: Public Docs Site (Post-Launch)

When The Seed launches publicly, create a professional docs site:

### Recommended Tool: GitBook

**Why GitBook:**

- Beautiful UI (better than GitHub Wiki)
- SEO-friendly (indexed by Google)
- Versioning (docs for v1.0, v2.0, etc.)
- Search (better than GitHub's)
- Analytics (see what users read)

**Pricing:**

- Free for open-source docs (if you make docs public)
- $29/month for private docs (team access)

**Setup:**

1. Create account at https://www.gitbook.com
2. Connect to GitHub repo
3. Auto-sync from `docs/user-guide/` folder
4. Publish to `docs.mustardseednetworks.com`

**Structure:**

```
docs.mustardseednetworks.com/
├── Getting Started
│   ├── Installation
│   ├── Quick Start
│   └── Configuration
├── Features
│   ├── Network Discovery
│   ├── WiFi Survey
│   ├── Vulnerability Scanning
│   └── Compliance Reporting
├── API Reference
│   ├── Authentication
│   ├── REST Endpoints
│   └── WebSocket Events
├── Hardware
│   └── Compatibility Matrix
└── Troubleshooting
    ├── Common Issues
    └── Error Messages
```

---

## Alternative: Docusaurus (Free, Open Source)

**Why Docusaurus:**

- 100% free (Facebook's tool)
- React-based (customizable)
- Versioning built-in
- Search built-in
- Markdown-based (easy to write)

**Setup:**

```bash
npx create-docusaurus@latest seed-docs classic
cd seed-docs
npm start
```

**Deploy to:** GitHub Pages, Netlify, or Vercel (all free)

---

## Internal Documentation (Future: Separate Repo or Notion)

**When team grows beyond 1 person**, separate internal docs:

### Option 1: Private GitHub Repo (Recommended for now)

Create `mustard-seed-networks/internal-docs` (private):

```
internal-docs/
├── engineering/
│   ├── architecture-decisions.md
│   ├── deployment-procedures.md
│   └── performance-benchmarks.md
├── sales/
│   ├── battle-cards.md
│   ├── pricing-negotiation.md
│   └── demo-scripts.md
├── support/
│   ├── advanced-troubleshooting.md
│   ├── known-issues.md
│   └── customer-quirks.md
└── business/
    ├── financial-models.md
    └── partnerships.md
```

**Access Control:**

- Employees: Full access
- Contractors: Engineering only (no sales/business)
- Partners: Support only (no engineering/sales)

### Option 2: Notion (Best for Growing Teams)

**When:** Team grows to 3+ people

**Why Notion:**

- Non-technical team can contribute (sales, support)
- Great search, templates, databases
- Beautiful UI, easy collaboration
- Mobile apps

**Pricing:** Free for <10 users, $8/user/month after

**Structure:**

```
Mustard Seed Networks Workspace
├── 📚 Engineering
│   ├── Architecture
│   ├── Deployment
│   └── Incident Post-Mortems
├── 💰 Sales
│   ├── Battle Cards
│   ├── Customer Profiles
│   └── Deal Tracker
├── 🎧 Support
│   ├── Troubleshooting
│   ├── Known Issues
│   └── Escalation Guide
└── 🏢 Business
    ├── Financial Models
    ├── OKRs
    └── Meeting Notes
```

---

## Documentation Workflow

### User-Facing (GitHub Wiki)

**When to Update:**

- New release (document new features)
- Bug fixes (update troubleshooting)
- Hardware tested (add to compatibility matrix)

**Who Updates:**

- Engineering team (feature docs)
- Support team (FAQ, troubleshooting)
- Community (via PRs to docs/ folder, then copied to wiki)

**Review Process:**

- Kris reviews all changes
- Test docs before publishing (ensure accuracy)

### Team Documentation (docs/ folder)

**When to Update:**

- Quarterly reviews (business plan, competitive analysis)
- Major milestones (funding, launch, hiring)
- Strategy changes (pricing, positioning)

**Who Updates:**

- Kris (business, strategy)
- Sales (playbooks, battle cards when hired)
- Engineering (technical docs)

**Review Process:**

- No formal review (private to team)
- Version controlled (git history)

---

## Maintenance Schedule

### Weekly

- Check GitHub issues for doc bugs
- Update FAQ if new common questions emerge

### Monthly

- Review wiki analytics (what are users reading?)
- Identify gaps (what's missing?)

### Quarterly

- Review and update:
  - BUSINESS_PLAN.md (financial projections)
  - COMPETITIVE_ANALYSIS.md (new competitors, pricing changes)
  - MARKETING_STRATEGY.md (campaign results)
- Check for outdated screenshots/examples

### Annually

- Major documentation refresh
- Competitive deep-dive
- User survey ("What docs do you need?")

---

## Quick Reference: What Goes Where?

| Content                | Location                     | Audience             |
| ---------------------- | ---------------------------- | -------------------- |
| Installation guide     | GitHub Wiki                  | Users                |
| Quick start tutorial   | GitHub Wiki                  | Users                |
| Feature documentation  | GitHub Wiki                  | Users                |
| Hardware compatibility | GitHub Wiki (docs/wiki/)     | Users                |
| API reference          | GitHub Wiki                  | Developers           |
| Troubleshooting        | GitHub Wiki                  | Users                |
| FAQ                    | GitHub Wiki                  | Users                |
| Business plan          | docs/BUSINESS_PLAN.md        | Team                 |
| Sales playbook         | docs/SALES_PLAYBOOK.md       | Team (Sales)         |
| Marketing strategy     | docs/MARKETING_STRATEGY.md   | Team (Marketing)     |
| Support strategy       | docs/SUPPORT_STRATEGY.md     | Team (Support)       |
| Competitive analysis   | docs/COMPETITIVE_ANALYSIS.md | Team (Sales)         |
| Brand guidelines       | docs/BRAND_GUIDELINES.md     | Team (Marketing)     |
| Testing requirements   | docs/TESTING_REQUIREMENTS.md | Team (Engineering)   |
| Architecture decisions | Future: internal-docs/       | Team (Engineering)   |
| Customer notes         | Future: internal-docs/       | Team (Support/Sales) |

---

## Action Items

- [x] Create team documentation (BUSINESS_PLAN, SALES_PLAYBOOK, etc.)
- [x] Update hardware wiki with correct naming
- [x] Create wiki setup script (`scripts/setup-wiki.sh`)
- [ ] Initialize GitHub Wiki (manual: create first page)
- [ ] Run wiki setup script to populate pages
- [ ] Create internal docs repo when team grows
- [ ] Consider GitBook/Docusaurus for public docs post-launch

---

**Document Owner:** Kris Armstrong **Last Updated:** December 2025 **Next Review:** After wiki
population, then quarterly
