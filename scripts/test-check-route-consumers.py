#!/usr/bin/env python3
"""Self-test for check-route-consumers.py: builds a throwaway tree and proves
the gate goes red for each failure class and green when the tree is clean."""

from __future__ import annotations

import importlib.util
import io
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("gate", HERE / "check-route-consumers.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)

ROUTES_GO = '''package api
func (s *Server) registerRoutes() {
	s.registerAll(mux, []apiRoute{
		{path: "/api/v1/config", handler: s.handleConfig},
		{path: APIVersionPrefix + "/profiles/", handler: s.handleProfiles},
		{path: "GET /api/v1/updates/check", handler: s.handleUpdateCheck},
		{path: "/api/v1/orphan", handler: s.handleOrphan},
	})
}
'''
SESSIONS_GO = '''package api
func (s *Server) sessionResourceHandler(resource string) (sessionHandler, bool) {
	handlers := map[string]sessionHandler{
		"devices": s.handleSessionDevices,
	}
	return handlers[resource], true
}
'''


class Tree:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        (self.root / "internal" / "api").mkdir(parents=True)
        (self.root / "ui" / "src").mkdir(parents=True)
        (self.root / "scripts").mkdir()
        (self.root / "internal" / "api" / "routes.go").write_text(ROUTES_GO)
        (self.root / "internal" / "api" / "handlers_sessions.go").write_text(SESSIONS_GO)
        (self.root / "internal" / "api" / "routes_test.go").write_text('{path: "/api/v1/only-in-test"}')
        (self.root / "ui" / "src" / "client.ts").write_text(
            "const updates = '/api/v1/updates';\n"
            "api.get('/api/v1/config');\n"
            "api.post(`/api/v1/profiles/${encodeURIComponent(id)}/duplicate`);\n"
            "fetch(`${updates}/check?x=1`);\n"
            "new EventSource(`/api/v1/sessions/${sid}/devices`);\n"
            "// api.get('/api/v1/commented-out');\n"
            " * api.get('/api/v1/in-jsdoc');\n"
            "fetch(`${API_BASE}/api/v1/config?x=1`);\n"
            "import { api } from '../api/client';\n"
        )
        (self.root / "ui" / "src" / "client.test.ts").write_text("api.get('/api/v1/never-registered');")
        self.baseline = self.root / "scripts" / "route-consumer-baseline.txt"

    def run(self, update: bool = False) -> tuple[int, str]:
        out = io.StringIO()
        code = gate.run(self.root, self.baseline, update=update, out=out)
        return code, out.getvalue()


class RouteConsumerGateTest(unittest.TestCase):
    def test_comments_and_const_definitions_are_not_consumers(self) -> None:
        t = Tree()
        consumers = gate.load_consumers(t.root)
        self.assertNotIn("/api/v1/commented-out", consumers)
        self.assertNotIn("/api/v1/in-jsdoc", consumers)
        self.assertNotIn("/api/v1/updates", consumers)
        self.assertNotIn("/api/client", consumers)
        self.assertIn("/api/v1/updates/check", consumers)
        self.assertIn("/api/v1/config", consumers)

    def test_bare_const_use_and_per_file_const_precedence(self) -> None:
        t = Tree()
        (t.root / "ui" / "src" / "a.ts").write_text("const ENDPOINT = '/api/v1/config';\napi.get(ENDPOINT);\n")
        (t.root / "ui" / "src" / "b.ts").write_text("const ENDPOINT = '/api/v1/profiles';\napi.get(`${ENDPOINT}/x`);\n")
        consumers = gate.load_consumers(t.root)
        self.assertIn("ui/src/a.ts:2", consumers["/api/v1/config"])
        self.assertIn("/api/v1/profiles/x", consumers)
        self.assertNotIn("/api/v1/profiles", consumers)

    def test_base_url_prefix_is_not_a_404_and_does_not_consume(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.root / "ui" / "src" / "base.ts").write_text("const b = `${origin}/api/v1/updates`;\n")
        code, out = t.run()
        self.assertEqual(code, 0, out)  # not a 404 ...
        self.assertIn("orphan", t.baseline.read_text())  # ... and /api/v1/updates/check stays consumed only by its real caller

    def test_orphan_route_without_baseline_fails(self) -> None:
        t = Tree()
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("/api/v1/orphan", out)
        self.assertNotIn("/api/v1/only-in-test", out)

    def test_update_then_clean_run_passes(self) -> None:
        t = Tree()
        self.assertEqual(t.run(update=True)[0], 0)
        self.assertIn("/api/v1/orphan", t.baseline.read_text())
        code, out = t.run()
        self.assertEqual(code, 0, out)
        self.assertIn("5 routes", out)

    def test_consumer_without_route_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.root / "ui" / "src" / "hook.ts").write_text("api.post('/api/v1/profiles-switch');")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("/api/v1/profiles-switch", out)
        self.assertIn("live 404s", out)

    def test_known_404_is_baselined_and_goes_stale_when_fixed(self) -> None:
        t = Tree()
        (t.root / "ui" / "src" / "hook.ts").write_text("api.get('/api/v1/missing');")
        self.assertEqual(t.run()[0], 1)
        t.run(update=True)
        self.assertIn("404 /api/v1/missing", t.baseline.read_text())
        code, out = t.run()
        self.assertEqual(code, 0, out)
        (t.root / "ui" / "src" / "hook.ts").write_text("")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("404 /api/v1/missing", out)

    def test_prefix_route_matches_deeper_consumer(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.root / "ui" / "src" / "hook.ts").write_text("api.patch(`/api/v1/profiles/${id}/settings`);")
        code, out = t.run()
        self.assertEqual(code, 0, out)

    def test_stale_baseline_entry_fails(self) -> None:
        t = Tree()
        t.run(update=True)
        (t.root / "ui" / "src" / "hook.ts").write_text("api.get('/api/v1/orphan');")
        code, out = t.run()
        self.assertEqual(code, 1)
        self.assertIn("remove them", out)
        self.assertIn("/api/v1/orphan", out)

    def test_session_resource_map_and_method_prefixed_routes_are_routes(self) -> None:
        t = Tree()
        routes = gate.load_routes(t.root)
        self.assertIn("/api/v1/sessions/*/devices", routes)
        self.assertIn("/api/v1/updates/check", routes)
        self.assertIn("/api/v1/profiles/", routes)


if __name__ == "__main__":
    sys.exit(unittest.main(verbosity=1))
