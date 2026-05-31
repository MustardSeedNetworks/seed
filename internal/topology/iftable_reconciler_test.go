package topology_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/topology"
)

type fakeIfStore struct {
	mu        sync.Mutex
	nodeFor   map[string]string // (client|target) -> nodeID
	lookupErr error
	upserts   []*database.TopologyInterface
	upsertErr error
}

func newFakeIfStore() *fakeIfStore {
	return &fakeIfStore{nodeFor: make(map[string]string)}
}

func (f *fakeIfStore) NodeIDForTarget(_ context.Context, clientID, targetID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.lookupErr != nil {
		return "", f.lookupErr
	}
	id, ok := f.nodeFor[clientID+"|"+targetID]
	if !ok {
		return "", database.ErrTopologyNodeNotFound
	}
	return id, nil
}

func (f *fakeIfStore) UpsertInterface(_ context.Context, iface *database.TopologyInterface) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserts = append(f.upserts, iface)
	return nil
}

func ifPayload(rows ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"Rows": rows})
	return string(b)
}

func ifObs(client, target string, observed time.Time, rows ...map[string]any) *database.SNMPObservation {
	return &database.SNMPObservation{
		ClientID:    client,
		TargetID:    target,
		Kind:        "if_table",
		ObservedAt:  observed,
		PayloadJSON: ifPayload(rows...),
	}
}

func TestNewIfTableReconciler_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  topology.IfTableConfig
	}{
		{"missing Observations", topology.IfTableConfig{Store: newFakeIfStore(), Settings: newFakeSettings()}},
		{"missing Store", topology.IfTableConfig{Observations: &fakeObservations{}, Settings: newFakeSettings()}},
		{"missing Settings", topology.IfTableConfig{Observations: &fakeObservations{}, Store: newFakeIfStore()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := topology.NewIfTableReconciler(tt.cfg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestIfTableReconcileOnce_UpsertsOneRowPerInterface(t *testing.T) {
	t.Parallel()
	store := newFakeIfStore()
	store.nodeFor["client-a|t-1"] = "node-abc"

	o := &fakeObservations{rows: []*database.SNMPObservation{
		ifObs("client-a", "t-1", at(),
			map[string]any{
				"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 1, "SpeedBps": 1000000000,
			},
			map[string]any{
				"IfIndex": 2, "IfName": "Gi0/1", "IfAdmin": 1, "IfOper": 2, "SpeedBps": 1000000000,
			},
		),
	}}
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if len(store.upserts) != 2 {
		t.Fatalf("upserts = %d, want 2", len(store.upserts))
	}
	if store.upserts[0].NodeID != "node-abc" {
		t.Errorf("NodeID = %q, want node-abc", store.upserts[0].NodeID)
	}
	if store.upserts[0].IfName != "Gi0/0" {
		t.Errorf("IfName = %q", store.upserts[0].IfName)
	}
}

func TestIfTableReconcileOnce_SkipsObservationsWithoutNodeMapping(t *testing.T) {
	t.Parallel()
	store := newFakeIfStore() // empty mapping
	o := &fakeObservations{rows: []*database.SNMPObservation{
		ifObs("c", "t-orphan", at(),
			map[string]any{"IfIndex": 1, "IfName": "eth0"},
		),
	}}
	settings := newFakeSettings()
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: o, Store: store, Settings: settings,
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if len(store.upserts) != 0 {
		t.Errorf("expected zero upserts for orphan target, got %d", len(store.upserts))
	}
	// High-water still advances so we don't reprocess every poll
	// forever waiting for sysinfo to catch up.
	hw, _ := settings.GetWithDefault(context.Background(), "topology.iftable.high_water", "")
	if hw == "" {
		t.Error("high-water should advance even when target has no node mapping")
	}
}

func TestIfTableReconcileOnce_PersistsMaxObservedAt(t *testing.T) {
	t.Parallel()
	store := newFakeIfStore()
	store.nodeFor["c|t-1"] = "node-1"
	store.nodeFor["c|t-2"] = "node-2"

	old := at().Add(-time.Hour)
	newer := at()
	o := &fakeObservations{rows: []*database.SNMPObservation{
		ifObs("c", "t-1", newer, map[string]any{"IfIndex": 1, "IfName": "e0"}),
		ifObs("c", "t-2", old, map[string]any{"IfIndex": 1, "IfName": "e0"}),
	}}
	s := newFakeSettings()
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: o, Store: store, Settings: s,
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	hw, _ := s.GetWithDefault(context.Background(), "topology.iftable.high_water", "")
	parsed, perr := time.Parse(time.RFC3339Nano, hw)
	if perr != nil {
		t.Fatalf("parse high water: %v", perr)
	}
	if !parsed.Equal(newer) {
		t.Errorf("high-water = %v, want %v", parsed, newer)
	}
}

func TestIfTableReconcileOnce_MalformedPayloadSkipsObservation(t *testing.T) {
	t.Parallel()
	store := newFakeIfStore()
	store.nodeFor["c|t-1"] = "node-1"
	o := &fakeObservations{rows: []*database.SNMPObservation{
		{
			ClientID: "c", TargetID: "t-1", Kind: "if_table",
			ObservedAt: at(), PayloadJSON: "{not valid json",
		},
	}}
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("malformed should not abort batch; got %v", err)
	}
	if len(store.upserts) != 0 {
		t.Errorf("malformed payload should yield zero upserts, got %d", len(store.upserts))
	}
}

func TestIfTableReconcileOnce_ZeroIfIndexSkipped(t *testing.T) {
	t.Parallel()
	store := newFakeIfStore()
	store.nodeFor["c|t-1"] = "node-1"
	o := &fakeObservations{rows: []*database.SNMPObservation{
		ifObs("c", "t-1", at(),
			map[string]any{"IfIndex": 0, "IfName": "invalid"},
			map[string]any{"IfIndex": 1, "IfName": "ok"},
		),
	}}
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())
	if len(store.upserts) != 1 || store.upserts[0].IfName != "ok" {
		t.Errorf("ifIndex 0 should be skipped, got upserts = %+v", store.upserts)
	}
}

func TestIfTableReconcileOnce_LookupErrorContinuesBatch(t *testing.T) {
	t.Parallel()
	store := newFakeIfStore()
	store.lookupErr = errors.New("db down")
	o := &fakeObservations{rows: []*database.SNMPObservation{
		ifObs("c", "t-1", at(), map[string]any{"IfIndex": 1, "IfName": "e0"}),
	}}
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: o, Store: store, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Errorf("lookup error should not abort batch, got %v", err)
	}
}

func TestIfTableReconcileOnce_ListErrorPropagates(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: &fakeObservations{listErr: errors.New("db down")},
		Store:        newFakeIfStore(),
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if err := r.ReconcileOnce(context.Background()); err == nil {
		t.Error("expected list error to propagate")
	}
}

func TestIfTableStartStop_Idempotent(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: &fakeObservations{},
		Store:        newFakeIfStore(),
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

func TestIfTableReconciler_EngineName(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: &fakeObservations{},
		Store:        newFakeIfStore(),
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if r.Name() != topology.IfTableReconcilerName {
		t.Errorf("Name() = %q, want %q", r.Name(), topology.IfTableReconcilerName)
	}
}
