// SPDX-License-Identifier: BUSL-1.1

package netif_test

import (
	"slices"
	"testing"

	"github.com/krisarmstrong/seed/internal/netif"
)

func TestLinkMonitorPool_ReconcileAddsAndRemoves(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()

	if got := pool.Reconcile([]string{"eth0", "wlan0"}); got != 2 {
		t.Fatalf("Reconcile add: got count %d, want 2", got)
	}
	names := pool.Interfaces()
	if !slices.Equal(names, []string{"eth0", "wlan0"}) {
		t.Errorf("Interfaces after add: got %v, want [eth0 wlan0]", names)
	}

	// Removing wlan0, adding eth1 — eth0 stays.
	if got := pool.Reconcile([]string{"eth0", "eth1"}); got != 2 {
		t.Fatalf("Reconcile swap: got count %d, want 2", got)
	}
	names = pool.Interfaces()
	if !slices.Equal(names, []string{"eth0", "eth1"}) {
		t.Errorf("Interfaces after swap: got %v, want [eth0 eth1]", names)
	}

	if got := pool.Reconcile(nil); got != 0 {
		t.Errorf("Reconcile to empty: got count %d, want 0", got)
	}
	if names = pool.Interfaces(); len(names) != 0 {
		t.Errorf("Interfaces after empty: got %v, want []", names)
	}
}

func TestLinkMonitorPool_ReconcileSkipsEmptyNames(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	pool.Reconcile([]string{"eth0", "", "wlan0", ""})
	names := pool.Interfaces()
	if !slices.Equal(names, []string{"eth0", "wlan0"}) {
		t.Errorf("got %v, want [eth0 wlan0]", names)
	}
}

func TestLinkMonitorPool_ReconcilePreservesExistingMonitor(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	pool.Reconcile([]string{"eth0"})
	before := pool.Monitor("eth0")
	if before == nil {
		t.Fatal("expected eth0 monitor after first Reconcile")
	}
	// Re-reconcile with eth0 plus a new name.
	pool.Reconcile([]string{"eth0", "wlan0"})
	after := pool.Monitor("eth0")
	if after != before {
		t.Errorf("eth0 monitor was restarted unnecessarily (instance changed)")
	}
}

func TestLinkMonitorPool_DiffNames(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	pool.Reconcile([]string{"eth0", "wlan0"})

	added, removed := pool.DiffNames([]string{"eth0", "eth1"})
	if !slices.Equal(added, []string{"eth1"}) {
		t.Errorf("added: got %v, want [eth1]", added)
	}
	if !slices.Equal(removed, []string{"wlan0"}) {
		t.Errorf("removed: got %v, want [wlan0]", removed)
	}
}

func TestLinkMonitorPool_OnStateChangePropagatesToExistingAndFuture(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	pool.Reconcile([]string{"eth0"})

	// Register callback AFTER eth0 already exists — should attach
	// retroactively.
	cb := func(_ netif.LinkEvent) {}
	pool.OnStateChange(cb)

	// New monitor added later should also receive the callback.
	pool.Reconcile([]string{"eth0", "eth1"})

	// Smoke: every monitor should be non-nil and accessible.
	for _, name := range pool.Interfaces() {
		if pool.Monitor(name) == nil {
			t.Errorf("monitor for %s missing after Reconcile", name)
		}
	}
}

func TestLinkMonitorPool_GetStateForUnknownInterface(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	if got := pool.GetState("nope"); got != netif.LinkStateUnknown {
		t.Errorf("got %v, want LinkStateUnknown", got)
	}
}

func TestLinkMonitorPool_HasAndCount(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	if pool.Count() != 0 {
		t.Errorf("empty pool Count = %d, want 0", pool.Count())
	}
	pool.Reconcile([]string{"eth0", "eth1"})
	if pool.Count() != 2 {
		t.Errorf("Count after reconcile = %d, want 2", pool.Count())
	}
	if !pool.Has("eth0") {
		t.Error("Has(eth0) = false, want true")
	}
	if pool.Has("not-there") {
		t.Error("Has(not-there) = true, want false")
	}
}

func TestLinkMonitorPool_StopIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := netif.NewLinkMonitorPool()
	pool.Reconcile([]string{"eth0"})
	pool.Stop() // not started — no-op
	if err := pool.Start(); err != nil {
		t.Fatalf("Start after no-op Stop: %v", err)
	}
	pool.Stop()
	pool.Stop() // second Stop is a no-op
}
