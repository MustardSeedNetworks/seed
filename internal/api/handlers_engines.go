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

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// EnginesResponse is the GET /api/v1/engines body.
type EnginesResponse struct {
	Count   int           `json:"count"`
	Engines []EngineEntry `json:"engines"`
}

// EngineEntry is one engine's status. LastTickAt is RFC 3339, or "" for an
// engine that has not ticked yet.
type EngineEntry struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	LastTickAt string `json:"lastTickAt"`
	LastError  string `json:"lastError"`
	Inflight   int    `json:"inflight"`
}

func (s *Server) handleEngines(w http.ResponseWriter, r *http.Request) {
	statuses := s.engineStatus.List()
	out := make([]EngineEntry, 0, len(statuses))
	for _, st := range statuses {
		out = append(out, EngineEntry{
			Name:       st.Name,
			State:      st.State,
			LastTickAt: formatTime(st.LastTickAt),
			LastError:  st.LastError,
			Inflight:   st.Inflight,
		})
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, EnginesResponse{
		Count:   len(out),
		Engines: out,
	})
}
