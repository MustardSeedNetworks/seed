#!/usr/bin/env python3
"""Self-test for check-feature-gate-parity.py against a throwaway tree."""

from __future__ import annotations

import importlib.util
import io
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("gate_parity", HERE / "check-feature-gate-parity.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)

POLICY_GO = '''package license
func starterFeatures() []string {
	return []string{
		"export_csv_json",
	}
}
func proFeatures() []string {
	pro := []string{
		// "commented_out" is not sold
		"path_analysis",
		"sso",
	}
	return append(starterFeatures(), pro...)
}
'''

ROUTES_GO = '''package api
var routes = []apiRoute{
	{path: "/path/path", feature: "path_analysis"},
	{path: "/reports", feature: "export_csv_json"},
}
'''

CATALOG_TS = """export const FEATURE_CATALOG = {
  path_analysis: { tier: 'Pro' },
  export_csv_json: { tier: 'Starter' },
} as const satisfies Record<string, CatalogEntry>;
"""


class Tree:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        for rel, text in (
            (gate.POLICY, POLICY_GO),
            (gate.ROUTES, ROUTES_GO),
            (gate.CATALOG, CATALOG_TS),
        ):
            path = self.root / rel
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8")

    def write(self, rel: str, text: str) -> None:
        (self.root / rel).write_text(text, encoding="utf-8")

    def run(self) -> tuple[int, str]:
        out = io.StringIO()
        return gate.run(self.root, out=out), out.getvalue()


class FeatureGateParityTest(unittest.TestCase):
    def test_agreeing_tree_passes(self) -> None:
        t = Tree()
        code, out = t.run()
        self.assertEqual(code, 0, out)
        self.assertIn("2 UI-gated features", out)

    def test_ui_gate_on_an_unenforced_feature_fails(self) -> None:
        t = Tree()
        t.write(gate.CATALOG, CATALOG_TS.replace("} as const", "  sso: { tier: 'Pro' },\n} as const"))
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("sso", out)
        self.assertIn("no route", out)

    def test_wrong_tier_fails(self) -> None:
        t = Tree()
        t.write(gate.CATALOG, CATALOG_TS.replace("export_csv_json: { tier: 'Starter' }", "export_csv_json: { tier: 'Pro' }"))
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("UI advertises Pro", out)

    def test_feature_missing_from_policy_fails(self) -> None:
        t = Tree()
        t.write(gate.POLICY, POLICY_GO.replace('\t\t"path_analysis",\n', ""))
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("absent from the catalog", out)

    def test_pro_reexport_of_a_starter_feature_reads_as_starter(self) -> None:
        """proFeatures() appends starterFeatures(), so membership in both is the
        normal case and must not read as Pro — the tier a customer buys is the
        lowest one that grants the feature."""
        t = Tree()
        self.assertEqual(gate.policy_tiers(t.root)["export_csv_json"], "Starter")

    def test_a_commented_out_feature_is_not_sold(self) -> None:
        t = Tree()
        self.assertNotIn("commented_out", gate.policy_tiers(t.root))


if __name__ == "__main__":
    sys.exit(unittest.main(verbosity=1))
