#!/usr/bin/env bash
# check-snmp-credential-surface.sh — the #1799 ratchet.
#
# SNMP credentials belong to the encrypted device-credential vault. Before this
# gate, they also lived in seed.json as plaintext community strings, were handed
# back by the settings API to any authenticated reader, and shipped with a
# `public` default that scanned every network the daemon could reach with the
# community every attacker tries first.
#
# Two checks, both over production code only:
#
#   1. No `public` SNMP default anywhere a shipped artefact reads it — the
#      default config, the schema, the database seed, the Go defaults.
#   2. No credential field on the config type, its schema, or the settings API.
#      A struct field is how the plaintext path came back last time; the wire
#      DTO is how it reached the client.
#
# Test fixtures are excluded on purpose: internal/config/testdata carries a real
# v0.200.0 config, `public` included, and it is what proves the upgrade path
# strips these keys rather than crash-looping the daemon (see
# TestRemovedKeysAreStripped).
#
# Run locally: scripts/check-snmp-credential-surface.sh

set -uo pipefail

FAIL=0

report() {
  echo "============================================================"
  echo "[#1799] $1"
  echo "$2"
  echo "$3"
  echo ""
  FAIL=1
}

# 1. Production `public` SNMP defaults.
DEFAULTS=$(grep -rEn '"public"|\[.public.\]' \
  configs/ internal/config/schema.json internal/config/config_defaults.go \
  internal/database/seed.go 2>/dev/null \
  | grep -v '/testdata/' || true)
if [ -n "$DEFAULTS" ]; then
  report "a 'public' SNMP default is back in a shipped artefact:" "$DEFAULTS" \
    "Credentials come from the vault; there is no default community."
fi

# 2. Credential fields on the config type, its schema, or the settings API.
SURFACE=$(grep -rEn 'Communities|V3Credentials|SNMPv3Credential|v3_credentials|"communities"' \
  internal/config/config_types_security.go internal/config/schema.json \
  internal/config/defaults.go internal/security/settings/settings.go \
  internal/api/handlers_security.go 2>/dev/null || true)
if [ -n "$SURFACE" ]; then
  report "an SNMP credential field is back on the config or settings surface:" "$SURFACE" \
    "Read and write them through the device-credential vault instead."
fi

if [ "$FAIL" -eq 0 ]; then
  echo "✔ SNMP credentials live only in the vault"
fi
exit "$FAIL"
