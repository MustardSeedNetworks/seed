// Package status holds the engine-status application (use-case) layer
// (ADR-0020). It owns the read-only query that lists every registered
// long-running subsystem and their health state, previously inlined in
// the api.Server handleEngines handler. Handlers keep transport concerns:
// request decode, authorization, response shaping. The adapter satisfying
// the port lives in the composition root (internal/app) and resolves the
// engine registry lazily, so a nil registry degrades to an empty list
// (the pre-strangle empty-response path) rather than panicking.
package status

import (
	"time"

	"github.com/MustardSeedNetworks/seed/internal/engine"
)

// Registry is the engine-registry surface the use-case reads, defined at
// the consumer (ADR-0020) and satisfied by an adapter over *engine.Registry
// in internal/app. Available reports whether a registry is wired (resolved
// per call so the use-case can degrade gracefully); Engines is only invoked
// once availability is confirmed.
type Registry interface {
	Available() bool
	Engines() []engine.Engine
}

// EngineStatus is the domain result for one engine entry. Transport
// encoding to the wire map stays in the handler so this struct is
// protocol-agnostic and testable without HTTP.
type EngineStatus struct {
	Name       string
	State      string
	LastTickAt time.Time
	LastError  string
	Inflight   int
}

// Service is the engine-status use-case.
type Service struct {
	reg Registry
}

// NewService builds the use-case over its narrow registry dependency.
func NewService(reg Registry) *Service {
	return &Service{reg: reg}
}

// List returns a domain snapshot of every registered engine and its status.
// Returns nil when the registry is not wired or reports unavailable —
// callers treat nil/empty identically (the pre-strangle empty-response path).
//
// The status logic mirrors the pre-strangle encodeEngineEntry: engines that
// implement engine.Reporter contribute their Status(); engines that don't
// default to engine.StatusOK(). An empty State from a Reporter is normalised
// to engine.StateOK so the wire response is uniform regardless of adoption.
func (s *Service) List() []EngineStatus {
	if s == nil || s.reg == nil || !s.reg.Available() {
		return nil
	}
	engines := s.reg.Engines()
	out := make([]EngineStatus, 0, len(engines))
	for _, e := range engines {
		st := engine.StatusOK()
		if reporter, ok := e.(engine.Reporter); ok {
			st = reporter.Status()
			if st.State == "" {
				st.State = engine.StateOK
			}
		}
		out = append(out, EngineStatus{
			Name:       e.Name(),
			State:      st.State,
			LastTickAt: st.LastTickAt,
			LastError:  st.LastError,
			Inflight:   st.Inflight,
		})
	}
	return out
}
