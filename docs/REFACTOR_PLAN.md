# Seed Module Refactor Plan

**Date:** 2025-12-30
**Target Architecture:** THE_SEED_ARCHITECTURE.md v2.0
**Goal:** Reorganize packages into module structure without feature loss

---

## 1. Package Mapping Table

### Current → Target Mapping

| Current Package | Target Module | Target Path | Notes |
|-----------------|---------------|-------------|-------|
| `internal/paths` | **Roots** 🟤 | `internal/roots/traceroute` | Traceroute, path analysis |
| `internal/publicip` | **Roots** 🟤 | `internal/roots/enrichment` | ASN lookup, geolocation |
| `internal/wifi` | **Canopy** 🟢 | `internal/canopy/wifi` | WiFi scanning, connection |
| `internal/survey` | **Canopy** 🟢 | `internal/canopy/survey` | WiFi surveys, floor plans |
| `internal/discovery` | **Shell** 🟠 | `internal/shell/discovery` | ARP, LLDP, CDP, NDP, device enumeration |
| `internal/discovery` (vuln) | **Shell** 🟠 | `internal/shell/vulnerability` | CVE lookup, KEV, risk scoring |
| `internal/dhcp` (rogue) | **Shell** 🟠 | `internal/shell/rogue` | Rogue DHCP detection |
| `internal/dhcp` (testing) | **Sap** 🔵 | `internal/sap/dhcp` | DHCP tester, timing |
| `internal/dns` | **Sap** 🔵 | `internal/sap/dns` | DNS testing |
| `internal/gateway` | **Sap** 🔵 | `internal/sap/gateway` | Gateway health monitoring |
| `internal/snmp` | **Sap** 🔵 | `internal/sap/snmp` | SNMP collector |
| `internal/cable` | **Sap** 🔵 | `internal/sap/cable` | TDR cable testing |
| `internal/iperf` | **Sap** 🔵 | `internal/sap/performance` | iPerf client/server |
| `internal/speedtest` | **Sap** 🔵 | `internal/sap/performance` | Speed testing |
| `internal/phy` | **Sap** 🔵 | `internal/sap/link` | Physical layer info |
| `internal/network` | **Sap** 🔵 | `internal/sap/link` | Link monitoring |
| `internal/vlan` | **Sap** 🔵 | `internal/sap/vlan` | VLAN management |
| (new) | **Harvest** 🟡 | `internal/harvest/generator` | PDF/HTML/CSV/JSON export |
| (new) | **Harvest** 🟡 | `internal/harvest/templates` | Report templates |

### Infrastructure (Unchanged)

| Package | Status | Notes |
|---------|--------|-------|
| `internal/api` | Restructure handlers | Split into module-based handler files |
| `internal/auth` | Keep | Authentication/authorization |
| `internal/config` | Keep + extend | Add deployment detection |
| `internal/database` | Keep | Database layer |
| `internal/i18n` | Keep | Internationalization |
| `internal/logging` | Keep | Structured logging |
| `internal/system` | Keep | System info |
| `internal/validation` | Keep | Input validation |
| `internal/version` | Keep | Version info |
| `internal/testutil` | Keep | Test utilities |
| `internal/constants` | Keep | Constants |
| `internal/oauth` | Keep | OAuth/SSO |
| `internal/mcp` | Keep | MCP provider |

---

## 2. API Endpoint Mapping

### Current → Target Route Mapping

| Current Route | Target Route | Module | Handler File |
|---------------|--------------|--------|--------------|
| `GET /api/paths/trace` | `GET /api/roots/traceroute` | Roots | `handlers_roots.go` |
| `POST /api/paths/trace` | `POST /api/roots/traceroute` | Roots | `handlers_roots.go` |
| `GET /api/network/publicip` | `GET /api/roots/enrichment/:ip` | Roots | `handlers_roots.go` |
| `GET /api/wifi/scan` | `GET /api/canopy/scan` | Canopy | `handlers_canopy.go` |
| `POST /api/wifi/connect` | `POST /api/canopy/connect` | Canopy | `handlers_canopy.go` |
| `GET /api/survey/*` | `GET /api/canopy/survey/*` | Canopy | `handlers_canopy.go` |
| `POST /api/discovery/start` | `POST /api/shell/discover` | Shell | `handlers_shell.go` |
| `GET /api/discovery/devices` | `GET /api/shell/devices` | Shell | `handlers_shell.go` |
| `GET /api/discovery/device/:id` | `GET /api/shell/device/:id` | Shell | `handlers_shell.go` |
| `GET /api/vulnerabilities` | `GET /api/shell/vulnerabilities` | Shell | `handlers_shell.go` |
| `POST /api/vulnerabilities/scan` | `POST /api/shell/scan/vulnerability` | Shell | `handlers_shell.go` |
| `GET /api/dhcp/test` | `GET /api/sap/dhcp/test` | Sap | `handlers_sap.go` |
| `POST /api/dhcp/test` | `POST /api/sap/dhcp/test` | Sap | `handlers_sap.go` |
| `GET /api/dns/test` | `GET /api/sap/dns/test` | Sap | `handlers_sap.go` |
| `POST /api/dns/test` | `POST /api/sap/dns/test` | Sap | `handlers_sap.go` |
| `GET /api/gateway/health` | `GET /api/sap/gateway/health` | Sap | `handlers_sap.go` |
| `GET /api/snmp/*` | `GET /api/sap/switch/*` | Sap | `handlers_sap.go` |
| `GET /api/cable/test` | `GET /api/sap/cable/test` | Sap | `handlers_sap.go` |
| `POST /api/speedtest` | `POST /api/sap/speedtest` | Sap | `handlers_sap.go` |
| `GET /api/iperf/*` | `GET /api/sap/iperf/*` | Sap | `handlers_sap.go` |
| `GET /api/network/status` | `GET /api/sap/link/status` | Sap | `handlers_sap.go` |
| `GET /api/vlan/*` | `GET /api/sap/vlan/*` | Sap | `handlers_sap.go` |
| `GET /api/system/health` | `GET /api/sap/system/health` | Sap | `handlers_sap.go` |
| (new) | `POST /api/harvest/generate` | Harvest | `handlers_harvest.go` |
| (new) | `GET /api/harvest/templates` | Harvest | `handlers_harvest.go` |
| (new) | `GET /api/harvest/reports` | Harvest | `handlers_harvest.go` |

### Legacy Route Support

Keep legacy routes working with HTTP redirects until v2.0:

```go
// Legacy → New redirects (deprecation warnings in response headers)
router.GET("/api/paths/trace", redirectTo("/api/roots/traceroute"))
router.GET("/api/wifi/scan", redirectTo("/api/canopy/scan"))
router.POST("/api/discovery/start", redirectTo("/api/shell/discover"))
```

---

## 3. Migration Phases

### Phase 1: Directory Structure (Week 1)

**Goal:** Create module directories without moving code

```bash
# Create module directory structure
mkdir -p internal/roots/{traceroute,topology,enrichment,analysis}
mkdir -p internal/canopy/{wifi,survey,channel,ai}
mkdir -p internal/shell/{discovery,vulnerability,posture,rogue}
mkdir -p internal/sap/{link,cable,dhcp,dns,gateway,snmp,performance,vlan,telemetry}
mkdir -p internal/harvest/{generator,templates,scheduler,aggregator}
```

**Checkpoint:** Directories exist, existing code unchanged, all tests pass

### Phase 2: Module Scaffolding (Week 1-2)

**Goal:** Create module interfaces and types

For each module, create:
- `module.go` - Module interface and registration
- `types.go` - Module-specific types
- `service.go` - Main service implementation

Example for Roots:
```go
// internal/roots/module.go
package roots

type Module struct {
    traceroute *TracerouteService
    topology   *TopologyService
    enrichment *EnrichmentService
}

func New(cfg *config.Config, db *database.DB) *Module {
    return &Module{...}
}
```

**Checkpoint:** Module scaffolds exist, can be imported, all tests pass

### Phase 3: Code Migration - Sap Module (Week 2-3)

**Why Sap first:** Most independent, clearest boundaries, heavily used

1. Move `internal/dns` → `internal/sap/dns` (rename package)
2. Move `internal/gateway` → `internal/sap/gateway`
3. Move `internal/cable` → `internal/sap/cable`
4. Move `internal/snmp` → `internal/sap/snmp`
5. Move `internal/speedtest` + `internal/iperf` → `internal/sap/performance`
6. Move `internal/phy` + `internal/network` (link parts) → `internal/sap/link`
7. Move `internal/vlan` → `internal/sap/vlan`
8. Split `internal/dhcp` (testing parts) → `internal/sap/dhcp`

**Checkpoint:** Sap module complete, all Sap-related tests pass

### Phase 4: Code Migration - Shell Module (Week 3-4)

1. Move `internal/discovery` (device enumeration) → `internal/shell/discovery`
2. Move vulnerability scanning → `internal/shell/vulnerability`
3. Move rogue detection from `internal/dhcp` → `internal/shell/rogue`

**Checkpoint:** Shell module complete, discovery tests pass

### Phase 5: Code Migration - Canopy Module (Week 4-5)

1. Move `internal/wifi` → `internal/canopy/wifi`
2. Move `internal/survey` → `internal/canopy/survey`

**Checkpoint:** Canopy module complete, WiFi/survey tests pass

### Phase 6: Code Migration - Roots Module (Week 5-6)

1. Move `internal/paths` → `internal/roots/traceroute`
2. Move `internal/publicip` → `internal/roots/enrichment`
3. Extract topology from discovery → `internal/roots/topology`

**Checkpoint:** Roots module complete, path analysis tests pass

### Phase 7: API Handler Reorganization (Week 6-7)

1. Create `handlers_roots.go` - consolidate path handlers
2. Create `handlers_canopy.go` - consolidate WiFi/survey handlers
3. Create `handlers_shell.go` - consolidate discovery/vuln handlers
4. Create `handlers_sap.go` - consolidate telemetry handlers
5. Create `handlers_harvest.go` - new report handlers
6. Add module-based route groups
7. Add legacy route redirects

**Checkpoint:** New routes work, legacy routes redirect, all API tests pass

### Phase 8: Harvest Module (Week 7-8)

1. Implement `internal/harvest/generator` - PDF/HTML/CSV export
2. Implement `internal/harvest/templates` - report templates
3. Add `handlers_harvest.go` endpoints

**Checkpoint:** Report generation works

### Phase 9: Cleanup (Week 8)

1. Remove old package directories
2. Remove legacy route redirects (or keep for v1.x compatibility)
3. Update all documentation
4. Final test sweep

**Checkpoint:** Clean codebase, all tests pass, no legacy paths

---

## 4. Risks & Mitigations

### HIGH Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking API contracts | Frontend stops working | Keep legacy routes, add redirects with deprecation headers |
| Import cycle creation | Build fails | Use interfaces at boundaries, dependency injection |
| Database schema changes | Data loss | No schema changes in this refactor - data layer unchanged |

### MEDIUM Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Test coverage gaps | Regressions undetected | Run full test suite at each phase checkpoint |
| WebSocket message format changes | Real-time updates break | Keep WebSocket layer unchanged initially |
| Configuration breaking | App won't start | Configuration paths unchanged |

### LOW Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Logging context lost | Debug harder | Ensure module field added to all logs |
| Documentation drift | Confusion | Update docs incrementally per phase |

---

## 5. Verification Checklist

### Per-Phase Verification

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `golangci-lint run` passes
- [ ] Frontend builds (`npm run build`)
- [ ] Frontend tests pass (`npm test`)
- [ ] Manual smoke test on localhost
- [ ] API endpoint smoke tests

### Final Verification

- [ ] All legacy routes redirect correctly
- [ ] WebSocket connections work
- [ ] Authentication flow unchanged
- [ ] Profile switching works
- [ ] Device discovery works end-to-end
- [ ] WiFi scanning works
- [ ] All health checks pass
- [ ] Report generation works

---

## 6. Rollback Plan

Each phase creates a git tag. Rollback procedure:

```bash
# If Phase N breaks, rollback to Phase N-1
git checkout phase-N-1
go build ./...
# Deploy
```

Tags to create:
- `refactor-phase-1-dirs`
- `refactor-phase-2-scaffold`
- `refactor-phase-3-sap`
- `refactor-phase-4-shell`
- `refactor-phase-5-canopy`
- `refactor-phase-6-roots`
- `refactor-phase-7-api`
- `refactor-phase-8-harvest`
- `refactor-phase-9-cleanup`

---

## 7. Next Steps

1. **Review this plan** - Get stakeholder sign-off
2. **Create tracking issues** - One GitHub issue per phase
3. **Start Phase 1** - Directory structure (low risk, no code changes)
4. **Iterate** - Complete each phase before starting next

---

## Appendix: File Count by Package

Current package sizes (for effort estimation):

| Package | Go Files | Lines of Code |
|---------|----------|---------------|
| `internal/api` | ~40 | ~8000 |
| `internal/discovery` | ~20 | ~4000 |
| `internal/wifi` | ~5 | ~1000 |
| `internal/dhcp` | ~5 | ~800 |
| `internal/dns` | ~3 | ~500 |
| `internal/gateway` | ~3 | ~400 |
| `internal/snmp` | ~5 | ~1200 |
| `internal/paths` | ~5 | ~600 |
| `internal/survey` | ~5 | ~800 |
| `internal/speedtest` | ~3 | ~400 |
| `internal/iperf` | ~5 | ~600 |
| `internal/cable` | ~2 | ~200 |
| `internal/vlan` | ~2 | ~300 |
| `internal/publicip` | ~2 | ~300 |
| `internal/phy` | ~2 | ~200 |
| `internal/network` | ~5 | ~800 |

**Total estimated effort:** 8-10 weeks at current pace
