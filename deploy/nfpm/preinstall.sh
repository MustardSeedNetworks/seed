#!/bin/sh
set -e

if ! getent group seed >/dev/null 2>&1; then
    groupadd --system seed
fi

if ! getent passwd seed >/dev/null 2>&1; then
    useradd --system \
        --gid seed \
        --home-dir /var/lib/seed \
        --no-create-home \
        --shell /usr/sbin/nologin \
        --comment "The Seed Network Diagnostic Tool" \
        seed
fi

exit 0
