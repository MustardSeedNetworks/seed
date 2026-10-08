package api

// jobs_path_discovery.go registers the two one-shot path engines as job kinds:
// multi-path egress detection (#395) and path MTU discovery (#435). Both run
// for seconds to minutes and probe the network as they go, so they belong on
// the runner for the same reasons the path monitor does -- cancel, the
// concurrency ceiling and a handle to poll -- and a cancelled run returns what
// it had measured.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// Registered kind names for the path engines. Both are sold under
// pathAnalysisFeature, like the monitor and the one-shot trace on /path.
const (
	multiPathJobKind = "path-multipath"
	pathMTUJobKind   = "path-mtu"
)

// The probe budget, on the same terms as the monitor's: one second per hop
// and no more hops than /path traces. A multi-path run is
// discovery.MultiPathAttempts traces, so its worst case is that many times
// the hop count in seconds; a path MTU search sends one probe at a time and
// waits pathMTUProbeTimeout for each.
const (
	multiPathHopTimeout = 1 * time.Second
	multiPathMaxHops    = tracerouteMaxHops
	pathMTUProbeTimeout = 1 * time.Second
)

// MultiPathRequest starts a multi-path egress run.
type MultiPathRequest struct {
	Destination string `json:"destination"       validate:"required"`
	MaxHops     int    `json:"maxHops,omitempty" validate:"omitempty,gte=1,lte=30"`
}

// PathMTURequest starts a path MTU discovery run.
type PathMTURequest struct {
	Destination string `json:"destination" validate:"required"`
}

// flowTrace is discovery.TraceFlow, behind a seam so the kind is testable
// without a raw socket.
type flowTrace func(
	ctx context.Context,
	target string,
	maxHops, flowID int,
	timeout time.Duration,
) (*discovery.TracerouteResult, error)

// pathMTUMeasure is discovery.MeasurePathMTU, behind the same kind of seam.
type pathMTUMeasure func(ctx context.Context, target string, timeout time.Duration) (*discovery.PMTUDResult, error)

// errPathDestinationRequired is returned when a run names no destination.
var errPathDestinationRequired = errors.New("destination is required")

// newMultiPathHandler returns the job Handler for the "path-multipath" kind.
func newMultiPathHandler(trace flowTrace) jobs.Handler {
	return func(ctx context.Context, params any, progress func(float64)) (any, error) {
		var req MultiPathRequest
		if err := decodeStrictJobParams(multiPathJobKind, params, &req); err != nil {
			return nil, err
		}
		if req.Destination == "" {
			return nil, errPathDestinationRequired
		}
		maxHops := multiPathMaxHops
		if req.MaxHops > 0 {
			maxHops = min(req.MaxHops, multiPathMaxHops)
		}

		// A trace that cannot start fails the same way on every attempt, so
		// the first such error ends the run rather than being folded into an
		// empty result.
		var traceErr error
		attempts := 0
		round := func(ctx context.Context, flowID int) *discovery.TracerouteResult {
			if traceErr != nil {
				return nil
			}
			result, err := trace(ctx, req.Destination, maxHops, flowID, multiPathHopTimeout)
			if err != nil {
				traceErr = err
				return nil
			}
			attempts++
			progress(float64(attempts) / discovery.MultiPathAttempts)
			return result
		}

		result := discovery.DiscoverPaths(ctx, req.Destination, round)
		if traceErr != nil {
			return nil, traceErr
		}
		return result, nil
	}
}

// newPathMTUHandler returns the job Handler for the "path-mtu" kind.
func newPathMTUHandler(measure pathMTUMeasure) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		var req PathMTURequest
		if err := decodeStrictJobParams(pathMTUJobKind, params, &req); err != nil {
			return nil, err
		}
		if req.Destination == "" {
			return nil, errPathDestinationRequired
		}
		return measure(ctx, req.Destination, pathMTUProbeTimeout)
	}
}

// decodeStrictJobParams reads a kind's request off the generic /jobs surface,
// refusing fields the request does not have so a misspelt option fails
// rather than being silently ignored.
func decodeStrictJobParams(kind string, params any, req any) error {
	raw, ok := params.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return fmt.Errorf("%s job requires params", kind)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(req); err != nil {
		return fmt.Errorf("invalid %s params: %w", kind, err)
	}
	return nil
}

// registerPathDiscoveryKinds registers both path engines with injectable
// network seams.
func (s *Server) registerPathDiscoveryKinds(trace flowTrace, measure pathMTUMeasure) {
	if err := s.jobsRunner().Register(multiPathJobKind, newMultiPathHandler(trace)); err != nil {
		logging.GetLogger().Error("failed to register path-multipath job kind", "error", err)
	}
	if err := s.jobsRunner().Register(pathMTUJobKind, newPathMTUHandler(measure)); err != nil {
		logging.GetLogger().Error("failed to register path-mtu job kind", "error", err)
	}
}
