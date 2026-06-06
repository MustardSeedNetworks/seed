#!/bin/sh
set -e

BINARY=/usr/bin/seed

if command -v setcap >/dev/null 2>&1; then
    setcap 'cap_net_raw,cap_net_admin=+ep' "$BINARY" || \
        echo "warning: could not set capabilities on $BINARY"
else
    echo "warning: setcap not found; install libcap/libcap2-bin for non-root diagnostics"
fi

PORT=8443

open_firewall() {
    opened=""
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
        if ufw allow ${PORT}/tcp comment 'Seed WebUI HTTPS' >/dev/null 2>&1; then
            opened="ufw"
        fi
    fi
    if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --permanent --add-port=${PORT}/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
        opened="${opened:+$opened, }firewalld"
    fi
    if [ -n "$opened" ]; then
        echo "Opened TCP ${PORT} on: $opened (SEED_OPEN_FIREWALL set)"
    fi
}

firewall_hint() {
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
        echo "  sudo ufw allow ${PORT}/tcp"
    elif command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        echo "  sudo firewall-cmd --permanent --add-port=${PORT}/tcp && sudo firewall-cmd --reload"
    fi
}

# Network exposure is the operator's choice. We do NOT open the firewall by
# default; set SEED_OPEN_FIREWALL=1 to opt in (e.g. automated provisioning).
case "${SEED_OPEN_FIREWALL:-0}" in
    1 | true | yes | TRUE | YES) open_firewall ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
    systemctl enable seed.service >/dev/null 2>&1 || true
    if systemctl is-active --quiet seed.service 2>/dev/null; then
        systemctl restart seed.service || true
    else
        systemctl start seed.service || true
    fi
fi

cat <<'EOF'

==========================================
  The Seed installed successfully
==========================================

Web interface: https://localhost:8443

Commands:
  View logs:  journalctl -u seed -f
  Restart:    sudo systemctl restart seed
  Status:     sudo systemctl status seed
  Stop:       sudo systemctl stop seed

EOF

# When we did not open the firewall, tell the operator how (only if one is active).
case "${SEED_OPEN_FIREWALL:-0}" in
    1 | true | yes | TRUE | YES) : ;;
    *)
        hint=$(firewall_hint)
        if [ -n "$hint" ]; then
            echo "An active firewall was detected; TCP ${PORT} is NOT open to the network."
            echo "To allow remote access to the web interface, run:"
            echo "$hint"
            echo "(or re-install with SEED_OPEN_FIREWALL=1 to open it automatically)"
            echo ""
        fi
        ;;
esac

exit 0
