package api

// jobs_multicast.go registers #399's active multicast listen as a job kind:
// join one group on one interface for a bounded window and report what
// arrived. It is a job rather than a route because it holds a group
// membership for seconds, and the runner already provides the cancel, the
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

// multicastListenJobKind is the registered kind name for a multicast listen.
const multicastListenJobKind = "multicast-listen"

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
