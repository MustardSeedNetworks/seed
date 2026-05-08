#!/bin/sh
set -e

is_purge=0
case "$1" in
    purge|0)
        is_purge=1
        ;;
esac

if [ "$is_purge" -eq 1 ]; then
    if command -v ufw >/dev/null 2>&1; then
        ufw delete allow 8443/tcp >/dev/null 2>&1 || true
    fi
    if command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --permanent --remove-port=8443/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
    fi

    if getent passwd seed >/dev/null 2>&1; then
        userdel seed >/dev/null 2>&1 || true
    fi
    if getent group seed >/dev/null 2>&1; then
        groupdel seed >/dev/null 2>&1 || true
    fi

    rm -rf /etc/seed /var/lib/seed /var/log/seed
else
    echo "Seed removed. Data preserved in /var/lib/seed"
fi

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

exit 0
