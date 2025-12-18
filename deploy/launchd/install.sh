#!/bin/bash
#
# The Seed Installation Script for macOS
# Installs The Seed as a launchd service on macOS
#
set -e

# Configuration
INSTALL_DIR="/usr/local/seed"
BINARY_NAME="seed"
PLIST_NAME="com.seed.plist"
PLIST_DEST="/Library/LaunchDaemons/$PLIST_NAME"

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
    echo "Usage: $0 [BINARY_PATH]"
    echo ""
    echo "Examples:"
    echo "  sudo $0                    # Use ./seed binary"
    echo "  sudo $0 /path/to/seed      # Use specified binary"
    echo "  sudo $0 --help             # Show this help message"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Check if this is macOS
if [[ "$(uname)" != "Darwin" ]]; then
    log_error "This script is for macOS only. Use systemd scripts for Linux."
    exit 1
fi

# Handle arguments
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    usage
    exit 0
fi

# Check if binary exists in current directory or is provided
BINARY_PATH=""
if [[ -n "$1" && -f "$1" ]]; then
    BINARY_PATH="$1"
elif [[ -f "./${BINARY_NAME}" ]]; then
    BINARY_PATH="./${BINARY_NAME}"
else
    log_error "The Seed binary not found."
    log_error "Please run from the directory containing the binary or provide the path."
    usage
    exit 1
fi

log_info "Installing The Seed on macOS..."
echo ""

# Get version of binary being installed
NEW_VERSION=$("$BINARY_PATH" version 2>/dev/null || echo "unknown")

echo "┌────────────────────────────────────────────────────────┐"
echo "│  Seed macOS Installation                               │"
echo "├────────────────────────────────────────────────────────┤"
printf "│  Version:     %-41s │\n" "$NEW_VERSION"
printf "│  Install dir: %-41s │\n" "$INSTALL_DIR"
echo "└────────────────────────────────────────────────────────┘"
echo ""

# Step 1: Create installation directory
log_step "1/6 Creating installation directory..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/configs"
mkdir -p "$INSTALL_DIR/logs"
mkdir -p "$INSTALL_DIR/data"
log_info "Created $INSTALL_DIR"

# Step 2: Stop existing service if running
log_step "2/6 Checking for existing service..."
if launchctl list | grep -q "com.seed"; then
    log_info "Stopping existing service..."
    launchctl unload "$PLIST_DEST" 2>/dev/null || true
    sleep 2
fi

# Step 3: Copy binary
log_step "3/6 Installing binary..."
cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"
log_info "Binary installed"

# Step 4: Install launchd plist
log_step "4/6 Installing launchd service..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/$PLIST_NAME" ]]; then
    cp "$SCRIPT_DIR/$PLIST_NAME" "$PLIST_DEST"
else
    # Create plist inline if not found
    cat > "$PLIST_DEST" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.seed</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/seed/seed</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/usr/local/seed</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/usr/local/seed/logs/seed.log</string>
    <key>StandardErrorPath</key>
    <string>/usr/local/seed/logs/seed.error.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>HOME</key>
        <string>/usr/local/seed</string>
    </dict>
    <key>ThrottleInterval</key>
    <integer>5</integer>
</dict>
</plist>
EOF
fi
chmod 644 "$PLIST_DEST"
chown root:wheel "$PLIST_DEST"
log_info "Launchd plist installed"

# Step 5: Load and start service
log_step "5/6 Starting service..."
launchctl load "$PLIST_DEST"

# Wait for startup
sleep 3

# Step 6: Verify
log_step "6/6 Verifying installation..."
echo ""

if launchctl list | grep -q "com.seed"; then
    # Check if process is actually running
    if pgrep -f "/usr/local/seed/seed" > /dev/null; then
        log_info "Installation successful!"
        echo ""
        echo "┌────────────────────────────────────────────────────────┐"
        echo "│  Status: RUNNING                                       │"
        echo "├────────────────────────────────────────────────────────┤"
        printf "│  Version: %-46s │\n" "$("$INSTALL_DIR/$BINARY_NAME" version 2>/dev/null || echo 'unknown')"
        printf "│  PID:     %-46s │\n" "$(pgrep -f '/usr/local/seed/seed')"
        printf "│  Logs:    %-46s │\n" "$INSTALL_DIR/logs/"
        echo "└────────────────────────────────────────────────────────┘"
        echo ""
        echo "Commands:"
        echo "  View logs:      tail -f $INSTALL_DIR/logs/seed.log"
        echo "  View errors:    tail -f $INSTALL_DIR/logs/seed.error.log"
        echo "  Stop:           sudo launchctl unload $PLIST_DEST"
        echo "  Start:          sudo launchctl load $PLIST_DEST"
        echo "  Restart:        sudo launchctl unload $PLIST_DEST && sudo launchctl load $PLIST_DEST"
        echo ""

        # Generate and display initial credentials
        CRED_FILE="$INSTALL_DIR/.seed-credentials"
        log_info "Generating initial admin credentials..."
        if "$INSTALL_DIR/$BINARY_NAME" credentials -config "$INSTALL_DIR/configs/seed.yaml" -file "$CRED_FILE" 2>/dev/null; then
            echo ""
            cat "$CRED_FILE"
            echo ""
            log_warn "Credentials saved to: $CRED_FILE"
            log_warn "DELETE this file after saving the credentials securely!"
            chmod 600 "$CRED_FILE"
        else
            log_warn "Could not generate credentials. Visit the web UI to complete setup."
        fi
    else
        log_error "Service loaded but process not running!"
        log_error "Check logs: tail -f $INSTALL_DIR/logs/seed.error.log"
        exit 1
    fi
else
    log_error "Failed to load launchd service!"
    log_error "Check: sudo launchctl load $PLIST_DEST"
    exit 1
fi
