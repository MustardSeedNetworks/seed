#!/usr/bin/env python3
"""Self-test for check-ui-generated-types.py: builds a throwaway tree and proves
the gate goes red for each failure class and green when the tree is clean."""

from __future__ import annotations

import importlib.util
import io
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("gate", HERE / "check-ui-generated-types.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)

GENERATED_TS = """export interface DiscoveredDevice {
  ip: string;
  snmpData?: SNMPFullData;
}
export interface SNMPFullData {
  sysName?: string;
}
"""


class Tree:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.src = self.root / "ui" / "src"
        (self.src / "types" / "generated").mkdir(parents=True)
        (self.root / "scripts").mkdir()
        (self.src / "types" / "generated" / "engine.ts").write_text(GENERATED_TS)
        self.baseline = self.root / "scripts" / "ui-generated-type-baseline.txt"

    def write(self, rel: str, text: str) -> None:
        path = self.src / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)

    def run(self) -> tuple[int, str]:
        out = io.StringIO()
        code = gate.run(self.root, self.baseline, out=out)
        return code, out.getvalue()


class GeneratedTypeGateTest(unittest.TestCase):
    def test_clean_tree_passes(self) -> None:
        t = Tree()
        t.write("hooks/useDevices.ts", "import type { DiscoveredDevice } from '../types/generated/engine';\n")
        code, out = t.run()
        self.assertEqual(code, 0, out)
        self.assertIn("0 shadows", out)

    def test_shadow_without_baseline_fails(self) -> None:
        t = Tree()
        t.write("hooks/useDevices.ts", "export interface DiscoveredDevice {\n  ip: string;\n}\n")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("ui/src/hooks/useDevices.ts:DiscoveredDevice", out)

    def test_baselined_shadow_with_reason_passes(self) -> None:
        t = Tree()
        t.write("hooks/useDevices.ts", "export interface DiscoveredDevice {\n  ip: string;\n}\n")
        t.baseline.write_text("ui/src/hooks/useDevices.ts:DiscoveredDevice # deliberate narrow view\n")
        code, out = t.run()
        self.assertEqual(code, 0, out)

    def test_baselined_shadow_without_reason_fails(self) -> None:
        t = Tree()
        t.write("hooks/useDevices.ts", "export interface DiscoveredDevice {\n  ip: string;\n}\n")
        t.baseline.write_text("ui/src/hooks/useDevices.ts:DiscoveredDevice\n")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("no `# reason`", out)

    def test_stale_baseline_entry_fails(self) -> None:
        t = Tree()
        t.baseline.write_text("ui/src/hooks/gone.ts:DiscoveredDevice # migrated away\n")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("no longer exist", out)

    def test_entries_are_keyed_on_name_not_line(self) -> None:
        """An unrelated edit above the declaration must not read as new + stale.

        scripts/ui-fetch-baseline.txt is line-anchored and seed#2513 hit exactly
        that: one moved site read as one new entry plus one stale one.
        """
        t = Tree()
        t.write("hooks/useDevices.ts", "export interface DiscoveredDevice {\n  ip: string;\n}\n")
        t.baseline.write_text("ui/src/hooks/useDevices.ts:DiscoveredDevice # deliberate narrow view\n")
        self.assertEqual(t.run()[0], 0)
        t.write("hooks/useDevices.ts", "// a new comment\n\nexport interface DiscoveredDevice {\n  ip: string;\n}\n")
        code, out = t.run()
        self.assertEqual(code, 0, out)

    def test_tests_stories_and_test_helpers_are_not_scanned(self) -> None:
        t = Tree()
        t.write("hooks/useDevices.test.ts", "interface DiscoveredDevice { ip: string }\n")
        t.write("hooks/useDevices.stories.tsx", "interface DiscoveredDevice { ip: string }\n")
        t.write("test/setup.ts", "interface DiscoveredDevice { ip: string }\n")
        code, out = t.run()
        self.assertEqual(code, 0, out)

    def test_non_shadowing_local_interface_is_ignored(self) -> None:
        t = Tree()
        t.write("hooks/useDevices.ts", "interface GroupedDevices {\n  routers: string[];\n}\n")
        code, out = t.run()
        self.assertEqual(code, 0, out)


if __name__ == "__main__":
    unittest.main(verbosity=2)
