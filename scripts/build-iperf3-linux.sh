#!/bin/bash
#------------------------------------------------------------------------------
# build-iperf3-linux.sh
#
# Cross-compiles iperf3 for Linux (AMD64 and ARM64) using Docker.
# This allows building Linux binaries from macOS for deployment.
#
# Usage:
#   ./build-iperf3-linux.sh          # Build both AMD64 and ARM64
#   ./build-iperf3-linux.sh amd64    # Build only AMD64
#   ./build-iperf3-linux.sh arm64    # Build only ARM64
#
# Requirements:
#   - Docker (with multi-platform support for ARM64)
#
# Output:
#   bin/iperf3-linux-amd64
#   bin/iperf3-linux-arm64
#
#------------------------------------------------------------------------------

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$PROJECT_ROOT/bin"

IPERF_VERSION="${IPERF_VERSION:-3.17.1}"
TARGET_ARCH="${1:-all}"  # amd64, arm64, or all

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Check for Docker
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is required for cross-compilation"
    echo "Install Docker Desktop from https://docker.com"
    exit 1
fi

build_for_arch() {
    local arch=$1
    local docker_platform=""
    local binary_name="iperf3-linux-${arch}"

    case "$arch" in
        amd64)
            docker_platform="linux/amd64"
            ;;
        arm64)
            docker_platform="linux/arm64"
            ;;
        *)
            echo "Unknown architecture: $arch"
            return 1
            ;;
    esac

    # Check if binary already exists
    if [ -f "$OUTPUT_DIR/$binary_name" ]; then
        echo "iperf3 Linux $arch binary already exists at $OUTPUT_DIR/$binary_name"
        "$OUTPUT_DIR/$binary_name" --version 2>/dev/null || true
        if [ "$FORCE" != "1" ]; then
            echo "Use FORCE=1 to rebuild"
            return 0
        fi
    fi

    echo ""
    echo "=== Building iperf3 $IPERF_VERSION for linux-${arch} ==="
    echo ""

    # Build using Docker with platform emulation
    docker run --rm --platform "$docker_platform" \
        -v "$PROJECT_ROOT:/build" \
        ubuntu:22.04 bash -c "
        set -e

        echo '==> Installing build dependencies...'
        apt-get update -qq
        apt-get install -y -qq curl build-essential autoconf automake libtool > /dev/null

        cd /tmp

        echo '==> Downloading iperf3 source...'
        curl -sL -o iperf3.tar.gz https://github.com/esnet/iperf/archive/refs/tags/$IPERF_VERSION.tar.gz
        tar xzf iperf3.tar.gz
        cd iperf-${IPERF_VERSION#v}

        echo '==> Configuring...'
        ./configure --prefix=/tmp/iperf3-install --disable-shared --enable-static-bin > /dev/null 2>&1 || \
        ./configure --prefix=/tmp/iperf3-install > /dev/null

        echo '==> Building...'
        make -j\$(nproc) > /dev/null
        make install > /dev/null

        echo '==> Copying binary...'
        cp /tmp/iperf3-install/bin/iperf3 /build/bin/$binary_name
        chmod +x /build/bin/$binary_name

        echo '==> Verifying...'
        file /build/bin/$binary_name
        /build/bin/$binary_name --version || echo '(Cannot run - different architecture)'
    "

    echo ""
    echo "Successfully built: $OUTPUT_DIR/$binary_name"
}

# Build requested architectures
case "$TARGET_ARCH" in
    amd64)
        build_for_arch amd64
        ;;
    arm64)
        build_for_arch arm64
        ;;
    all)
        build_for_arch amd64
        build_for_arch arm64
        ;;
    *)
        echo "Usage: $0 [amd64|arm64|all]"
        exit 1
        ;;
esac

echo ""
echo "=== Available iperf3 binaries ==="
ls -la "$OUTPUT_DIR"/iperf3* 2>/dev/null || echo "No binaries found"
