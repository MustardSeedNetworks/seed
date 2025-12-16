# CI/Build Tooling Gap Analysis

**Last Updated:** December 2025 **Purpose:** Identify professional-grade CI/build tools missing from
the pipeline

---

## ✅ Current Tooling (Excellent Coverage)

### Go Backend

- ✅ **golangci-lint** - Comprehensive linting (36+ linters enabled)
- ✅ **staticcheck** - Advanced Go static analysis
- ✅ **go vet** - Go suspicious construct detection
- ✅ **gofmt** - Code formatting
- ✅ **gosec** - Security vulnerability scanning
- ✅ **govulncheck** - Go vulnerability database check
- ✅ **gocyclo** - Cyclomatic complexity tracking
- ✅ **Race detector** - Concurrent access detection
- ✅ **Code coverage** - 40% minimum enforced (target: 90%)
- ✅ **Codecov** - Coverage reporting and tracking

### Frontend

- ✅ **ESLint** - TypeScript/React linting
- ✅ **Prettier** - Code formatting
- ✅ **TypeScript** - Type checking
- ✅ **Vitest** - Unit testing with coverage
- ✅ **Playwright** - E2E browser testing (Chromium, Firefox, WebKit)
- ✅ **Storybook** - Component documentation and visual testing

### Security

- ✅ **gitleaks** - Secret detection
- ✅ **Trivy** - Container/filesystem vulnerability scanning
- ✅ **npm audit** - Node.js dependency security
- ✅ **Pre-commit hooks** - Automated checks before commit

### Build & Deploy

- ✅ **Multi-arch builds** - AMD64 + ARM64
- ✅ **Deterministic builds** - `-trimpath -buildvcs=false`
- ✅ **Docker builds** - Multi-stage with testing
- ✅ **Smoke tests** - Automated deployment verification
- ✅ **Makefile** - Comprehensive build automation

---

## 🔴 HIGH PRIORITY - Missing Critical Tools

### 1. **Automated Dependency Updates**

**Problem:** Manual dependency updates are time-consuming and often forgotten.

**Recommendation:** Dependabot (GitHub native, free for private repos)

**Implementation:**

```yaml
# .github/dependabot.yml
version: 2
updates:
  # Go modules
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
    labels:
      - "dependencies"
      - "go"

  # npm packages
  - package-ecosystem: "npm"
    directory: "/web"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
    labels:
      - "dependencies"
      - "frontend"

  # Docker base images
  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"

  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

**Benefits:**

- Automated security updates
- Keeps dependencies current
- No cost (built into GitHub)
- Auto-merges patch updates (configurable)

**Alternative:** Renovate (more features, requires setup)

**Priority:** 🔴 Add immediately - security and maintenance

---

### 2. **License Compliance Checking**

**Problem:** Healthcare/enterprise customers require license compliance verification.

**Recommendation:** go-licenses + license-checker (npm)

**Implementation:**

```yaml
# .github/workflows/ci.yml - add to quality job
- name: Check Go license compliance
  run: |
    go install github.com/google/go-licenses@latest
    go-licenses check ./... --disallowed_types=forbidden,restricted

- name: Check npm license compliance
  working-directory: web
  run: |
    npx license-checker --production --onlyAllow "MIT;Apache-2.0;BSD-2-Clause;BSD-3-Clause;ISC;CC0-1.0"
```

**Makefile target:**

```makefile
license-check: ## Check license compliance
	@echo "Checking Go licenses..."
	go-licenses check ./... --disallowed_types=forbidden,restricted
	@echo "Checking npm licenses..."
	cd web && npx license-checker --production --onlyAllow "MIT;Apache-2.0;BSD-2-Clause;BSD-3-Clause;ISC;CC0-1.0"
```

**Benefits:**

- Avoid GPL/AGPL contamination
- Required for enterprise/healthcare sales
- Automated enforcement

**Priority:** 🔴 Add before first enterprise customer

---

### 3. **SBOM Generation (Software Bill of Materials)**

**Problem:** Required for government contracts, healthcare compliance, supply chain security.

**Recommendation:** syft + grype (Anchore)

**Implementation:**

```yaml
# .github/workflows/release.yml
- name: Generate SBOM
  uses: anchore/sbom-action@v0
  with:
    path: ./
    format: spdx-json
    output-file: luminetiq-sbom.spdx.json

- name: Scan SBOM for vulnerabilities
  uses: anchore/scan-action@v5
  with:
    sbom: luminetiq-sbom.spdx.json
    fail-build: true
    severity-cutoff: high
```

**Benefits:**

- Supply chain transparency
- Required for healthcare/gov compliance
- Free and open source

**Priority:** 🔴 Add before HIPAA certification

---

## 🟡 MEDIUM PRIORITY - Quality & Productivity

### 4. **Changelog Automation**

**Problem:** Manual changelog writing is inconsistent and often forgotten.

**Recommendation:** conventional-changelog (uses commit messages)

**Implementation:**

```json
// web/package.json - add scripts
"changelog": "conventional-changelog -p angular -i CHANGELOG.md -s",
"changelog:all": "conventional-changelog -p angular -i CHANGELOG.md -s -r 0"
```

**GitHub Action:**

```yaml
# .github/workflows/release.yml
- name: Generate changelog
  run: |
    npm install -g conventional-changelog-cli
    conventional-changelog -p angular -i CHANGELOG.md -s
```

**Benefits:**

- Auto-generated from commit messages
- Follows semantic versioning
- Reduces release overhead

**Priority:** 🟡 Add when releasing v1.0

---

### 5. **Bundle Size Tracking**

**Problem:** Frontend bundle size creep degrades performance.

**Recommendation:** bundlesize + size-limit

**Implementation:**

```json
// web/package.json
{
  "bundlesize": [
    {
      "path": "./dist/assets/*.js",
      "maxSize": "250 kB"
    },
    {
      "path": "./dist/assets/*.css",
      "maxSize": "50 kB"
    }
  ],
  "scripts": {
    "size": "size-limit",
    "size:why": "size-limit --why"
  },
  "size-limit": [
    {
      "path": "dist/assets/index-*.js",
      "limit": "250 KB"
    }
  ]
}
```

**CI integration:**

```yaml
# .github/workflows/ci.yml - add to frontend job
- name: Check bundle size
  working-directory: web
  run: npm run size
```

**Benefits:**

- Prevents performance regression
- Alerts on bloat before merge
- ~$0 cost

**Priority:** 🟡 Add when focusing on performance

---

### 6. **API Documentation Generation**

**Problem:** API docs are manually maintained and often outdated.

**Recommendation:** swag (Go Swagger) for auto-generated OpenAPI docs

**Implementation:**

```go
// Add Swagger annotations to handlers
// @Summary Get system status
// @Description Returns overall system health and status
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /api/status [get]
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

**Makefile:**

```makefile
docs-api: ## Generate API documentation
	swag init -g cmd/luminetiq/main.go -o docs/api
```

**CI check:**

```yaml
- name: Verify API docs are up to date
  run: |
    make docs-api
    git diff --exit-code docs/api/
```

**Benefits:**

- Always up-to-date API docs
- Interactive Swagger UI
- API testing interface

**Priority:** 🟡 Add when API is stable (post-v1.0)

---

### 7. **Performance Budget Enforcement (Lighthouse CI)**

**Problem:** No automated performance regression detection.

**Recommendation:** Lighthouse CI for performance/accessibility audits

**Implementation:**

```yaml
# .github/workflows/ci.yml
- name: Run Lighthouse CI
  uses: treosh/lighthouse-ci-action@v12
  with:
    urls: |
      https://localhost:8443
      https://localhost:8443/login
    uploadArtifacts: true
    temporaryPublicStorage: true
```

**lighthouserc.json:**

```json
{
  "ci": {
    "assert": {
      "assertions": {
        "categories:performance": ["error", { "minScore": 0.9 }],
        "categories:accessibility": ["error", { "minScore": 0.9 }],
        "categories:best-practices": ["error", { "minScore": 0.9 }],
        "first-contentful-paint": ["error", { "maxNumericValue": 2000 }]
      }
    }
  }
}
```

**Benefits:**

- Catch performance regressions
- Accessibility compliance
- SEO optimization

**Priority:** 🟡 Add when optimizing UX (Year 1)

---

### 8. **Code Quality Metrics (SonarCloud)**

**Problem:** No centralized code quality dashboard.

**Recommendation:** SonarCloud (free for open source, affordable for private)

**Implementation:**

```yaml
# .github/workflows/ci.yml
- name: SonarCloud Scan
  uses: SonarSource/sonarcloud-github-action@master
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
```

**Benefits:**

- Code smell detection
- Security hotspot tracking
- Technical debt measurement
- Coverage visualization

**Cost:** $10/month for private repos (or free if open source)

**Priority:** 🟡 Add when team grows (3+ developers)

---

## 🟢 LOW PRIORITY - Nice to Have

### 9. **Visual Regression Testing**

**Tool:** Chromatic (Storybook) or Percy

**Use case:** Catch unintended UI changes

**Cost:** $149/month (Chromatic) - expensive for early stage

**Priority:** 🟢 Add when UI is mature and team is larger

---

### 10. **Mutation Testing**

**Tool:** Stryker (JS) or go-mutesting (Go)

**Use case:** Verify test quality by mutating code

**Cost:** Time overhead (slow builds)

**Priority:** 🟢 Add when aiming for 95%+ coverage

---

### 11. **Release Automation**

**Tool:** semantic-release or goreleaser

**Use case:** Automated GitHub releases with binaries

**Priority:** 🟢 Add when releasing frequently (v1.0+)

---

### 12. **Container Image Signing**

**Tool:** cosign (Sigstore)

**Use case:** Verify Docker image authenticity

**Priority:** 🟢 Add when distributing containers to customers

---

## 📋 Recommended Implementation Plan

### Phase 1: Immediate (This Week)

1. ✅ Add format/format:check scripts to package.json (DONE)
2. 🔴 Add Dependabot configuration
3. 🔴 Add license compliance checking

### Phase 2: Before First Enterprise Customer (1-2 Months)

1. 🔴 SBOM generation (syft)
2. 🟡 Changelog automation
3. 🟡 Bundle size tracking

### Phase 3: Before v1.0 Release (3-6 Months)

1. 🟡 API documentation generation (Swagger)
2. 🟡 Lighthouse CI for performance budgets

### Phase 4: When Team Grows (6-12 Months)

1. 🟡 SonarCloud for code quality metrics
2. 🟢 Visual regression testing (Chromatic)
3. 🟢 Release automation (goreleaser)

### Phase 5: Enterprise/Gov Customers (Year 2)

1. 🟢 Container image signing
2. 🟢 Mutation testing
3. 🟢 SLSA provenance (supply chain security)

---

## 💰 Cost Analysis

| Tool                 | Cost        | Priority  | ROI        |
| -------------------- | ----------- | --------- | ---------- |
| Dependabot           | Free        | 🔴 High   | ⭐⭐⭐⭐⭐ |
| License checker      | Free        | 🔴 High   | ⭐⭐⭐⭐⭐ |
| SBOM (syft)          | Free        | 🔴 High   | ⭐⭐⭐⭐   |
| Changelog automation | Free        | 🟡 Medium | ⭐⭐⭐⭐   |
| Bundle size          | Free        | 🟡 Medium | ⭐⭐⭐⭐   |
| Lighthouse CI        | Free        | 🟡 Medium | ⭐⭐⭐⭐   |
| Swagger/OpenAPI      | Free        | 🟡 Medium | ⭐⭐⭐⭐   |
| SonarCloud           | $10/mo      | 🟡 Medium | ⭐⭐⭐     |
| Chromatic            | $149/mo     | 🟢 Low    | ⭐⭐       |
| Mutation testing     | Free (time) | 🟢 Low    | ⭐⭐       |

**Total additional monthly cost:** $10-159 (SonarCloud optional, Chromatic later)

---

## 🎯 Quick Wins (Add Today)

### 1. Dependabot

- 5 minutes to configure
- Immediate security benefit
- Zero cost

### 2. License compliance

- 10 minutes to add CI job
- Required for enterprise sales
- Zero cost

### 3. Bundle size tracking

- 15 minutes to configure
- Prevents performance regression
- Zero cost

**Total setup time:** ~30 minutes for significant quality improvements

---

## 🚫 Tools NOT Recommended

### CircleCI/Travis CI

**Why not:** GitHub Actions is superior and native

### Jenkins

**Why not:** Overkill for this project, maintenance overhead

### Snyk

**Why not:** Trivy + govulncheck + npm audit already cover this

### Code Climate (paid tier)

**Why not:** SonarCloud offers better value

### New Relic/DataDog APM

**Why not:** Premature for current stage (add at scale)

---

## Summary

**Current State:** Your CI pipeline is already **A+ quality** with comprehensive linting, testing,
and security scanning.

**Top 3 Missing Tools:**

1. 🔴 **Dependabot** - Automated dependency updates (add immediately)
2. 🔴 **License compliance** - Required for enterprise (add before first customer)
3. 🔴 **SBOM generation** - Required for healthcare/gov (add before HIPAA)

**Total cost to add all high-priority tools:** $0

**Next step:** Add Dependabot configuration today (5 minutes).
