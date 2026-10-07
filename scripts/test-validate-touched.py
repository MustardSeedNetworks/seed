#!/usr/bin/env python3
"""Self-test for validate-touched.py's selection logic, on a synthetic module."""

from __future__ import annotations

import importlib.util
import re
import sys
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("validate_touched", HERE / "validate-touched.py")
vt = importlib.util.module_from_spec(spec)
assert spec.loader is not None
sys.modules[spec.name] = vt  # dataclasses resolve annotations through it
spec.loader.exec_module(vt)

M = "example.com/m"
EXCLUDE = re.compile(r"/cmd/|/ui$")


def pkg(rel: str, imports: tuple[str, ...] = (), test_imports: tuple[str, ...] = ()) -> vt.Package:
    return vt.Package(
        import_path=f"{M}/{rel}",
        rel_dir=rel,
        imports=frozenset(f"{M}/{i}" for i in imports),
        test_imports=frozenset(f"{M}/{i}" for i in test_imports),
    )


# leaf <- mid <- top <- cmd/app; helper's tests import mid; other is unrelated.
PACKAGES = [
    pkg("internal/leaf"),
    pkg("internal/mid", imports=("internal/leaf",)),
    pkg("internal/top", imports=("internal/mid",)),
    pkg("internal/helper", test_imports=("internal/mid",)),
    pkg("internal/helperuser", test_imports=("internal/helper",)),
    pkg("internal/other"),
    pkg("cmd/app", imports=("internal/top",)),
]


def plan(changed: list[str], deleted: frozenset[str] = frozenset()) -> list[str]:
    steps = vt.plan(changed, PACKAGES, EXCLUDE, "golangci-lint", "0.0.0", deleted)
    return [step.render() for step in steps]


def go_test_line(lines: list[str]) -> str:
    return next(line for line in lines if line.startswith("go test"))


class Selection(unittest.TestCase):
    def test_leaf_change_tests_reverse_dependencies(self) -> None:
        lines = plan(["internal/leaf/leaf.go"])
        self.assertIn("golangci-lint run ./internal/leaf", lines)
        tested = go_test_line(lines).split()[3:]
        self.assertEqual(
            tested,
            [f"{M}/internal/helper", f"{M}/internal/leaf", f"{M}/internal/mid", f"{M}/internal/top"],
        )

    def test_test_imports_do_not_propagate(self) -> None:
        tested = go_test_line(plan(["internal/mid/mid.go"]))
        self.assertIn(f"{M}/internal/helper", tested)
        self.assertNotIn("helperuser", tested)
        self.assertNotIn("internal/other", tested)

    def test_make_test_exclusions_apply(self) -> None:
        lines = plan(["internal/leaf/leaf.go"])
        self.assertNotIn("cmd/app", go_test_line(lines))

    def test_cmd_change_lints_but_runs_no_excluded_tests(self) -> None:
        lines = plan(["cmd/app/main.go"])
        self.assertIn("golangci-lint run ./cmd/app", lines)
        self.assertFalse(any(line.startswith("go test") for line in lines))

    def test_testdata_belongs_to_its_package(self) -> None:
        lines = plan(["internal/other/testdata/golden.json"])
        self.assertEqual(go_test_line(lines).split()[3:], [f"{M}/internal/other"])

    def test_go_mod_selects_everything(self) -> None:
        tested = go_test_line(plan(["go.mod"])).split()[3:]
        self.assertEqual(len(tested), len(PACKAGES) - 1)

    def test_docs_only_runs_no_go(self) -> None:
        lines = plan(["docs/guide.md"])
        self.assertFalse(any(line.startswith(("go ", "golangci-lint")) for line in lines))
        self.assertIn("npx --yes markdownlint-cli2@0.0.0 docs/guide.md", lines)
        self.assertIn("python3 scripts/check-banned-vocabulary.py", lines)
        self.assertNotIn("./scripts/check-route-policy.sh", lines)

    def test_api_change_runs_api_gates_and_windows_vet(self) -> None:
        lines = vt.plan(
            ["internal/api/routes.go"],
            [pkg("internal/api")],
            EXCLUDE,
            "golangci-lint",
            "0.0.0",
        )
        rendered = [s.render() for s in lines]
        self.assertIn("GOOS=windows CGO_ENABLED=0 go vet ./internal/api/...", rendered)
        self.assertIn("./scripts/check-route-policy.sh", rendered)
        self.assertIn("./scripts/check-error-shape.sh", rendered)

    def test_ui_change_runs_related_vitest_and_biome(self) -> None:
        lines = plan(["ui/src/App.tsx"])
        self.assertIn("(cd ui && npx @biomejs/biome check src/App.tsx)", lines)
        self.assertTrue(any("vitest.mjs related --run --passWithNoTests src/App.tsx" in line for line in lines))
        self.assertIn("(cd ui && ../scripts/check-token-discipline.sh)", lines)

    def test_deleted_files_select_gates_but_skip_per_file_tools(self) -> None:
        lines = plan(["ui/src/Gone.tsx"], deleted=frozenset({"ui/src/Gone.tsx"}))
        self.assertFalse(any("Gone.tsx" in line for line in lines))
        self.assertIn("./scripts/check-tailwind-classes.sh", lines)

    def test_editing_a_gate_runs_that_gate(self) -> None:
        lines = plan(["scripts/check-json-casing.sh"])
        self.assertIn("./scripts/check-json-casing.sh", lines)
        lines = plan(["scripts/test-check-feature-catalog.py"])
        self.assertIn("python3 scripts/check-feature-catalog.py", lines)

    def test_make_fragments_set_every_value_read(self) -> None:
        self.assertRegex(vt.make_var("lint.mk", "GOLANGCI_LINT_VERSION"), r"^v\d+\.\d+\.\d+$")
        self.assertRegex(vt.make_var("lint.mk", "MARKDOWNLINT_CLI2_VERSION"), r"^\d+\.\d+\.\d+$")
        exclude = re.compile(vt.make_var("test.mk", "TEST_PKG_EXCLUDE"))
        self.assertTrue(exclude.search("github.com/MustardSeedNetworks/seed/internal/api/ui"))
        self.assertFalse(exclude.search("github.com/MustardSeedNetworks/seed/internal/uix"))

    def test_every_gate_script_exists(self) -> None:
        root = HERE.parent
        for gate in vt.GATES:
            for path in gate.own_files():
                self.assertTrue((root / path).is_file(), f"{gate.name}: {path} is missing")


if __name__ == "__main__":
    unittest.main()
