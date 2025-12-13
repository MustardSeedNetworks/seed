#!/bin/bash
#
# LuminetIQ Installation Script
# Installs LuminetIQ as a systemd service on Linux
#
set -e

# Configuration
INSTALL_DIR="/usr/local/luminetiq"
SERVICE_USER="luminetiq"
SERVICE_GROUP="luminetiq"
BINARY_NAME="luminetiq"

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

# Check if systemd is available
if ! command -v systemctl &> /dev/null; then
    log_error "systemd is not available on this system"
    exit 1
fi

# Check if binary exists in current directory or is provided
BINARY_PATH=""
if [[ -f "./${BINARY_NAME}" ]]; then
    BINARY_PATH="./${BINARY_NAME}"
elif [[ -f "$1" ]]; then
    BINARY_PATH="$1"
else
    log_error "LuminetIQ binary not found. Please run this script from the directory containing the binary or provide the path as an argument."
    exit 1
fi

log_info "Installing LuminetIQ..."

# Create service user and group if they don't exist
if ! getent group "$SERVICE_GROUP" > /dev/null 2>&1; then
    log_info "Creating group: $SERVICE_GROUP"
    groupadd --system "$SERVICE_GROUP"
fi

if ! getent passwd "$SERVICE_USER" > /dev/null 2>&1; then
    log_info "Creating user: $SERVICE_USER"
    useradd --system --gid "$SERVICE_GROUP" --home-dir "$INSTALL_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
fi

# Create installation directory
log_info "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/configs"
mkdir -p "$INSTALL_DIR/logs"

# Stop existing service if running
if systemctl is-active --quiet luminetiq; then
    log_info "Stopping existing LuminetIQ service..."
    systemctl stop luminetiq
fi

# Copy binary
log_info "Installing binary..."
cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"

# Set ownership
log_info "Setting ownership..."
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"

# Set capabilities for raw socket access
log_info "Setting capabilities for raw socket access..."
setcap cap_net_raw=+ep "$INSTALL_DIR/$BINARY_NAME"

# Install systemd service file
log_info "Installing systemd service..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/luminetiq.service" ]]; then
    cp "$SCRIPT_DIR/luminetiq.service" /etc/systemd/system/luminetiq.service
else
    # Create service file inline if not found
    cat > /etc/systemd/system/luminetiq.service << 'EOF'
[Unit]
Description=LuminetIQ Network Diagnostics
Documentation=https://github.com/krisarmstrong/luminetiq
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=luminetiq
Group=luminetiq
WorkingDirectory=/usr/local/luminetiq
ExecStartPre=/sbin/setcap cap_net_raw=+ep /usr/local/luminetiq/luminetiq
ExecStart=/usr/local/luminetiq/luminetiq
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=no
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/usr/local/luminetiq/configs /usr/local/luminetiq/logs
PrivateTmp=true

# Environment
Environment=HOME=/usr/local/luminetiq

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd
log_info "Reloading systemd daemon..."
systemctl daemon-reload

# Enable service
log_info "Enabling LuminetIQ service..."
systemctl enable luminetiq

# Start service
log_info "Starting LuminetIQ service..."
systemctl start luminetiq

# Wait a moment for startup
sleep 3

# Check status
if systemctl is-active --quiet luminetiq; then
    log_info "LuminetIQ installed and running successfully!"
    echo ""
    echo "Service Status:"
    systemctl status luminetiq --no-pager
    echo ""
    echo "Commands:"
    echo "  View logs:      journalctl -u luminetiq -f"
    echo "  Restart:        systemctl restart luminetiq"
    echo "  Stop:           systemctl stop luminetiq"
    echo "  Status:         systemctl status luminetiq"
    echo ""

    # Show initial credentials if available
    if [[ -f "$INSTALL_DIR/configs/luminetiq.yaml" ]]; then
        log_info "Configuration file created at: $INSTALL_DIR/configs/luminetiq.yaml"
    fi

    # Check for first-run password in journal
    echo "Check logs for initial admin password: journalctl -u luminetiq | grep Password"
else
    log_error "LuminetIQ failed to start. Check logs with: journalctl -u luminetiq"
    exit 1
fi
