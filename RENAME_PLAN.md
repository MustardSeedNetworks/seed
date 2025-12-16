# Mustard Seed Networks - Comprehensive Rename Plan

## Overview

This document outlines the complete plan to rename the project from **NetScope/LuminetIQ** to
**Mustard Seed Networks**.

## Brand Hierarchy

| Level            | Name                               | Description                   |
| ---------------- | ---------------------------------- | ----------------------------- |
| **Company**      | Mustard Seed Networks              | The company/organization name |
| **Platform**     | Mustard Seed Networks Platform     | The overall product suite     |
| **Core Product** | The Seed                           | The main application (binary) |
| **Modules**      | Roots, Canopy, Shell, Sap, Harvest | Feature modules               |

## Module Definitions

| Module      | Function                                      | Metaphor                       |
| ----------- | --------------------------------------------- | ------------------------------ |
| **Roots**   | Path Analysis (traceroute, deep connectivity) | Foundation beneath the surface |
| **Canopy**  | Wi-Fi Planning & Surveys                      | The expansive coverage above   |
| **Shell**   | Security Posture & Hardening                  | Protective outer layer         |
| **Sap**     | Live Telemetry & Monitoring                   | Lifeblood flowing through      |
| **Harvest** | Reports, compliance, exports                  | The fruits of your labor       |

## Naming Convention

| Context          | Format                          | Example                              |
| ---------------- | ------------------------------- | ------------------------------------ |
| Company/Brand    | Mustard Seed Networks           | "Welcome to Mustard Seed Networks"   |
| Platform         | Mustard Seed Networks Platform  | "The Mustard Seed Networks Platform" |
| Core Product     | The Seed                        | "The Seed v1.0.0"                    |
| Binary/CLI       | `seed`                          | `./seed --version`                   |
| Go Module        | `github.com/krisarmstrong/seed` | import paths                         |
| Package Name     | `seed`                          | Go package name                      |
| Storage Keys     | `seed-*`                        | `seed-token`, `seed-theme`           |
| Service Account  | `seed`                          | systemd user                         |
| Config Directory | `~/.config/seed/`               | user config                          |
| System Config    | `/etc/seed/`                    | system config                        |

## Current → New Mapping

| Current                | New                                                  |
| ---------------------- | ---------------------------------------------------- |
| NetScope               | Mustard Seed Networks (company) / The Seed (product) |
| LuminetIQ              | Mustard Seed Networks (company) / The Seed (product) |
| `luminetiq` (binary)   | `seed`                                               |
| `netscope` (code refs) | `seed`                                               |
| `netscope-*` (storage) | `seed-*`                                             |

---

## Phase 1: Pre-Rename Preparation

### 1.1 Secure External Assets

- [ ] Register domain: `mustardseednetworks.com`
- [ ] Register domain: `mustardseednetworks.io`
- [ ] Create GitHub organization: `mustardseednetworks`
- [ ] Secure Twitter/X handle: `@mustardseednet` or `@theseednet`
- [ ] Secure LinkedIn company page: `/company/mustardseednetworks`
- [ ] Consult trademark attorney

### 1.2 Rename GitHub Repository

- [ ] Rename `krisarmstrong/netscope` to `krisarmstrong/seed`
- [ ] Update repository description
- [ ] Configure repository settings (branch protection, etc.)
- [ ] Update GitHub Actions secrets if needed

> **Note:** Business GitHub organization (`mustardseednetworks`) will be created later. Initial
> development continues on personal account.

### 1.3 Backup Current State

- [ ] Tag current release: `pre-rename-backup`
- [ ] Export all issues and PRs
- [ ] Document current CI/CD configuration

---

## Phase 2: Go Module Rename

### 2.1 Update go.mod

**File:** `go.mod`

```diff
- module github.com/krisarmstrong/luminetiq
+ module github.com/krisarmstrong/seed
```

### 2.2 Update All Go Imports (30+ files, ~150 occurrences)

**Pattern to replace:**

```text
github.com/krisarmstrong/luminetiq/internal/*
→
github.com/krisarmstrong/seed/internal/*
```

**Files affected:**

- `cmd/luminetiq/main.go` → `cmd/seed/main.go`
- `internal/api/*.go` (15+ files)
- `internal/auth/*.go`
- `internal/cable/*.go`
- `internal/config/*.go`
- `internal/dhcp/*.go`
- `internal/discovery/*.go`
- `internal/dns/*.go`
- `internal/gateway/*.go`
- `internal/iperf/*.go`
- `internal/logging/*.go`
- `internal/network/*.go`
- `internal/publicip/*.go`
- `internal/snmp/*.go`
- `internal/speedtest/*.go`
- `internal/survey/*.go`
- `internal/system/*.go`
- `internal/validation/*.go`
- `internal/version/*.go`
- `internal/vlan/*.go`
- `internal/wifi/*.go`
- All `*_test.go` files

### 2.3 Update .golangci.yml

```diff
- - github.com/krisarmstrong/luminetiq
+ - github.com/krisarmstrong/seed
```

---

## Phase 3: Directory Rename

### 3.1 Rename Command Directory

```bash
git mv cmd/luminetiq cmd/seed
```

### 3.2 Rename Config File

```bash
git mv configs/netscope.yaml configs/seed.yaml
```

### 3.3 Rename Packaging Files

```bash
git mv packaging/luminetiq.spec packaging/seed.spec
git mv packaging/luminetiq.service packaging/seed.service
git mv deploy/systemd/luminetiq.service deploy/systemd/seed.service
git mv deploy/luminetiq-dev.service deploy/seed-dev.service
```

### 3.4 Rename Web Assets

```bash
git mv web/public/luminetiq.svg web/public/seed.svg
```

---

## Phase 4: Build System Updates

### 4.1 Makefile Changes

| Line  | Current                     | New                                        |
| ----- | --------------------------- | ------------------------------------------ |
| 43    | `BINARY_NAME=luminetiq`     | `BINARY_NAME=seed`                         |
| 61-63 | LDFLAGS with luminetiq path | Update module path to `krisarmstrong/seed` |
| All   | `./cmd/luminetiq`           | `./cmd/seed`                               |
| All   | Deploy paths with luminetiq | Deploy paths with seed                     |

### 4.2 Dockerfile

```diff
- go build ... -o /luminetiq ./cmd/luminetiq
+ go build ... -o /seed ./cmd/seed
```

---

## Phase 5: Frontend Updates

### 5.1 Storage Keys (web/src/hooks/useAuth.ts)

```diff
- "netscope-token"
- "netscope-token-expiry"
- "netscope-username"
+ "seed-token"
+ "seed-token-expiry"
+ "seed-username"
```

### 5.2 Theme Storage (web/src/hooks/useTheme.ts)

```diff
- const STORAGE_KEY = "netscope-theme"
+ const STORAGE_KEY = "seed-theme"
```

### 5.3 HTML Updates (web/index.html)

```diff
- <link rel="icon" type="image/svg+xml" href="/luminetiq.svg" />
- <title>LuminetIQ</title>
- <meta name="description" content="LuminetIQ - Illuminate Your Network" />
+ <link rel="icon" type="image/svg+xml" href="/seed.svg" />
+ <title>The Seed | Mustard Seed Networks</title>
+ <meta name="description" content="The Seed - Network diagnostic platform by Mustard Seed Networks" />
```

### 5.4 SVG Label (web/public/seed.svg)

```diff
- aria-label="NetScope"
+ aria-label="The Seed"
```

### 5.5 Component Updates

- `web/src/App.tsx` - Update comments
- `web/src/main.tsx` - Update comments
- `web/src/components/settings/SettingsDrawer.tsx` - Update version display
- `web/src/lib/api.ts` - Update comments
- `web/src/hooks/useWebSocket.ts` - Update comments

---

## Phase 6: Backend Code Updates

### 6.1 Display Text (cmd/seed/main.go)

- Update ASCII art banner to "THE SEED" or "MUSTARD SEED NETWORKS"
- Update version display: `"The Seed %s (Mustard Seed Networks)\n"`
- Update setup wizard text
- Update credential display

### 6.2 ICMP/Network Payloads

**internal/discovery/icmp.go:**

```diff
- data := []byte("netscope")
+ data := []byte("seed")
```

**internal/discovery/traceroute.go:**

```diff
- data := []byte("NETSCOPE")
+ data := []byte("SEED")
```

### 6.3 Config Paths (internal/discovery/oui.go)

```diff
- filepath.Join(os.Getenv("HOME"), ".config", "netscope", "oui.txt")
+ filepath.Join(os.Getenv("HOME"), ".config", "seed", "oui.txt")
```

### 6.4 Export Filenames (internal/api/handlers_status.go)

```diff
- "netscope-export.json"
+ "seed-export.json"
```

### 6.5 Default Password (internal/auth/auth.go)

Consider updating default password from "netscope" to "seed"

---

## Phase 7: Service & Deployment Updates

### 7.1 Systemd Service Files

**deploy/systemd/seed.service:**

```diff
- Description=LuminetIQ Network Diagnostic Tool
- ExecStart=/usr/local/bin/luminetiq
- User=luminetiq
- Group=luminetiq
+ Description=The Seed - Mustard Seed Networks Platform
+ ExecStart=/usr/local/bin/seed
+ User=seed
+ Group=seed
```

### 7.2 Install/Uninstall Scripts

- `deploy/systemd/install.sh` - Update all references
- `deploy/systemd/uninstall.sh` - Update all references
- `scripts/deploy-systemd.sh` - Update all references

### 7.3 Packaging Files

**packaging/control:**

```diff
- Package: netscope
- Description: NetScope is...
+ Package: seed
+ Description: The Seed is the core product of Mustard Seed Networks...
```

**packaging/seed.spec:**

- Update Name, URL, description
- Update all file paths

---

## Phase 8: CI/CD Updates

### 8.1 GitHub Workflows

**.github/workflows/ci.yml:**

- Update comment header to "The Seed - Mustard Seed Networks"
- Update binary artifact names: `seed-linux-amd64`, `seed-darwin-amd64`, etc.

**.github/workflows/release-please.yml:**

- Update all binary references
- Update artifact names to `seed-*`

**.github/workflows/release.yml:**

- Update binary names
- Update download URLs

**.github/workflows/docker-publish.yml:**

- Update comment header
- Update image names if applicable

### 8.2 Issue Templates

- `.github/ISSUE_TEMPLATE/hardware-report.yml` - Update branding to "The Seed"
- `.github/ISSUE_TEMPLATE/bug_report.md` - Update version label to "The Seed Version"

---

## Phase 9: Documentation Updates

### 9.1 Root Documentation

| File              | Changes Required                                    |
| ----------------- | --------------------------------------------------- |
| `README.md`       | Title, description, installation commands, examples |
| `CONTRIBUTING.md` | All NetScope/LuminetIQ references                   |
| `SECURITY.md`     | Product name references                             |
| `LICENSE`         | "Licensed Work: Mustard Seed Networks"              |
| `STYLE_GUIDE.md`  | Title, import examples                              |
| `PROJECT_PLAN.md` | Project name references                             |
| `HARDWARE.md`     | Product name references                             |

### 9.2 Docs Directory

- `docs/VERIFICATION.md` - All LuminetIQ references
- All wiki files in `docs/wiki/` (25+ files)

### 9.3 Web Documentation

- `web/docs/ui-style.md` - Title update
- `web/THEMING.md` - Check for references

### 9.4 Claude Configuration

Claude Code reads CLAUDE.md from multiple locations (all are merged):

1. `~/.claude/CLAUDE.md` - User-level global (not project-specific)
2. `PROJECT_ROOT/CLAUDE.md` - Project root
3. `PROJECT_ROOT/.claude/CLAUDE.md` - Project hidden directory

#### File: `.claude/CLAUDE.md`

This file requires extensive updates:

| Line  | Current                                       | New                                          |
| ----- | --------------------------------------------- | -------------------------------------------- |
| 1     | `# Claude Operating Guidelines for LuminetIQ` | `# Claude Operating Guidelines for The Seed` |
| 35    | `cd ~/luminetiq && git pull`                  | `cd ~/seed && git pull`                      |
| 63    | `/home/krisarmstrong/luminetiq`               | `/home/krisarmstrong/seed`                   |
| 64    | `./luminetiq`                                 | `./seed`                                     |
| 75-76 | `systemctl ... luminetiq`                     | `systemctl ... seed`                         |
| 76    | `journalctl -u luminetiq`                     | `journalctl -u seed`                         |
| 78-81 | `luminetiq credentials`                       | `seed credentials`                           |
| 104   | `luminetiq/`                                  | `seed/`                                      |
| 105   | `cmd/luminetiq/`                              | `cmd/seed/`                                  |

**Full rewrite recommended** to reflect:

- Brand: "The Seed" (product) by "Mustard Seed Networks" (company)
- Binary: `seed`
- Service: `seed.service`
- Paths: `/home/krisarmstrong/seed`, `cmd/seed/`
- Module metaphors: Roots, Canopy, Shell, Sap, Harvest

#### File: `.claude/settings.local.json`

This file contains permission allowlist entries with old paths/names:

| Line(s)        | Current                  | New        |
| -------------- | ------------------------ | ---------- |
| 7              | `netscope` in SSH path   | `seed`     |
| 23             | `netscope` in deploy cmd | `seed`     |
| 34, 56-57      | `/netscope` paths        | `/seed`    |
| 59-60          | `netscope` binary refs   | `seed`     |
| 70             | `./luminetiq:*`          | `./seed:*` |
| 92-99, 124-125 | `/netscope` in git paths | `/seed`    |

**After rename:** These permissions may need regenerating as paths change. Consider clearing and
re-allowing as you use the new paths.

---

## Phase 10: Test Updates

### 10.1 Go Tests

- `internal/auth/auth_test.go` - Update test passwords
- All test files with import path updates

### 10.2 Frontend Tests

- `web/src/App.test.tsx` - Update storage key references
- `web/src/hooks/useAuth.test.ts` - Update all netscope references

---

## Phase 11: GitHub Repository Rename

### 11.1 Rename Repository

```bash
# Rename repository via GitHub UI or CLI
gh repo rename seed --repo krisarmstrong/netscope

# Update local remote
git remote set-url origin git@github.com:krisarmstrong/seed.git
```

### 11.2 Update All References

- Update any hardcoded GitHub URLs in code/docs
- Update CI/CD badge URLs in README
- Update issue/PR templates with new repo URL

### 11.3 Future: Business Organization Migration

When ready to move to business account:

```bash
# Transfer repository to organization
# Done via GitHub UI: Settings > Transfer ownership

# Update remote after transfer
git remote set-url origin git@github.com:mustardseednetworks/seed.git
```

---

## Phase 12: Verification

### 12.1 Build Verification

```bash
make clean
make all
./seed version
```

### 12.2 Search for Remaining References

```bash
# Check for any remaining old names (should return no results)
grep -ri "netscope" --include="*.go" --include="*.ts" --include="*.tsx" \
  --include="*.md" --include="*.yaml" --include="*.yml" --include="*.json" .

grep -ri "luminetiq" --include="*.go" --include="*.ts" --include="*.tsx" \
  --include="*.md" --include="*.yaml" --include="*.yml" --include="*.json" .
```

### 12.3 Test All Features

- [ ] Build passes
- [ ] All tests pass
- [ ] Frontend builds
- [ ] Docker image builds
- [ ] Service starts correctly
- [ ] Web UI loads
- [ ] Authentication works
- [ ] All API endpoints respond

---

## Execution Order

1. **Phase 1** - Secure external assets (domains, GitHub org, social)
2. **Phase 2** - Go module rename (most critical)
3. **Phase 3** - Directory renames
4. **Phase 4** - Build system updates
5. **Phase 5** - Frontend updates
6. **Phase 6** - Backend code updates
7. **Phase 7** - Service & deployment updates
8. **Phase 8** - CI/CD updates
9. **Phase 9** - Documentation updates
10. **Phase 10** - Test updates
11. **Phase 11** - GitHub repository migration
12. **Phase 12** - Verification

---

## Rollback Plan

If issues arise:

1. Revert to `pre-rename-backup` tag
2. Keep old repository active
3. Document lessons learned

---

## Post-Rename Tasks

- [ ] Update any external links
- [ ] Notify users of name change
- [ ] Update package registries
- [ ] Update Docker Hub/container registries
- [ ] Monitor for broken links/references
- [ ] Close issues #501, #504, #505

---

## Detailed File Audit

### Files Containing "netscope" (39 unique files)

| Category       | File                                             | Reference Type                                    |
| -------------- | ------------------------------------------------ | ------------------------------------------------- |
| **Auth**       | `internal/auth/auth.go`                          | Default password "netscope", storage key comments |
| **Auth**       | `internal/auth/auth_test.go`                     | Test password references                          |
| **Discovery**  | `internal/discovery/icmp.go`                     | ICMP payload `[]byte("netscope")`                 |
| **Discovery**  | `internal/discovery/traceroute.go`               | Traceroute payload `[]byte("NETSCOPE")`           |
| **Discovery**  | `internal/discovery/oui.go`                      | Config path `~/.config/netscope/`                 |
| **API**        | `internal/api/handlers_status.go`                | Export filename `netscope-export.json`            |
| **Config**     | `internal/config/config.go`                      | Example domain references                         |
| **Config**     | `configs/netscope.yaml`                          | **FILE RENAME REQUIRED**                          |
| **Frontend**   | `web/src/hooks/useAuth.ts`                       | Storage keys `netscope-token`, etc.               |
| **Frontend**   | `web/src/hooks/useAuth.test.ts`                  | Test storage key references                       |
| **Frontend**   | `web/src/hooks/useTheme.ts`                      | Storage key `netscope-theme`                      |
| **Frontend**   | `web/src/types/settings.ts`                      | Storage key references                            |
| **Frontend**   | `web/src/App.test.tsx`                           | Test storage keys                                 |
| **Frontend**   | `web/src/index.css`                              | CSS comment "NetScope Theme"                      |
| **Assets**     | `web/public/luminetiq.svg`                       | aria-label="NetScope"                             |
| **Build**      | `Dockerfile`                                     | Build output path references                      |
| **Packaging**  | `packaging/luminetiq.spec`                       | GitHub URL                                        |
| **Packaging**  | `packaging/control`                              | Package name "netscope"                           |
| **Scripts**    | `scripts/build-iperf3.sh`                        | NetScope references                               |
| **Docs**       | `README.md`                                      | Product name references                           |
| **Docs**       | `CONTRIBUTING.md`                                | NetScope references                               |
| **Docs**       | `SECURITY.md`                                    | NetScope references                               |
| **Docs**       | `PROJECT_PLAN.md`                                | NetScope references                               |
| **Docs**       | `LICENSE`                                        | "Licensed Work" reference                         |
| **Docs**       | `web/docs/ui-style.md`                           | NetScope references                               |
| **Issues**     | `.github/ISSUE_TEMPLATE/bug_report.md`           | "NetScope Version" label                          |
| **Types**      | `web/src/types/index.ts`                         | NetScope type references                          |
| **Components** | `web/src/components/cards/SystemHealthCard.tsx`  | NetScope references                               |
| **Components** | `web/src/components/settings/SettingsDrawer.tsx` | NetScope references                               |

### Files Containing "luminetiq" (99 unique files)

| Category          | Count     | Examples                                                              |
| ----------------- | --------- | --------------------------------------------------------------------- |
| **Go Imports**    | 48 files  | All `internal/*/*.go` files with `github.com/krisarmstrong/luminetiq` |
| **Makefile**      | 1 file    | BINARY_NAME, LDFLAGS, all paths                                       |
| **Dockerfile**    | 1 file    | Build output, entry point                                             |
| **Web Config**    | 8 files   | vite.config.ts, eslint.config.mjs, package.json, etc.                 |
| **Service Files** | 5 files   | systemd services, install/uninstall scripts                           |
| **Workflows**     | 4 files   | CI, release, docker-publish                                           |
| **Documentation** | 25+ files | Wiki, templates, README, guides                                       |
| **Frontend**      | 15+ files | Components, hooks, lib, styles                                        |

### Directories Requiring Rename

| Current Path                       | New Path                      |
| ---------------------------------- | ----------------------------- |
| `cmd/luminetiq/`                   | `cmd/seed/`                   |
| `configs/netscope.yaml`            | `configs/seed.yaml`           |
| `web/public/luminetiq.svg`         | `web/public/seed.svg`         |
| `packaging/luminetiq.spec`         | `packaging/seed.spec`         |
| `packaging/luminetiq.service`      | `packaging/seed.service`      |
| `deploy/systemd/luminetiq.service` | `deploy/systemd/seed.service` |
| `deploy/luminetiq-dev.service`     | `deploy/seed-dev.service`     |

### Special Cases

1. **CSS Theme Comment** (`web/src/index.css` line 48):

   ```css
   /* NetScope Theme - WiFi Vigilante Color Scheme */
   ```

   → Change to: `/* The Seed Theme - Mustard Seed Networks */`

2. **HTML Meta/Title** (`web/index.html`):
   - favicon: `/luminetiq.svg` → `/seed.svg`
   - apple-touch-icon: `/luminetiq.svg` → `/seed.svg`
   - description: "LuminetIQ - Illuminate Your Network" → "The Seed - Network diagnostics by Mustard
     Seed Networks"
   - title: "LuminetIQ" → "The Seed | Mustard Seed Networks"

3. **SVG aria-label** (`web/public/luminetiq.svg`):

   ```svg
   aria-label="NetScope"
   ```

   → Change to: `aria-label="The Seed"`

4. **Makefile grep pattern** (line 216):

   ```makefile
   ps aux | grep '[l]uminetiq'
   ```

   → Change to: `ps aux | grep '[s]eed'`

5. **Go module path in 48 files**:

   ```go
   github.com/krisarmstrong/luminetiq/internal/*
   ```

   → Change to: `github.com/krisarmstrong/seed/internal/*`

---

## Files Summary

### Total Files to Modify: ~100+

| Category               | Count |
| ---------------------- | ----- |
| Go source files        | 35+   |
| Go test files          | 15+   |
| TypeScript/React files | 20+   |
| Documentation (MD)     | 30+   |
| Configuration files    | 10+   |
| Workflow files         | 5     |
| Shell scripts          | 8     |
| Service files          | 4     |
| HTML files             | 2     |
| Other                  | 5+    |

---

## Related Issues

### Master Tracker

- **#621** - Master Tracker: Mustard Seed Networks Rebrand

### Phase Issues

| Phase | Issue | Description                  |
| ----- | ----- | ---------------------------- |
| 1     | #609  | Pre-Rename Preparation       |
| 2     | #610  | Go Module Rename (CRITICAL)  |
| 3     | #611  | Directory Renames            |
| 4     | #612  | Build System Updates         |
| 5     | #613  | Frontend Updates             |
| 6     | #614  | Backend Code Updates         |
| 7     | #615  | Service & Deployment Updates |
| 8     | #616  | CI/CD Updates                |
| 9     | #617  | Documentation Updates        |
| 10    | #618  | Test Updates                 |
| 11    | #619  | GitHub Repository Rename     |
| 12    | #620  | Verification & Cleanup       |

### Supporting Issues

- #502 - Brand Identity (Logo, Colors, Typography)
- #503 - Marketing Website & Brand Presence
- #506 - Onboarding Flows for The Seed

### Closed (Superseded)

- ~~#501~~ - Kernel Networks naming (closed - superseded)
- ~~#504~~ - Kernel Networks renaming (closed - superseded)
- ~~#505~~ - Documentation updates (closed - superseded)

---

## Gap Analysis Update (December 16, 2025)

**Automated scan results:**

- 302 occurrences of "netscope" across 59 files (plan documented 39)
- 767 occurrences of "luminetiq" across 133 files (plan documented 99)

### Missing E2E Tests (31 files) - All contain "luminetiq" references

| File                                              | Occurrences |
| ------------------------------------------------- | ----------- |
| `web/e2e/auth-complete.spec.ts`                   | 11          |
| `web/e2e/settings.spec.ts`                        | 10          |
| `web/e2e/vulnerability-scanning-complete.spec.ts` | 10          |
| `web/e2e/responsive.spec.ts`                      | 7           |
| `web/e2e/vlan.spec.ts`                            | 4           |
| `web/e2e/error-scenarios.spec.ts`                 | 4           |
| `web/e2e/websocket-realtime.spec.ts`              | 3           |
| `web/e2e/snmp-settings.spec.ts`                   | 3           |
| `web/e2e/link-card.spec.ts`                       | 3           |
| `web/e2e/dns-card.spec.ts`                        | 3           |
| `web/e2e/cable-diagnostics.spec.ts`               | 3           |
| `web/e2e/gateway.spec.ts`                         | 2           |
| `web/e2e/iperf.spec.ts`                           | 2           |
| `web/e2e/auth.spec.ts`                            | 2           |
| `web/e2e/system-health.spec.ts`                   | 2           |
| `web/e2e/theme-and-help.spec.ts`                  | 1           |
| `web/e2e/speed-test.spec.ts`                      | 1           |
| `web/e2e/speed-test-complete.spec.ts`             | 1           |
| `web/e2e/setup-wizard.spec.ts`                    | 1           |
| `web/e2e/smoke.spec.ts`                           | 1           |
| `web/e2e/public-ip.spec.ts`                       | 1           |
| `web/e2e/network-discovery-complete.spec.ts`      | 1           |
| `web/e2e/network-discovery.spec.ts`               | 1           |
| `web/e2e/wifi-survey.spec.ts`                     | 1           |
| `web/e2e/wifi-survey-complete.spec.ts`            | 1           |
| `web/e2e/websocket.spec.ts`                       | 1           |
| `web/e2e/vulnerabilities.spec.ts`                 | 1           |
| `web/e2e/interface-switching.spec.ts`             | 1           |
| `web/e2e/fab.spec.ts`                             | 1           |
| `web/e2e/dashboard.spec.ts`                       | 1           |
| `web/e2e/card-lifecycle.spec.ts`                  | 1           |

### Missing Internal Documentation (10 files)

| File                                          | Contains                     |
| --------------------------------------------- | ---------------------------- |
| `docs/internal/COMPLIANCE_MAPPINGS.md`        | luminetiq (62 occurrences)   |
| `docs/internal/HEALTHCARE_MARKET_STRATEGY.md` | luminetiq (41 occurrences)   |
| `docs/internal/WIFI_COMPETITIVE_ANALYSIS.md`  | luminetiq (35 occurrences)   |
| `docs/internal/VERIFICATION.md`               | luminetiq (23), netscope (7) |
| `docs/internal/HARDWARE_PHASE5_PLAN.md`       | luminetiq (16)               |
| `docs/internal/LICENSING_STRATEGY.md`         | luminetiq (15)               |
| `docs/internal/HARDWARE_PHASE4_PLAN.md`       | luminetiq (13)               |
| `docs/internal/AI_INTEGRATION_PLAN.md`        | luminetiq (5)                |
| `docs/internal/AI_ISSUES_SUMMARY.md`          | luminetiq (5)                |
| `docs/internal/PRODUCT_ROADMAP.md`            | luminetiq (1), netscope (4)  |
| `docs/internal/BUSINESS_PLAN.md`              | netscope (2)                 |
| `docs/internal/MARKETING_STRATEGY.md`         | luminetiq (1)                |

### Missing Reference/Template Documentation (5 files)

| File                                     | Contains      |
| ---------------------------------------- | ------------- |
| `docs/reference/CI_TOOLING_ANALYSIS.md`  | luminetiq (3) |
| `docs/reference/SETUP_COMPLETE.md`       | netscope (1)  |
| `docs/templates/ETHERNET_TEST_REPORT.md` | luminetiq (3) |
| `docs/templates/WIFI_TEST_REPORT.md`     | luminetiq (3) |
| `docs/DOCUMENTATION_STRUCTURE.md`        | netscope (4)  |
| `docs/WIKI_CONTENT.md`                   | netscope (28) |

### Missing Wiki Documentation (7 files)

| File                              | Contains     |
| --------------------------------- | ------------ |
| `docs/wiki/FAQ.md`                | netscope (5) |
| `docs/wiki/Installation-Linux.md` | netscope (3) |
| `docs/wiki/Home.md`               | netscope (2) |
| `docs/wiki/Installation-macOS.md` | netscope (2) |
| `docs/wiki/Network-Discovery.md`  | netscope (1) |
| `docs/wiki/Quick-Start-Guide.md`  | netscope (1) |

### Missing Config Files (8 files)

| File                      | Contains                    |
| ------------------------- | --------------------------- |
| `.gitleaks.toml`          | luminetiq (2)               |
| `.pre-commit-config.yaml` | luminetiq (1)               |
| `web/vitest.config.ts`    | luminetiq (1)               |
| `web/vite.config.ts`      | luminetiq (1)               |
| `web/vite.config.js`      | luminetiq (1)               |
| `web/vite.config.d.ts`    | luminetiq (1)               |
| `web/postcss.config.js`   | luminetiq (1)               |
| `docs/AI_README.md`       | luminetiq (3), netscope (4) |

### Missing Storybook Stories (2 files)

| File                                                    | Contains      |
| ------------------------------------------------------- | ------------- |
| `web/src/components/cards/SystemHealthCard.stories.tsx` | netscope (8)  |
| `web/src/components/help/ImprovedHelpModal.stories.tsx` | luminetiq (1) |

### Missing Scripts (4 files)

| File                                     | Contains       |
| ---------------------------------------- | -------------- |
| `scripts/test-dhcp-rogue.sh`             | luminetiq (13) |
| `scripts/setup-wiki.sh`                  | netscope (28)  |
| `scripts/test-hardware-compatibility.sh` | luminetiq (3)  |
| `scripts/re-version.sh`                  | luminetiq (1)  |

### Missing Planning Documents (2 files)

| File                     | Contains      |
| ------------------------ | ------------- |
| `IMPLEMENTATION_PLAN.md` | luminetiq (1) |
| `HARDWARE.md`            | luminetiq (7) |

### Generated/Build Artifacts (Ignore but note)

| File                                 | Contains                    | Action               |
| ------------------------------------ | --------------------------- | -------------------- |
| `web/playwright-report/results.json` | netscope (44)               | Ignore - regenerated |
| `web/package-lock.json`              | luminetiq (2), netscope (2) | Auto-updated by npm  |
| `package-lock.json`                  | netscope (2)                | Auto-updated by npm  |

### Updated Totals

| Category               | Original Count | Actual Count | Delta |
| ---------------------- | -------------- | ------------ | ----- |
| Files with "netscope"  | 39             | 59           | +20   |
| Files with "luminetiq" | 99             | 133          | +34   |
| **Total unique files** | ~100           | ~145         | +45   |

---

---

## Execution Plan

### Recommended Approach: Phased PRs

Execute the rename in 5 focused PRs to make review manageable:

| PR       | Phases | Files | Description                            |
| -------- | ------ | ----- | -------------------------------------- |
| **PR 1** | 2-4    | ~55   | Go module + directories + build system |
| **PR 2** | 5-6    | ~22   | Frontend + backend code                |
| **PR 3** | 7-8    | ~12   | Services + CI/CD                       |
| **PR 4** | 9-10   | ~68   | Documentation + tests                  |
| **PR 5** | 11-12  | -     | Repository rename + verification       |

### Pre-Requisites (Before PR 1)

1. Complete Phase 1 external asset preparation (#609)
2. Tag current release: `git tag pre-rename-backup && git push --tags`
3. Ensure all CI checks pass on main branch

### Execution Commands

#### PR 1: Go Module + Build System

```bash
# Update go.mod
sed -i '' 's|github.com/krisarmstrong/luminetiq|github.com/krisarmstrong/seed|g' go.mod

# Update all Go imports
find . -name '*.go' -exec sed -i '' 's|github.com/krisarmstrong/luminetiq|github.com/krisarmstrong/seed|g' {} +

# Update .golangci.yml
sed -i '' 's|github.com/krisarmstrong/luminetiq|github.com/krisarmstrong/seed|g' .golangci.yml

# Rename directories
git mv cmd/luminetiq cmd/seed
git mv configs/netscope.yaml configs/seed.yaml

# Update Makefile
sed -i '' 's|BINARY_NAME=luminetiq|BINARY_NAME=seed|g' Makefile
sed -i '' 's|./cmd/luminetiq|./cmd/seed|g' Makefile

# Verify
go mod tidy && go build ./... && go test ./...
```

#### PR 2: Frontend + Backend

```bash
# Update storage keys
find web/src -name '*.ts' -o -name '*.tsx' | xargs sed -i '' 's/netscope-/seed-/g'

# Update HTML
sed -i '' 's/luminetiq/seed/g' web/index.html
sed -i '' 's/LuminetIQ/The Seed/g' web/index.html

# Update backend payloads
sed -i '' 's/netscope/seed/g' internal/discovery/icmp.go
sed -i '' 's/NETSCOPE/SEED/g' internal/discovery/traceroute.go

# Rename web assets
git mv web/public/luminetiq.svg web/public/seed.svg
```

#### PR 3: Services + CI/CD

```bash
# Rename service files
git mv deploy/systemd/luminetiq.service deploy/systemd/seed.service
git mv deploy/luminetiq-dev.service deploy/seed-dev.service
git mv packaging/luminetiq.spec packaging/seed.spec
git mv packaging/luminetiq.service packaging/seed.service

# Update service content
find deploy scripts packaging -type f -exec sed -i '' 's/luminetiq/seed/g' {} +

# Update workflows
find .github -name '*.yml' -exec sed -i '' 's/luminetiq/seed/g' {} +
```

#### PR 4: Documentation + Tests

```bash
# Update docs
find docs -name '*.md' -exec sed -i '' 's/netscope/seed/g' {} +
find docs -name '*.md' -exec sed -i '' 's/luminetiq/seed/g' {} +
find docs -name '*.md' -exec sed -i '' 's/NetScope/The Seed/g' {} +
find docs -name '*.md' -exec sed -i '' 's/LuminetIQ/Mustard Seed Networks/g' {} +

# Update E2E tests
find web/e2e -name '*.spec.ts' -exec sed -i '' 's/luminetiq/seed/g' {} +

# Update root docs
sed -i '' 's/netscope/seed/g' README.md CONTRIBUTING.md SECURITY.md
```

#### PR 5: Repository Rename

```bash
# After all code PRs merged:
gh repo rename seed --repo krisarmstrong/netscope
git remote set-url origin git@github.com:krisarmstrong/seed.git

# Final verification
grep -ri 'netscope' --include='*.go' --include='*.ts' --include='*.md' .
grep -ri 'luminetiq' --include='*.go' --include='*.ts' --include='*.md' .
# Both should return NO results
```

### Time Estimate

| Activity                | Time                |
| ----------------------- | ------------------- |
| Phase 1 prep            | 1-2 days (external) |
| PR 1 (Go/build)         | 2-3 hours           |
| PR 2 (frontend/backend) | 2-3 hours           |
| PR 3 (services/CI)      | 1-2 hours           |
| PR 4 (docs/tests)       | 3-4 hours           |
| PR 5 (repo rename)      | 1 hour              |
| **Total code work**     | **~12-15 hours**    |

---

_Plan created: December 15, 2025_ _Gap analysis update: December 16, 2025_ _Execution plan added:
December 16, 2025_ _Target completion: TBD_
