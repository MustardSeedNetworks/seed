#!/bin/bash
#
# Notarize and staple the Seed Wi-Fi Helper bundle.
#
# Usage:
#   ./notarize-helper.sh setup-key     # store an App Store Connect API key (recommended)
#   ./notarize-helper.sh setup-appleid # store an Apple ID + app-specific password
#   ./notarize-helper.sh               # notarize + staple using the stored profile
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PROFILE="${SEED_NOTARY_PROFILE:-seed-notary}"
TEAM_ID="${SEED_TEAM_ID:-X6JWYP43HG}"
BUNDLE="${SEED_HELPER_BUNDLE:-$REPO_ROOT/dist/macos-helper/Seed Wi-Fi Helper.app}"
ZIP="${BUNDLE%.app}.zip"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
die() { echo -e "${RED}$*${NC}" >&2; exit 1; }
ok()  { echo -e "${GREEN}$*${NC}"; }
note(){ echo -e "${YELLOW}$*${NC}"; }

setup_key() {
    echo "App Store Connect API key setup."
    echo "Create at: App Store Connect > Users and Access > Integrations > App Store Connect API"
    echo "Role: Developer or higher. The .p8 downloads once."
    echo
    read -r -p "Path to AuthKey_XXXXXXXX.p8: " KEY_PATH
    [[ -f "$KEY_PATH" ]] || die "No such file: $KEY_PATH"
    read -r -p "Key ID (10 chars, from the filename): " KEY_ID
    read -r -p "Issuer ID (UUID, shown above the key list): " ISSUER

    xcrun notarytool store-credentials "$PROFILE" \
        --key "$KEY_PATH" --key-id "$KEY_ID" --issuer "$ISSUER"
    ok "Stored profile: $PROFILE"
}

setup_appleid() {
    echo "Apple ID setup."
    echo "The Apple ID must own team $TEAM_ID (check developer.apple.com/account > Membership)."
    echo "You will be prompted for an app-specific credential from appleid.apple.com"
    echo "> Sign-In and Security > App-Specific Passwords. Format: xxxx-xxxx-xxxx-xxxx"
    echo
    read -r -p "Apple ID email: " APPLE_ID
    xcrun notarytool store-credentials "$PROFILE" \
        --apple-id "$APPLE_ID" --team-id "$TEAM_ID"
    ok "Stored profile: $PROFILE"
}

notarize() {
    [[ -d "$BUNDLE" ]] || die "Bundle not found: $BUNDLE
Build it first: $SCRIPT_DIR/build-helper.sh"

    codesign --verify --strict "$BUNDLE" 2>/dev/null \
        || die "Bundle is not validly signed. Rebuild: $SCRIPT_DIR/build-helper.sh"

    echo "Submitting: $BUNDLE"
    rm -f "$ZIP"
    # notarytool takes an archive, never a bundle directory.
    ditto -c -k --keepParent "$BUNDLE" "$ZIP"

    if ! xcrun notarytool submit "$ZIP" --keychain-profile "$PROFILE" --wait; then
        echo
        note "If this failed on credentials, the profile is wrong or missing:"
        note "  $0 setup-key        (recommended, also works in CI)"
        note "  $0 setup-appleid"
        note "If it failed on review, get the detail with:"
        note "  xcrun notarytool log <submission-id> --keychain-profile $PROFILE"
        exit 1
    fi

    xcrun stapler staple "$BUNDLE"
    rm -f "$ZIP"

    echo
    ok "Notarized and stapled."
    xcrun stapler validate "$BUNDLE"
    spctl -a -vv -t exec "$BUNDLE" 2>&1 | head -3 || true

    echo
    note "Now test whether the Location prompt fires on its own:"
    note "  1. tccutil reset Location net.mustardseed.seed.wifihelper"
    note "  2. open \"$BUNDLE\""
    note "  3. Watch for a Location Services prompt."
    note "     Prompt appears  -> notarization fixes onboarding."
    note "     No prompt       -> manual System Settings step is permanent."
    note "  Check the result:"
    note "  plutil -p /var/db/locationd/clients.plist | grep -A4 seed.wifihelper"
}

case "${1:-notarize}" in
    setup-key)     setup_key ;;
    setup-appleid) setup_appleid ;;
    notarize)      notarize ;;
    *)             die "Usage: $0 [setup-key|setup-appleid|notarize]" ;;
esac
