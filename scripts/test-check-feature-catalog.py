#!/usr/bin/env python3
"""Self-test for check-feature-catalog.py against a throwaway tree."""

from __future__ import annotations

import importlib.util
import io
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("catalog_gate", HERE / "check-feature-catalog.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)

POLICY_GO = '''package license
const productName = "seed"
func starterFeatures() []string {
	return []string{
		"export_csv_json",
		"topology_local", // built, ungated
	}
}
func proFeatures() []string {
	pro := []string{
		"sso",
		// "commented_out" is not sold
		"white_label",
	}
	return append(starterFeatures(), pro...)
}
'''


class Tree:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        (self.root / "internal" / "license").mkdir(parents=True)
        (self.root / "internal" / "api").mkdir(parents=True)
        (self.root / "scripts").mkdir()
        (self.root / "internal" / "license" / "policy.go").write_text(POLICY_GO)
        (self.root / "internal" / "api" / "routes.go").write_text(
            'var r = []apiRoute{{feature: "sso"}}\nfunc x() { if mgr.HasFeature("export_csv_json") {} }\n')
        (self.root / "internal" / "api" / "routes_test.go").write_text('requireFeature("white_label")')
        self.baseline = self.root / "scripts" / "feature-catalog-baseline.txt"

    def run(self, update: bool = False) -> tuple[int, str]:
        out = io.StringIO()
        return gate.run(self.root, self.baseline, update=update, out=out), out.getvalue()


class FeatureCatalogGateTest(unittest.TestCase):
    def test_catalog_parsing_ignores_comments_and_non_catalog_strings(self) -> None:
        t = Tree()
        self.assertEqual(gate.catalog(t.root), {"export_csv_json", "topology_local", "sso", "white_label"})

    def test_unbacked_without_baseline_fails(self) -> None:
        t = Tree()
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("topology_local", out)
        self.assertIn("white_label", out)  # a test-only reference does not count
        self.assertNotIn("  sso", out)

    def test_update_then_clean_run_passes(self) -> None:
        t = Tree()
        self.assertEqual(t.run(update=True)[0], 0)
        code, out = t.run()
        self.assertEqual(code, 0, out)
        self.assertIn("4 sold features, 2 backed, 2 unbacked (2 baselined)", out)

    def test_new_unbacked_feature_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        p = t.root / "internal" / "license" / "policy.go"
        p.write_text(p.read_text().replace('"white_label",', '"white_label",\n\t\t"bgp_monitoring",'))
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("bgp_monitoring", out)

    def test_stale_baseline_entry_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.root / "internal" / "api" / "gate.go").write_text('requireFeature("topology_local")')
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("remove them", out)
        self.assertIn("topology_local", out)


if __name__ == "__main__":
    sys.exit(unittest.main(verbosity=1))
