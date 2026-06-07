package topology_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

type fakeARPStore struct {
	mu sync.Mutex

	targetMap map[string]string // (client|target) -> node_id
	macMap    map[string]string // mac -> node_id

	bindings   []*database.TopologyARPBinding
	primaryIPs map[string]string // node_id -> ip

	upsertErr error
	setIPErr  error
}

func newFakeARPStore() *fakeARPStore {
	return &fakeARPStore{
		targetMap:  map[string]string{},
		macMap:     map[string]string{},
		primaryIPs: map[string]string{},
	}
}

func (f *fakeARPStore) NodeIDForTarget(_ context.Context, clientID, targetID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.targetMap[clientID+"|"+targetID]
	if !ok {
		return "", database.ErrTopologyNodeNotFound
	}
	return id, nil
}

func (f *fakeARPStore) NodeIDForMAC(_ context.Context, _, mac string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.macMap[mac]
	if !ok {
		return "", database.ErrTopologyNodeNotFound
	}
	return id, nil
}

func (f *fakeARPStore) UpsertARPBinding(_ context.Context, b *database.TopologyARPBinding) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.bindings = append(f.bindings, b)
	return nil
}

func (f *fakeARPStore) SetNodePrimaryIP(_ context.Context, nodeID, ip string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setIPErr != nil {
		return f.setIPErr
	}
	f.primaryIPs[nodeID] = ip
	return nil
}

func arpObs(target string, observed time.Time, entries []map[string]any) *database.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Entries": entries})
	return &database.SNMPObservation{
		ClientID:    "c",
		TargetID:    target,
		Kind:        "arp",
		ObservedAt:  observed,
		PayloadJSON: string(b),
	}
}

func TestNewARPReconciler_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  topology.ARPConfig
	}{
		{"missing Observations", topology.ARPConfig{Store: newFakeARPStore(), Settings: newFakeSettings()}},
		{"missing Store", topology.ARPConfig{Observations: &fakeObservations{}, Settings: newFakeSettings()}},
		{"missing Settings", topology.ARPConfig{Observations: &fakeObservations{}, Store: newFakeARPStore()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := topology.NewARPReconciler(tt.cfg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestARPReconcileOnce_UpsertsOneBindingPerEntry(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "10.0.0.1", "MACAddress": "aa:bb:cc:dd:ee:01", "MediaType": 3},
			{"IfIndex": 1, "IPAddress": "10.0.0.2", "MACAddress": "aa:bb:cc:dd:ee:02", "MediaType": 3},
		}),
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if len(store.bindings) != 2 {
		t.Fatalf("bindings = %d, want 2", len(store.bindings))
	}
	if store.bindings[0].SourceNodeID != "node-source" {
		t.Errorf("SourceNodeID = %q", store.bindings[0].SourceNodeID)
	}
	if store.bindings[0].IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %q", store.bindings[0].IPAddress)
	}
}

func TestARPReconcileOnce_BackfillsPrimaryIPOnMACMatch(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"
	// Node-B's primary MAC matches an ARP entry — its primary_ip
	// should be backfilled.
	store.macMap["aa:bb:cc:dd:ee:99"] = "node-B"

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "192.0.2.10", "MACAddress": "aa:bb:cc:dd:ee:99"},
		}),
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if got := store.primaryIPs["node-B"]; got != "192.0.2.10" {
		t.Errorf("primary_ip[node-B] = %q, want 192.0.2.10", got)
	}
}

func TestARPReconcileOnce_UnmatchedMACSkipsBackfill(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"
	// macMap deliberately empty — no node matches the binding's MAC.

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "10.0.0.5", "MACAddress": "11:22:33:44:55:66"},
		}),
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.primaryIPs) != 0 {
		t.Errorf("no MAC match should mean no backfills, got %d", len(store.primaryIPs))
	}
	// Binding should still be written.
	if len(store.bindings) != 1 {
		t.Errorf("binding should still be persisted, got %d", len(store.bindings))
	}
}

func TestARPReconcileOnce_SourceNotMappedSkips(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore() // empty target map -> source unknown

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-orphan", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "10.0.0.1", "MACAddress": "aa:aa:aa:aa:aa:aa"},
		}),
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.bindings) != 0 {
		t.Errorf("source-unknown observation should yield zero bindings, got %d",
			len(store.bindings))
	}
}

func TestARPReconcileOnce_EmptyIPOrMACSkipped(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "10.0.0.1", "MACAddress": ""},                  // bad
			{"IfIndex": 1, "IPAddress": "", "MACAddress": "aa:bb:cc:dd:ee:ff"},         // bad
			{"IfIndex": 1, "IPAddress": "10.0.0.2", "MACAddress": "11:22:33:44:55:66"}, // good
		}),
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(store.bindings) != 1 {
		t.Errorf("empty IP/MAC rows should be skipped, got %d bindings", len(store.bindings))
	}
}

func TestARPReconcileOnce_UpsertErrorSkipsBackfill(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"
	store.macMap["aa:aa:aa:aa:aa:aa"] = "node-B"
	store.upsertErr = errors.New("constraint")

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "10.0.0.1", "MACAddress": "aa:aa:aa:aa:aa:aa"},
		}),
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	// Backfill must not run if the binding upsert failed.
	if _, ok := store.primaryIPs["node-B"]; ok {
		t.Error("backfill should not happen when upsert errored")
	}
}

func TestARPReconcileOnce_MalformedPayloadSkipsObservation(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"
	o := &fakeObservations{rows: []*database.SNMPObservation{
		{
			ClientID: "c", TargetID: "t-1", Kind: "arp",
			ObservedAt: at(), PayloadJSON: "{not valid",
		},
	}}
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("malformed should not abort: %v", err)
	}
	if len(store.bindings) != 0 {
		t.Errorf("malformed payload should yield zero bindings, got %d", len(store.bindings))
	}
}

func TestARPReconcileOnce_PersistsHighWater(t *testing.T) {
	t.Parallel()
	store := newFakeARPStore()
	store.targetMap["c|t-1"] = "node-source"

	o := &fakeObservations{rows: []*database.SNMPObservation{
		arpObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IPAddress": "10.0.0.1", "MACAddress": "aa:bb:cc:dd:ee:ff"},
		}),
	}}
	s := newFakeSettings()
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: o, Store: store, Settings: s,
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	hw, _ := s.GetWithDefault(context.Background(), "topology.arp.high_water", "")
	parsed, perr := time.Parse(time.RFC3339Nano, hw)
	if perr != nil {
		t.Fatalf("parse high-water: %v", perr)
	}
	if !parsed.Equal(at()) {
		t.Errorf("high-water = %v, want %v", parsed, at())
	}
}

func TestARPReconciler_EngineName(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: &fakeObservations{},
		Store:        newFakeARPStore(),
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if r.Name() != topology.ARPReconcilerName {
		t.Errorf("Name() = %q, want %q", r.Name(), topology.ARPReconcilerName)
	}
}

func TestARPStartStop_Idempotent(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewARPReconciler(topology.ARPConfig{
		Observations: &fakeObservations{},
		Store:        newFakeARPStore(),
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
