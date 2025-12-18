#!/bin/bash
#
# The Seed Upgrade Script
# Upgrades an existing Seed installation to the latest version
#
# Usage:
#   ./upgrade.sh              # Upgrade using binary in current directory
#   ./upgrade.sh /path/to/seed # Upgrade using specified binary
#   ./upgrade.sh --build      # Pull, build, and upgrade (run from repo root)
#
set -e

# Configuration
INSTALL_DIR="/usr/local/seed"
CONFIG_FILE="/etc/seed/seed.yaml"
SERVICE_NAME="seed"
BINARY_NAME="seed"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS] [BINARY_PATH]"
    echo ""
    echo "Options:"
    echo "  --build     Pull latest code, build, and upgrade"
    echo "  --help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  sudo $0                    # Use ./seed binary"
    echo "  sudo $0 /path/to/seed      # Use specified binary"
    echo "  sudo $0 --build            # Build and upgrade (from repo root)"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check if service is installed
check_installed() {
    if [[ ! -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
        log_error "Seed service is not installed. Run install.sh first."
        exit 1
    fi
}

# Get current version
get_current_version() {
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        "$INSTALL_DIR/$BINARY_NAME" version 2>/dev/null || echo "unknown"
    else
        echo "not installed"
    fi
}

# Get new version
get_new_version() {
    local binary="$1"
    if [[ -f "$binary" ]]; then
        "$binary" version 2>/dev/null || echo "unknown"
    else
        echo "unknown"
    fi
}

# Build from source
build_from_source() {
    log_step "Building from source..."

    # Check if we're in a git repo with Makefile
    if [[ ! -f "Makefile" ]]; then
        log_error "Makefile not found. Run this from the repository root."
        exit 1
    fi

    # Pull latest changes
    log_info "Pulling latest changes..."
    git pull

    # Build
    log_info "Building..."
    make build

    # Return path to built binary
    echo "./seed"
}

# Main upgrade function
do_upgrade() {
    local binary_path="$1"

    log_info "Starting Seed upgrade..."
    echo ""

    # Get versions
    local current_version=$(get_current_version)
    local new_version=$(get_new_version "$binary_path")

    echo "┌────────────────────────────────────────────────────────┐"
    echo "│  Seed Upgrade                                          │"
    echo "├────────────────────────────────────────────────────────┤"
    printf "│  Current version: %-37s │\n" "$current_version"
    printf "│  New version:     %-37s │\n" "$new_version"
    echo "└────────────────────────────────────────────────────────┘"
    echo ""

    # Step 1: Stop service
    log_step "1/5 Stopping service..."
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        systemctl stop "$SERVICE_NAME"
        log_info "Service stopped"
    else
        log_warn "Service was not running"
    fi

    # Step 2: Backup current binary (optional)
    log_step "2/5 Backing up current binary..."
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        cp "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}.bak"
        log_info "Backup created: ${BINARY_NAME}.bak"
    fi

    # Step 3: Copy new binary
    log_step "3/5 Installing new binary..."
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    chown seed:seed "$INSTALL_DIR/$BINARY_NAME"
    log_info "Binary installed"

    # Step 4: Set capabilities
    log_step "4/5 Setting capabilities..."
    setcap cap_net_raw=+ep "$INSTALL_DIR/$BINARY_NAME"
    log_info "Capabilities set (cap_net_raw)"

    # Step 5: Start service
    log_step "5/5 Starting service..."
    systemctl start "$SERVICE_NAME"

    # Wait for startup
    sleep 2

    # Verify
    echo ""
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "Upgrade successful!"
        echo ""
        echo "┌────────────────────────────────────────────────────────┐"
        echo "│  Status: RUNNING                                       │"
        echo "├────────────────────────────────────────────────────────┤"
        printf "│  Version: %-46s │\n" "$($INSTALL_DIR/$BINARY_NAME version 2>/dev/null || echo 'unknown')"
        printf "│  Config:  %-46s │\n" "$CONFIG_FILE"
        echo "└────────────────────────────────────────────────────────┘"
        echo ""
        systemctl status "$SERVICE_NAME" --no-pager | head -15
    else
        log_error "Service failed to start!"
        echo ""
        log_warn "Rolling back to previous version..."
        if [[ -f "$INSTALL_DIR/${BINARY_NAME}.bak" ]]; then
            cp "$INSTALL_DIR/${BINARY_NAME}.bak" "$INSTALL_DIR/$BINARY_NAME"
            setcap cap_net_raw=+ep "$INSTALL_DIR/$BINARY_NAME"
            systemctl start "$SERVICE_NAME"
            log_info "Rollback complete"
        fi
        echo ""
        echo "Check logs with: journalctl -u seed -n 50"
        exit 1
    fi
}

# Parse arguments
BUILD_MODE=false
BINARY_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD_MODE=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            BINARY_PATH="$1"
            shift
            ;;
    esac
done

# Main
check_root
check_installed

if [[ "$BUILD_MODE" == true ]]; then
    BINARY_PATH=$(build_from_source)
elif [[ -z "$BINARY_PATH" ]]; then
    # Default to ./seed in current directory
    if [[ -f "./$BINARY_NAME" ]]; then
        BINARY_PATH="./$BINARY_NAME"
    else
        log_error "Binary not found. Specify path or use --build flag."
        usage
        exit 1
    fi
fi

# Verify binary exists
if [[ ! -f "$BINARY_PATH" ]]; then
    log_error "Binary not found: $BINARY_PATH"
    exit 1
fi

# Verify it's executable
if [[ ! -x "$BINARY_PATH" ]]; then
    chmod +x "$BINARY_PATH"
fi

do_upgrade "$BINARY_PATH"
