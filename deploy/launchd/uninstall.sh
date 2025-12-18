#!/bin/bash
#
# The Seed Uninstall Script for macOS
# Removes The Seed launchd service and optionally all data
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
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --purge     Remove all data including configs, logs, and surveys"
    echo "  --help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  sudo $0           # Uninstall, keep data"
    echo "  sudo $0 --purge   # Uninstall and remove all data"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Check if this is macOS
if [[ "$(uname)" != "Darwin" ]]; then
    log_error "This script is for macOS only."
    exit 1
fi

# Parse arguments
PURGE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --purge)
            PURGE=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

log_info "Uninstalling The Seed from macOS..."
echo ""

# Step 1: Stop and unload service
log_step "1/3 Stopping service..."
if launchctl list | grep -q "com.seed"; then
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    sleep 2
    log_info "Service stopped"
else
    log_warn "Service was not loaded"
fi

# Force kill if still running
if pgrep -f "/usr/local/seed/seed" > /dev/null 2>&1; then
    log_warn "Force stopping process..."
    pkill -f "/usr/local/seed/seed" || true
    sleep 1
fi

# Step 2: Remove launchd plist
log_step "2/3 Removing launchd configuration..."
if [[ -f "$PLIST_PATH" ]]; then
    rm -f "$PLIST_PATH"
    log_info "Removed $PLIST_PATH"
else
    log_warn "Plist not found"
fi

# Step 3: Remove files
log_step "3/3 Removing files..."

if [[ "$PURGE" == true ]]; then
    log_warn "Purge mode: Removing ALL data including configs and surveys"
    if [[ -d "$INSTALL_DIR" ]]; then
        rm -rf "$INSTALL_DIR"
        log_info "Removed $INSTALL_DIR and all contents"
    fi
else
    # Keep configs, logs, data - just remove binary
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        rm -f "$INSTALL_DIR/$BINARY_NAME"
        rm -f "$INSTALL_DIR/${BINARY_NAME}.bak"
        log_info "Removed binary"
    fi

    if [[ -d "$INSTALL_DIR" ]]; then
        log_warn "Data preserved in $INSTALL_DIR"
        log_warn "  - Configs: $INSTALL_DIR/configs/"
        log_warn "  - Logs:    $INSTALL_DIR/logs/"
        log_warn "  - Data:    $INSTALL_DIR/data/"
        echo ""
        log_info "To remove all data, run: sudo $0 --purge"
    fi
fi

echo ""
log_info "Uninstall complete!"

if [[ "$PURGE" != true && -d "$INSTALL_DIR" ]]; then
    echo ""
    echo "Note: Configuration and data were preserved."
    echo "Run 'sudo $0 --purge' to remove everything."
fi
