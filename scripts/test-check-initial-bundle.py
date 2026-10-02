#!/usr/bin/env python3
"""Self-test for check-initial-bundle.py: builds a throwaway dist and proves the
gate counts every first-load chunk, skips vendor chunks and lazy chunks, and
goes red over budget."""

from __future__ import annotations

import importlib.util
import io
import os
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("gate", HERE / "check-initial-bundle.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)

INDEX = """<!doctype html><html><head>
<script type="module" crossorigin src="/assets/index-a.js"></script>
<link rel="modulepreload" crossorigin href="/assets/vendor-react-b.js">
<link rel="modulepreload" crossorigin href="/assets/shared-c.js">
<link rel="stylesheet" crossorigin href="/assets/index-d.css">
</head></html>"""


class Dist:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        (self.root / "assets").mkdir()
        (self.root / "index.html").write_text(INDEX)
        # Random bytes do not compress, so gzip size tracks the byte count.
        for name, size in {
            "index-a.js": 400,
            "vendor-react-b.js": 5000,
            "shared-c.js": 300,
            "LazyPage-e.js": 5000,
        }.items():
            (self.root / "assets" / name).write_bytes(os.urandom(size))

    def run(self, budget: int) -> tuple[int, str]:
        out = io.StringIO()
        return gate.main(dist=self.root, budget=budget, out=out), out.getvalue()


class InitialBundleGate(unittest.TestCase):
    def setUp(self) -> None:
        self.dist = Dist()
        self.addCleanup(self.dist.tmp.cleanup)

    def test_counts_preloaded_app_chunks_not_just_the_entry(self) -> None:
        code, out = self.dist.run(budget=500)
        self.assertEqual(code, 1, out)

    def test_vendor_and_lazy_chunks_are_outside_the_budget(self) -> None:
        code, out = self.dist.run(budget=1000)
        self.assertEqual(code, 0, out)
        self.assertIn("vendor-react-b.js", out)
        self.assertNotIn("LazyPage-e.js", out)

    def test_missing_build_is_its_own_failure(self) -> None:
        (self.dist.root / "index.html").unlink()
        code, _ = self.dist.run(budget=1000)
        self.assertEqual(code, 2)


if __name__ == "__main__":
    unittest.main()
