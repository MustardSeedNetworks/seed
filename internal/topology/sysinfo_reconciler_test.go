package topology_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/topology"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
func at() time.Time              { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

type fakeObservations struct {
	mu      sync.Mutex
	rows    []*database.SNMPObservation
	listErr error
}

func (f *fakeObservations) List(_ context.Context, _ database.ListOptions) ([]*database.SNMPObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*database.SNMPObservation, len(f.rows))
	copy(out, f.rows)
	return out, nil
}

type fakeNodes struct {
	mu        sync.Mutex
	upserts   []*database.TopologyNode
	upsertErr error
}

func (f *fakeNodes) Upsert(_ context.Context, node *database.TopologyNode) (*database.TopologyNode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	f.upserts = append(f.upserts, node)
	return node, nil
}

type fakeSettings struct {
	mu     sync.Mutex
	values map[string]string
}

func newFakeSettings() *fakeSettings {
	return &fakeSettings{values: make(map[string]string)}
}

func (f *fakeSettings) GetWithDefault(_ context.Context, key, def string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.values[key]
	if !ok {
		return def, nil
	}
	return v, nil
}

func (f *fakeSettings) Set(_ context.Context, key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = value
	return nil
}

func payload(client, target, sysName, sysObjectID string) string {
	b, _ := json.Marshal(map[string]string{
		"ClientID":    client,
		"TargetID":    target,
		"SysName":     sysName,
		"SysObjectID": sysObjectID,
		"SysDescr":    "Test Device",
	})
	return string(b)
}

func obs(client, target, sysName, sysObjectID string, observed time.Time) *database.SNMPObservation {
	return &database.SNMPObservation{
		ClientID:    client,
		TargetID:    target,
		Kind:        "sys_info",
		ObservedAt:  observed,
		PayloadJSON: payload(client, target, sysName, sysObjectID),
	}
}

func TestNew_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  topology.Config
	}{
		{"missing Observations", topology.Config{Nodes: &fakeNodes{}, Settings: newFakeSettings()}},
		{"missing Nodes", topology.Config{Observations: &fakeObservations{}, Settings: newFakeSettings()}},
		{"missing Settings", topology.Config{Observations: &fakeObservations{}, Nodes: &fakeNodes{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := topology.NewSysInfoReconciler(tt.cfg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestNew_EngineName(t *testing.T) {
	t.Parallel()
	r, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: &fakeObservations{},
		Nodes:        &fakeNodes{},
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r.Name() != topology.SysInfoReconcilerName {
		t.Errorf("Name() = %q, want %q", r.Name(), topology.SysInfoReconcilerName)
	}
}

func TestReconcileOnce_UpsertsOneNodePerObservation(t *testing.T) {
	t.Parallel()
	o := &fakeObservations{rows: []*database.SNMPObservation{
		obs("client-a", "t-1", "router-1", "1.3.6.1.4.1.9.1.123", at()),
		obs("client-a", "t-2", "router-2", "1.3.6.1.4.1.9.1.456", at()),
	}}
	n := &fakeNodes{}
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: o, Nodes: n, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if len(n.upserts) != 2 {
		t.Fatalf("upserts = %d, want 2", len(n.upserts))
	}
	if n.upserts[0].DeviceType != "cisco" {
		t.Errorf("expected cisco DeviceType from 1.3.6.1.4.1.9 OID, got %q",
			n.upserts[0].DeviceType)
	}
	if n.upserts[0].SysName != "router-1" {
		t.Errorf("SysName = %q", n.upserts[0].SysName)
	}
}

func TestReconcileOnce_PersistsHighWaterMark(t *testing.T) {
	t.Parallel()
	old := at().Add(-1 * time.Hour)
	newer := at()
	o := &fakeObservations{rows: []*database.SNMPObservation{
		obs("c", "t-1", "h-1", "1.3.6.1.4.1.9.1.1", newer),
		obs("c", "t-2", "h-2", "1.3.6.1.4.1.9.1.2", old),
	}}
	s := newFakeSettings()
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: o, Nodes: &fakeNodes{}, Settings: s,
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	saved, _ := s.GetWithDefault(context.Background(), "topology.sysinfo.high_water", "")
	parsed, perr := time.Parse(time.RFC3339Nano, saved)
	if perr != nil {
		t.Fatalf("parse high water %q: %v", saved, perr)
	}
	if !parsed.Equal(newer) {
		t.Errorf("high-water = %v, want %v (newest of batch)", parsed, newer)
	}
}

func TestReconcileOnce_IdentityHashMergesObservationsFromSameDevice(t *testing.T) {
	t.Parallel()
	// Same client + sysName + sysObjectID across two observations
	// at different times -> same identity_hash -> same node.
	o := &fakeObservations{rows: []*database.SNMPObservation{
		obs("c", "t-1", "router-1", "1.3.6.1.4.1.9.1.1", at().Add(-1*time.Minute)),
		obs("c", "t-1", "router-1", "1.3.6.1.4.1.9.1.1", at()),
	}}
	n := &fakeNodes{}
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: o, Nodes: n, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if len(n.upserts) != 2 {
		t.Fatalf("upserts = %d, want 2", len(n.upserts))
	}
	if n.upserts[0].IdentityHash != n.upserts[1].IdentityHash {
		t.Errorf("same device should hash identically; got %q vs %q",
			n.upserts[0].IdentityHash, n.upserts[1].IdentityHash)
	}
}

func TestReconcileOnce_DifferentClientsGetDifferentNodes(t *testing.T) {
	t.Parallel()
	o := &fakeObservations{rows: []*database.SNMPObservation{
		obs("client-a", "t-1", "router-1", "1.3.6.1.4.1.9.1.1", at()),
		obs("client-b", "t-1", "router-1", "1.3.6.1.4.1.9.1.1", at()),
	}}
	n := &fakeNodes{}
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: o, Nodes: n, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())

	if n.upserts[0].IdentityHash == n.upserts[1].IdentityHash {
		t.Error("two clients with identical sysName/OID must not collapse to one node")
	}
}

func TestReconcileOnce_EmptyObservationsListIsNoOp(t *testing.T) {
	t.Parallel()
	s := newFakeSettings()
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: &fakeObservations{},
		Nodes:        &fakeNodes{},
		Settings:     s,
		Logger:       silentLogger(),
		Now:          at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("empty list: %v", err)
	}
	v, _ := s.GetWithDefault(context.Background(), "topology.sysinfo.high_water", "")
	if v != "" {
		t.Errorf("high-water should not be touched on empty batch, got %q", v)
	}
}

func TestReconcileOnce_ListErrorPropagates(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: &fakeObservations{listErr: errors.New("db down")},
		Nodes:        &fakeNodes{},
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if err := r.ReconcileOnce(context.Background()); err == nil {
		t.Error("expected list error to propagate")
	}
}

func TestReconcileOnce_UpsertFailureContinuesBatch(t *testing.T) {
	t.Parallel()
	o := &fakeObservations{rows: []*database.SNMPObservation{
		obs("c", "t-1", "h-1", "1.3.6.1.4.1.9.1.1", at()),
		obs("c", "t-2", "h-2", "1.3.6.1.4.1.9.1.2", at()),
	}}
	// upsertErr fails the first call only? our fakeNodes always
	// errors when upsertErr is set; that's fine for "all rows
	// fail" coverage. The contract is: a per-row failure does
	// not abort the batch — high-water still advances.
	n := &fakeNodes{upsertErr: errors.New("constraint")}
	s := newFakeSettings()
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: o, Nodes: n, Settings: s,
		Logger: silentLogger(), Now: at,
	})
	if err := r.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce should continue past row errors; got %v", err)
	}
	saved, _ := s.GetWithDefault(context.Background(), "topology.sysinfo.high_water", "")
	if saved == "" {
		t.Error("high-water should still advance even when every upsert fails")
	}
}

func TestReconcileOnce_SkipsObservationWithoutIdentity(t *testing.T) {
	t.Parallel()
	// Empty SysName + SysObjectID — buildNode rejects this.
	o := &fakeObservations{rows: []*database.SNMPObservation{
		obs("c", "t-1", "", "", at()),
	}}
	n := &fakeNodes{}
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: o, Nodes: n, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = r.ReconcileOnce(context.Background())
	if len(n.upserts) != 0 {
		t.Errorf("expected zero upserts for empty-identity row, got %d", len(n.upserts))
	}
}

func TestStartStop_Idempotent(t *testing.T) {
	t.Parallel()
	r, _ := topology.NewSysInfoReconciler(topology.Config{
		Observations: &fakeObservations{},
		Nodes:        &fakeNodes{},
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
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := r.Stop(stopCtx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}
