//go:build niacacceptance

// Built only by scripts/snmp-acceptance-niac.sh, which starts the pack this
// needs. It has its own tag because the nightly `integration` job fails on a
// skipped test, and without a running pack there is nothing to run.

package orchestrator_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpclient"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// packPollers bounds how many agents are polled at once. Each runs its ten
// collectors in sequence, so this is also the number of requests in flight.
const packPollers = 16

// packCollectTimeout bounds one collector on one agent. The largest table in
// a pack is an FDB of a few hundred rows; a minute is generous for that and
// still ends a run against a pack that has stopped answering.
const packCollectTimeout = time.Minute

// faultGrace is how long after its due time the fault is read. The harness
// dates the fault from when it saw NIAC report the simulation started, which
// is after the timeline's clock began, so this covers only timer latency.
const faultGrace = 2 * time.Second

// packTargets is what scripts/snmp-acceptance-niac.sh reads out of a
// generated pack: every device that runs an SNMP agent, at its first address,
// every pair of those agents the pack links, and the interface it has NIAC
// take down.
type packTargets struct {
	Pack   string      `json:"pack"`
	Agents []packAgent `json:"agents"`
	Links  []packLink  `json:"links"`
	Fault  packFault   `json:"fault"`
}

type packAgent struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Community string `json:"community"`
}

// packManifest is the part of NIAC's scenario manifest (schema version 4) this
// suite asserts. NIAC keys expectedObservations by seed's collector names; an
// absent key means the pack promises nothing for that collector, which is not
// the same claim as zero.
type packManifest struct {
	SchemaVersion int                        `json:"schemaVersion"`
	Expected      map[string]packObservation `json:"expectedObservations"`
}

// manifestSchemaVersion is the NIAC manifest version that introduced
// expectedObservations. An older manifest has no promises to check, and
// reading it as "every collector unpromised" would pass vacuously.
const manifestSchemaVersion = 4

func readJSON(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if decodeErr := json.Unmarshal(data, into); decodeErr != nil {
		return fmt.Errorf("decode %s: %w", path, decodeErr)
	}
	return nil
}

// TestNIACPack is the acceptance against a running NIAC pack: seed's ten
// collectors, over the wire, against every SNMP agent, compared with the
// pack's manifest (seed plan S4-2); then seed's topology reconcilers over what
// they stored, compared with the links the pack authors; then seed's alert
// pipeline across a fault NIAC injects after that poll (niac plan P3-2).
// scripts/snmp-acceptance-niac.sh starts the pack and runs this with
// SEED_NIAC_TARGETS, SEED_NIAC_MANIFEST, SEED_NIAC_POLLER_MAC and
// SEED_NIAC_FAULT_AT set.
func TestNIACPack(t *testing.T) {
	targets, manifest, poller, due := packInputs(t)
	db := dbtest.Open(t)
	store := sink.New(db.SNMPObservations(), slog.New(slog.DiscardHandler), nil)
	results := pollPack(t.Context(), targets.Agents, poller, store)

	t.Run("observations", func(t *testing.T) {
		// One line per agent, so a finding can be traced to the devices behind it.
		for _, line := range agentRows(results) {
			t.Log(line)
		}
		tallies := tallyResults(results)
		for _, name := range collectorNames(manifest.Expected, tallies) {
			want, promised := manifest.Expected[name]
			got := tallies[name]
			if !promised {
				t.Logf("%-15s found %3d devices %5d rows  (no promise), %d errors",
					name, got.Devices, got.Rows, len(got.Errors))
				continue
			}
			t.Logf("%-15s found %3d devices %5d rows  want %3d devices %5d rows, %d errors",
				name, got.Devices, got.Rows, want.Devices, want.Rows, len(got.Errors))
		}
		findings := packFindings(manifest.Expected, tallies)
		for _, finding := range findings {
			t.Errorf("%s: %s", targets.Pack, finding)
		}
		t.Logf("%s: %d agents, %d findings", targets.Pack, len(targets.Agents), len(findings))
	})

	t.Run("topology", func(t *testing.T) {
		reconcileTopology(t, db)
		drawn, findings := drawnLinks(t.Context(), db, targets.Agents)
		findings = append(findings, topologyFindings(targets.Links, drawn)...)
		findings = append(findings, primaryMACFindings(t, db, targets.Agents)...)
		for _, finding := range findings {
			t.Errorf("%s: %s", targets.Pack, finding)
		}
		t.Logf("%s: topology found %d links, want %d, %d findings",
			targets.Pack, len(drawn), len(targets.Links), len(findings))
	})

	t.Run("alerts", func(t *testing.T) {
		checkAlerts(t, db, store, targets, manifest, poller, due)
	})
}

// packInputs reads what scripts/snmp-acceptance-niac.sh hands the test, and
// stops it when any part would leave a check asserting nothing.
func packInputs(t *testing.T) (packTargets, packManifest, string, time.Time) {
	t.Helper()
	targetsPath, manifestPath := os.Getenv("SEED_NIAC_TARGETS"), os.Getenv("SEED_NIAC_MANIFEST")
	poller := os.Getenv("SEED_NIAC_POLLER_MAC")
	faultAt, faultAtErr := strconv.ParseInt(os.Getenv("SEED_NIAC_FAULT_AT"), 10, 64)
	if targetsPath == "" || manifestPath == "" || poller == "" || faultAtErr != nil {
		t.Fatal("SEED_NIAC_TARGETS, SEED_NIAC_MANIFEST, SEED_NIAC_POLLER_MAC or SEED_NIAC_FAULT_AT is unset; " +
			"run scripts/snmp-acceptance-niac.sh")
	}
	var targets packTargets
	if err := readJSON(targetsPath, &targets); err != nil {
		t.Fatalf("read targets: %v", err)
	}
	var manifest packManifest
	if err := readJSON(manifestPath, &manifest); err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest.SchemaVersion < manifestSchemaVersion || len(manifest.Expected) == 0 {
		t.Fatalf("manifest schema %d promises no observations; this run would assert nothing",
			manifest.SchemaVersion)
	}
	if len(targets.Agents) == 0 {
		t.Fatal("the pack has no SNMP agents; this run would assert nothing")
	}
	if len(targets.Links) == 0 {
		t.Fatal("the pack links none of its SNMP agents; the topology check would assert nothing")
	}
	if targets.Fault.Device == "" || targets.Fault.Interface == "" {
		t.Fatal("the harness names no fault; the alert check would assert nothing")
	}
	return targets, manifest, poller, time.Unix(faultAt, 0)
}

// checkAlerts runs seed's observation alert pipeline over the poll above, then
// again after the fault NIAC injects at due, and compares what it raised with
// that fault.
func checkAlerts(
	t *testing.T, db *database.DB, store orchestrator.Publisher,
	targets packTargets, manifest packManifest, poller string, due time.Time,
) {
	t.Helper()
	watcher, err := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: db.SNMPObservations(), Alerts: db.Alerts(), Settings: db.Settings(),
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("observation pipeline: %v", err)
	}
	// The first pass learns every interface's state from the poll above;
	// the second sees the fault, which is due once the first is done.
	if scanErr := watcher.ScanOnce(t.Context()); scanErr != nil {
		t.Fatalf("baseline scan: %v", scanErr)
	}
	if !time.Now().Before(due) {
		t.Fatalf("the baseline ended %s after the fault was due; start the fault later",
			time.Since(due).Round(time.Second))
	}
	// The margin is the run's timing budget: the nightly job's log shows it
	// shrinking long before a slow runner turns it into the failure above.
	t.Logf("%s: baseline done %s before the fault was due", targets.Pack, time.Until(due).Round(time.Second))
	time.Sleep(time.Until(due.Add(faultGrace)))

	// Every agent again, not just the faulted one: an interface the fault
	// did not touch going down is as much a finding as the faulted one
	// staying up, and an agent the fault cut off fails its collect.
	ifTable := map[string]packObservation{"if_table": manifest.Expected["if_table"]}
	findings := packFindings(ifTable, tallyResults(
		pollPack(t.Context(), targets.Agents, poller, store, "if_table")))
	if scanErr := watcher.ScanOnce(t.Context()); scanErr != nil {
		t.Fatalf("fault scan: %v", scanErr)
	}
	raised, err := db.Alerts().List(t.Context(), alerts.ListOptions{})
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	findings = append(findings, alertFindings(targets.Fault, raised)...)
	for _, finding := range findings {
		t.Errorf("%s: %s", targets.Pack, finding)
	}
	t.Logf("%s: alerts with %s %s down: %d raised, %d findings",
		targets.Pack, targets.Fault.Device, targets.Fault.Interface, len(raised), len(findings))
}

// reconcileTopology runs the topology reconcilers once each, in the order the
// daemon registers them (internal/api initTopologyReconcilers). One pass
// reads at most 500 observations of a kind and a pack stores one per agent; a
// pack past that would show up as missing links, not as a pass.
func reconcileTopology(t *testing.T, db *database.DB) {
	t.Helper()
	obs, topo, settings := db.SNMPObservations(), db.Topology(), db.Settings()
	logger := slog.New(slog.DiscardHandler)
	sysInfo, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: obs, Nodes: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("sysinfo reconciler: %v", err)
	}
	ifTable, err := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("iftable reconciler: %v", err)
	}
	edge, err := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("edge reconciler: %v", err)
	}
	arp, err := topology.NewARPReconciler(topology.ARPConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	})
	if err != nil {
		t.Fatalf("arp reconciler: %v", err)
	}
	for _, reconciler := range []interface {
		Name() string
		ReconcileOnce(ctx context.Context) error
	}{sysInfo, ifTable, edge, arp} {
		if reconcileErr := reconciler.ReconcileOnce(t.Context()); reconcileErr != nil {
			t.Fatalf("%s: %v", reconciler.Name(), reconcileErr)
		}
	}
}

// drawnLinks reads back every link the reconcilers drew, named by the agents
// at either end. An agent with no node, or a link to a node no agent owns, is
// a finding: neither can be compared with the pack.
func drawnLinks(ctx context.Context, db *database.DB, agents []packAgent) ([]packLink, []string) {
	var findings []string
	agentByNode := make(map[string]string, len(agents))
	for _, agent := range agents {
		nodeID, err := db.Topology().NodeIDForTarget(ctx, database.DefaultClientID, agent.Name)
		if err != nil {
			findings = append(findings, fmt.Sprintf("topology: no node for agent %s: %v", agent.Name, err))
			continue
		}
		agentByNode[nodeID] = agent.Name
	}
	seen := make(map[string]bool)
	var drawn []packLink
	for _, nodeID := range slices.Sorted(maps.Keys(agentByNode)) {
		links, err := db.Topology().ListLinks(ctx, nodeID)
		if err != nil {
			findings = append(findings, fmt.Sprintf("topology: list links of %s: %v", agentByNode[nodeID], err))
			continue
		}
		for _, link := range links {
			if seen[link.ID] {
				continue
			}
			seen[link.ID] = true
			source, sourceKnown := agentByNode[link.SourceNodeID]
			target, targetKnown := agentByNode[link.TargetNodeID]
			if !sourceKnown || !targetKnown {
				findings = append(findings, fmt.Sprintf("topology: %s link %s joins a node no agent owns",
					link.LinkType, link.ID))
				continue
			}
			drawn = append(drawn, newPackLink(source, target))
		}
	}
	return drawn, findings
}

// primaryMACFindings reports every agent seed knows an interface MAC for
// whose node still serves no primaryMac (seed#2574).
func primaryMACFindings(t *testing.T, db *database.DB, agents []packAgent) []string {
	t.Helper()
	ctx := t.Context()
	nodes, err := db.Topology().List(ctx, topology.ListOptions{Limit: 5000})
	if err != nil {
		return []string{fmt.Sprintf("topology: list nodes: %v", err)}
	}
	primaryMAC := make(map[string]string, len(nodes))
	for _, n := range nodes {
		primaryMAC[n.ID] = n.PrimaryMAC
	}
	var findings []string
	withMAC := 0
	for _, agent := range agents {
		nodeID, lookupErr := db.Topology().NodeIDForTarget(ctx, database.DefaultClientID, agent.Name)
		if lookupErr != nil {
			continue // drawnLinks already reports an agent with no node
		}
		ifaces, listErr := db.Topology().ListInterfaces(ctx, nodeID)
		if listErr != nil {
			findings = append(findings, fmt.Sprintf("topology: list interfaces of %s: %v", agent.Name, listErr))
			continue
		}
		if !slices.ContainsFunc(ifaces, func(i *topology.Interface) bool {
			return i.IfPhysAddr != "" && i.IfPhysAddr != "00:00:00:00:00:00"
		}) {
			continue
		}
		withMAC++
		if primaryMAC[nodeID] == "" {
			findings = append(findings, fmt.Sprintf("topology: %s has an interface MAC but no primaryMac", agent.Name))
		}
	}
	t.Logf("topology: %d agents with an interface MAC, %d without a primaryMac", withMAC, len(findings))
	return findings
}

// pollPack runs the named collectors, or every collector when none is named,
// against every agent with the production client factory, handing every
// observation on to store.
func pollPack(
	ctx context.Context, agents []packAgent, poller string, store orchestrator.Publisher, only ...string,
) []packResult {
	factory := snmpclient.NewFactory(snmpclient.Options{})
	queue := make(chan packAgent)
	var (
		mu      sync.Mutex
		results []packResult
		wg      sync.WaitGroup
	)
	for range packPollers {
		wg.Go(func() {
			recorder := &rowRecorder{poller: poller, next: store}
			collectors := orchestrator.Collectors(factory, recorder, nil)
			if len(only) > 0 {
				collectors = slices.DeleteFunc(collectors, func(c snmp.Collector) bool {
					return !slices.Contains(only, c.Name())
				})
			}
			for agent := range queue {
				target := snmp.Target{
					ID: agent.Name, ClientID: database.DefaultClientID, Name: agent.Name,
					IPAddress: agent.Address, SNMPVersion: "2c",
				}
				creds := snmp.ResolvedCredentials{SNMPCommunity: agent.Community}
				for _, collector := range collectors {
					recorder.rows = 0
					collectCtx, cancel := context.WithTimeout(ctx, packCollectTimeout)
					err := collector.Collect(collectCtx, target, creds)
					cancel()
					mu.Lock()
					results = append(results, packResult{
						Collector: collector.Name(), Agent: agent.Name, Rows: recorder.rows, Err: err,
					})
					mu.Unlock()
				}
			}
		})
	}
	for _, agent := range agents {
		queue <- agent
	}
	close(queue)
	wg.Wait()
	return results
}

// collectorNames lists the collectors a tally or manifest mentions, for the
// report, in one stable order.
func collectorNames(expected map[string]packObservation, tallies map[string]packTally) []string {
	names := slices.Collect(maps.Keys(tallies))
	for name := range expected {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}
