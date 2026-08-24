#!/usr/bin/env python3
"""Run the structural i18n rules and report findings as GitHub annotations.

Kept as a file rather than inline in validate.sh: the reporting needs quoting
that does not survive being nested inside a shell function, and having it here
means it can be run and tested directly.

Exit status is 1 if anything is found, 0 otherwise, and 0 with a notice when
semgrep is not installed — CI installs it; a laptop without it should not fail
the whole gate.
"""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
RULES = ROOT / "scripts" / "i18n" / "semgrep-i18n.yml"
UI_SRC = ROOT / "ui" / "src"


def main() -> int:
    if shutil.which("semgrep") is None:
        print("semgrep not installed; skipping structural i18n rules")
        return 0
    if not RULES.is_file():
        print(f"missing rules file: {RULES}")
        return 1

    proc = subprocess.run(
        ["semgrep", "--config", str(RULES), "--metrics=off", "--json", str(UI_SRC)],
        capture_output=True,
        text=True,
        check=False,
    )
    try:
        results = json.loads(proc.stdout).get("results", [])
    except json.JSONDecodeError:
        print("semgrep produced no parseable output:")
        print(proc.stderr.strip()[:2000])
        return 1

    if not results:
        return 0

    print(f"::error::{len(results)} banned t() fallback pattern(s):")
    for r in sorted(results, key=lambda x: (x["path"], x["start"]["line"])):
        rule = r["check_id"].rsplit(".", 1)[-1]
        line = r["start"]["line"]
        try:
            path = str(Path(r["path"]).resolve().relative_to(ROOT))
        except ValueError:
            path = r["path"]
        # semgrep's extra.lines can report a propagated definition rather than
        # the call site, which reads as the wrong code. Quote the file.
        try:
            src = (ROOT / path).read_text().split("\n")[line - 1]
        except (OSError, IndexError):
            src = ""
        snippet = " ".join(src.split())[:100]
        print(f"  {path}:{line}: {rule}: {snippet}")
        print(f"::error file={path},line={line}::{rule} — put the copy in a locale file")
    return 1


if __name__ == "__main__":
    sys.exit(main())
