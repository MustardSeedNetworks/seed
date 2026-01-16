<!--
LOG DECISIONS WHEN:
- Choosing between architectural approaches
- Selecting libraries or tools
- Making security-related choices
- Deviating from standard patterns

This is append-only. Never delete entries.
-->

# Decision Log

Track key architectural and implementation decisions.

## Format
```
## [YYYY-MM-DD] Decision Title

**Decision**: What was decided
**Context**: Why this decision was needed
**Options Considered**: What alternatives existed
**Choice**: Which option was chosen
**Reasoning**: Why this choice was made
**Trade-offs**: What we gave up
**References**: Related code/docs
```

---

## [2025-01-09] Skills System Initialization

**Decision**: Add Claude skills system to existing project
**Context**: Project had CLAUDE.md but lacked structured skills and session management
**Options Considered**:
1. Keep existing CLAUDE.md as-is
2. Replace with skills system
3. Augment existing CLAUDE.md with skills references
**Choice**: Option 3 - Augment with skills references
**Reasoning**: Preserve project-specific conventions while adding reusable skill patterns
**Trade-offs**: Slightly larger configuration surface
**References**: .claude/skills/, CLAUDE.md

---

## [2026-01-16] Configuration Architecture - JSON Only

**Decision**: Use JSON-only configuration with environment variable overrides
**Context**: Previously used YAML for config files, but needed a consistent approach that can be shared across seed, stem, and niac/go projects
**Options Considered**:
1. Keep YAML config files
2. Support both YAML and JSON
3. JSON-only with env var overrides (12-factor app style)
4. TOML config
**Choice**: Option 3 - JSON-only with SEED_* env var prefix
**Reasoning**:
- JSON is universally supported and easier to parse
- JSON Schema provides robust validation
- Environment variables follow 12-factor app principles
- Cleaner separation: system config (JSON file + env vars) vs profile config (JSON in database)
- One format reduces complexity and potential bugs
**Trade-offs**:
- Less human-friendly than YAML for hand-editing
- Requires migration from existing YAML configs
**References**: internal/config/, internal/config/schema.json

### Architecture

**System Config** (`config.json` or `seed.json`):
- Loaded once at startup from JSON file
- Environment variables override with SEED_* prefix (e.g., `SEED_SERVER_PORT=8443`)
- Validated against JSON Schema (Draft 2020-12)
- Contains server settings, auth, logging, thresholds

**Profile Config**:
- Stored as JSON in SQLite database
- Per-user or per-profile settings
- Can be updated at runtime

**Key Files**:
- `internal/config/config.go` - Config struct and loading
- `internal/config/schema.json` - JSON Schema validation
- `internal/config/defaults.go` - Default values
- `internal/paths/paths.go` - Config file resolution

**Migration Guide for stem/niac**:
1. Replace `yaml.Unmarshal` → `json.Unmarshal`
2. Rename config files: `.yaml` → `.json`
3. Update JSON tags in structs to use camelCase
4. Add JSON Schema validation
5. Implement env var override with project prefix (STEM_*, NIAC_*)
6. Remove `gopkg.in/yaml.v3` dependency
