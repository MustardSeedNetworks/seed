package orchestrator_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/bgp4"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/cdp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/lldp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/routing"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/sysinfo"
)

// The NIAC pack acceptance (seed plan S4-2) runs every collector against every
// SNMP agent of a NIAC scenario pack and compares what they found with the
// pack's manifest. The live half needs a running pack and lives in
// pack_acceptance_niac_test.go; this file holds the parts that decide
// the verdict, so they are tested without one.

// packObservation is one collector's entry in NIAC's expectedObservations.
type packObservation struct {
	Devices int `json:"devices"`
	Rows    int `json:"rows,omitempty"`
}

// rowRecorder is the Publisher for one device. Collect publishes
// synchronously, so the recorder holds the row count of whichever collector
// ran last; the caller reads it after each Collect.
type rowRecorder struct {
	rows int
}

func (r *rowRecorder) record(rows int) error {
	r.rows = rows
	return nil
}

func (r *rowRecorder) PublishSysInfo(context.Context, sysinfo.Observation) error {
	return r.record(1)
}

func (r *rowRecorder) PublishIfTable(_ context.Context, obs iftable.Observation) error {
	return r.record(len(obs.Rows))
}

func (r *rowRecorder) PublishLLDP(_ context.Context, obs lldp.Observation) error {
	return r.record(len(obs.Neighbors))
}

func (r *rowRecorder) PublishCDP(_ context.Context, obs cdp.Observation) error {
	return r.record(len(obs.Neighbors))
}

func (r *rowRecorder) PublishARP(_ context.Context, obs arp.Observation) error {
	return r.record(len(obs.Entries))
}

// PublishFDB counts bridge ports, not MAC entries: NIAC's manifest counts the
// switch ports an endpoint is learned on, and one port can carry many MACs.
func (r *rowRecorder) PublishFDB(_ context.Context, obs fdb.Observation) error {
	ports := make(map[uint32]struct{}, len(obs.Entries))
	for _, entry := range obs.Entries {
		ports[entry.BridgePort] = struct{}{}
	}
	return r.record(len(ports))
}

func (r *rowRecorder) PublishRouting(_ context.Context, obs routing.Observation) error {
	return r.record(len(obs.Routes))
}

func (r *rowRecorder) PublishHostResources(_ context.Context, obs hostresources.Observation) error {
	return r.record(len(obs.Storage) + len(obs.Processors))
}

func (r *rowRecorder) PublishBGP4(_ context.Context, obs bgp4.Observation) error {
	return r.record(len(obs.Peers))
}

// packResult is one collector's outcome on one agent.
type packResult struct {
	Collector string
	Agent     string
	Rows      int
	Err       error
}

// agentRows renders each agent's row count per collector on one line, agents
// and collectors in a stable order.
func agentRows(results []packResult) []string {
	byAgent := make(map[string][]packResult)
	for _, result := range results {
		byAgent[result.Agent] = append(byAgent[result.Agent], result)
	}
	lines := make([]string, 0, len(byAgent))
	for _, agent := range slices.Sorted(maps.Keys(byAgent)) {
		var line strings.Builder
		line.WriteString(agent + ":")
		for _, result := range byAgent[agent] {
			if result.Err != nil {
				fmt.Fprintf(&line, " %s=error", result.Collector)
				continue
			}
			fmt.Fprintf(&line, " %s=%d", result.Collector, result.Rows)
		}
		lines = append(lines, line.String())
	}
	return lines
}

// packTally is what one collector found across a pack.
type packTally struct {
	Devices int
	Rows    int
	Errors  []string
}

// tallyResults sums per-agent results by collector. An agent counts toward a
// collector when that collector found at least one row there, which is how
// NIAC counts the devices that author something the collector reads.
func tallyResults(results []packResult) map[string]packTally {
	tallies := make(map[string]packTally)
	for _, result := range results {
		tally := tallies[result.Collector]
		switch {
		case result.Err != nil:
			tally.Errors = append(tally.Errors, fmt.Sprintf("%s: %v", result.Agent, result.Err))
		case result.Rows > 0:
			tally.Devices++
			tally.Rows += result.Rows
		}
		tallies[result.Collector] = tally
	}
	return tallies
}

// packFindings compares the tallies with the manifest. A collector that
// errors on an agent is a finding whether or not the pack promises anything
// for it: an agent that serves no table should answer with an empty one.
// Rows are compared only where the manifest counts them.
func packFindings(expected map[string]packObservation, tallies map[string]packTally) []string {
	var findings []string
	for _, collector := range slices.Sorted(maps.Keys(tallies)) {
		for _, failure := range tallies[collector].Errors {
			findings = append(findings, fmt.Sprintf("%s: collect failed on %s", collector, failure))
		}
	}
	for _, collector := range slices.Sorted(maps.Keys(expected)) {
		want, got := expected[collector], tallies[collector]
		if got.Devices != want.Devices {
			findings = append(findings, fmt.Sprintf(
				"%s: found rows on %d devices, the manifest promises %d", collector, got.Devices, want.Devices))
		}
		if want.Rows > 0 && got.Rows != want.Rows {
			findings = append(findings, fmt.Sprintf(
				"%s: found %d rows, the manifest promises %d", collector, got.Rows, want.Rows))
		}
	}
	return findings
}
