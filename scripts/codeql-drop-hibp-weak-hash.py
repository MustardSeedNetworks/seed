#!/usr/bin/env python3
"""Drop the HIBP SHA-1 result from a CodeQL SARIF log.

go/weak-sensitive-data-hashing stays enabled for the rest of the repository.
A query filter cannot name a path, and the Go extractor ignores paths-ignore,
so the one contractual SHA-1 call in internal/auth/hibp.go is removed here
before the SARIF is uploaded.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

RULE_ID = "go/weak-sensitive-data-hashing"
HIBP_PATH = "internal/auth/hibp.go"


def _uri(location: dict) -> str:
    physical = location.get("physicalLocation") or {}
    artifact = physical.get("artifactLocation") or {}
    return str(artifact.get("uri") or "").replace("\\", "/")


def _is_hibp(uri: str) -> bool:
    normalized = uri[2:] if uri.startswith("./") else uri
    return normalized == HIBP_PATH or normalized.endswith("/" + HIBP_PATH)


def _rule_id(result: dict) -> str:
    rule_id = result.get("ruleId")
    if isinstance(rule_id, str) and rule_id:
        return rule_id
    rule = result.get("rule") or {}
    nested = rule.get("id")
    return nested if isinstance(nested, str) else ""


def _primary_uri(result: dict) -> str:
    locations = result.get("locations") or []
    if not locations:
        return ""
    return _uri(locations[0])


def drop_hibp_weak_hash(log: dict) -> int:
    """Remove weak-hash results whose primary location is the HIBP file."""
    dropped = 0
    for run in log.get("runs") or []:
        kept = []
        for result in run.get("results") or []:
            if _rule_id(result) == RULE_ID and _is_hibp(_primary_uri(result)):
                dropped += 1
                continue
            kept.append(result)
        run["results"] = kept
    return dropped


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print(f"usage: {argv[0]} SARIF", file=sys.stderr)
        return 2
    path = Path(argv[1])
    log = json.loads(path.read_text(encoding="utf-8"))
    dropped = drop_hibp_weak_hash(log)
    path.write_text(json.dumps(log) + "\n", encoding="utf-8")
    print(f"dropped {dropped} {RULE_ID} result(s) in {HIBP_PATH}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
