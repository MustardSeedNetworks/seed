package topology_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

type fakeEdgeStore struct {
	mu sync.Mutex

	// (client|target) -> node_id for sysinfo-resolved sources.
	targetMap map[string]string
	// sys_name -> node_id for remote-neighbor resolution.
	sysNameMap map[string]string

	links     []*topology.Link
	upsertErr error
}

func newFakeEdgeStore() *fakeEdgeStore {
	return &fakeEdgeStore{
		targetMap:  map[string]string{},
		sysNameMap: map[string]string{},
	}
}

func (f *fakeEdgeStore) NodeIDForTarget(_ context.Context, clientID, targetID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.targetMap[clientID+"|"+targetID]
	if !ok {
		return "", topology.ErrTopologyNodeNotFound
	}
	return id, nil
}

func (f *fakeEdgeStore) NodeIDForSysName(_ context.Context, _, sysName string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.sysNameMap[sysName]
	if !ok {
		return "", topology.ErrTopologyNodeNotFound
	}
	return id, nil
}

func (f *fakeEdgeStore) UpsertLink(_ context.Context, link *topology.Link) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.links = append(f.links, link)
	return nil
}

// All edge tests use client "c" for simplicity; multi-client
// isolation is covered by the sysinfo reconciler tests where the
// client_id matters for identity merging.
const edgeTestClient = "c"

func lldpObs(target string, observed time.Time, neighbors []map[string]any) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Neighbors": neighbors})
	return &observation.SNMPObservation{
		ClientID: edgeTestClient, TargetID: target, Kind: "lldp",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

func cdpObs(target, kind string, observed time.Time, neighbors []map[string]any) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Neighbors": neighbors})
	return &observation.SNMPObservation{
		ClientID: edgeTestClient, TargetID: target, Kind: kind,
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

func TestNewEdgeReconciler_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  topology.EdgeConfig
	}{
		{"missing Observations", topology.EdgeConfig{Store: newFakeEdgeStore(), Settings: newFakeSettings()}},
		{"missing Store", topology.EdgeConfig{Observations: &fakeObservations{}, Settings: newFakeSettings()}},
		{"missing Settings", topology.EdgeConfig{Observations: &fakeObservations{}, Store: newFakeEdgeStore()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := topology.NewEdgeReconciler(tt.cfg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestEdgeReconcileOnce_LLDPEmitsOneLinkPerNeighbor(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-source"] = "node-A"
	store.sysNameMap["core-sw"] = "node-B"

	o := &fakeObservations{rows: []*observation.SNMPObservation{
		lldpObs("t-source", at(), []map[string]any{
			{"LocalPortNum": 24, "PortID": "Gi0/24", "SysName": "core-sw"},
		}),
	}}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if len(store.links) != 1 {
		t.Fatalf("links = %d, want 1", len(store.links))
	}
	link := store.links[0]
	if link.LinkType != "lldp" {
		t.Errorf("LinkType = %q", link.LinkType)
	}
	// Either endpoint order acceptable as long as both nodes are present.
	if (link.SourceNodeID != "node-A" || link.TargetNodeID != "node-B") &&
		(link.SourceNodeID != "node-B" || link.TargetNodeID != "node-A") {
		t.Errorf("endpoints = (%s, %s)", link.SourceNodeID, link.TargetNodeID)
	}
}

func TestEdgeReconcileOnce_OrphanNeighborSkipped(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-source"] = "node-A"
	// sysNameMap deliberately empty — remote unknown.

	o := &fakeObservations{rows: []*observation.SNMPObservation{
		lldpObs("t-source", at(), []map[string]any{
			{"LocalPortNum": 1, "SysName": "unknown-edge"},
		}),
	}}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.links) != 0 {
		t.Errorf("orphan remote should be skipped, got %d links", len(store.links))
	}
}

func TestEdgeReconcileOnce_SameLinkFromBothSidesMergesToOneRow(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-A"] = "node-A"
	store.targetMap["c|t-B"] = "node-B"
	store.sysNameMap["sw-A"] = "node-A"
	store.sysNameMap["sw-B"] = "node-B"

	// Two observations describing the same cable from each end.
	// Source A sees B as neighbor on local port 24; source B sees
	// A as neighbor on local port 12.
	// Because linkIDFor sorts endpoints lexicographically, both
	// upserts must hit the same ID — but interfaces differ between
	// the two views, so the resulting row will reflect whichever
	// arrived last (deterministic given high-water ordering).
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		lldpObs("t-A", at(), []map[string]any{
			{"LocalPortNum": 24, "PortID": "Gi0/12", "SysName": "sw-B"},
		}),
		lldpObs("t-B", at(), []map[string]any{
			{"LocalPortNum": 12, "PortID": "Gi0/24", "SysName": "sw-A"},
		}),
	}}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.links) != 2 {
		t.Fatalf("expected 2 upserts (one per side), got %d", len(store.links))
	}
	// Both upserts must share the same canonical link ID — that's
	// how the database upsert collapses them in production.
	if store.links[0].ID != store.links[1].ID {
		t.Errorf("two observations of the same cable should produce the same link ID; got %q vs %q",
			store.links[0].ID, store.links[1].ID)
	}
	if !strings.HasPrefix(store.links[0].ID, "link-") {
		t.Errorf("link ID should be prefixed link-, got %q", store.links[0].ID)
	}
}

func TestEdgeReconcileOnce_CDPAlsoCreatesLinks(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-A"] = "node-A"
	store.sysNameMap["fqdn-core-sw"] = "node-B"

	o := &fakeObservations{rows: []*observation.SNMPObservation{
		cdpObs("t-A", "cdp", at(), []map[string]any{
			{"LocalIfIndex": 5, "DeviceID": "fqdn-core-sw", "DevicePort": "Gi1/0/24"},
		}),
	}}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.links) != 1 {
		t.Fatalf("expected 1 link from cdp, got %d", len(store.links))
	}
	if store.links[0].LinkType != "cdp" {
		t.Errorf("LinkType = %q, want cdp", store.links[0].LinkType)
	}
}

func TestEdgeReconcileOnce_FDPRoutedThroughCDPPayloadShape(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-A"] = "node-A"
	store.sysNameMap["foundry-edge"] = "node-B"

	o := &fakeObservations{rows: []*observation.SNMPObservation{
		cdpObs("t-A", "fdp", at(), []map[string]any{
			{"LocalIfIndex": 1, "DeviceID": "foundry-edge", "DevicePort": "ether-1"},
		}),
	}}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.links) != 1 || store.links[0].LinkType != "fdp" {
		t.Errorf("expected one fdp link, got %+v", store.links)
	}
}

func TestEdgeReconcileOnce_SourceNotKnownSkips(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	// targetMap empty -> source unknown.
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		lldpObs("t-A", at(), []map[string]any{
			{"LocalPortNum": 1, "SysName": "core"},
		}),
	}}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())
	if len(store.links) != 0 {
		t.Errorf("unknown source should yield zero links, got %d", len(store.links))
	}
}

func TestEdgeReconcileOnce_ListErrorOnOneKindContinuesOthers(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-A"] = "node-A"
	store.sysNameMap["core"] = "node-B"

	// fakeObservations.List ignores kind filter — so a list error
	// kills all kinds at once. Use a per-kind capable fake.
	pc := &perKindObservations{
		byKind: map[string][]*observation.SNMPObservation{
			"lldp": {lldpObs("t-A", at(), []map[string]any{
				{"LocalPortNum": 1, "SysName": "core"},
			})},
		},
		errByKind: map[string]error{
			"cdp": errors.New("cdp index dead"),
		},
	}
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: pc, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("per-kind error should not abort the pass; got %v", err)
	}
	if len(store.links) != 1 {
		t.Errorf("expected lldp link despite cdp failure, got %d links", len(store.links))
	}
}

// perKindObservations is a fake that respects opts.Kind so we can
// test the per-kind error-isolation behavior.
type perKindObservations struct {
	mu        sync.Mutex
	byKind    map[string][]*observation.SNMPObservation
	errByKind map[string]error
}

func (p *perKindObservations) List(
	_ context.Context,
	opts observation.ListOptions,
) ([]*observation.SNMPObservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err, ok := p.errByKind[opts.Kind]; ok {
		return nil, err
	}
	return p.byKind[opts.Kind], nil
}

func TestEdgeReconcileOnce_MaxObservedAtAcrossKinds(t *testing.T) {
	t.Parallel()
	store := newFakeEdgeStore()
	store.targetMap["c|t-A"] = "node-A"
	store.sysNameMap["b"] = "node-B"

	older := at().Add(-time.Hour)
	newer := at()
	pc := &perKindObservations{byKind: map[string][]*observation.SNMPObservation{
		"lldp": {lldpObs("t-A", older, []map[string]any{{"LocalPortNum": 1, "SysName": "b"}})},
		"cdp":  {cdpObs("t-A", "cdp", newer, []map[string]any{{"LocalIfIndex": 1, "DeviceID": "b"}})},
	}}
	s := newFakeSettings()
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: pc, Store: store, Settings: s,
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	hw, _ := s.GetWithDefault(context.Background(), "topology.edge.high_water", "")
	parsed, perr := time.Parse(time.RFC3339Nano, hw)
	if perr != nil {
		t.Fatalf("parse high-water: %v", perr)
	}
	if !parsed.Equal(newer) {
		t.Errorf("high-water = %v, want %v (max across kinds)", parsed, newer)
	}
}

func TestEdgeReconciler_EngineName(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: &fakeObservations{},
		Store:        newFakeEdgeStore(),
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if r.Name() != topology.EdgeReconcilerName {
		t.Errorf("Name() = %q, want %q", r.Name(), topology.EdgeReconcilerName)
	}
}

func TestEdgeStartStop_Idempotent(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: &fakeObservations{},
		Store:        newFakeEdgeStore(),
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
		Interval:     500 * time.Millisecond,
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
