package api

// jobs_engine.go registers the discovery engine scan as a unified job kind
// (ADR-0005). Like the vuln-scan kind it is a thin additive wrapper over the
// existing ctx-aware engine methods behind an interface seam — no
// discovery-internal refactor. engine.Scan is synchronous and exposes no
// progress fraction (only IsScanning), so the kind reports no granular progress;
// the runner's queued->running->succeeded transitions carry status. The legacy
// /discovery/engine/scan endpoint is unchanged (retire at the Phase-7 cutover).

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
	"github.com/krisarmstrong/seed/internal/services/discovery"
)

// engineScanJobKind is the registered kind name for a discovery engine scan.
const engineScanJobKind = "engine-scan"

// engineScanner is the slice of *discovery.Engine behaviour the kind needs.
// The real engine satisfies it; a fake drives the tests.
type engineScanner interface {
	Scan(ctx context.Context, opts *discovery.ScanOptions) (*discovery.ScanResult, error)
	GetDevices() []*discovery.DiscoveredDevice
	GetStats() *discovery.EngineStats
	GetCapabilities() map[string]bool
}

// newEngineScanHandler returns the job Handler for the "engine-scan" kind. It
// runs one scan (cancellable via the job context) and returns the same
// EngineDiscoveryResponse the legacy endpoint produces. The report callback is
// unused: the engine surfaces no progress fraction, only running/idle.
func newEngineScanHandler(newEngine func() engineScanner) jobs.Handler {
	return func(ctx context.Context, params any, _ func(float64)) (any, error) {
		opts, err := engineScanOptsFromParams(params)
		if err != nil {
			return nil, err
		}

		engine := newEngine()
		result, scanErr := engine.Scan(ctx, opts)
		if scanErr != nil {
			return nil, scanErr
		}
		return EngineDiscoveryResponse{
			Devices:      engine.GetDevices(),
			Stats:        engine.GetStats(),
			ScanResult:   result,
			Capabilities: engine.GetCapabilities(),
		}, nil
	}
}

// engineScanOptsFromParams parses the optional job params into scan options.
// Absent params default to a quick scan (matching the legacy no-body behaviour);
// malformed params are an error so the job fails rather than silently running a
// different scan.
func engineScanOptsFromParams(params any) (*discovery.ScanOptions, error) {
	raw, ok := params.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return discovery.DefaultQuickScanOpts(), nil
	}
	var req EngineScanRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("invalid engine-scan params: %w", err)
	}
	return scanOptsFromRequest(req), nil
}

// registerEngineScanKind registers the engine-scan kind with an injectable
// engine factory (the seam that makes the wiring testable without the engine).
func (s *Server) registerEngineScanKind(newEngine func() engineScanner) {
	if err := s.jobsRunner().Register(engineScanJobKind, newEngineScanHandler(newEngine)); err != nil {
		logging.GetLogger().Error("failed to register engine-scan job kind", "error", err)
	}
}
