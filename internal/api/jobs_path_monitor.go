package api

// jobs_path_monitor.go registers continuous path monitoring as a job kind
// (#165) -- the mtr experience, in-product.
//
// It is a job rather than a bespoke goroutine because everything continuous
// monitoring needs already exists there: cancellation, a concurrency ceiling,
// a handle the caller can poll, and interrupted-on-restart bookkeeping. The
// one thing it does differently is that it has no natural end -- it runs until
// cancelled -- so progress is a round count folded into a fraction rather than
// a distance travelled.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// pathMonitorJobKind is the registered kind name for continuous path monitoring.
const pathMonitorJobKind = "path-monitor"

// The probe budget. docs/EDITIONS.md targets a ~5W Pi-class "Lite (portable)"
// unit, so this is a number rather than an adjective: at most one probe per hop
// per second, over at most pathMonitorMaxHops hops.
const (
	// pathMonitorMinInterval is the floor between the start of one round and
	// the next. A round is already paced by its per-hop timeout; this bounds
	// the case where every hop answers instantly.
	pathMonitorMinInterval = 1 * time.Second

	// pathMonitorMaxInterval bounds what a caller may ask for, so a monitor
	// cannot be parked for an hour and still hold a slot.
	pathMonitorMaxInterval = 60 * time.Second

	// pathMonitorHopTimeout is the per-hop wait. One second per hop is the
	// other half of the budget.
	pathMonitorHopTimeout = 1 * time.Second

	// pathMonitorMaxHops caps the hops probed per round.
	pathMonitorMaxHops = tracerouteMaxHops

	// pathMonitorMaxRounds stops a monitor that nobody ever cancels. At the
	// default interval this is a little over two hours.
	pathMonitorMaxRounds = 8192

	// pathMonitorIdleGrace is how long a monitor keeps running with no SSE
	// client attached. /api/events is a broadcast, so the server cannot tell
	// which page navigated away; the explicit cancel is the happy path and
	// this is the guard for when it does not arrive.
	pathMonitorIdleGrace = 30 * time.Second
)

// PathMonitorRequest starts a continuous monitor.
type PathMonitorRequest struct {
	Destination     string `json:"destination"     validate:"required"`
	IntervalSeconds int    `json:"intervalSeconds" validate:"omitempty,gte=1,lte=60"`
	MaxHops         int    `json:"maxHops"         validate:"omitempty,gte=1,lte=30"`
}

// PathMonitorUpdate is the SSE payload carrying one round's accumulated view.
//
// It carries the destination rather than a job id because a job handler is not
// told its own id, and an always-empty field on the wire is worse than one that
// is absent. Correlation is by destination, which holds because the kind allows
// one monitor per destination at a time.
type PathMonitorUpdate struct {
	discovery.PathMonitorSnapshot
}

// pathTracer is the slice of *discovery.Tracer the monitor needs, and the seam
// that lets the loop be tested without a raw socket.
type pathTracer interface {
	TraceICMP(ctx context.Context, target string) *discovery.TracerouteResult
}

// pathMonitorDeps is what the handler needs from the server, injected so the
// loop is testable without one.
type pathMonitorDeps struct {
	newTracer   func(maxHops int) pathTracer
	publish     func(PathMonitorUpdate)
	clientCount func() int
	now         func() time.Time
}

// errPathMonitorDestinationRequired is returned when a monitor is started with
// no destination, which is a caller error rather than a network one.
var errPathMonitorDestinationRequired = errors.New("destination is required")

// newPathMonitorHandler returns the job Handler for the "path-monitor" kind.
//
// The job runs until its context is cancelled, the round ceiling is reached, or
// no client has been listening for pathMonitorIdleGrace. Every exit returns the
// snapshot accumulated so far: a monitor that was cancelled after ten minutes
// has ten minutes of evidence in it, and discarding that because the operator
// pressed stop would throw away the answer they were waiting for.
func newPathMonitorHandler(deps pathMonitorDeps) jobs.Handler {
	return func(ctx context.Context, params any, progress func(float64)) (any, error) {
		req, err := decodePathMonitorParams(params)
		if err != nil {
			return nil, err
		}
		if req.Destination == "" {
			return nil, errPathMonitorDestinationRequired
		}

		return runPathMonitor(ctx, req, deps, progress), nil
	}
}

// runPathMonitor is the loop itself, separated from param decoding so a test
// drives it with a fake tracer and a clock.
func runPathMonitor(
	ctx context.Context,
	req PathMonitorRequest,
	deps pathMonitorDeps,
	progress func(float64),
) discovery.PathMonitorSnapshot {
	interval := pathMonitorInterval(req.IntervalSeconds)
	maxHops := pathMonitorHops(req.MaxHops)

	tracer := deps.newTracer(maxHops)
	monitor := discovery.NewPathMonitor(req.Destination)
	idleSince := time.Time{}

	for round := range pathMonitorMaxRounds {
		started := deps.now()

		monitor.Observe(tracer.TraceICMP(ctx, req.Destination))
		snapshot := monitor.Snapshot()

		if deps.publish != nil {
			deps.publish(PathMonitorUpdate{PathMonitorSnapshot: snapshot})
		}
		if progress != nil {
			progress(float64(round+1) / float64(pathMonitorMaxRounds))
		}

		var stop bool
		idleSince, stop = pathMonitorIdleCheck(deps, idleSince)
		if stop {
			return snapshot
		}

		if !sleepUntil(ctx, started.Add(interval), deps.now) {
			return snapshot
		}
	}

	return monitor.Snapshot()
}

// pathMonitorIdleCheck tracks how long the monitor has run unobserved and
// reports whether the grace period has elapsed.
func pathMonitorIdleCheck(deps pathMonitorDeps, idleSince time.Time) (time.Time, bool) {
	if deps.clientCount == nil || deps.clientCount() > 0 {
		return time.Time{}, false
	}
	if idleSince.IsZero() {
		return deps.now(), false
	}
	return idleSince, deps.now().Sub(idleSince) >= pathMonitorIdleGrace
}

// sleepUntil waits for a deadline, reporting false if the context ended first.
func sleepUntil(ctx context.Context, deadline time.Time, now func() time.Time) bool {
	remaining := deadline.Sub(now())
	if remaining <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(remaining)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// pathMonitorInterval clamps a requested interval into the probe budget.
func pathMonitorInterval(seconds int) time.Duration {
	if seconds <= 0 {
		return pathMonitorMinInterval
	}
	interval := time.Duration(seconds) * time.Second
	return min(max(interval, pathMonitorMinInterval), pathMonitorMaxInterval)
}

// pathMonitorHops clamps a requested hop count into the probe budget.
func pathMonitorHops(hops int) int {
	if hops <= 0 {
		return pathMonitorMaxHops
	}
	return min(hops, pathMonitorMaxHops)
}

// registerPathMonitorKind registers the path-monitor kind against the server's
// SSE hub, with the tracer behind a factory so tests can substitute one.
func (s *Server) registerPathMonitorKind(newTracer func(maxHops int) pathTracer) {
	deps := pathMonitorDeps{
		newTracer: newTracer,
		now:       time.Now,
		publish: func(update PathMonitorUpdate) {
			if s.sseHub() == nil {
				return
			}
			s.sseHub().Broadcast(Message{Type: "pathMonitor", Payload: update})
		},
		clientCount: func() int {
			if s.sseHub() == nil {
				return 0
			}
			return s.sseHub().ClientCount()
		},
	}

	if err := s.jobsRunner().Register(pathMonitorJobKind, newPathMonitorHandler(deps)); err != nil {
		logging.GetLogger().Error("failed to register path-monitor job kind", "error", err)
	}
}

// defaultPathTracer builds the tracer the monitor uses in production. PTR
// lookups are on: a continuous view is read over minutes, so the name is worth
// the lookup that a one-shot trace skips for latency.
func defaultPathTracer(maxHops int) pathTracer {
	return discovery.NewTracerWithPTR(pathMonitorHopTimeout, maxHops)
}

// decodePathMonitorParams reads the request off the generic /jobs surface.
func decodePathMonitorParams(params any) (PathMonitorRequest, error) {
	raw, ok := params.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return PathMonitorRequest{}, errors.New("path-monitor job requires params")
	}
	var req PathMonitorRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return PathMonitorRequest{}, fmt.Errorf("invalid path-monitor params: %w", err)
	}
	return req, nil
}
