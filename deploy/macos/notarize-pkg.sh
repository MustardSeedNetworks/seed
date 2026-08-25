#!/bin/bash
#
# Sign, notarize and staple the macOS installer package.
#
# Runs the sequence that makes a .pkg installable on a machine other than the
# one that built it: productsign with a Developer ID *Installer* certificate —
# a different certificate from the Application one used on the binaries —
# then notarization, then stapling so the result validates offline.
#
# Usage:
#   ./notarize-pkg.sh PKG_PATH [OUTPUT_PATH]
#
# Requirements:
#   - "Developer ID Installer" identity in the keychain
#   - a notarytool keychain profile (see notarize-helper.sh setup-key)
#
set -euo pipefail

PKG="${1:-}"
[[ -n "$PKG" && -f "$PKG" ]] || { echo "usage: $0 PKG_PATH [OUTPUT_PATH]" >&2; exit 1; }

OUTPUT="${2:-${PKG%.pkg}-signed.pkg}"
PROFILE="${SEED_NOTARY_PROFILE:-seed-notary}"
INSTALLER_IDENTITY="${SEED_INSTALLER_IDENTITY:-Developer ID Installer}"

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
die() { echo -e "${RED}$*${NC}" >&2; exit 1; }

security find-identity -v | grep -q "$INSTALLER_IDENTITY" \
    || die "No '$INSTALLER_IDENTITY' identity in the keychain.
This is a different certificate from 'Developer ID Application'. Create it at
developer.apple.com > Certificates > Developer ID Installer."

echo "Signing installer..."
productsign --sign "$INSTALLER_IDENTITY" "$PKG" "$OUTPUT"

echo "Notarizing..."
if ! xcrun notarytool submit "$OUTPUT" --keychain-profile "$PROFILE" --wait; then
    die "Notarization failed. For the specific reason:
  xcrun notarytool log <submission-id> --keychain-profile $PROFILE
The usual cause is an unsigned executable inside the package. Every binary needs
a Developer ID signature, the hardened runtime, and a secure timestamp —
build-pkg.sh and build-helper.sh do this, so a failure here normally means a
binary was added to the payload without being signed."
fi

xcrun stapler staple "$OUTPUT"

echo
echo -e "${GREEN}Signed, notarized and stapled:${NC} $OUTPUT"
spctl -a -vv -t install "$OUTPUT" 2>&1 | head -3
pkgutil --check-signature "$OUTPUT" | head -4
