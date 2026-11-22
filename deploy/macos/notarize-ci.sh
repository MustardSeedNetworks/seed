#!/bin/bash
#
# Submit a file to Apple's notary service from CI and fail unless it is
# Accepted. Takes an archive or a .pkg (never a bare .app), and the App Store
# Connect API key from the environment so nothing touches a keychain profile.
#
# Usage:
#   NOTARY_KEY_BASE64=... NOTARY_KEY_ID=... NOTARY_ISSUER_ID=... ./notarize-ci.sh FILE
#
set -euo pipefail

FILE="${1:?usage: notarize-ci.sh FILE}"

if [ -z "${NOTARY_KEY_BASE64:-}" ] || [ -z "${NOTARY_KEY_ID:-}" ] || [ -z "${NOTARY_ISSUER_ID:-}" ]; then
    echo "APPLE_NOTARY_KEY_P8_BASE64 / APPLE_NOTARY_KEY_ID / APPLE_NOTARY_ISSUER_ID are not set;" \
        "an un-notarized artifact is refused by Gatekeeper on every other Mac." >&2
    exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
echo "$NOTARY_KEY_BASE64" | base64 --decode > "$WORK/key.p8"

# notarytool exits 0 for an Invalid submission, so read the verdict.
xcrun notarytool submit "$FILE" --key "$WORK/key.p8" --key-id "$NOTARY_KEY_ID" \
    --issuer "$NOTARY_ISSUER_ID" --wait --output-format json > "$WORK/result.json"
cat "$WORK/result.json"
test "$(jq -r .status "$WORK/result.json")" = "Accepted"
