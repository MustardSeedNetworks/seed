#!/bin/bash
# Build iperf3 from source for bundling with NetScope
# This script downloads the latest iperf3 release from GitHub and compiles it

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
    echo "Failed to fetch latest version, using fallback 3.17.1"
    LATEST_VERSION="3.17.1"
fi

echo "Latest iperf3 version: $LATEST_VERSION"

# Check if we already have this version built
BINARY_NAME="iperf3"
if [ "$OS" = "darwin" ]; then
    BINARY_PATH="$OUTPUT_DIR/$BINARY_NAME"
else
    BINARY_PATH="$OUTPUT_DIR/$BINARY_NAME-$OS-$ARCH"
fi

if [ -f "$BINARY_PATH" ]; then
    EXISTING_VERSION=$("$BINARY_PATH" --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' || echo "")
    if [ "$EXISTING_VERSION" = "${LATEST_VERSION#v}" ] || [ "$EXISTING_VERSION" = "$LATEST_VERSION" ]; then
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
rm -rf "iperf-$LATEST_VERSION" "iperf-${LATEST_VERSION#v}"
tar -xzf "$TARBALL_FILE"

# Find the extracted directory (could be iperf-3.17.1 or iperf-v3.17.1)
SOURCE_DIR=$(ls -d iperf-* 2>/dev/null | head -1)
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
make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

echo "Installing to $BUILD_DIR/install..."
make install

# Copy binary to output directory
echo "Copying binary to $BINARY_PATH..."
cp "$BUILD_DIR/install/bin/iperf3" "$BINARY_PATH"
chmod +x "$BINARY_PATH"

# Verify the binary works
echo "Verifying build..."
"$BINARY_PATH" --version

echo ""
echo "Successfully built iperf3 at: $BINARY_PATH"
echo "Version: $("$BINARY_PATH" --version | head -1)"
