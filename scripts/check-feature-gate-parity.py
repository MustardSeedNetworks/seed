#!/usr/bin/env python3
"""check-feature-gate-parity.py — UI page gate ⇄ requireFeature gate.

The UI gates whole pages on a licence feature (ui/src/constants/featureCatalog.ts,
read by <GatedPreview>), and the API gates routes on the same features
(`feature: "x"` rows in internal/api/server_routes.go, enforced by
requireFeature). Nothing held the two together: a UI gate could name a feature
no route enforces — a page hidden behind a pitch the server would have served
anyway — or advertise a tier the licence policy does not grant it in (#2669).

Two checks over every entry in the UI catalog:

  A. The feature is enforced by requireFeature on at least one route.
  B. The tier the UI advertises is the tier internal/license/policy.go grants
     the feature in (starterFeatures -> Starter, proFeatures -> Pro).

This does not map a page to the API paths its components request; only a test
through the real mux proves a given page's fetches are the gated ones.

Run locally: scripts/check-feature-gate-parity.py
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROUTES = "internal/api/server_routes.go"
POLICY = "internal/license/policy.go"
CATALOG = "ui/src/constants/featureCatalog.ts"

ROUTE_FEATURE = re.compile(r'\bfeature:\s*"([a-z][a-z0-9_]*)"')
CATALOG_BODY = re.compile(r"export const FEATURE_CATALOG = \{(.*?)\n\} as const", re.S)
CATALOG_ENTRY = re.compile(r"(\w+):\s*\{\s*tier:\s*'(\w+)'")
POLICY_FUNC = re.compile(r"func (starterFeatures|proFeatures)\(\) \[\]string \{(.*?)\n\}", re.S)
FEATURE_STRING = re.compile(r'"([a-z][a-z0-9_]*)"')


def ui_catalog(root: Path) -> dict[str, str]:
    body = CATALOG_BODY.search((root / CATALOG).read_text(encoding="utf-8"))
    if not body:
        raise SystemExit(f"{CATALOG}: FEATURE_CATALOG not found — is the regex stale?")
    return dict(CATALOG_ENTRY.findall(body.group(1)))


def policy_tiers(root: Path) -> dict[str, str]:
    """Feature -> tier name. proFeatures() appends starterFeatures(), so a
    feature listed in the Starter body is Starter even though Pro also grants
    it; the tier a customer must buy is the lowest one that includes it."""
    text = (root / POLICY).read_text(encoding="utf-8")
    tiers: dict[str, str] = {}
    for func, body in POLICY_FUNC.findall(text):
        tier = "Starter" if func == "starterFeatures" else "Pro"
        for feature in FEATURE_STRING.findall(re.sub(r"//[^\n]*", "", body)):
            tiers.setdefault(feature, tier)
    return tiers


def run(root: Path, out=sys.stdout) -> int:
    catalog = ui_catalog(root)
    enforced = set(ROUTE_FEATURE.findall((root / ROUTES).read_text(encoding="utf-8")))
    tiers = policy_tiers(root)

    failures: list[str] = []
    for feature, tier in sorted(catalog.items()):
        if feature not in enforced:
            failures.append(
                f"{feature}: gated in the UI but no route in {ROUTES} carries feature: \"{feature}\""
            )
        actual = tiers.get(feature)
        if actual is None:
            failures.append(f"{feature}: gated in the UI but absent from the catalog in {POLICY}")
        elif actual != tier:
            failures.append(f"{feature}: UI advertises {tier}, {POLICY} grants it in {actual}")

    if failures:
        print("::error::a UI page gate disagrees with the feature the API enforces:", file=out)
        for line in failures:
            print(f"  {line}", file=out)
        return 1

    print(f"Feature-gate parity: {len(catalog)} UI-gated features agree with {ROUTES}.", file=out)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    return run(parser.parse_args().root)


if __name__ == "__main__":
    sys.exit(main())
