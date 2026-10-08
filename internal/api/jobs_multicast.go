package api

// jobs_multicast.go registers #399's two multicast checks as job kinds. A
// listen joins one group on one interface for a bounded window and reports
// what arrived; an observation watches the segment's IGMP and MLD for a
// bounded window and reports the groups, their reporters and the querier.
// They are jobs rather than routes because each holds the interface for
// seconds to minutes, and the runner already provides the cancel, the
// concurrency ceiling and the handle a caller polls.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/multicast"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

const (
	// multicastListenJobKind is the registered kind name for a multicast listen.
	multicastListenJobKind = "multicast-listen"
	// multicastObserveJobKind is the registered kind name for a passive
	// IGMP/MLD observation.
	multicastObserveJobKind = "multicast-observe"
)

// multicastListen is multicast.Listen, behind a seam so the kind is testable
// without joining a group.
type multicastListen func(context.Context, multicast.ListenRequest) (*multicast.ListenResult, error)

// errMulticastListenParams is returned when a listen is submitted with no
// params: group, port and interface are all required.
var errMulticastListenParams = errors.New("multicast-listen job requires params")

// newMulticastListenHandler returns the job Handler for the kind. A cancelled
// listen returns what it heard, so the job succeeds with a partial result.
func newMulticastListenHandler(listen multicastListen) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		raw, ok := params.(json.RawMessage)
		if !ok || len(raw) == 0 {
			return nil, errMulticastListenParams
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var req multicast.ListenRequest
		if err := dec.Decode(&req); err != nil {
			return nil, fmt.Errorf("invalid multicast-listen params: %w", err)
		}
		return listen(ctx, req)
	}
}

// registerMulticastListenKind registers the kind with an injectable listen.
func (s *Server) registerMulticastListenKind(listen multicastListen) {
	if err := s.jobsRunner().Register(multicastListenJobKind, newMulticastListenHandler(listen)); err != nil {
		logging.GetLogger().Error("failed to register multicast-listen job kind", "error", err)
	}
}

// multicastObserve is multicast.Observe bound to the capture port, behind a
// seam so the kind is testable without a capture handle.
type multicastObserve func(context.Context, multicast.ObserveRequest, func(float64)) (*multicast.ObserveResult, error)

// newMulticastObserveHandler returns the job Handler for the kind. Like a
// packet capture, an observation that names no interface runs on the
// configured default one. A cancelled observation returns what it saw.
func newMulticastObserveHandler(observe multicastObserve, defaultInterface func() string) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		var req multicast.ObserveRequest
		if raw, ok := params.(json.RawMessage); ok && len(raw) > 0 {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil {
				return nil, fmt.Errorf("invalid multicast-observe params: %w", err)
			}
		}
		if req.Interface == "" {
			req.Interface = defaultInterface()
		}
		return observe(ctx, req, report)
	}
}

// registerMulticastObserveKind registers the kind with an injectable observe.
func (s *Server) registerMulticastObserveKind(observe multicastObserve) {
	if err := s.jobsRunner().
		Register(multicastObserveJobKind, newMulticastObserveHandler(observe, s.defaultInterface)); err != nil {
		logging.GetLogger().Error("failed to register multicast-observe job kind", "error", err)
	}
}
