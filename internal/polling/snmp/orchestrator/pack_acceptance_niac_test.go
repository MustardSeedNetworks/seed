//go:build niacacceptance

// Built only by scripts/snmp-acceptance-niac.sh, which starts the pack this
// needs. It has its own tag because the nightly `integration` job fails on a
// skipped test, and without a running pack there is nothing to run.

package orchestrator_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpclient"
)

// packPollers bounds how many agents are polled at once. Each runs its ten
// collectors in sequence, so this is also the number of requests in flight.
const packPollers = 16

// packCollectTimeout bounds one collector on one agent. The largest table in
// a pack is an FDB of a few hundred rows; a minute is generous for that and
// still ends a run against a pack that has stopped answering.
const packCollectTimeout = time.Minute

// packTargets is what scripts/snmp-acceptance-niac.sh reads out of a
// generated pack: every device that runs an SNMP agent, at its first address.
type packTargets struct {
	Pack   string      `json:"pack"`
	Agents []packAgent `json:"agents"`
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

// TestNIACPackObservations is the S4-2 acceptance: seed's ten collectors,
// over the wire, against every SNMP agent of a running NIAC pack, compared
// with the pack's manifest. scripts/snmp-acceptance-niac.sh starts the pack
// and runs this with SEED_NIAC_TARGETS and SEED_NIAC_MANIFEST set.
func TestNIACPackObservations(t *testing.T) {
	targetsPath, manifestPath := os.Getenv("SEED_NIAC_TARGETS"), os.Getenv("SEED_NIAC_MANIFEST")
	if targetsPath == "" || manifestPath == "" {
		t.Fatal("SEED_NIAC_TARGETS and SEED_NIAC_MANIFEST are unset; run scripts/snmp-acceptance-niac.sh")
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

	results := pollPack(t.Context(), targets.Agents)
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
}

// pollPack runs every collector against every agent with the production
// client factory.
func pollPack(ctx context.Context, agents []packAgent) []packResult {
	factory := snmpclient.NewFactory(snmpclient.Options{})
	queue := make(chan packAgent)
	var (
		mu      sync.Mutex
		results []packResult
		wg      sync.WaitGroup
	)
	for range packPollers {
		wg.Go(func() {
			recorder := &rowRecorder{}
			collectors := orchestrator.Collectors(factory, recorder, nil)
			for agent := range queue {
				target := snmp.Target{
					ID: agent.Name, ClientID: "niac-acceptance", Name: agent.Name,
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
