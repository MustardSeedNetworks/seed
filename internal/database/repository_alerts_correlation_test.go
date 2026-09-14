package database_test

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// The correlation columns are only useful if they survive the round trip, and
// both read paths must see them — Get and List were separate scanners before
// this change, which is exactly how a new column gets read by one and not the
// other.
func TestAlertRuleAndRootCauseRoundTrip(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	cause := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityWarning,
		Title: "Interface eth0 down on t-1", Message: "ifOperStatus up -> down",
		Source: "t-1", Rule: "iface.down",
	}
	if err := repo.Create(ctx, cause); err != nil {
		t.Fatalf("create cause: %v", err)
	}
	if cause.ID == 0 {
		t.Fatal("Create did not assign an id to the cause")
	}

	effect := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "BGP peer 10.0.0.2 left Established on t-1", Message: "state 6 -> 1",
		Source: "t-1", Rule: "bgp.flap", RootCauseID: &cause.ID,
	}
	if err := repo.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}

	got, err := repo.Get(ctx, effect.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Rule != "bgp.flap" {
		t.Errorf("Get rule = %q, want bgp.flap", got.Rule)
	}
	if got.RootCauseID == nil || *got.RootCauseID != cause.ID {
		t.Errorf("Get rootCauseID = %v, want %d", got.RootCauseID, cause.ID)
	}

	listed, err := repo.List(ctx, alerts.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *alerts.Alert
	for _, a := range listed {
		if a.ID == effect.ID {
			found = a
		}
	}
	if found == nil {
		t.Fatal("List did not return the effect alert")
	}
	if found.Rule != "bgp.flap" {
		t.Errorf("List rule = %q, want bgp.flap", found.Rule)
	}
	if found.RootCauseID == nil || *found.RootCauseID != cause.ID {
		t.Errorf("List rootCauseID = %v, want %d", found.RootCauseID, cause.ID)
	}
}

// Retention prunes by age and the cause is always older than the effect, so
// the cause is deleted first. Losing the explanation must not take the alert
// with it — ON DELETE SET NULL, not CASCADE.
func TestDeletingTheCauseKeepsTheEffect(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Alerts()

	cause := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityWarning,
		Title: "Interface down", Source: "t-1", Rule: "iface.down",
	}
	if err := repo.Create(ctx, cause); err != nil {
		t.Fatalf("create cause: %v", err)
	}
	effect := &alerts.Alert{
		Type: alerts.TypeConnectivity, Severity: alerts.SeverityError,
		Title: "BGP flap", Source: "t-1", Rule: "bgp.flap", RootCauseID: &cause.ID,
	}
	if err := repo.Create(ctx, effect); err != nil {
		t.Fatalf("create effect: %v", err)
	}

	if err := repo.Delete(ctx, cause.ID); err != nil {
		t.Fatalf("delete cause: %v", err)
	}

	got, err := repo.Get(ctx, effect.ID)
	if err != nil {
		t.Fatalf("the effect did not survive its cause being pruned: %v", err)
	}
	if got.RootCauseID != nil {
		t.Errorf("rootCauseID = %d after the cause was deleted, want nil", *got.RootCauseID)
	}
}
