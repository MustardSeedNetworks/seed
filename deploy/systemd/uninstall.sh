#!/bin/bash
#
# The Seed Uninstallation Script
# Removes The Seed systemd service and files from Linux
#
set -e

# Configuration
INSTALL_DIR="/usr/local/seed"
SERVICE_USER="seed"
SERVICE_GROUP="seed"
SERVICE_NAME="seed"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Parse arguments
REMOVE_DATA=false
REMOVE_USER=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --purge)
            REMOVE_DATA=true
            REMOVE_USER=true
            shift
            ;;
        --keep-data)
            REMOVE_DATA=false
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --purge      Remove all data, configs, logs, and user account"
            echo "  --keep-data  Keep configuration and log files (default)"
            echo "  --help, -h   Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

log_info "Uninstalling The Seed..."

# Stop and disable service
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    log_info "Stopping The Seed service..."
    systemctl stop "$SERVICE_NAME"
fi

if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    log_info "Disabling The Seed service..."
    systemctl disable "$SERVICE_NAME"
fi

# Remove service file
if [[ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
    log_info "Removing systemd service file..."
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload
fi

# Remove binary
if [[ -f "$INSTALL_DIR/seed" ]]; then
    log_info "Removing The Seed binary..."
    rm -f "$INSTALL_DIR/seed"
fi

# Remove data if requested
if [[ "$REMOVE_DATA" == true ]]; then
    if [[ -d "$INSTALL_DIR" ]]; then
        log_info "Removing all data from $INSTALL_DIR..."
        rm -rf "$INSTALL_DIR"
    fi
else
    log_info "Keeping configuration and log files in $INSTALL_DIR"
    log_warn "To remove all data, run: $0 --purge"
fi

# Remove user and group if requested
if [[ "$REMOVE_USER" == true ]]; then
    if getent passwd "$SERVICE_USER" > /dev/null 2>&1; then
        log_info "Removing user: $SERVICE_USER"
        userdel "$SERVICE_USER" 2>/dev/null || true
    fi

    if getent group "$SERVICE_GROUP" > /dev/null 2>&1; then
        log_info "Removing group: $SERVICE_GROUP"
        groupdel "$SERVICE_GROUP" 2>/dev/null || true
    fi
fi

log_info "The Seed has been uninstalled."

if [[ "$REMOVE_DATA" == false ]]; then
    echo ""
    echo "Configuration and logs preserved at: $INSTALL_DIR"
    echo "To completely remove, run: $0 --purge"
fi
