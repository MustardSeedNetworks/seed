#!/bin/bash
#
# The Seed Installation Script
# Installs The Seed as a systemd service on Linux
#
set -e

# Configuration
INSTALL_DIR="/usr/local/seed"
SERVICE_USER="seed"
SERVICE_GROUP="seed"
BINARY_NAME="seed"

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
    log_error "The Seed binary not found. Please run this script from the directory containing the binary or provide the path as an argument."
    exit 1
fi

log_info "Installing The Seed..."

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
if systemctl is-active --quiet seed; then
    log_info "Stopping existing The Seed service..."
    systemctl stop seed
fi

# Copy binary
log_info "Installing binary..."
cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"

# Set ownership
log_info "Setting ownership..."
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"

# Set capabilities needed for ICMP/ARP scans and MTU/Wi-Fi control
log_info "Setting capabilities for network operations (raw + admin)..."
setcap cap_net_raw,cap_net_admin=+ep "$INSTALL_DIR/$BINARY_NAME"

# Install systemd service file
log_info "Installing systemd service..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/seed.service" ]]; then
    cp "$SCRIPT_DIR/seed.service" /etc/systemd/system/seed.service
else
    # Create service file inline if not found
    cat > /etc/systemd/system/seed.service << 'EOF'
[Unit]
Description=The Seed - Mustard Seed Networks
Documentation=https://github.com/krisarmstrong/seed
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=seed
Group=seed
WorkingDirectory=/usr/local/seed
ExecStart=/usr/local/seed/seed
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
# Note: Capabilities are set at install time via install.sh
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/usr/local/seed/configs /usr/local/seed/logs
PrivateTmp=true

# Environment
Environment=HOME=/usr/local/seed

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd
log_info "Reloading systemd daemon..."
systemctl daemon-reload

# Enable service
log_info "Enabling The Seed service..."
systemctl enable seed

# Start service
log_info "Starting The Seed service..."
systemctl start seed

# Wait a moment for startup
sleep 3

# Check status
if systemctl is-active --quiet seed; then
    log_info "The Seed installed and running successfully!"
    echo ""
    echo "Service Status:"
    systemctl status seed --no-pager
    echo ""
    echo "Commands:"
    echo "  View logs:      journalctl -u seed -f"
    echo "  Restart:        systemctl restart seed"
    echo "  Stop:           systemctl stop seed"
    echo "  Status:         systemctl status seed"
    echo ""

    # Show initial credentials if available
    if [[ -f "$INSTALL_DIR/configs/seed.yaml" ]]; then
        log_info "Configuration file created at: $INSTALL_DIR/configs/seed.yaml"
    fi

    # Generate and display initial credentials (fixes #489)
    CRED_FILE="$INSTALL_DIR/.seed-credentials"
    log_info "Generating initial admin credentials..."
    if "$INSTALL_DIR/$BINARY_NAME" credentials -config "$INSTALL_DIR/configs/seed.yaml" -file "$CRED_FILE"; then
        echo ""
        cat "$CRED_FILE"
        echo ""
        log_warn "Credentials saved to: $CRED_FILE (mode 0600)"
        log_warn "DELETE this file after saving the credentials securely!"
        # Set ownership on credentials file
        chown "$SERVICE_USER:$SERVICE_GROUP" "$CRED_FILE"
    else
        log_warn "Could not generate credentials. Visit the web UI to complete setup."
    fi
else
    log_error "The Seed failed to start. Check logs with: journalctl -u seed"
    exit 1
fi
