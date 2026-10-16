#!/usr/bin/env python3
"""Self-tests for the HIBP CodeQL SARIF filter."""

from __future__ import annotations

import importlib.util
import json
import subprocess
import tempfile
import unittest
from pathlib import Path

FILTER = Path(__file__).with_name("codeql-drop-hibp-weak-hash.py")


def load_filter():
    spec = importlib.util.spec_from_file_location("codeql_drop_hibp_weak_hash", FILTER)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load {FILTER}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


FILTER_MOD = load_filter()


def result(rule_id: str, uri: str, rule_key: str = "ruleId") -> dict:
    body = {
        "message": {"text": "weak hash"},
        "locations": [
            {"physicalLocation": {"artifactLocation": {"uri": uri}}},
        ],
    }
    if rule_key == "ruleId":
        body["ruleId"] = rule_id
    else:
        body["rule"] = {"id": rule_id}
    return body


class DropHibpWeakHashTest(unittest.TestCase):
    def test_drops_hibp_weak_hash_only(self) -> None:
        log = {
            "runs": [
                {
                    "results": [
                        result("go/weak-sensitive-data-hashing", "internal/auth/hibp.go"),
                        result("go/weak-sensitive-data-hashing", "internal/auth/password.go"),
                        result("go/clear-text-logging", "internal/auth/hibp.go"),
                    ]
                }
            ]
        }
        dropped = FILTER_MOD.drop_hibp_weak_hash(log)
        self.assertEqual(1, dropped)
        kept = log["runs"][0]["results"]
        self.assertEqual(
            ["internal/auth/password.go", "internal/auth/hibp.go"],
            [item["locations"][0]["physicalLocation"]["artifactLocation"]["uri"] for item in kept],
        )
        self.assertEqual("go/clear-text-logging", kept[1]["ruleId"])

    def test_absolute_and_nested_rule_id(self) -> None:
        log = {
            "runs": [
                {
                    "results": [
                        result(
                            "go/weak-sensitive-data-hashing",
                            "/home/runner/work/seed/seed/internal/auth/hibp.go",
                            rule_key="rule",
                        ),
                        result(
                            "go/weak-sensitive-data-hashing",
                            ".\\internal\\auth\\hibp.go",
                        ),
                    ]
                }
            ]
        }
        self.assertEqual(2, FILTER_MOD.drop_hibp_weak_hash(log))
        self.assertEqual([], log["runs"][0]["results"])

    def test_cli_rewrites_the_file(self) -> None:
        sarif = {
            "runs": [
                {
                    "results": [
                        result("go/weak-sensitive-data-hashing", "internal/auth/hibp.go"),
                        result("go/weak-sensitive-data-hashing", "internal/other.go"),
                    ]
                }
            ]
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "go.sarif"
            path.write_text(json.dumps(sarif), encoding="utf-8")
            completed = subprocess.run(
                ["python3", str(FILTER), str(path)],
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(0, completed.returncode, completed.stderr)
            self.assertIn("dropped 1", completed.stdout)
            rewritten = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(1, len(rewritten["runs"][0]["results"]))
            self.assertEqual(
                "internal/other.go",
                rewritten["runs"][0]["results"][0]["locations"][0]["physicalLocation"][
                    "artifactLocation"
                ]["uri"],
            )


if __name__ == "__main__":
    unittest.main()
