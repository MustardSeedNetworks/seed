package api

// jobs_iperf.go registers the iperf3 client test as a unified job kind
// (ADR-0005), the second real long-op on the runner after speedtest. iperf3 has
// no native Go protocol library, so the existing wrapper drives the bundled
// binary via exec.CommandContext — which already gives real subprocess
// cancellation through the job's context. The legacy /telemetry/iperf/client +
// /status endpoints are unchanged (retire at the Phase-7 frontend cutover); this
// kind is additive and adds no routes.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/diagnostics/iperf"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

const (
	// iperfJobKind is the registered kind name for an iperf3 client test.
	iperfJobKind = "iperf"

	// iperfProgressPoll is how often the kind samples the client's phase
	// progress to report it onto the job.
	iperfProgressPoll = 250 * time.Millisecond

	// iperfProgressMax is the client's progress scale (0–100); the job model
	// reports a [0,1] fraction.
	iperfProgressMax = 100.0
)

// iperfClient is the slice of [iperf.Manager] behaviour the job kind needs.
// Depending on an interface keeps the kind unit-testable without launching the
// iperf3 binary and isolates the kind from the wrapper's other concerns.
type iperfClient interface {
	RunClient(ctx context.Context, config *iperf.ClientConfig) (*iperf.Result, error)
	GetClientStatus() iperf.ClientStatus
}

// newIperfHandler returns the job Handler for the "iperf" kind. The job params
// are an IperfClientRequest (the same shape the legacy endpoint accepts);
// newClient produces a fresh manager per job so concurrent runs never share
// singleton client state. The handler runs the test, samples its phase progress
// onto the job, and returns the result as an IperfResultResponse.
func newIperfHandler(newClient func() iperfClient) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		req, err := decodeIperfParams(params)
		if err != nil {
			return nil, err
		}
		// Semantic validation lives with the domain, not the generic /jobs
		// surface: a bad request surfaces as a failed job, not a panic.
		if verr := validateIperfClientRequest(&req); verr != nil {
			return nil, verr
		}

		config := iperf.ClientConfig{
			Server:    req.Server,
			Port:      req.Port,
			Protocol:  req.Protocol,
			Reverse:   req.Reverse,
			Direction: req.Direction,
			Duration:  req.Duration,
			Parallel:  req.Parallel,
		}

		client := newClient()
		type outcome struct {
			res *iperf.Result
			err error
		}
		done := make(chan outcome, 1)
		go func() {
			res, runErr := client.RunClient(ctx, &config)
			done <- outcome{res: res, err: runErr}
		}()

		ticker := time.NewTicker(iperfProgressPoll)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				report(client.GetClientStatus().Progress / iperfProgressMax)
			case out := <-done:
				if out.err != nil {
					return nil, out.err
				}
				report(1)
				return toIperfResultResponse(out.res), nil
			}
		}
	}
}

// decodeIperfParams parses the opaque job params into an IperfClientRequest.
func decodeIperfParams(params any) (IperfClientRequest, error) {
	raw, ok := params.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return IperfClientRequest{}, errors.New("iperf job requires params")
	}
	var req IperfClientRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return IperfClientRequest{}, fmt.Errorf("invalid iperf params: %w", err)
	}
	return req, nil
}

// registerIperfKind registers the iperf kind with an injectable client factory
// (the seam that makes the wiring testable without the binary).
func (s *Server) registerIperfKind(newClient func() iperfClient) {
	if err := s.jobsRunner().Register(iperfJobKind, newIperfHandler(newClient)); err != nil {
		logging.GetLogger().Error("failed to register iperf job kind", "error", err)
	}
}
