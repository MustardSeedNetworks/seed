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
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
)

// The NIAC pack acceptance (seed plan S4-2, niac plan P3-2) runs every
// collector against every SNMP agent of a NIAC scenario pack and compares what
// they found with the pack's manifest, then what seed's topology drew from it
// with the pack's links. The live half needs a running pack and lives in
// pack_acceptance_niac_test.go; this file holds the parts that decide
// the verdict, so they are tested without one.

// packObservation is one collector's entry in NIAC's expectedObservations.
type packObservation struct {
	Devices int `json:"devices"`
	Rows    int `json:"rows,omitempty"`
}

// rowRecorder is the Publisher for one device. Collect publishes
// synchronously, so the recorder holds the row count of whichever collector
// ran last; the caller reads it after each Collect. Every observation is then
// handed on to next, the sink the topology consumers read from.
type rowRecorder struct {
	rows int
	// poller is the polling host's own MAC, in the collector's canonical
	// form. NIAC places the poller on a spare port of its attachment pool, so
	// the pool switch learns it on a port the manifest does not describe.
	poller string
	next   orchestrator.Publisher
}

func (r *rowRecorder) PublishSysInfo(ctx context.Context, obs sysinfo.Observation) error {
	r.rows = 1
	return r.next.PublishSysInfo(ctx, obs)
}

func (r *rowRecorder) PublishIfTable(ctx context.Context, obs iftable.Observation) error {
	r.rows = len(obs.Rows)
	return r.next.PublishIfTable(ctx, obs)
}

func (r *rowRecorder) PublishLLDP(ctx context.Context, obs lldp.Observation) error {
	r.rows = len(obs.Neighbors)
	return r.next.PublishLLDP(ctx, obs)
}

func (r *rowRecorder) PublishCDP(ctx context.Context, obs cdp.Observation) error {
	r.rows = len(obs.Neighbors)
	return r.next.PublishCDP(ctx, obs)
}

func (r *rowRecorder) PublishARP(ctx context.Context, obs arp.Observation) error {
	r.rows = len(obs.Entries)
	return r.next.PublishARP(ctx, obs)
}

// PublishFDB counts bridge ports, not MAC entries: NIAC's manifest counts the
// switch ports an endpoint is learned on, and one port can carry many MACs.
// The poller's own entry is left out; it belongs to the test, not the pack.
func (r *rowRecorder) PublishFDB(ctx context.Context, obs fdb.Observation) error {
	ports := make(map[uint32]struct{}, len(obs.Entries))
	for _, entry := range obs.Entries {
		if entry.MACAddress == r.poller {
			continue
		}
		ports[entry.BridgePort] = struct{}{}
	}
	r.rows = len(ports)
	return r.next.PublishFDB(ctx, obs)
}

func (r *rowRecorder) PublishRouting(ctx context.Context, obs routing.Observation) error {
	r.rows = len(obs.Routes)
	return r.next.PublishRouting(ctx, obs)
}

func (r *rowRecorder) PublishHostResources(ctx context.Context, obs hostresources.Observation) error {
	r.rows = len(obs.Storage) + len(obs.Processors)
	return r.next.PublishHostResources(ctx, obs)
}

func (r *rowRecorder) PublishBGP4(ctx context.Context, obs bgp4.Observation) error {
	r.rows = len(obs.Peers)
	return r.next.PublishBGP4(ctx, obs)
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

// packLink is one pair of the pack's SNMP agents that the pack cables
// together, or whose forwarding database places one behind the other's port.
// Seed draws one link per pair of nodes whatever the number of cables, so the
// pair, ordered by name, is the unit both sides are compared in.
type packLink [2]string

func newPackLink(a, b string) packLink {
	if b < a {
		a, b = b, a
	}
	return packLink{a, b}
}

func (l packLink) String() string { return l[0] + " -- " + l[1] }

// topologyFindings compares the links seed's topology reconcilers drew with
// the pack's authored ones. Both directions are findings: a missing link is a
// cable the topology map leaves out, an extra one is a cable it invents.
func topologyFindings(authored, drawn []packLink) []string {
	want := make(map[packLink]bool, len(authored))
	for _, link := range authored {
		want[newPackLink(link[0], link[1])] = true
	}
	got := make(map[packLink]bool, len(drawn))
	for _, link := range drawn {
		got[newPackLink(link[0], link[1])] = true
	}
	var findings []string
	for _, link := range slices.SortedFunc(maps.Keys(want), comparePackLinks) {
		if !got[link] {
			findings = append(findings, "topology: no link drawn for authored "+link.String())
		}
	}
	for _, link := range slices.SortedFunc(maps.Keys(got), comparePackLinks) {
		if !want[link] {
			findings = append(findings, "topology: link drawn for "+link.String()+", which the pack does not author")
		}
	}
	return findings
}

func comparePackLinks(a, b packLink) int {
	return strings.Compare(a.String(), b.String())
}
