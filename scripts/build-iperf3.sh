#!/bin/bash
#------------------------------------------------------------------------------
# build-iperf3.sh
#
# This script automates the process of downloading, building, and bundling the
# latest release of iperf3 from source for use with NetScope. It performs the
# following steps:
#
# 1. Detects the operating system and architecture to ensure compatibility.
# 2. Creates necessary build and output directories.
# 3. Fetches the latest iperf3 release version from GitHub.
# 4. Checks if the latest version is already built and skips rebuild if so.
# 5. Downloads and extracts the iperf3 source tarball.
# 6. Ensures required build dependencies (autoconf, automake, libtool) are installed.
# 7. Configures the build for static linking where possible for portability.
# 8. Builds iperf3 from source.
# 9. Copies the built binary to the output directory, naming it according to OS/arch.
# 10. Verifies the built binary and displays its version.
#
# Usage:
#   ./build-iperf3.sh
#
# Requirements:
#   - curl
#   - autoconf, automake, libtool (will attempt to install if missing)
#   - make, gcc (standard build tools)
#
# Output:
#   The built iperf3 binary will be placed in the bin/ directory of the project,
#   named according to the detected OS and architecture.
#
#------------------------------------------------------------------------------

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build/iperf3"
OUTPUT_DIR="$PROJECT_ROOT/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
esac

echo "Building iperf3 for $OS-$ARCH"

# Create directories
mkdir -p "$BUILD_DIR"
mkdir -p "$OUTPUT_DIR"

# Get latest release version from GitHub API
echo "Fetching latest iperf3 release..."
LATEST_VERSION=$(curl -s https://api.github.com/repos/esnet/iperf/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "Failed to fetch latest version, using fallback v3.20"
    LATEST_VERSION="v3.20"
fi

echo "Latest iperf3 version: $LATEST_VERSION"

# Determine binary name based on OS/arch
# For consistency, always use platform suffix except for local dev
BINARY_NAME="iperf3"
BINARY_PATH="$OUTPUT_DIR/$BINARY_NAME-$OS-$ARCH"

# Also create a symlink without suffix for local dev convenience
LOCAL_BINARY="$OUTPUT_DIR/$BINARY_NAME"

if [ -f "$BINARY_PATH" ]; then
    EXISTING_VERSION=$("$BINARY_PATH" --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' || echo "")
    EXISTING_VERSION_STRIPPED="${EXISTING_VERSION#v}"
    LATEST_VERSION_STRIPPED="${LATEST_VERSION#v}"
    if [ "$EXISTING_VERSION_STRIPPED" = "$LATEST_VERSION_STRIPPED" ]; then
        echo "iperf3 $EXISTING_VERSION already built at $BINARY_PATH"
        exit 0
    fi
fi

# Download source
TARBALL_URL="https://github.com/esnet/iperf/archive/refs/tags/$LATEST_VERSION.tar.gz"
TARBALL_FILE="$BUILD_DIR/iperf3-$LATEST_VERSION.tar.gz"

if [ ! -f "$TARBALL_FILE" ]; then
    echo "Downloading iperf3 source..."
    curl -L -o "$TARBALL_FILE" "$TARBALL_URL"
fi

# Extract
echo "Extracting source..."
cd "$BUILD_DIR"
# Remove all old iperf-* directories to avoid confusion
find . -maxdepth 1 -type d -name "iperf-*" -exec rm -rf {} +
tar -xzf "$TARBALL_FILE"

# Find the extracted directory (should match the version)
SOURCE_DIR=$(find . -maxdepth 1 -type d -name "iperf-${LATEST_VERSION#v}" | head -1)
if [ -z "$SOURCE_DIR" ]; then
    # Fallback: try matching with the full tag name
    SOURCE_DIR=$(find . -maxdepth 1 -type d -name "iperf-$LATEST_VERSION" | head -1)
fi
if [ -z "$SOURCE_DIR" ]; then
    echo "Error: Could not find extracted source directory"
    exit 1
fi

cd "$SOURCE_DIR"

# Build dependencies check
echo "Checking build dependencies..."
if ! command -v autoconf &> /dev/null; then
    echo "Warning: autoconf not found, attempting to install..."
    if [ "$OS" = "darwin" ]; then
        brew install autoconf automake libtool
    elif command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y autoconf automake libtool
    elif command -v yum &> /dev/null; then
        sudo yum install -y autoconf automake libtool
    fi
fi

# Configure and build
echo "Configuring iperf3..."

# Run bootstrap if configure doesn't exist
if [ ! -f "configure" ]; then
    if [ -f "bootstrap.sh" ]; then
        ./bootstrap.sh
    else
        autoreconf -i
    fi
fi

# Configure with static linking where possible for portability
./configure --prefix="$BUILD_DIR/install" --disable-shared --enable-static-bin 2>/dev/null || \
./configure --prefix="$BUILD_DIR/install"

echo "Building iperf3..."
make clean 2>/dev/null || true
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 2)
make install

# Copy binary to output directory
if [ ! -f "$BUILD_DIR/install/bin/iperf3" ]; then
    echo "Error: Built iperf3 binary not found at $BUILD_DIR/install/bin/iperf3"
    exit 1
fi
echo "Copying binary to $BINARY_PATH..."
cp "$BUILD_DIR/install/bin/iperf3" "$BINARY_PATH"
chmod +x "$BINARY_PATH"

# Create local symlink for dev convenience
if [ "$BINARY_PATH" != "$LOCAL_BINARY" ]; then
    ln -sf "$(basename "$BINARY_PATH")" "$LOCAL_BINARY"
    echo "Created symlink: $LOCAL_BINARY -> $(basename "$BINARY_PATH")"
fi

# Verify the binary works
echo "Verifying build..."
"$BINARY_PATH" --version

echo ""
echo "Successfully built iperf3 at: $BINARY_PATH"
echo "Version: $("$BINARY_PATH" --version | head -1)"
echo ""
echo "Available binaries:"
ls -la "$OUTPUT_DIR"/iperf3* 2>/dev/null || true
