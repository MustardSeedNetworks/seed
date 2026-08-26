package iftable_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpwalkfile"
)

// The tests in iftable_test.go drive the collector through a hand-written fake
// that returns whatever the test author decided an agent emits. These replay
// recorded walks from eight vendors instead, so the collector meets the types,
// the sparse columns and the odd encodings those devices actually produced.

type capturingPublisher struct{ obs []iftable.Observation }

func (p *capturingPublisher) PublishIfTable(_ context.Context, o iftable.Observation) error {
	p.obs = append(p.obs, o)
	return nil
}

// walkFixtures returns every recorded walk, so adding a vendor to testdata
// extends coverage without editing this file.
func walkFixtures(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(
		"..", "..", "snmpwalkfile", "testdata", "*.walk"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no walk fixtures found; this test would assert nothing")
	}
	return paths
}

func collectFrom(t *testing.T, path string) iftable.Observation {
	t.Helper()

	walk, err := snmpwalkfile.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	client := walk.Client()

	pub := &capturingPublisher{}
	collector := iftable.New(
		func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) {
			return client, nil
		},
		pub,
		func() time.Time { return time.Unix(0, 0).UTC() },
	)

	target := snmp.Target{ID: "t1", ClientID: "c1", IPAddress: "192.0.2.1", SNMPVersion: "2c"}
	if collectErr := collector.Collect(
		context.Background(), target, snmp.ResolvedCredentials{},
	); collectErr != nil {
		t.Fatalf("Collect: %v", collectErr)
	}
	if len(pub.obs) != 1 {
		t.Fatalf("published %d observations, want 1", len(pub.obs))
	}
	return pub.obs[0]
}

// TestCollectFromRecordedWalks is the point of the fixtures: the collector runs
// against eight real devices and has to produce a coherent table for each.
func TestCollectFromRecordedWalks(t *testing.T) {
	for _, path := range walkFixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			obs := collectFrom(t, path)

			if len(obs.Rows) == 0 {
				t.Fatal("no interfaces decoded from a walk that carries an ifTable")
			}

			seen := make(map[uint32]bool, len(obs.Rows))
			for i, row := range obs.Rows {
				if seen[row.IfIndex] {
					t.Errorf("ifIndex %d appears twice; rows were merged wrongly",
						row.IfIndex)
				}
				seen[row.IfIndex] = true
				assertRowIsSane(t, i, row)
			}
		})
	}
}

// assertRowIsSane checks one decoded interface against IF-MIB's own ranges.
// A value outside them means a column was read from the wrong OID or decoded
// as the wrong type — the divergence a hand-written fake cannot show.
func assertRowIsSane(t *testing.T, i int, row iftable.Row) {
	t.Helper()

	if row.IfIndex == 0 {
		t.Errorf("row %d has ifIndex 0, which no interface uses", i)
	}
	if row.IfDescr == "" && row.IfName == "" {
		t.Errorf("ifIndex %d has neither a description nor a name", row.IfIndex)
	}
	if row.IfAdmin < 0 || row.IfAdmin > 3 {
		t.Errorf("ifIndex %d has ifAdminStatus %d, outside 1..3",
			row.IfIndex, row.IfAdmin)
	}
	if row.IfOper < 0 || row.IfOper > 7 {
		t.Errorf("ifIndex %d has ifOperStatus %d, outside 1..7",
			row.IfIndex, row.IfOper)
	}
}

// TestRowsAreSortedByIfIndex pins the ordering the Observation doc promises.
// Downstream alerting diffs consecutive observations, so unstable ordering
// would report interfaces as appearing and disappearing.
func TestRowsAreSortedByIfIndex(t *testing.T) {
	for _, path := range walkFixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			obs := collectFrom(t, path)
			for i := 1; i < len(obs.Rows); i++ {
				if obs.Rows[i-1].IfIndex >= obs.Rows[i].IfIndex {
					t.Fatalf("rows %d and %d are out of order: %d then %d",
						i-1, i, obs.Rows[i-1].IfIndex, obs.Rows[i].IfIndex)
				}
			}
		})
	}
}

// TestSpeedIsPlausible pins the ifSpeed/ifHighSpeed reconciliation against real
// data. ifSpeed is a 32-bit bits-per-second gauge that saturates at ~4.29 Gb/s,
// so anything faster reports through ifHighSpeed in megabits instead; a
// collector reading only one of them silently mis-states every fast port.
func TestSpeedIsPlausible(t *testing.T) {
	const maxPlausibleBps = uint64(800) * 1_000_000_000 // 800 Gb/s

	for _, path := range walkFixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			obs := collectFrom(t, path)
			var withSpeed int
			for _, row := range obs.Rows {
				if row.SpeedBps == 0 {
					continue // down or unreported, which is legitimate
				}
				withSpeed++
				if row.SpeedBps > maxPlausibleBps {
					t.Errorf("ifIndex %d reports %d bps, faster than any port "+
						"in the corpus — the wrong column was read",
						row.IfIndex, row.SpeedBps)
				}
			}
			if withSpeed == 0 {
				t.Log("no interface reported a speed; fixture may lack ifSpeed")
			}
		})
	}
}
