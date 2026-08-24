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
# The Wi-Fi helper runs as a per-user LaunchAgent, so it has to be booted out
# of the console user's GUI session before its plist is removed.
HELPER_LABEL="net.mustardseed.seed.wifihelper"
HELPER_PLIST_PATH="/Library/LaunchAgents/$HELPER_LABEL.plist"
HELPER_APP="/Library/Application Support/Seed/Seed Wi-Fi Helper.app"

CONSOLE_USER=$(stat -f%Su /dev/console 2>/dev/null || echo "root")
if [[ "$CONSOLE_USER" != "root" && -n "$CONSOLE_USER" ]]; then
    CONSOLE_UID=$(id -u "$CONSOLE_USER" 2>/dev/null || echo "")
    if [[ -n "$CONSOLE_UID" ]]; then
        launchctl bootout "gui/$CONSOLE_UID/$HELPER_LABEL" 2>/dev/null || true
    fi
fi

if [[ -f "$HELPER_PLIST_PATH" ]]; then
    rm -f "$HELPER_PLIST_PATH"
    log_info "Removed $HELPER_PLIST_PATH"
fi

if [[ -d "$HELPER_APP" ]]; then
    rm -rf "$HELPER_APP"
    log_info "Removed $HELPER_APP"
fi

# The Location Services entry survives removing the bundle; macOS keeps it
# until the user clears it themselves, so say so rather than leave it puzzling.
log_info "Note: 'Seed Wi-Fi Helper' may remain listed in System Settings >"
log_info "      Privacy & Security > Location Services. macOS keeps that entry;"
log_info "      it can be removed there."

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
