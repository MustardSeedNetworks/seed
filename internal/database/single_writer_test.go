package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// openWithShortBusyTimeout opens a database whose SQLite busy timeout is short
// enough to make lock contention observable inside a unit test. Production uses
// 5 s; the behaviour under test is what happens once that budget is spent.
func openWithShortBusyTimeout(t *testing.T) *database.DB {
	t.Helper()
	cfg := database.DefaultConfig(filepath.Join(t.TempDir(), "seed.db"))
	cfg.BusyTimeout = int(busyTimeoutForTest / time.Millisecond)
	db, err := database.OpenWithConfig(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

const (
	// busyTimeoutForTest is the SQLite busy timeout the contention tests run
	// with; lockHoldForTest exceeds it, so a second writer that competes for the
	// lock at the SQLite level is guaranteed to exhaust the timeout.
	busyTimeoutForTest = 200 * time.Millisecond
	lockHoldForTest    = 800 * time.Millisecond
)

// TestWriteWaitsForHeldTransaction is the regression for #2453: on CT313 ten of
// 74 SNMP polling targets ended a cycle in error with
// "database is locked (5) (SQLITE_BUSY)", losing the observation, because a
// write held elsewhere in the process outlasted the busy timeout. Writes are
// serialised on one connection, so a competing writer queues behind the
// transaction instead of racing it at the SQLite level and being rejected.
func TestWriteWaitsForHeldTransaction(t *testing.T) {
	db := openWithShortBusyTimeout(t)
	ctx := context.Background()

	holding := make(chan struct{})
	released := make(chan struct{})
	holderErr := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		holderErr <- db.WithTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx,
				`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
				 VALUES ('default', 'holder', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
			if err != nil {
				return err
			}
			close(holding)
			time.Sleep(lockHoldForTest)
			close(released)
			return nil
		})
	})

	// The write lock is held only once that first statement has executed.
	<-holding

	_, err := db.Exec(ctx,
		`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
		 VALUES ('default', 'collector', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err, "a collector insert must never be rejected because another write is in flight")

	select {
	case <-released:
	default:
		t.Fatal("the competing insert committed before the holding transaction released the lock")
	}
	wg.Wait()
	require.NoError(t, <-holderErr)

	var count int
	require.NoError(t, db.QueryRow(ctx,
		`SELECT count(*) FROM snmp_observations WHERE target_id = 'collector'`).Scan(&count))
	require.Equal(t, 1, count, "the observation is persisted, not dropped")
}

// TestWriteWaitIsBoundedByContext keeps the queue from becoming an unbounded
// wait: serialising writers means a caller waits for the connection, and the
// only thing that may end that wait early is its own deadline.
func TestWriteWaitIsBoundedByContext(t *testing.T) {
	db := openWithShortBusyTimeout(t)

	holding := make(chan struct{})
	blocked := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		_ = db.WithTx(context.Background(), func(tx *sql.Tx) error {
			_, err := tx.ExecContext(context.Background(),
				`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
				 VALUES ('default', 'holder', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
			if err != nil {
				return err
			}
			close(holding)
			<-blocked
			return nil
		})
	})
	t.Cleanup(func() { close(blocked); wg.Wait() })
	<-holding

	ctx, cancel := context.WithTimeout(context.Background(), lockHoldForTest)
	defer cancel()
	_, err := db.Exec(ctx,
		`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
		 VALUES ('default', 'late', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestUpdateReturningWaitsForHeldTransaction covers the write that does not
// look like one: profile Update/UpdateIfMatch are UPDATE ... RETURNING, so they
// read a row back and would sit on the read pool if written with QueryRow.
func TestUpdateReturningWaitsForHeldTransaction(t *testing.T) {
	db := openWithShortBusyTimeout(t)
	ctx := context.Background()
	repo := db.Profiles()
	require.NoError(t, repo.Create(ctx, &database.Profile{ID: "p1", Name: "orig", ConfigJSON: "{}"}))

	holding := make(chan struct{})
	released := make(chan struct{})
	holderErr := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		holderErr <- db.WithTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx,
				`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
				 VALUES ('default', 'holder', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
			if err != nil {
				return err
			}
			close(holding)
			time.Sleep(lockHoldForTest)
			close(released)
			return nil
		})
	})
	<-holding

	require.NoError(t, repo.Update(ctx, &database.Profile{ID: "p1", Name: "renamed", ConfigJSON: "{}"}))

	select {
	case <-released:
	default:
		t.Fatal("the update committed before the holding transaction released the lock")
	}
	wg.Wait()
	require.NoError(t, <-holderErr)
}
