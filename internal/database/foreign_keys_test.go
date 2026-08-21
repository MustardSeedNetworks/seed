package database_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

const foreignKeyTestClientID = "foreign-key-parent"

func TestForeignKeysEnforcedOnEveryPooledConnection(t *testing.T) {
	t.Run("invalid child inserts", func(t *testing.T) {
		db := openPooledTestDB(t, filepath.Join(t.TempDir(), "inserts.db"))
		assertInvalidChildrenRejected(t, holdTwoConnections(t, db))
	})

	t.Run("restrictive deletes", func(t *testing.T) {
		db := openPooledTestDB(t, filepath.Join(t.TempDir(), "deletes.db"))
		insertClientWithCredential(t, db)
		assertParentDeletesRejected(t, holdTwoConnections(t, db))
	})
}

func TestForeignKeysRemainEnabledAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	db := openPooledTestDB(t, path)
	require.NoError(t, db.Close())

	reopened := openPooledTestDB(t, path)
	assertInvalidChildrenRejected(t, holdTwoConnections(t, reopened))
}

func openPooledTestDB(t *testing.T, path string) *database.DB {
	t.Helper()
	cfg := database.DefaultConfig(path)
	cfg.MaxOpenConns = 2
	cfg.MaxIdleConns = 2
	db, err := database.OpenWithConfig(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func holdTwoConnections(t *testing.T, db *database.DB) []*sql.Conn {
	t.Helper()
	connections := make([]*sql.Conn, 2)
	for i := range connections {
		conn, err := db.Conn().Conn(t.Context())
		require.NoError(t, err)
		connections[i] = conn
		t.Cleanup(func() { require.NoError(t, conn.Close()) })
	}
	return connections
}

func assertInvalidChildrenRejected(t *testing.T, connections []*sql.Conn) {
	t.Helper()
	for i, conn := range connections {
		_, err := conn.ExecContext(t.Context(), `
			INSERT INTO device_credentials
				(id, name, kind, snmp_community_enc, created_at, updated_at, client_id)
			VALUES (?, 'orphan', 'v2c', CAST('enc:v1:x' AS BLOB), ?, ?, 'missing-client')`,
			fmt.Sprintf("orphan-%d", i), time.Now(), time.Now())
		require.ErrorContains(t, err, "FOREIGN KEY constraint failed", "connection %d", i+1)
	}
}

func insertClientWithCredential(t *testing.T, db *database.DB) {
	t.Helper()
	now := time.Now()
	_, err := db.Exec(t.Context(), `
		INSERT INTO clients (id, name, slug, created_at, updated_at)
		VALUES (?, 'FK parent', 'fk-parent', ?, ?)`, foreignKeyTestClientID, now, now)
	require.NoError(t, err)
	_, err = db.Exec(t.Context(), `
		INSERT INTO device_credentials
			(id, name, kind, snmp_community_enc, created_at, updated_at, client_id)
		VALUES ('fk-child', 'FK child', 'v2c', CAST('enc:v1:x' AS BLOB), ?, ?, ?)`,
		now, now, foreignKeyTestClientID)
	require.NoError(t, err)
}

func assertParentDeletesRejected(t *testing.T, connections []*sql.Conn) {
	t.Helper()
	for i, conn := range connections {
		_, err := conn.ExecContext(t.Context(),
			"DELETE FROM clients WHERE id = ?", foreignKeyTestClientID)
		require.ErrorContains(t, err, "FOREIGN KEY constraint failed", "connection %d", i+1)
	}
}
