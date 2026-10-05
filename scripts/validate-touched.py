#!/usr/bin/env python3
"""Validate only what the branch's changes can affect (`make validate-touched`).

This is the inner loop. It lints the changed Go packages, runs `go test -race`
on them and on every in-module package whose test binary depends on them,
runs Vitest `related` and Biome on the changed UI files, markdownlint on the
changed Markdown, and every CI gate in scripts/ whose inputs changed. Each
command is printed before it runs. The full `make test` still runs once before
the PR: a gate's input list here is a judgement, CI's is the whole tree.

The changed set is everything between the merge base with BASE (default
origin/main) and the working tree, plus untracked files.
"""

from __future__ import annotations

import argparse
import fnmatch
import json
import os
import re
import shlex
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parent.parent


@dataclass(frozen=True)
class Step:
    argv: tuple[str, ...]
    cwd: str = "."
    env: tuple[tuple[str, str], ...] = ()

    def render(self) -> str:
        prefix = "".join(f"{k}={v} " for k, v in self.env)
        where = "" if self.cwd == "." else f"(cd {self.cwd} && "
        return f"{where}{prefix}{shlex.join(self.argv)}{')' if where else ''}"


@dataclass(frozen=True)
class Gate:
    name: str
    steps: tuple[Step, ...]
    # fnmatch patterns over repo-relative paths; `*` crosses directories. The
    # gate's own scripts are inputs too and need not be listed.
    inputs: tuple[str, ...]

    def own_files(self) -> set[str]:
        files = set()
        for step in self.steps:
            for arg in step.argv:
                if "scripts/" in arg:
                    files.add(os.path.normpath(os.path.join(step.cwd, arg)))
        return files

    def touched_by(self, changed: list[str]) -> bool:
        own = self.own_files()
        return any(p in own or any(fnmatch.fnmatchcase(p, pat) for pat in self.inputs) for p in changed)


def _py(script: str) -> Step:
    return Step(("python3", f"scripts/{script}"))


def _sh(script: str) -> Step:
    return Step((f"./scripts/{script}",))


def _py_pair(gate: str) -> tuple[Step, ...]:
    return (_py(f"test-check-{gate}.py"), _py(f"check-{gate}.py"))


# The gates ci.yml runs on every PR, as ci.yml runs them. Gates that need a
# built UI (initial-bundle, webkit-libsoup), the network (release-changelog
# --pending) or a daemon are left to CI.
GATES: tuple[Gate, ...] = (
    Gate("banned-vocabulary", _py_pair("banned-vocabulary"), ("*",)),
    Gate("product-name", _py_pair("product-name"), ("*",)),
    Gate("component-variants", _py_pair("component-variants"), ("ui/*",)),
    Gate("error-shape", (_sh("check-error-shape.sh"),), ("internal/api/*",)),
    Gate(
        "feature-catalog",
        _py_pair("feature-catalog"),
        ("internal/*.go", "cmd/*.go", "scripts/feature-catalog-baseline.txt"),
    ),
    Gate(
        "feature-gate-parity",
        _py_pair("feature-gate-parity"),
        ("internal/api/*", "internal/license/*", "ui/src/*"),
    ),
    Gate(
        "file-size",
        (_sh("check-file-size.sh"),),
        ("*.go", "*.ts", "*.tsx", "scripts/file-size-baseline.txt"),
    ),
    Gate("filename-policy", (_sh("check-filename-policy.sh"),), ("internal/*",)),
    Gate(
        "hardware-matrix",
        (_sh("check-hardware-matrix.sh"),),
        ("HARDWARE.md", "cmd/seed-hardware/*", "internal/capabilities/*"),
    ),
    Gate("json-casing", (_sh("check-json-casing.sh"),), ("internal/*.go",)),
    Gate("output-escaping", (_sh("check-output-escaping.sh"),), ("internal/api/*", "ui/src/*")),
    Gate(
        "package-reachability",
        (_sh("check-package-reachability.sh"),),
        ("*.go", "go.mod", "scripts/package-reachability-baseline.txt"),
    ),
    Gate(
        "route-consumers",
        _py_pair("route-consumers"),
        ("internal/api/*", "internal/cliclient/*", "cmd/*", "ui/src/*", "scripts/route-consumer-baseline.txt"),
    ),
    Gate("route-policy", (_sh("check-route-policy.sh"),), ("internal/api/*",)),
    Gate(
        "schema-drift",
        (_sh("check-schema-drift.sh"),),
        ("internal/*.go", "cmd/seed-schema/*", "docs/schemas/api/*"),
    ),
    Gate("single-writer", (_sh("check-single-writer.sh"),), ("internal/*.go", "cmd/*.go")),
    Gate("snmp-credential-surface", (_sh("check-snmp-credential-surface.sh"),), ("internal/*", "configs/*")),
    Gate("tailwind-classes", (_sh("check-tailwind-classes.sh"),), ("ui/src/*",)),
    Gate("token-discipline", (Step(("../scripts/check-token-discipline.sh",), cwd="ui"),), ("ui/*",)),
    Gate("tsconfig-flags", (_py("check-tsconfig-flags.py"),), ("ui/tsconfig*",)),
    Gate(
        "types-drift",
        (_sh("check-types-drift.sh"),),
        ("docs/schemas/api/*", "ui/src/types/generated/*", "ui/scripts/gen-types.mjs"),
    ),
    Gate("ui-fetch", (_py("check-ui-fetch.py"),), ("ui/src/*", "scripts/ui-fetch-baseline.txt")),
    Gate(
        "ui-generated-types",
        _py_pair("ui-generated-types"),
        ("ui/src/*", "docs/schemas/api/*", "scripts/ui-generated-type-baseline.txt"),
    ),
    Gate(
        "help-i18n",
        (
            Step(("node", "--test", "scripts/check-help-i18n.test.ts")),
            Step(("node", "scripts/check-help-i18n.ts")),
        ),
        ("internal/i18n/locales/*", "ui/src/*"),
    ),
    Gate("validate-touched", (_py("test-validate-touched.py"),), ("scripts/validate-touched.py", "mk/*", "Makefile")),
    Gate(
        "release-workflow-contract",
        (_sh("test-check-release-workflow-contract.sh"),),
        (".github/*",),
    ),
)


@dataclass(frozen=True)
class Package:
    import_path: str
    rel_dir: str
    imports: frozenset[str]
    test_imports: frozenset[str]


def owning_package(path: str, by_dir: dict[str, Package]) -> Package | None:
    """The package whose build or tests can read `path`: its nearest enclosing
    package directory, so testdata, golden files and go:embed inputs count."""
    for parent in PurePosixPath(path).parents:
        pkg = by_dir.get(str(parent))
        if pkg is not None:
            return pkg
    return None


def changed_packages(changed: list[str], packages: list[Package]) -> set[str]:
    if any(p in ("go.mod", "go.sum") for p in changed):
        return {pkg.import_path for pkg in packages}
    by_dir = {pkg.rel_dir: pkg for pkg in packages}
    hit = set()
    for path in changed:
        pkg = owning_package(path, by_dir)
        if pkg is not None:
            hit.add(pkg.import_path)
    return hit


def affected_packages(changed: set[str], packages: list[Package]) -> set[str]:
    """Every package whose test binary links a changed package: the reverse
    closure over production imports, plus packages whose tests import a member
    of that closure. Test imports do not propagate further, since no other
    package compiles a package's tests."""
    importers: dict[str, set[str]] = {}
    for pkg in packages:
        for dep in pkg.imports:
            importers.setdefault(dep, set()).add(pkg.import_path)
    closure = set(changed)
    frontier = list(changed)
    while frontier:
        for importer in importers.get(frontier.pop(), ()):
            if importer not in closure:
                closure.add(importer)
                frontier.append(importer)
    return closure | {pkg.import_path for pkg in packages if pkg.test_imports & closure}


def plan(
    changed: list[str],
    packages: list[Package],
    test_exclude: re.Pattern[str],
    golangci_lint: str,
    markdownlint_version: str,
    deleted: frozenset[str] = frozenset(),
) -> list[Step]:
    """A deleted file still selects its package and gates, but is not handed to
    a per-file tool."""
    steps: list[Step] = []
    rel = {pkg.import_path: pkg.rel_dir for pkg in packages}

    lint_pkgs = sorted(changed_packages(changed, packages))
    if lint_pkgs:
        steps.append(Step((golangci_lint, "run", *(f"./{rel[p]}" for p in lint_pkgs))))
        if any(rel[p] == "internal/api" or rel[p].startswith("internal/api/") for p in lint_pkgs):
            steps.append(
                Step(("go", "vet", "./internal/api/..."), env=(("GOOS", "windows"), ("CGO_ENABLED", "0")))
            )
        test_pkgs = sorted(
            p for p in affected_packages(set(lint_pkgs), packages) if not test_exclude.search(p)
        )
        if test_pkgs:
            steps.append(Step(("go", "test", "-race", *test_pkgs)))

    present = [p for p in changed if p not in deleted]
    ui_files = [p.removeprefix("ui/") for p in present if p.startswith("ui/") and p.endswith((".ts", ".tsx"))]
    if ui_files:
        steps.append(Step(("npx", "@biomejs/biome", "check", *ui_files), cwd="ui"))
        steps.append(
            Step(
                (
                    "node",
                    "--disable-warning=DEP0205",
                    "./node_modules/vitest/vitest.mjs",
                    "related",
                    "--run",
                    "--passWithNoTests",
                    *ui_files,
                ),
                cwd="ui",
            )
        )

    md_files = [p for p in present if p.endswith(".md")]
    if md_files:
        steps.append(Step(("npx", "--yes", f"markdownlint-cli2@{markdownlint_version}", *md_files)))

    for gate in GATES:
        if gate.touched_by(changed):
            steps.extend(gate.steps)
    return steps


def git_lines(*args: str) -> list[str]:
    out = subprocess.run(["git", *args], cwd=ROOT, check=True, capture_output=True, text=True).stdout
    return [line for line in out.splitlines() if line]


def changed_files(base: str) -> list[str]:
    merge_base = git_lines("merge-base", base, "HEAD")[0]
    paths = set(git_lines("diff", "--name-only", merge_base)) | set(git_lines("ls-files", "--others", "--exclude-standard"))
    return sorted(paths)


def go_packages() -> list[Package]:
    out = subprocess.run(
        ["go", "list", "-e", "-json=ImportPath,Dir,Imports,TestImports,XTestImports", "./..."],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout
    decoder = json.JSONDecoder()
    packages, pos = [], 0
    while (pos := _skip_ws(out, pos)) < len(out):
        obj, pos = decoder.raw_decode(out, pos)
        packages.append(
            Package(
                import_path=obj["ImportPath"],
                rel_dir=Path(obj["Dir"]).relative_to(ROOT).as_posix(),
                imports=frozenset(obj.get("Imports", ())),
                test_imports=frozenset([*obj.get("TestImports", ()), *obj.get("XTestImports", ())]),
            )
        )
    return packages


def _skip_ws(text: str, pos: int) -> int:
    while pos < len(text) and text[pos].isspace():
        pos += 1
    return pos


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--base", default=os.environ.get("BASE", "origin/main"))
    parser.add_argument("--dry-run", action="store_true", help="print the commands without running them")
    # Pinned by the Makefile, so the inner loop cannot pass what CI rejects.
    parser.add_argument("--golangci-lint", required=True, help="path to the pinned golangci-lint")
    parser.add_argument("--markdownlint-version", required=True)
    parser.add_argument("--test-exclude", required=True, help="regexp of packages `make test` skips")
    args = parser.parse_args()

    changed = changed_files(args.base)
    print(f"{len(changed)} changed file(s) since the merge base with {args.base}")
    steps = plan(
        changed,
        go_packages(),
        re.compile(args.test_exclude),
        args.golangci_lint,
        args.markdownlint_version,
        frozenset(p for p in changed if not (ROOT / p).exists()),
    )
    if not steps:
        print("nothing to validate")
        return 0

    if args.dry_run:
        for step in steps:
            print(f"+ {step.render()}")
        return 0
    failed = []
    for step in steps:
        print(f"+ {step.render()}", flush=True)
        result = subprocess.run(step.argv, cwd=ROOT / step.cwd, env={**os.environ, **dict(step.env)})
        if result.returncode != 0:
            failed.append(step.render())
    if failed:
        print(f"\n{len(failed)} of {len(steps)} command(s) failed:", file=sys.stderr)
        for cmd in failed:
            print(f"  {cmd}", file=sys.stderr)
        return 1
    print(f"\n{len(steps)} command(s) passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
