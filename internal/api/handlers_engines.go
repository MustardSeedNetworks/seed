package api

// /api/v1/engines — Stage A5.8 + #1383 enhancement. Read-only admin
// endpoint listing every long-running subsystem registered with
// services.Engines (probe, retention, snmp-poller, the 4 topology
// reconcilers, the 2 alert pipelines, plus any opt-in listeners).
//
//   GET /api/v1/engines
//     -> { count: N, "engines": [{
//            "name": "...",
//            "state": "ok" | "degraded" | "stopped",
//            "lastTickAt": "RFC3339" | "",
//            "lastError": "" | "...",
//            "inflight": N
//          }, ...] }
//
// Engines opt into rich status by implementing engine.Reporter. Those
// that don't get a default {state: "ok"} payload — backward-compatible
// with the V1.0 wire format and avoids forcing every engine to grow a
// Status() method before operators see any signal.

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/engine"
)

func (s *Server) handleEngines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.services == nil || s.services.Engines == nil {
		writeJSON(w, r, map[string]any{jsonKeyCount: 0, "engines": []any{}})
		return
	}
	engines := s.services.Engines.Engines()
	out := make([]map[string]any, 0, len(engines))
	for _, e := range engines {
		out = append(out, encodeEngineEntry(e))
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(out),
		"engines":    out,
	})
}

// encodeEngineEntry flattens one engine into the wire shape. Engines
// that implement engine.Reporter get their Status() surfaced; engines
// that don't get StatusOK() so the response is uniform.
func encodeEngineEntry(e engine.Engine) map[string]any {
	status := engine.StatusOK()
	if reporter, ok := e.(engine.Reporter); ok {
		status = reporter.Status()
		if status.State == "" {
			status.State = engine.StateOK
		}
	}
	return map[string]any{
		jsonKeyName:  e.Name(),
		"state":      status.State,
		"lastTickAt": formatTime(status.LastTickAt),
		"lastError":  status.LastError,
		"inflight":   status.Inflight,
	}
}
