#!/bin/sh
set -e

BINARY=/usr/bin/seed

if command -v setcap >/dev/null 2>&1; then
    setcap 'cap_net_raw,cap_net_admin=+ep' "$BINARY" || \
        echo "warning: could not set capabilities on $BINARY"
else
    echo "warning: setcap not found; install libcap/libcap2-bin for non-root diagnostics"
fi

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
    ufw allow 8443/tcp comment 'Seed WebUI HTTPS' >/dev/null 2>&1 || true
fi

if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
    firewall-cmd --permanent --add-port=8443/tcp >/dev/null 2>&1 || true
    firewall-cmd --reload >/dev/null 2>&1 || true
fi

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

exit 0
