#!/usr/bin/env python3
"""Report a bundle's Location Services authorization.

Reading /var/db/locationd/clients.plist with grep or an awk range is
unreliable: client keys are long composites of user UUID and a mangled
identifier, several entries share prefixes, and a range like
``/pattern/,/^  }/`` silently reports a neighbouring block. That produced wrong
answers more than once during the macOS Wi-Fi work.

``plutil -convert json`` is not an option either — the plist stores binary
client tokens, which JSON cannot represent. plistlib reads it directly.

Exit status: 0 authorized, 1 registered but not authorized, 2 unknown to
locationd, so this can gate a script rather than only inform a human.
"""

from __future__ import annotations

import plistlib
import sys

CLIENTS = "/var/db/locationd/clients.plist"
DEFAULT_BUNDLE = "net.mustardseed.seed.wifihelper"


def main() -> int:
    bundle = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_BUNDLE

    try:
        with open(CLIENTS, "rb") as fh:
            clients = plistlib.load(fh)
    except OSError as err:
        print(f"cannot read {CLIENTS}: {err}", file=sys.stderr)
        return 2

    # Match the BundleId field exactly. Never match on the client key.
    hits = [
        value
        for value in clients.values()
        if isinstance(value, dict) and value.get("BundleId") == bundle
    ]

    if not hits:
        print(f"{bundle}: not registered with locationd")
        print("  The bundle has never asked. Launch it once with `open`, not by")
        print("  executing the binary inside it — a direct exec registers a")
        print("  different client identity, which holds no grant.")
        return 2

    entry = hits[0]
    authorized = bool(entry.get("Authorized", False))

    print(f"{bundle}: {'AUTHORIZED' if authorized else 'NOT authorized'}")
    print(f"  registered : {bool(entry.get('Registered', False))}")
    print(f"  bundle     : {entry.get('BundlePath', '(none)')}")

    if not authorized:
        print("  Enable it in System Settings > Privacy & Security > Location Services.")
        print("  If it never prompts, the bundle lacks the entitlement")
        print("  com.apple.security.personal-information.location. Confirm with:")
        print("    log show --last 5m --predicate 'process == \"locationd\"' --info \\")
        print(f"      | grep -i {bundle}")
        return 1

    print("  Scans will return network names and BSSIDs — provided the process is")
    print("  launched so LaunchServices attributes it to this bundle.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
