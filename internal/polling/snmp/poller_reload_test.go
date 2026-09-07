package snmp_test

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// An operator who adds, edits or removes a target through the API expects
// the running poller to follow. Before Reload existed the table was read once
// at Start, and 74 targets created on CT313 sat unpolled until a restart.
func TestPoller_Reload_FollowsTheTargetTable(t *testing.T) {
	a := &polling.Target{ID: "a", IPAddress: "10.0.0.1", PollIntervalSec: 60, Enabled: true}
	b := &polling.Target{ID: "b", IPAddress: "10.0.0.2", PollIntervalSec: 60, Enabled: true}
	store := &fakeStorage{targets: []*polling.Target{a}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(store, sched, nil)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	before, registered := sched.jobs["snmp:a"]
	if !registered {
		t.Fatal("start did not register a")
	}

	// b appears, a's interval changes.
	store.targets = []*polling.Target{
		{ID: "a", IPAddress: "10.0.0.1", PollIntervalSec: 30, Enabled: true},
		b,
	}
	if err := p.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := sched.jobs["snmp:b"]; !ok {
		t.Error("reload did not register the new target b")
	}
	if after := sched.jobs["snmp:a"]; after == before {
		t.Error("reload kept the old job for a although its interval changed")
	}

	// a disappears (deleted or disabled).
	store.targets = []*polling.Target{b}
	if err := p.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := sched.jobs["snmp:a"]; ok {
		t.Error("reload left a registered after it was removed from the table")
	}
	if len(sched.jobs) != 1 {
		t.Errorf("scheduler holds %d jobs, want 1", len(sched.jobs))
	}
}

// Reload on a poller that never started must not touch the scheduler.
func TestPoller_Reload_BeforeStartIsANoOp(t *testing.T) {
	store := &fakeStorage{targets: []*polling.Target{{ID: "a", Enabled: true}}}
	sched := newFakeScheduler()
	p := snmp.NewPoller(store, sched, nil)
	if err := p.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(sched.jobs) != 0 {
		t.Errorf("reload before start registered %d jobs", len(sched.jobs))
	}
}
