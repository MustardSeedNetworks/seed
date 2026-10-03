#!/bin/sh
set -e

# Stop the service only when the package is actually going away.
#
# RPM runs the OLD package's %preun *after* the NEW package's %post, so an
# unconditional stop here undid the upgrade: postinstall had just enabled and
# restarted seed, this stopped and disabled it, and the upgrade finished with
# the service down (#2861). dpkg runs prerm before the new postinst, which is
# why only the .rpm broke.
#
# RPM passes the number of versions that will remain (0 on removal, 1 or more
# during an upgrade); dpkg passes a word. postremove.sh keys off the same
# convention. niac-go#2085 and stem#1448 fixed the identical script.
case "${1:-}" in
    0 | remove | purge) ;;
    *) exit 0 ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet seed.service 2>/dev/null; then
        systemctl stop seed.service || true
    fi
    if systemctl is-enabled --quiet seed.service 2>/dev/null; then
        systemctl disable seed.service || true
    fi
fi

exit 0
