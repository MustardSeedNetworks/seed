// Package database provides SQLite database management for The Seed.
//
// It handles connection pooling, schema migrations, and provides a clean
// interface for data persistence operations. Uses modernc.org/sqlite for
// pure Go SQLite implementation (no CGO required).
//
// Features:
// - Automatic schema migrations with versioning
// - Connection pooling and health checks
// - Support for profiles, metrics, devices, and alerts storage
// - Data retention policies with automatic cleanup
//
// Usage:
//
//	db, err := database.Open("/path/to/seed.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Use repositories for data access
//	profiles := db.Profiles()
//	metrics := db.Metrics()
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

// DB represents the database connection and provides access to repositories.
type DB struct {
	conn   *sql.DB
	path   string
	mu     sync.RWMutex
	closed bool

	// Repositories - lazily initialized
	profiles *ProfileRepository
	metrics  *MetricsRepository
	devices  *DeviceRepository
	alerts   *AlertRepository
	settings *SettingsRepository
}

// Config holds database configuration options.
type Config struct {
	// Path to the SQLite database file
	Path string

	// MaxOpenConns sets the maximum number of open connections
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum lifetime of a connection
	ConnMaxLifetime time.Duration

	// RetentionDays sets how many days of data to retain (0 = forever)
	RetentionDays int

	// EnableWAL enables Write-Ahead Logging for better concurrency
	EnableWAL bool

	// BusyTimeout sets the timeout for waiting on locked database (ms)
	BusyTimeout int
}

// DefaultConfig returns sensible defaults for database configuration.
func DefaultConfig(path string) Config {
	return Config{
		Path:            path,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		RetentionDays:   90, // Keep 90 days of data by default
		EnableWAL:       true,
		BusyTimeout:     5000, // 5 seconds
	}
}

// Open creates a new database connection with default configuration.
func Open(path string) (*DB, error) {
	return OpenWithConfig(DefaultConfig(path))
}

// OpenWithConfig creates a new database connection with custom configuration.
func OpenWithConfig(cfg Config) (*DB, error) {
	if cfg.Path == "" {
		return nil, errors.New("database path is required")
	}

	// Build connection string with pragmas
	dsn := fmt.Sprintf("file:%s?_txlock=immediate", cfg.Path)
	if cfg.BusyTimeout > 0 {
		dsn += fmt.Sprintf("&_busy_timeout=%d", cfg.BusyTimeout)
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Apply pragmas for performance and safety
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -64000", // 64MB cache
		"PRAGMA temp_store = MEMORY",
	}

	if !cfg.EnableWAL {
		pragmas[1] = "PRAGMA journal_mode = DELETE"
	}

	for _, pragma := range pragmas {
		if _, err := conn.Exec(pragma); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	// Verify connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn: conn,
		path: cfg.Path,
	}

	// Run migrations
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Seed default profile if database is empty
	if err := db.seedDefaultProfile(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to seed default profile: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	db.closed = true

	// Checkpoint WAL before closing for clean shutdown
	if _, err := db.conn.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		// Log but don't fail - this is a cleanup operation
		fmt.Printf("warning: failed to checkpoint WAL: %v\n", err)
	}

	return db.conn.Close()
}

// Ping checks database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return errors.New("database is closed")
	}

	return db.conn.PingContext(ctx)
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

// Stats returns database connection statistics.
func (db *DB) Stats() sql.DBStats {
	return db.conn.Stats()
}

// Profiles returns the profile repository.
func (db *DB) Profiles() *ProfileRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.profiles == nil {
		db.profiles = &ProfileRepository{db: db}
	}
	return db.profiles
}

// Metrics returns the metrics repository.
func (db *DB) Metrics() *MetricsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.metrics == nil {
		db.metrics = &MetricsRepository{db: db}
	}
	return db.metrics
}

// Devices returns the device repository.
func (db *DB) Devices() *DeviceRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.devices == nil {
		db.devices = &DeviceRepository{db: db}
	}
	return db.devices
}

// Alerts returns the alert repository.
func (db *DB) Alerts() *AlertRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.alerts == nil {
		db.alerts = &AlertRepository{db: db}
	}
	return db.alerts
}

// Settings returns the settings repository.
func (db *DB) Settings() *SettingsRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.settings == nil {
		db.settings = &SettingsRepository{db: db}
	}
	return db.settings
}

// Exec executes a query without returning any rows.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	return db.conn.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
// Caller is responsible for closing the returned rows.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	//nolint:sqlclosecheck // Caller is responsible for closing rows
	return db.conn.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.conn.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, errors.New("database is closed")
	}

	return db.conn.BeginTx(ctx, opts)
}

// WithTx executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
