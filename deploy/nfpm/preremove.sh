#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet seed.service 2>/dev/null; then
        systemctl stop seed.service || true
    fi
    if systemctl is-enabled --quiet seed.service 2>/dev/null; then
        systemctl disable seed.service || true
    fi
fi

exit 0
