package api

// /api/v1/engines — Stage A5.8. Read-only admin endpoint listing
// every long-running subsystem registered with services.Engines
// (probe, retention, snmp-poller, the 4 topology reconcilers, the
// 2 alert pipelines, plus any opt-in listeners).
//
//   GET /api/v1/engines
//     -> { "count": N, "engines": [{ "name": "..." }, ...] }
//
// The endpoint deliberately does NOT expose per-engine status or
// last-tick timestamps for V1.0 — the engine.Engine interface
// only carries Name + Start + Stop, and inventing a richer status
// surface would require touching every engine implementation. A
// future iteration can add an optional Status() method and a
// Reporter interface; until then, "is this engine in the registry"
// is the useful signal operators need.

import (
	"net/http"
)

func (s *Server) handleEngines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.services == nil || s.services.Engines == nil {
		writeJSON(w, r, map[string]any{"count": 0, "engines": []any{}})
		return
	}
	engines := s.services.Engines.Engines()
	out := make([]map[string]any, 0, len(engines))
	for _, e := range engines {
		out = append(out, map[string]any{"name": e.Name()})
	}
	writeJSON(w, r, map[string]any{
		"count":   len(out),
		"engines": out,
	})
}
