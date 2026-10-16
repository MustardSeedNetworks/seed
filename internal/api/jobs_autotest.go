package api

// jobs_autotest.go registers AutoTest (P-E1) as a job kind: the link, cable,
// DHCP, DNS, gateway, VLAN and internet checks run in turn on the selected
// interface and come back as one verdict with every stage's answer. It is a
// job because the run takes seconds, and the runner already gives a caller the
// handle, the progress and the cancel.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// autotestJobKind is the registered kind name for an AutoTest run.
const autotestJobKind = "autotest"

// newAutotestHandler returns the job Handler for the kind. iface names the
// interface a run checks; stages are the checks bound to it.
func newAutotestHandler(iface func() string, stages []autotest.Stage) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		var req autotest.Spec
		if raw, ok := params.(json.RawMessage); ok && len(raw) > 0 {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil {
				return nil, fmt.Errorf("invalid autotest params: %w", err)
			}
		}
		return autotest.Run(ctx, iface(), req, stages, report), nil
	}
}

// autotestStages binds each stage to the tester the matching per-check page
// uses, so a stage and its page cannot disagree about the same port.
func (s *Server) autotestStages() []autotest.Stage {
	readLink := func(iface string) (*netif.LinkStatus, error) {
		if err := s.netManager().RefreshInterfaces(); err != nil {
			return nil, err
		}
		return s.netManager().GetLinkStatus(iface)
	}
	return []autotest.Stage{
		autotest.LinkStage(readLink),
		autotest.CableStage(readLink, func() *cable.TestResult { return s.cableTester().Test() }),
		autotest.DHCPStage(readLink, dhcp.CurrentLease),
		autotest.DNSStage(func(ctx context.Context) *dns.TestResult { return s.dnsTester().Test(ctx) }),
		autotest.GatewayStage(func() *gateway.PingStats { return s.gatewayTester().Test() }),
		autotest.VLANStage(s.getVLANsFromDiscovery),
		autotest.InternetStage(s.autotestInternetTargets, autotest.PingHost),
	}
}

// autotestInternetTargets returns the enabled ping targets of the
// health-check settings: the hosts the operator already chose to prove
// internet reachability with.
func (s *Server) autotestInternetTargets(ctx context.Context) ([]autotest.Target, error) {
	st, err := s.healthSettings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("health-check settings: %w", err)
	}
	var targets []autotest.Target
	for _, t := range st.Endpoints.PingTargets {
		if t.Enabled {
			targets = append(targets, autotest.Target{Name: t.Name, Host: t.Host})
		}
	}
	return targets, nil
}

// registerAutotestKind registers the kind on the selected interface.
func (s *Server) registerAutotestKind() {
	handler := newAutotestHandler(func() string { return s.netManager().GetCurrentInterface() }, s.autotestStages())
	if err := s.jobsRunner().Register(autotestJobKind, handler); err != nil {
		logging.GetLogger().Error("failed to register autotest job kind", "error", err)
	}
}
