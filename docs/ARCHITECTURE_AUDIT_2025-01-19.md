# Seed / Stem / NIAC Architecture Audit Report

**Date:** 2025-01-19
**Auditor:** Claude (Opus 4.5)
**Role:** Senior Platform Architect, DevOps Lead, Documentation Systems Auditor
**Scope:** Holistic analysis of Seed, Stem, and NIAC repositories

---

## 1. Executive Summary

### Critical Conceptual Clarification

**The premise of this audit request contains a fundamental misunderstanding.**

The request describes Seed/Stem/NIAC as a "layered, opinionated reference architecture" where:
- Seed = foundational patterns, primitives, templates
- Stem = applied implementations built from Seed
- NIAC = production-grade deployable systems

**This is incorrect.** These are **three independent products** from the same developer:

| Repo | Actual Purpose | Domain |
|------|----------------|--------|
| **Seed** | Network diagnostic appliance | Plug-in network analyzer |
| **Stem** | Network performance testing | RFC 2544, Y.1564 testing |
| **NIAC** | Network device simulator | SNMP device simulation |

They share:
- **Developer conventions** (coding style, tooling choices)
- **Infrastructure patterns** (Makefile structure, CI/CD)
- **UI framework** (React, Vite, Biome)

They do NOT share:
- Code inheritance
- Shared libraries
- Runtime dependencies

**The correct framing:** These are **sibling products** that should maintain **convention consistency**, not architectural layering.

---

### Overall Maturity Score: 7.5/10

**Significantly improved from previous 4/10 score.** Recent migration work addressed most structural issues.

### Top 5 Remaining Issues

| # | Issue | Severity | Impact |
|---|-------|----------|--------|
| 1 | **Cruft files committed** | High | `.gitignore.tmp`, `.gitignore.new`, `coverage.out` in repos |
| 2 | **LICENSE inconsistency** | High | Seed/Stem use BSL 1.1, NIAC uses MIT |
| 3 | **Go module naming drift** | Medium | `niac-go` vs `niac` pattern |
| 4 | **Documentation depth variance** | Medium | NIAC over-documented (30 files), Stem sparse (6 files) |
| 5 | **UI structure micro-drift** | Low | Minor differences in ui/src/ organization |

---

## 2. Consistency Findings

### 2.1 Directory Structure (✅ GOOD)

All three repos now follow the canonical structure:

```
{product}/
├── cmd/{product}/          ✓ All repos
├── internal/               ✓ All repos
│   ├── api/                ✓ All repos (standardized)
│   ├── services/           ✓ Seed/Stem (NIAC different domain)
│   ├── config/             ✓ All repos
│   └── logging/            ✓ All repos
├── ui/                     ✓ All repos
│   └── src/
│       ├── api/            ✓ All repos
│       ├── components/     ✓ All repos
│       ├── contexts/       ✓ All repos (plural)
│       ├── hooks/          ✓ All repos
│       ├── stores/         ✓ All repos
│       ├── styles/         ✓ All repos
│       ├── test/           ✓ All repos
│       ├── types/          ✓ All repos
│       │   └── generated/  ✓ All repos
│       └── utils/          ✓ All repos
├── tests/                  ✓ All repos
│   ├── integration/        ✓ All repos
│   ├── load/               ✓ All repos
│   └── smoke/              ✓ All repos
├── deploy/                 ✓ All repos
│   ├── deb/                ✓ All repos (standardized)
│   ├── rpm/                ✓ Seed/Stem
│   ├── systemd/            ✓ All repos
│   └── kubernetes/         ✓ NIAC only (appropriate)
├── configs/                ✓ All repos
├── scripts/                ✓ All repos
├── docs/                   ✓ All repos
├── mk/                     ✓ All repos (identical structure)
├── bin/                    ✓ All repos
└── _project_specs/         ✓ All repos
```

**Verdict:** Structure is now **consistent**. No action required.

### 2.2 Root-Level Files

| File | Seed | Stem | NIAC | Status |
|------|------|------|------|--------|
| `.clang-format` | ✓ | ✓ | ✗ | ⚠️ NIAC missing (has no C code, acceptable) |
| `.clang-tidy` | ✓ | ✓ | ✗ | ⚠️ NIAC missing (has no C code, acceptable) |
| `.editorconfig` | ✓ | ✓ | ✓ | ✅ |
| `.gitignore` | ✓ | ✓ | ✓ | ✅ |
| `.gitignore.new` | ✗ | ✗ | ✗ | ❌ CRUFT - Delete from Seed/Stem |
| `.gitignore.tmp` | ✗ | ✗ | ✗ | ❌ CRUFT - Delete from Seed/Stem |
| `.gitleaks.toml` | ✓ | ✓ | ✓ | ✅ |
| `.golangci.yml` | ✓ | ✓ | ✓ | ✅ |
| `.markdownlint.json` | ✓ | ✓ | ✓ | ✅ |
| `.nvmrc` | ✓ | ✓ | ✓ | ✅ |
| `.pre-commit-config.yaml` | ✓ | ✓ | ✓ | ✅ |
| `AGENTS.md` | ✓ | ✓ | ✓ | ✅ |
| `biome.json` | ✓ | ✓ | ✓ | ✅ |
| `CHANGELOG.md` | ✓ | ✓ | ✓ | ✅ |
| `commitlint.config.js` | ✓ | ✓ | ✓ | ✅ |
| `CONTRIBUTING.md` | ✓ | ✓ | ✓ | ✅ |
| `coverage.out` | ✗ | ✗ | ✗ | ❌ CRUFT - Should be gitignored |
| `Dockerfile` | ✓ | ✗ | ✗ | ⚠️ Stem/NIAC missing |
| `go.mod` | ✓ | ✓ | ✓ | ✅ |
| `LICENSE` | BSL | BSL | MIT | ⚠️ INCONSISTENT |
| `Makefile` | ✓ | ✓ | ✓ | ✅ |
| `package.json` | ✓ | ✓ | ✓ | ✅ |
| `project.toml` | ✓ | ✓ | ✓ | ✅ |
| `README.md` | ✓ | ✓ | ✓ | ✅ |
| `SECURITY.md` | ✓ | ✓ | ✓ | ✅ |
| `typos.toml` | ✓ | ✓ | ✓ | ✅ |

### 2.3 File Naming (✅ GOOD)

| Category | Convention | Status |
|----------|------------|--------|
| Markdown docs | `SCREAMING_SNAKE.md` | ✅ Consistent |
| Config files | `kebab-case.yaml` | ✅ Consistent |
| Go files | `snake_case.go` | ✅ Consistent |
| TypeScript | `PascalCase.tsx`, `camelCase.ts` | ✅ Consistent |
| Scripts | `kebab-case.sh` | ✅ Consistent |

### 2.4 mk/ Files (✅ IDENTICAL)

All three repos have identical mk/ structure:

```
mk/
├── build.mk
├── deps.mk
├── lint.mk
├── package.mk
├── security.mk
├── test.mk
└── vars.mk
```

### 2.5 npm Scripts (✅ IDENTICAL)

All three repos have identical package.json scripts:

```json
{
  "dev": "vite",
  "build": "tsc -b && vite build",
  "lint": "biome check src/",
  "lint:fix": "biome check --write src/",
  "format": "biome format --write src/",
  "test": "vitest run --reporter=verbose",
  "test:watch": "vitest --reporter=dot",
  "test:coverage": "vitest run --coverage --reporter=verbose",
  "test:e2e": "playwright test --reporter=list",
  "storybook": "storybook dev -p 6006",
  "build-storybook": "storybook build"
}
```

### 2.6 UI Package Naming (✅ CONSISTENT)

| Repo | Package Name |
|------|--------------|
| Seed | `seed-ui` |
| Stem | `stem-ui` |
| NIAC | `niac-ui` |

**Pattern:** `{product}-ui` ✅

---

## 3. Best-Practice Gaps

### 3.1 What Should Be Deleted

| File | Repo | Reason |
|------|------|--------|
| `.gitignore.new` | Seed, Stem | Temp file cruft |
| `.gitignore.tmp` | Seed, Stem | Temp file cruft |
| `coverage.out` | Seed, NIAC | Build artifact (should be gitignored) |
| `license_coverage.out` | Stem | Build artifact |
| `netif_coverage.out` | Stem | Build artifact |
| `niac-go.code-workspace` | NIAC | IDE-specific file |
| `SECURITY_REMEDIATION_PLAN.md` | NIAC | Should be in docs/archive/ |
| `VERSION` | NIAC | Redundant (version in go.mod) |
| `pyproject.toml` | NIAC | No Python in project |
| `config.yaml` | Seed | Should be in configs/ |

### 3.2 What Should Be Added

| Item | Repo | Reason |
|------|------|--------|
| `Dockerfile` | Stem, NIAC | Container deployment support |
| `i18n/` | NIAC UI | Internationalization (low priority) |

### 3.3 What Should Be Merged/Moved

| From | To | Reason |
|------|-----|--------|
| `SECURITY_REMEDIATION_PLAN.md` | `docs/archive/` | Internal tracking doc |
| `config.yaml` (Seed root) | `configs/seed.yaml` | Consistent location |
| `seed-dev.service` (deploy/) | `deploy/systemd/` | Consistent structure |

---

## 4. Recommended Canonical Standards

### 4.1 Directory Structure (Canonical)

```
{product}/
├── cmd/{product}/              # Binary entry point
├── internal/                   # Private packages
│   ├── api/                    # HTTP handlers
│   ├── config/                 # Configuration
│   ├── logging/                # Logging
│   └── services/               # Business logic
├── ui/                         # React frontend
│   ├── e2e/                    # Playwright tests
│   └── src/
│       ├── api/                # API client
│       ├── components/         # React components
│       ├── contexts/           # React contexts (plural)
│       ├── hooks/              # Custom hooks
│       ├── stores/             # State management
│       ├── styles/             # CSS/styles
│       ├── test/               # Test utilities
│       ├── types/generated/    # Generated types
│       └── utils/              # Utilities
├── tests/                      # Backend tests
│   ├── integration/
│   ├── load/
│   └── smoke/
├── deploy/                     # Deployment configs
│   ├── deb/                    # Debian packaging
│   ├── rpm/                    # RPM packaging
│   └── systemd/                # Systemd services
├── configs/                    # Default configs
├── scripts/                    # Build/dev scripts
├── docs/                       # Documentation
│   └── archive/                # Historical docs
├── mk/                         # Makefile includes
│   ├── build.mk
│   ├── deps.mk
│   ├── lint.mk
│   ├── package.mk
│   ├── security.mk
│   ├── test.mk
│   └── vars.mk
├── bin/                        # Built binaries (gitignored)
├── _project_specs/             # AI/session specs
├── .claude/                    # Claude config
├── .github/                    # GitHub config
│   ├── CODEOWNERS
│   ├── ISSUE_TEMPLATE/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── workflows/
└── [root config files]
```

### 4.2 Root Config Files (Required)

```
.editorconfig
.gitignore
.gitleaks.toml
.golangci.yml
.markdownlint.json
.nvmrc
.pre-commit-config.yaml
AGENTS.md
biome.json
CHANGELOG.md
commitlint.config.js
CONTRIBUTING.md
Dockerfile
go.mod
go.sum
LICENSE
Makefile
package.json
package-lock.json
project.toml
README.md
SECURITY.md
typos.toml
```

### 4.3 File Naming Rules

| Type | Convention | Example |
|------|------------|---------|
| Markdown docs | SCREAMING_SNAKE | `API_REFERENCE.md` |
| YAML configs | kebab-case | `docker-compose.yaml` |
| Shell scripts | kebab-case | `build-release.sh` |
| Go files | snake_case | `http_handler.go` |
| Go test files | snake_case_test | `http_handler_test.go` |
| TypeScript components | PascalCase | `DeviceList.tsx` |
| TypeScript utilities | camelCase | `formatDate.ts` |
| CSS/styles | kebab-case | `device-list.css` |

### 4.4 Documentation Rules

1. **README.md** - Product overview, quick start, badges
2. **ARCHITECTURE.md** - System design, component relationships
3. **DEVELOPMENT.md** - Local setup, build instructions
4. **DEPLOYMENT.md** - Production deployment guide
5. **DISTRIBUTION.md** - Packaging and release info
6. **API_REFERENCE.md** - API documentation (if applicable)
7. **CLI_REFERENCE.md** - CLI documentation (if applicable)

**Archive Policy:** Planning docs, reports, and historical content → `docs/archive/`

---

## 5. Actionable Refactor Plan

### Phase 1: Immediate Cleanup (Safe, No Breaking Changes)

```bash
# Delete cruft files
rm /Users/krisarmstrong/Developer/seed/.gitignore.new
rm /Users/krisarmstrong/Developer/seed/.gitignore.tmp
rm /Users/krisarmstrong/Developer/stem/.gitignore.new
rm /Users/krisarmstrong/Developer/stem/.gitignore.tmp
rm /Users/krisarmstrong/Developer/seed/coverage.out
rm /Users/krisarmstrong/Developer/stem/license_coverage.out
rm /Users/krisarmstrong/Developer/stem/netif_coverage.out
rm /Users/krisarmstrong/Developer/niac/go/coverage.out
rm /Users/krisarmstrong/Developer/niac/go/niac-go.code-workspace
rm /Users/krisarmstrong/Developer/niac/go/VERSION
rm /Users/krisarmstrong/Developer/niac/go/pyproject.toml

# Move misplaced files
mv /Users/krisarmstrong/Developer/niac/go/SECURITY_REMEDIATION_PLAN.md \
   /Users/krisarmstrong/Developer/niac/go/docs/archive/reports/
mv /Users/krisarmstrong/Developer/seed/config.yaml \
   /Users/krisarmstrong/Developer/seed/configs/seed.yaml
mv /Users/krisarmstrong/Developer/seed/deploy/seed-dev.service \
   /Users/krisarmstrong/Developer/seed/deploy/systemd/

# Ensure coverage.out is gitignored
echo "coverage.out" >> /Users/krisarmstrong/Developer/seed/.gitignore
echo "coverage.out" >> /Users/krisarmstrong/Developer/niac/go/.gitignore
echo "*.coverage.out" >> /Users/krisarmstrong/Developer/stem/.gitignore
```

**Effort:** 5 minutes
**Risk:** None

### Phase 2: License Decision (Requires Decision)

**Current State:**
- Seed: BSL 1.1
- Stem: BSL 1.1
- NIAC: MIT

**Options:**
1. Keep as-is (acceptable if intentional)
2. Standardize all to BSL 1.1
3. Standardize all to MIT

**Recommendation:** Make intentional decision and document why.

### Phase 3: Go Module Naming (Optional)

**Current:**
- `github.com/krisarmstrong/seed`
- `github.com/krisarmstrong/stem`
- `github.com/krisarmstrong/niac-go`

**Recommendation:** Keep as-is. The `-go` suffix in NIAC is because it lives in a `go/` subdirectory (repo also has `java/`). This is intentional and appropriate.

### Phase 4: Documentation Consolidation (Optional Improvement)

NIAC has 30 docs files vs Stem's 6. Consider:

1. Moving product-specific detailed docs to a central `msn-docs-internal` repo
2. Keeping only essential docs in each product repo:
   - README.md
   - ARCHITECTURE.md
   - DEVELOPMENT.md
   - DEPLOYMENT.md
   - DISTRIBUTION.md

**This is optional** - current state is functional.

---

## 6. Optional Enhancements (Clearly Marked)

### 6.1 Shared Conventions Package (OPTIONAL)

Create a shared repo for:
- `.golangci.yml` template
- `biome.json` template
- `Makefile` templates
- GitHub Actions workflows

**Benefit:** Single source of truth for conventions
**Effort:** Medium
**Recommendation:** Not urgent - current copy-paste approach works

### 6.2 Monorepo Consideration (OPTIONAL, NOT RECOMMENDED)

Could consolidate all three into a monorepo.

**Pros:**
- Single clone for all products
- Shared CI/CD

**Cons:**
- Products are independent
- Different release cycles
- Increased complexity

**Recommendation:** Keep separate repos. They are independent products.

### 6.3 Documentation Site (OPTIONAL)

Create MkDocs/Docusaurus site for all MSN products.

**Benefit:** Professional documentation portal
**Effort:** High
**Recommendation:** Future consideration, not urgent

---

## 7. Summary

### What's Good

1. ✅ Directory structure is now consistent
2. ✅ File naming conventions are consistent
3. ✅ npm scripts are identical
4. ✅ mk/ structure is identical
5. ✅ UI package naming follows pattern
6. ✅ Root config files are consistent
7. ✅ E2E tests exist in all repos

### What Needs Fixing (Immediate)

1. ❌ Delete cruft files (`.gitignore.tmp`, `.gitignore.new`, `coverage.out`, etc.)
2. ❌ Move misplaced files (`config.yaml`, `SECURITY_REMEDIATION_PLAN.md`)
3. ❌ Add `coverage.out` to `.gitignore`

### What Needs Decision

1. ⚠️ License consistency (BSL vs MIT)
2. ⚠️ Dockerfile for Stem/NIAC (if containerization needed)

### What's Optional

1. Shared conventions repo
2. Documentation consolidation
3. Documentation site

---

**End of Audit**
