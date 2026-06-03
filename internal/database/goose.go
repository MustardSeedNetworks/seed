package database

// goose.go is the migration engine (ADR-0006, Phase 5b). It replaces the
// homegrown index+1 runner with goose over a single embedded baseline
// (migrations/0001_init.sql). The connection setup in database.go is unchanged;
// only how the schema is created/tracked moved. The baseline's faithfulness to
// the legacy schema is locked by TestGooseBaselineReproducesSchema +
// TestSchemaSnapshot.

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// gooseProvider builds a goose provider over the embedded migrations for conn.
func gooseProvider(conn *sql.DB) (*goose.Provider, error) {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrations fs: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, sub)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}
	return provider, nil
}

// migrate brings the schema up to date by applying all embedded migrations. It
// runs during Open before the DB is shared, so it takes no lock.
func (db *DB) migrate() error {
	provider, err := gooseProvider(db.conn)
	if err != nil {
		return err
	}
	if _, upErr := provider.Up(context.Background()); upErr != nil {
		return fmt.Errorf("failed to apply migrations: %w", upErr)
	}
	return nil
}

// MigrationInfo represents the status of a migration.
type MigrationInfo struct {
	Version     int
	Description string
	Applied     bool
	AppliedAt   time.Time
}

// MigrationStatus returns the status of every known migration.
func (db *DB) MigrationStatus(ctx context.Context) ([]MigrationInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return nil, errors.New("database is closed")
	}

	provider, err := gooseProvider(db.conn)
	if err != nil {
		return nil, err
	}
	status, err := provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}

	result := make([]MigrationInfo, 0, len(status))
	for _, s := range status {
		result = append(result, MigrationInfo{
			Version:     int(s.Source.Version),
			Description: s.Source.Path,
			Applied:     s.State == goose.StateApplied,
			AppliedAt:   s.AppliedAt,
		})
	}
	return result, nil
}

// SchemaVersion returns the current applied schema version.
func (db *DB) SchemaVersion(ctx context.Context) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return 0, errors.New("database is closed")
	}

	provider, err := gooseProvider(db.conn)
	if err != nil {
		return 0, err
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}
	return int(version), nil
}
