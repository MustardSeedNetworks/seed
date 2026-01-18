#!/bin/bash
#
# Build macOS .pkg installer for The Seed
#
# Usage:
#   ./build-pkg.sh [BINARY_PATH] [VERSION]
#
# Examples:
#   ./build-pkg.sh                                    # Use ./seed and auto-detect version
#   ./build-pkg.sh ./seed-darwin-arm64                # Use specified binary
#   ./build-pkg.sh ./seed-darwin-arm64 0.165.34       # Use specified binary and version
#
# Requirements:
#   - Xcode command line tools (pkgbuild, productbuild)
#   - The Seed binary built for macOS
#

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PKG_ID="com.seed"
PKG_NAME="seed"
INSTALL_LOCATION="/usr/local/seed"
BUILD_DIR="$REPO_ROOT/dist/macos-pkg"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

# Find binary
BINARY_PATH=""
if [[ -n "$1" && -f "$1" ]]; then
    BINARY_PATH="$1"
elif [[ -f "$REPO_ROOT/seed-darwin-$(uname -m)" ]]; then
    BINARY_PATH="$REPO_ROOT/seed-darwin-$(uname -m)"
elif [[ -f "$REPO_ROOT/seed" ]]; then
    BINARY_PATH="$REPO_ROOT/seed"
else
    log_error "Cannot find seed binary. Please build first with: make build-darwin"
    exit 1
fi

# Get version
VERSION="${2:-}"
if [[ -z "$VERSION" ]]; then
    VERSION=$("$BINARY_PATH" version 2>/dev/null | head -1 || echo "0.0.0")
    # Strip 'v' prefix if present
    VERSION="${VERSION#v}"
fi

# Architecture
ARCH=$(uname -m)
[[ "$ARCH" == "x86_64" ]] && ARCH="amd64"

log_info "Building macOS .pkg installer"
echo "┌────────────────────────────────────────────────────────┐"
printf "│  Binary:      %-42s │\n" "$BINARY_PATH"
printf "│  Version:     %-42s │\n" "$VERSION"
printf "│  Architecture:%-42s │\n" "$ARCH"
echo "└────────────────────────────────────────────────────────┘"
echo

# Clean and create build directory
log_step "1/6 Preparing build directory..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION/launchd"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION/configs"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION/logs"
mkdir -p "$BUILD_DIR/payload$INSTALL_LOCATION/data"
mkdir -p "$BUILD_DIR/scripts"
mkdir -p "$BUILD_DIR/resources"

# Copy binary
log_step "2/6 Copying binary..."
cp "$BINARY_PATH" "$BUILD_DIR/payload$INSTALL_LOCATION/$PKG_NAME"
chmod 755 "$BUILD_DIR/payload$INSTALL_LOCATION/$PKG_NAME"

# Copy launchd plist
log_step "3/6 Copying launchd configuration..."
cp "$REPO_ROOT/deploy/launchd/com.seed.plist" "$BUILD_DIR/payload$INSTALL_LOCATION/launchd/"

# Copy scripts
log_step "4/6 Preparing installation scripts..."
cp "$SCRIPT_DIR/scripts/preinstall" "$BUILD_DIR/scripts/"
cp "$SCRIPT_DIR/scripts/postinstall" "$BUILD_DIR/scripts/"
chmod 755 "$BUILD_DIR/scripts/preinstall"
chmod 755 "$BUILD_DIR/scripts/postinstall"

# Copy resources (welcome, conclusion)
cp "$SCRIPT_DIR/resources/welcome.html" "$BUILD_DIR/resources/"
cp "$SCRIPT_DIR/resources/conclusion.html" "$BUILD_DIR/resources/"

# Build component package
log_step "5/6 Building component package..."
pkgbuild \
    --root "$BUILD_DIR/payload" \
    --identifier "$PKG_ID.pkg" \
    --version "$VERSION" \
    --scripts "$BUILD_DIR/scripts" \
    --install-location "/" \
    "$BUILD_DIR/seed-component.pkg"

# Create distribution.xml with version substituted
sed "s/__VERSION__/$VERSION/g" "$SCRIPT_DIR/distribution.xml" > "$BUILD_DIR/distribution.xml"

# Build final product package
log_step "6/6 Building final package..."
PKG_OUTPUT="$REPO_ROOT/dist/seed-${VERSION}-${ARCH}.pkg"
mkdir -p "$(dirname "$PKG_OUTPUT")"

productbuild \
    --distribution "$BUILD_DIR/distribution.xml" \
    --resources "$BUILD_DIR/resources" \
    --package-path "$BUILD_DIR" \
    "$PKG_OUTPUT"

# Clean up intermediate files
rm -rf "$BUILD_DIR"

echo
log_info "Package built successfully!"
echo "┌────────────────────────────────────────────────────────┐"
echo "│  Package ready:                                        │"
printf "│  %-54s │\n" "$PKG_OUTPUT"
echo "├────────────────────────────────────────────────────────┤"
echo "│  To install:                                           │"
printf "│  %-54s │\n" "sudo installer -pkg $PKG_OUTPUT -target /"
echo "│                                                        │"
echo "│  Or double-click the .pkg file in Finder               │"
echo "└────────────────────────────────────────────────────────┘"
echo
