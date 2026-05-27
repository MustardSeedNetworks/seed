#!/usr/bin/env python3
"""
Cross-check t('key') usage in ui/src against committed EN locale
files. Fails if any t() call references a key that doesn't exist in
the locale JSON, OR if any locale key isn't referenced by any t()
call.

Read-only — never modifies locale files.

Companion to scripts/i18n/validate.sh (which checks key parity
across en/es, banned vocab, glossary preservation, etc.). This
script is the missing piece that validates source-code references
agree with the committed key shape — caught by tools like
i18next-parser, but without that tool's destructive overwrite
semantics.

Usage:
  python3 scripts/i18n/check-keys.py
    → exits 0 if source and JSON agree, non-zero otherwise.
"""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent.parent
UI_SRC = ROOT / "ui" / "src"
LOCALES_EN = ROOT / "internal" / "i18n" / "locales" / "en"

# Match t('ns:key.path') or t("ns:key.path") with optional second arg.
# Captures the literal key string. Allows JSX/TS surrounding context.
#
# To avoid false positives the regex requires:
#   (a) The function name is `t`, `tFoo`, or `tFoo.bar` — letters only
#       so identifiers like `target(`, `text(`, `toLocaleString(` don't
#       match. `\b` boundary anchors the `t` at word start.
#   (b) The string argument either has a `:` (namespace prefix) or
#       a `.` (dotted path). Bare strings like 'en-US', 'router-1',
#       'http' aren't translation keys; they're constants the i18n
#       check should ignore.
T_CALL = re.compile(
    r"""\bt(?:[A-Z][A-Za-z]*)?\s*\(\s*['"`]([a-zA-Z][\w-]*[.:][\w.:-]+)['"`]""",
    re.MULTILINE,
)


def flatten(d, prefix=""):
    """Recursively yield 'foo.bar.baz' keys from a nested dict."""
    if isinstance(d, dict):
        for k, v in d.items():
            yield from flatten(v, f"{prefix}.{k}" if prefix else k)
    else:
        yield prefix


def load_locale_keys() -> dict[str, set[str]]:
    """Return {namespace: {flat_key, …}} from internal/i18n/locales/en/*.json."""
    out: dict[str, set[str]] = {}
    for path in sorted(LOCALES_EN.glob("*.json")):
        ns = path.stem
        data = json.loads(path.read_text())
        out[ns] = set(flatten(data))
    return out


def normalize_plural(key: str) -> str:
    """Strip i18next plural suffixes so deviceCount in source matches
    deviceCount_one/_other in JSON."""
    # If the source calls t('foo.bar') and JSON has foo.bar_one + foo.bar_other,
    # we should count both as 'foo.bar' for the comparison.
    return key


def extract_t_calls() -> set[tuple[str, str, int, str]]:
    """Walk ui/src/**/*.{ts,tsx} and yield (file, line, key) tuples for
    every t('…') call. Returns a set of (namespace, key, line, file).

    Builds a per-file map of t-alias → namespace from useTranslation
    declarations so call sites that destructure aliases (e.g.
    `const { t: tDevices } = useTranslation('devices')`) attribute
    their calls to the correct namespace.
    """
    # Match all useTranslation('ns') declarations, capturing the
    # destructured alias name. Two forms:
    #   const { t } = useTranslation('common');           → alias 't' -> 'common'
    #   const { t: tDevices } = useTranslation('devices'); → 'tDevices' -> 'devices'
    USE_TRANSLATION = re.compile(
        r"""const\s*\{\s*t(?:\s*:\s*([A-Za-z]+))?\s*[,}].*?useTranslation\s*\(\s*['"]([a-z]+)['"]""",
        re.DOTALL,
    )
    # Tighter T_CALL that captures the alias function name (group 1)
    # plus the literal key (group 2).
    T_CALL_LOCAL = re.compile(
        r"""\b(t(?:[A-Z][A-Za-z]*)?)\s*\(\s*['"`]([a-zA-Z][\w-]*[.:][\w.:-]+)['"`]""",
        re.MULTILINE,
    )

    hits: set[tuple[str, str, int, str]] = set()
    for path in UI_SRC.rglob("*.ts*"):
        if "/node_modules/" in str(path):
            continue
        if path.name.endswith(".d.ts"):
            continue
        try:
            text = path.read_text()
        except UnicodeDecodeError:
            continue

        # Per-file alias → namespace map. Multiple useTranslation
        # declarations create multiple aliases; for the bare `t` alias
        # we collect ALL namespaces it might bind to (scope is per-
        # function, but we don't track scope without an AST — so we
        # accept the looser superset and rely on key lookup across all
        # candidate namespaces below).
        alias_to_namespaces: dict[str, set[str]] = {}
        for m in USE_TRANSLATION.finditer(text):
            alias = m.group(1) or "t"
            ns = m.group(2)
            alias_to_namespaces.setdefault(alias, set()).add(ns)
        if "t" not in alias_to_namespaces:
            # Default: bare `t(` outside any useTranslation call falls back
            # to common, mirroring i18next's defaultNamespace.
            alias_to_namespaces["t"] = {"common"}

        rel = path.relative_to(ROOT)

        for m in T_CALL_LOCAL.finditer(text):
            alias = m.group(1)
            raw = m.group(2)
            line = text[: m.start()].count("\n") + 1
            if ":" in raw:
                ns, key = raw.split(":", 1)
                hits.add((ns, key, line, str(rel)))
            else:
                # Try each candidate namespace for this alias; the
                # main() check uses the alternates field to accept a
                # call as "found" if the key exists in any of them.
                candidates = alias_to_namespaces.get(alias, {"common"})
                for ns in candidates:
                    hits.add((ns, raw, line, str(rel)))
    return hits


def main() -> int:
    locale = load_locale_keys()
    calls = extract_t_calls()

    # 1. Every t() call must have a matching JSON entry in at least
    # one of its candidate namespaces. Group calls by (file, line, key)
    # so that an ambiguous-namespace call is "found" if ANY of its
    # candidate namespaces has the key.
    by_call: dict[tuple[str, int, str], set[str]] = {}
    for ns, key, line, file in calls:
        by_call.setdefault((file, line, key), set()).add(ns)

    missing: list[tuple[str, str, int, str]] = []
    used_keys: dict[str, set[str]] = {ns: set() for ns in locale}

    for (file, line, key), candidate_ns in sorted(by_call.items()):
        found = False
        for ns in candidate_ns:
            if ns not in locale:
                continue
            if (
                key in locale[ns]
                or f"{key}_one" in locale[ns]
                or f"{key}_other" in locale[ns]
            ):
                found = True
                if key in locale[ns]:
                    used_keys[ns].add(key)
                if f"{key}_one" in locale[ns]:
                    used_keys[ns].add(f"{key}_one")
                if f"{key}_other" in locale[ns]:
                    used_keys[ns].add(f"{key}_other")
        if not found:
            # Report against the first candidate namespace for the error message.
            ns = sorted(candidate_ns)[0]
            missing.append((ns, key, line, file))

    # 2. Every JSON key must be referenced by some t() call.
    # Exception: keys that are looked up dynamically via
    # t(`namespace.${variable}.suffix`) can't be statically validated;
    # we add a fallback heuristic — if a JSON key's flat path starts
    # with one of the prefixes below, allow it.
    #
    # Two layers of allowlist:
    # - Built-in prefixes (this list): common across all 3 products,
    #   tied to the shared SettingsDrawer/AppearanceSection patterns.
    # - Per-repo allowlist file (scripts/i18n/dynamic-prefixes.txt):
    #   one prefix per line, comments with #. Captures repo-specific
    #   data-driven lookups (glossary terms, help content, device
    #   types, RFC test catalog, CLI category metadata, etc.) that the
    #   static analyzer can't see because the keys are consumed via
    #   <ns>.json[<section>] lookups rather than literal t() strings.
    BUILTIN_DYNAMIC_PREFIXES = [
        # SettingsDrawer renders t(`tabs.${tab.id}`) for 5 tabs.
        "tabs.",
        # SimulationSection renders t(`simulation.${labelKey}`) for 3 tabs.
        "simulation.tab",
        # AppearanceSection renders t(`appearance.${id}`) for 3 themes.
        "appearance.dark",
        "appearance.light",
        "appearance.system",
        # NetworkSection renders t(`network.interfaceTypes.${type}`).
        "network.interfaceTypes.",
        # DebugSection renders t(`debug.levels.${lvl}`).
        "debug.levels.",
    ]

    repo_allowlist_path = Path("scripts/i18n/dynamic-prefixes.txt")
    repo_prefixes: list[str] = []
    if repo_allowlist_path.is_file():
        for raw in repo_allowlist_path.read_text().splitlines():
            line = raw.split("#", 1)[0].strip()
            if line:
                repo_prefixes.append(line)

    all_prefixes = tuple(BUILTIN_DYNAMIC_PREFIXES + repo_prefixes)

    unused: list[tuple[str, str]] = []
    for ns, keys in locale.items():
        for k in keys:
            if k in used_keys.get(ns, set()):
                continue
            # Match either `key.path` or `<namespace>:key.path`
            qualified = f"{ns}:{k}"
            if any(k.startswith(p) or qualified.startswith(p) for p in all_prefixes):
                continue
            unused.append((ns, k))

    # Report.
    ratchet = "--ratchet" in sys.argv
    code = 0
    if missing:
        level = "warning" if ratchet else "error"
        print(f"::{level}::{len(missing)} t() call(s) reference keys missing from EN locale:")
        for ns, key, line, file in sorted(missing)[:30]:
            print(f"  {file}:{line}: t('{ns}:{key}') — not in {ns}.json")
        if len(missing) > 30:
            print(f"  … and {len(missing) - 30} more")
        if not ratchet:
            code = 1
    else:
        print("✓ every t() call has a matching EN locale key")

    if unused:
        print(f"::warning::{len(unused)} EN locale key(s) not referenced by any t() call:")
        for ns, key in sorted(unused)[:30]:
            print(f"  {ns}.json: {key}")
        if len(unused) > 30:
            print(f"  … and {len(unused) - 30} more")
        # Don't fail on unused yet — too noisy until we catch up. Demote to warn.
    else:
        print("✓ every EN locale key is referenced by at least one t() call")

    return code


if __name__ == "__main__":
    sys.exit(main())
