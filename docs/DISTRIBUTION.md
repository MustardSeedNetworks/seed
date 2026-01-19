# Distribution & Licensing Roadmap

**Product:** Seed
**Model:** Commercial (paid license)
**Status:** Development - NOT FOR PUBLIC DISTRIBUTION

---

## Current State (Development)

All distribution channels are **locked down**:

| Channel | Status | Notes |
|---------|--------|-------|
| Container registry | DISABLED | No `container-push` target |
| Public downloads | DISABLED | No public artifacts |
| Package repos | DISABLED | .deb/.rpm stay local |

### Local Development Only

```bash
make container   # Builds locally only
make deb         # Creates dist/seed_*.deb (local)
make rpm         # Creates dist/seed_*.rpm (local)
```

---

## Distribution Strategy

### License Validation Required

Before any public/commercial distribution:

1. **License server integration** or offline license validation
2. **Hardware fingerprinting** (optional, for appliance model)
3. **Expiration/renewal** handling
4. **Feature gating** by license tier

### Proposed License Tiers

| Tier | Features | Target |
|------|----------|--------|
| Trial | Full features, 30-day limit | Evaluation |
| Standard | Core diagnostics | SMB |
| Professional | Full features + API | Enterprise |
| OEM | White-label, volume | Partners |

---

## Deployment Channels (Future)

When ready for commercial release:

### 1. Private Container Registry
```bash
# Future - requires auth
CONTAINER_REGISTRY=registry.mustardseednetworks.com
make container-push
```

### 2. Customer Portal
- Authenticated download of .deb/.rpm/.pkg
- License key provisioning
- Update notifications

### 3. Appliance Image
- Pre-installed on Dell Mini PC
- Hardware-locked license
- Factory reset capability

---

## Pre-Release Checklist

- [ ] License validation implemented (`internal/license/`)
- [ ] License server deployed (or offline validation)
- [ ] Private registry configured
- [ ] Customer portal ready
- [ ] EULA/Terms of Service finalized
- [ ] Pricing determined
- [ ] Support infrastructure ready

---

## Version Strategy

**Single source of truth:** Git tags

```bash
git tag v1.0.0          # Creates version
make build              # Embeds version via ldflags
./bin/seed --version    # Shows v1.0.0
```

- `package.json` version is `0.0.0` (ignored, real version from API)
- Container tags match git tags
- All artifacts include version in filename

---

## Security Considerations

1. **No secrets in containers** - Config injected at runtime
2. **License validation on startup** - App won't run without valid license
3. **Tamper detection** - Binary signing (future)
4. **Update mechanism** - Secure, authenticated updates

---

*Last updated: 2025-01-19*
*Status: Development lockdown*
