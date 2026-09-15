package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
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
	// with. A serialised writer queues in Go and never reaches SQLite, so this
	// budget should never be spent; keeping it short means that if
	// serialisation ever regresses, the competing writer is rejected promptly
	// and the test says so instead of hanging for the production 5 s.
	busyTimeoutForTest = 200 * time.Millisecond

	// queueDeadline bounds the wait for a writer to appear in the pool's
	// queue. It is a safety net, not a timing assumption: the queue entry is
	// observable within tens of microseconds, and exceeding this means no
	// writer ever queued, which is the failure the test exists to catch.
	queueDeadline = 5 * time.Second

	// blockedWriteDeadline is how long a writer that can never acquire the
	// connection is given before its own context must end the wait. Any value
	// proves the same thing — the holder never releases — so it is small.
	blockedWriteDeadline = 50 * time.Millisecond
)

// awaitQueuedWriter blocks until one more caller is waiting for the write
// connection than there was at baseline.
//
// This is what "the writer waits" actually means: the write handle has
// MaxOpenConns 1, so a second writer queues for that connection inside
// database/sql instead of opening its own and racing the first at the SQLite
// level, where the loser is rejected with SQLITE_BUSY rather than queued.
// Observing the queue entry is both faster and stricter than inferring it from
// elapsed time, which is what these tests did before: they slept past the busy
// timeout and checked the ordering afterwards.
func awaitQueuedWriter(t *testing.T, db *database.DB, baseline int64) {
	t.Helper()
	deadline := time.Now().Add(queueDeadline)
	for time.Now().Before(deadline) {
		if db.WriteConn().Stats().WaitCount > baseline {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("no writer queued for the write connection within %s: "+
		"the competing write is not waiting for the held transaction", queueDeadline)
}

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
	release := make(chan struct{})
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
			<-release
			return nil
		})
	})

	// The write lock is held only once that first statement has executed.
	<-holding
	baseline := db.WriteConn().Stats().WaitCount

	collectorErr := make(chan error, 1)
	go func() {
		_, err := db.Exec(ctx,
			`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
			 VALUES ('default', 'collector', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
		collectorErr <- err
	}()

	awaitQueuedWriter(t, db, baseline)
	select {
	case err := <-collectorErr:
		t.Fatalf("the competing insert finished (%v) while the holding transaction still held the lock", err)
	default:
	}

	close(release)
	require.NoError(t, <-collectorErr,
		"a collector insert must never be rejected because another write is in flight")
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
	baseline := db.WriteConn().Stats().WaitCount

	ctx, cancel := context.WithTimeout(context.Background(), blockedWriteDeadline)
	defer cancel()
	lateErr := make(chan error, 1)
	go func() {
		_, err := db.Exec(ctx,
			`INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
			 VALUES ('default', 'late', 'arp', '2026-01-01T00:00:00Z', '{}', '2026-01-01T00:00:00Z')`)
		lateErr <- err
	}()

	// Both halves are needed, and the second is why the deadline may be
	// short. DeadlineExceeded alone does not say the writer queued: a writer
	// racing SQLite on its own connection also fails, and with any deadline
	// below the busy timeout it fails the same way. Asserting the pool wait
	// first pins which of the two happened.
	awaitQueuedWriter(t, db, baseline)
	require.ErrorIs(t, <-lateErr, context.DeadlineExceeded)
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
	release := make(chan struct{})
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
			<-release
			return nil
		})
	})
	<-holding
	baseline := db.WriteConn().Stats().WaitCount

	updateErr := make(chan error, 1)
	go func() {
		updateErr <- repo.Update(ctx, &database.Profile{ID: "p1", Name: "renamed", ConfigJSON: "{}"})
	}()

	awaitQueuedWriter(t, db, baseline)
	select {
	case err := <-updateErr:
		t.Fatalf("the update finished (%v) while the holding transaction still held the lock", err)
	default:
	}

	close(release)
	require.NoError(t, <-updateErr)
	wg.Wait()
	require.NoError(t, <-holderErr)
}

// TestPragmasApplyToEveryConnection guards the DSN placement: pragmas are
// connection-scoped, so one applied after Open lands on a single pooled
// connection and every later one — including the replacement for a connection
// recycled at ConnMaxLifetime — reverts to the SQLite default. On the write
// handle that would mean an fsync per commit.
func TestPragmasApplyToEveryConnection(t *testing.T) {
	cfg := database.DefaultConfig(filepath.Join(t.TempDir(), "seed.db"))
	cfg.ConnMaxLifetime = time.Millisecond
	db, err := database.OpenWithConfig(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Long enough that every connection opened at startup is past its lifetime
	// and the pool must hand out fresh ones.
	time.Sleep(20 * time.Millisecond)

	for _, want := range []struct {
		pragma string
		value  string
	}{
		{"journal_mode", "wal"},
		{"synchronous", "1"}, // NORMAL
		{"foreign_keys", "1"},
		{"temp_store", "2"}, // MEMORY
	} {
		var got string
		require.NoError(t, db.QueryRow(context.Background(), "PRAGMA "+want.pragma).Scan(&got),
			"read %s", want.pragma)
		require.Equal(t, want.value, got, "read pragma %s", want.pragma)

		require.NoError(t, db.QueryRowWrite(context.Background(), "PRAGMA "+want.pragma).Scan(&got),
			"write %s", want.pragma)
		require.Equal(t, want.value, got, "write pragma %s", want.pragma)
	}
}
