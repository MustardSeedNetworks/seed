package api

// /api/v1/engines — Stage A5.8 + #1383 enhancement. Read-only admin
// endpoint listing every long-running subsystem registered with the
// engine registry (probe, retention, snmp-poller, the 4 topology
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
// The status logic lives in the engine-status use-case (s.engineStatus,
// ADR-0020): engines that implement engine.Reporter contribute their
// Status(), engines that don't default to {state: "ok"}. This handler
// keeps only transport concerns — encoding the domain snapshot to the
// V1.0 wire shape. No raw engine-registry reference remains in transport.

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/engine/status"
)

func (s *Server) handleEngines(w http.ResponseWriter, r *http.Request) {
	statuses := s.engineStatus.List()
	out := make([]map[string]any, 0, len(statuses))
	for _, st := range statuses {
		out = append(out, encodeEngineEntry(st))
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(out),
		"engines":    out,
	})
}

// encodeEngineEntry flattens one engine-status snapshot into the wire
// shape. The status determination (Reporter vs StatusOK fallback) happens
// in the use-case; this only shapes the response.
func encodeEngineEntry(st status.EngineStatus) map[string]any {
	return map[string]any{
		jsonKeyName:  st.Name,
		"state":      st.State,
		"lastTickAt": formatTime(st.LastTickAt),
		"lastError":  st.LastError,
		"inflight":   st.Inflight,
	}
}
