package api

// jobs_speedtest.go registers the first real long-op as a unified job kind
// (ADR-0005): "speedtest". It is the proof that the runner drives a real
// operation end-to-end — submit, progress, cancel, result — over the /jobs
// surface. The legacy /telemetry/speedtest + /status endpoints are unchanged and
// stay until the frontend cuts over to /jobs (Phase 7); this kind is additive.

import (
	"context"
	"time"

	"github.com/krisarmstrong/seed/internal/diagnostics/iperf"
	"github.com/krisarmstrong/seed/internal/diagnostics/speedtest"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

const (
	// speedtestJobKind is the registered kind name for a speedtest job.
	speedtestJobKind = "speedtest"

	// speedtestProgressPoll is how often the kind samples the tester's phase
	// progress to report it onto the job.
	speedtestProgressPoll = 250 * time.Millisecond

	// speedtestProgressMax is the tester's progress scale (0–100); the job model
	// reports a [0,1] fraction.
	speedtestProgressMax = 100.0
)

// speedTester is the slice of [speedtest.Tester] behaviour the job kind needs.
// Depending on an interface keeps the kind unit-testable without the network and
// leaves room to swap the implementation (e.g. a home-grown tester) without
// touching the kind, the runner, or the HTTP surface.
type speedTester interface {
	RunTest(ctx context.Context) (*speedtest.Result, error)
	GetStatus() speedtest.Status
}

// newSpeedtestHandler returns the job Handler for the "speedtest" kind. newTester
// produces a fresh tester per job so concurrent runs never share singleton
// state. The handler runs the test, samples its phase progress onto the job, and
// returns the result as a SpeedtestResponse; a cancelled job context unwinds the
// underlying run at its next phase boundary.
func newSpeedtestHandler(newTester func() speedTester) jobs.Handler {
	return func(ctx context.Context, _ any, report func(float64)) (any, error) {
		tester := newTester()

		type outcome struct {
			res *speedtest.Result
			err error
		}
		done := make(chan outcome, 1)
		go func() {
			res, err := tester.RunTest(ctx)
			done <- outcome{res: res, err: err}
		}()

		ticker := time.NewTicker(speedtestProgressPoll)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				report(tester.GetStatus().Progress / speedtestProgressMax)
			case out := <-done:
				if out.err != nil {
					return nil, out.err
				}
				report(1)
				return toSpeedtestResponse(out.res), nil
			}
		}
	}
}

// registerJobKinds registers the built-in job kinds on the runner at startup.
func (s *Server) registerJobKinds() {
	s.registerSpeedtestKind(func() speedTester { return speedtest.NewTester() })
	s.registerIperfKind(func() iperfClient { return iperf.NewManager() })
	s.registerVulnScanKind(func() vulnScanService {
		return serverVulnScanService{scanner: s.vulnScanner(), devices: s.deviceDiscovery()}
	})
	s.registerEngineScanKind(func() engineScanner { return s.services.Discovery.Engine })
}

// registerSpeedtestKind registers the speedtest kind with an injectable tester
// factory (the seam that makes the wiring testable without the network).
func (s *Server) registerSpeedtestKind(newTester func() speedTester) {
	if err := s.jobsRunner().Register(speedtestJobKind, newSpeedtestHandler(newTester)); err != nil {
		logging.GetLogger().Error("failed to register speedtest job kind", "error", err)
	}
}
