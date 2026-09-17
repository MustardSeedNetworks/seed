package database_test

// D-SEED-21 (seed#2692) reports lastStatus and lastPolledAt as null on the
// targets page after successful polls. UpdateLastPoll had no test against the
// real table at all -- internal/polling/snmp exercises it through a fake
// storage, which records whatever it is handed and so cannot fail on the SQL
// -- and the poller only logs the error it returns. These cases close that
// hole: they read the columns back through the same query the API serves from.

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// lastPollTarget returns a DB holding one target to poll.
func lastPollTarget(t *testing.T) (*database.DB, *polling.Target) {
	t.Helper()

	db, cleanup := testDB(t)
	t.Cleanup(cleanup)

	target := &polling.Target{
		ClientID: database.DefaultClientID, Name: "core-switch",
		IPAddress: "10.51.200.1", SNMPVersion: "v2c", PollIntervalSec: 60,
		Enabled: true, CollectorChain: []string{"sys_info"},
	}
	if err := db.PollingTargets().Create(context.Background(), target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	return db, target
}

// TestUpdateLastPollRecordsASuccess is the row's claim, against the real table.
func TestUpdateLastPollRecordsASuccess(t *testing.T) {
	db, target := lastPollTarget(t)
	ctx := context.Background()

	if err := db.PollingTargets().UpdateLastPoll(ctx, target.ID, "ok", ""); err != nil {
		t.Fatalf("UpdateLastPoll: %v", err)
	}

	got, err := db.PollingTargets().Get(ctx, database.DefaultClientID, target.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.LastStatus != "ok" {
		t.Errorf("lastStatus = %q, want ok", got.LastStatus)
	}
	if got.LastPolledAt.IsZero() {
		t.Error("lastPolledAt is the zero time, want the poll's timestamp")
	}
	if got.LastError != "" {
		t.Errorf("lastError = %q, want empty after a success", got.LastError)
	}
}

// TestUpdateLastPollRecordsAFailureAndThenClearsIt covers the operator-visible
// sequence: a target that failed and then answered must not keep showing the
// old reason next to an "ok" status.
func TestUpdateLastPollRecordsAFailureAndThenClearsIt(t *testing.T) {
	db, target := lastPollTarget(t)
	ctx := context.Background()

	if err := db.PollingTargets().UpdateLastPoll(
		ctx, target.ID, "error", "no configured credential succeeded",
	); err != nil {
		t.Fatalf("UpdateLastPoll(error): %v", err)
	}

	got, err := db.PollingTargets().Get(ctx, database.DefaultClientID, target.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.LastStatus != "error" || got.LastError == "" {
		t.Errorf("lastStatus = %q, lastError = %q, want error + a reason",
			got.LastStatus, got.LastError)
	}

	if updErr := db.PollingTargets().UpdateLastPoll(ctx, target.ID, "ok", ""); updErr != nil {
		t.Fatalf("UpdateLastPoll(ok): %v", updErr)
	}

	got, err = db.PollingTargets().Get(ctx, database.DefaultClientID, target.ID)
	if err != nil {
		t.Fatalf("Get after recovery: %v", err)
	}
	if got.LastStatus != "ok" {
		t.Errorf("lastStatus = %q, want ok", got.LastStatus)
	}
	if got.LastError != "" {
		t.Errorf("lastError = %q, want the stale reason cleared", got.LastError)
	}
}
