#!/bin/sh
set -e

# RPM only. %posttrans runs once the whole transaction is done, after the old
# package's %preun and %postun, so nothing can stop seed after this starts it.
# postinstall.sh leaves the service to this script on RPM (#2861).
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
    systemctl enable seed.service >/dev/null 2>&1 || true
    if systemctl is-active --quiet seed.service 2>/dev/null; then
        systemctl restart seed.service || true
    else
        systemctl start seed.service || true
    fi
fi

exit 0
