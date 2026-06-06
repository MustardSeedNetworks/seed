# Seed configuration

The Seed loads a single JSON config file. `seed.json` in this directory is the
**default config** — identical to what `seed install` writes on first run
(`config.DefaultConfig()`), so it always parses and is schema-valid.

## Format

- **JSON**, snake_case keys (see ADR-0010). The loader is `encoding/json`
  (`internal/config`). JSON input is required; the older `.yaml` names are read
  as a legacy fallback only.
- **Keys are documented by the JSON Schema**, not inline comments. See
  [`../docs/schemas/config.schema.json`](../docs/schemas/config.schema.json) for
  every field, its type, value constraints, and description. Point your editor at
  it (`json.schemas` in VS Code — see [`../docs/schemas/README.md`](../docs/schemas/README.md))
  for inline validation and hover docs while editing.

## Where it loads from

Resolution order (`internal/paths.ResolveConfigPath`):

1. `--config <path>` flag
2. `SEED_CONFIG_PATH` env var
3. Canonical path for the install mode:
   - system / systemd: `/etc/seed/seed.json`
   - user (XDG): `$XDG_CONFIG_HOME/seed/seed.json` (default `~/.config/seed/seed.json`)

   If `seed.json` is absent but a legacy-named file exists in the same dir
   (`config.yaml`, `seed.yaml`, `config.json`, `.seed.yaml`), it is loaded so
   pre-rename installs keep working — the content has always been JSON.

## Precedence

`flags > SEED_* env vars > config file > built-in defaults`. **Secrets**
(`jwt_secret`, credential keys) are auto-generated on first run or supplied via
`SEED_*` env vars — do not commit real secrets into a config file.

## Editing

The file is machine-managed: the first-run setup wizard and the running API
write to it (with backup-on-save). Hand-edits to the live file may be overwritten
by an API config change. Treat `seed.json` here as the canonical template.
