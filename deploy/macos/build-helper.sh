#!/bin/bash
#
# Build and sign the Seed Wi-Fi Helper application bundle.
#
# macOS redacts Wi-Fi network names and BSSIDs from any process that does not
# hold Location Services authorization. That grant is given per user, inside a
# login session, to a signed application bundle — so the helper must be a signed
# .app, not a bare executable. An unsigned build will be refused by the daemon,
# which verifies the connecting process against a code-signing requirement.
#
# Usage:
#   ./build-helper.sh [VERSION] [OUTPUT_DIR]
#
# Requirements:
#   - Go toolchain
#   - codesign, and a "Developer ID Application" identity in the keychain
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

VERSION="${1:-0.0.0}"
VERSION="${VERSION#v}"
OUTPUT_DIR="${2:-$REPO_ROOT/dist/macos-helper}"

BUNDLE_ID="net.mustardseed.seed.wifihelper"
BUNDLE_NAME="Seed Wi-Fi Helper.app"
BUNDLE="$OUTPUT_DIR/$BUNDLE_NAME"
BINARY_NAME="seed-wifi-helper"

# The identity must match the certificate the daemon's requirement names. An
# ad-hoc or Apple Development signature will build, but the daemon will refuse
# the resulting helper at runtime.
SIGN_IDENTITY="${SEED_SIGN_IDENTITY:-Developer ID Application}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Building $BUNDLE_NAME $VERSION"

rm -rf "$BUNDLE"
mkdir -p "$BUNDLE/Contents/MacOS"

# Apple Silicon only, matching the fleet's release targets.
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build \
    -trimpath \
    -o "$BUNDLE/Contents/MacOS/$BINARY_NAME" \
    "$REPO_ROOT/cmd/seed-wifi-helper"

sed -e "s|<string>0\.0\.0</string>|<string>$VERSION</string>|g" \
    "$SCRIPT_DIR/helper/Info.plist" > "$BUNDLE/Contents/Info.plist"

plutil -lint "$BUNDLE/Contents/Info.plist" > /dev/null

if ! security find-identity -v -p codesigning | grep -q "$SIGN_IDENTITY"; then
    echo -e "${RED}No '$SIGN_IDENTITY' identity found in the keychain.${NC}"
    echo "The helper must be signed with the identity the daemon's code requirement names;"
    echo "an unsigned helper is refused at runtime. Set SEED_SIGN_IDENTITY to override."
    exit 1
fi

# The hardened runtime is required for notarization, which is what lets macOS
# offer the Location permission for this bundle in the first place.
codesign --force --timestamp --options runtime \
    --identifier "$BUNDLE_ID" \
    --sign "$SIGN_IDENTITY" \
    "$BUNDLE"

codesign --verify --strict --verbose=2 "$BUNDLE"

echo -e "${GREEN}Built and signed:${NC} $BUNDLE"
echo
echo -e "${YELLOW}Not done here — notarization.${NC}"
echo "Notarizing requires Apple credentials this script deliberately does not handle:"
echo
echo "  xcrun notarytool submit \"$BUNDLE\" --keychain-profile <profile> --wait"
echo "  xcrun stapler staple \"$BUNDLE\""
echo
echo "Until the bundle is notarized, macOS may not present the Location Services"
echo "prompt at all, and an operator has to enable the permission by hand in"
echo "System Settings > Privacy & Security > Location Services."
