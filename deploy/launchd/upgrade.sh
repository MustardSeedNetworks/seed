#!/bin/bash
#
# The Seed Upgrade Script for macOS
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
BINARY_NAME="seed"
PLIST_NAME="com.seed.plist"
PLIST_PATH="/Library/LaunchDaemons/$PLIST_NAME"

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

# Check if this is macOS
check_macos() {
    if [[ "$(uname)" != "Darwin" ]]; then
        log_error "This script is for macOS only. Use systemd scripts for Linux."
        exit 1
    fi
}

# Check if service is installed
check_installed() {
    if [[ ! -f "$PLIST_PATH" ]]; then
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

    # Build for macOS
    log_info "Building..."
    make build

    # Return path to built binary
    echo "./seed"
}

# Check if service is running
is_service_running() {
    pgrep -f "/usr/local/seed/seed" > /dev/null 2>&1
}

# Main upgrade function
do_upgrade() {
    local binary_path="$1"

    log_info "Starting Seed upgrade on macOS..."
    echo ""

    # Get versions
    local current_version=$(get_current_version)
    local new_version=$(get_new_version "$binary_path")

    echo "┌────────────────────────────────────────────────────────┐"
    echo "│  Seed macOS Upgrade                                    │"
    echo "├────────────────────────────────────────────────────────┤"
    printf "│  Current version: %-37s │\n" "$current_version"
    printf "│  New version:     %-37s │\n" "$new_version"
    echo "└────────────────────────────────────────────────────────┘"
    echo ""

    # Step 1: Stop service
    log_step "1/4 Stopping service..."
    if is_service_running; then
        launchctl unload "$PLIST_PATH" 2>/dev/null || true
        sleep 2
        # Force kill if still running
        if is_service_running; then
            log_warn "Force stopping..."
            pkill -f "/usr/local/seed/seed" || true
            sleep 1
        fi
        log_info "Service stopped"
    else
        log_warn "Service was not running"
    fi

    # Step 2: Backup current binary
    log_step "2/4 Backing up current binary..."
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        cp "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}.bak"
        log_info "Backup created: ${BINARY_NAME}.bak"
    fi

    # Step 3: Copy new binary
    log_step "3/4 Installing new binary..."
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    log_info "Binary installed"

    # Step 4: Start service
    log_step "4/4 Starting service..."
    launchctl load "$PLIST_PATH"

    # Wait for startup
    sleep 3

    # Verify
    echo ""
    if is_service_running; then
        log_info "Upgrade successful!"
        echo ""
        echo "┌────────────────────────────────────────────────────────┐"
        echo "│  Status: RUNNING                                       │"
        echo "├────────────────────────────────────────────────────────┤"
        printf "│  Version: %-46s │\n" "$("$INSTALL_DIR/$BINARY_NAME" version 2>/dev/null || echo 'unknown')"
        printf "│  PID:     %-46s │\n" "$(pgrep -f '/usr/local/seed/seed')"
        echo "└────────────────────────────────────────────────────────┘"
        echo ""
        log_info "View logs: tail -f $INSTALL_DIR/logs/seed.log"
    else
        log_error "Service failed to start!"
        echo ""
        log_warn "Rolling back to previous version..."
        if [[ -f "$INSTALL_DIR/${BINARY_NAME}.bak" ]]; then
            cp "$INSTALL_DIR/${BINARY_NAME}.bak" "$INSTALL_DIR/$BINARY_NAME"
            launchctl load "$PLIST_PATH"
            sleep 2
            if is_service_running; then
                log_info "Rollback successful - running previous version"
            else
                log_error "Rollback failed!"
            fi
        fi
        echo ""
        echo "Check logs: tail -f $INSTALL_DIR/logs/seed.error.log"
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
check_macos
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
