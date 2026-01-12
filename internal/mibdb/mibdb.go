// Package mibdb provides MIB database management for SNMP OID resolution.
// It uses the application's SQLite database for persistent storage and
// provides OID name-to-numeric resolution for SNMP operations.
package mibdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrNotFound is returned when an OID entry is not found.
var ErrNotFound = errors.New("OID entry not found")

// DB represents the MIB database interface.
type DB struct {
	db *sql.DB
	mu sync.RWMutex
}

// OIDEntry represents a named OID mapping.
type OIDEntry struct {
	Name     string // Human-readable name (e.g., "sysDescr")
	OID      string // Numeric OID (e.g., "1.3.6.1.2.1.1.1")
	FullPath string // Full human-readable path (optional)
	MIBName  string // Source MIB name (e.g., "SNMPv2-MIB")
}

// New creates a new MIB database interface using an existing SQLite connection.
// The schema must already exist (created via database migrations).
func New(db *sql.DB) *DB {
	return &DB{db: db}
}

// LoadBuiltinOIDs loads all built-in OID definitions into the database.
// This is typically called during application initialization.
func (d *DB) LoadBuiltinOIDs() error {
	return d.AddOIDs(builtinOIDs)
}

// AddOID adds a single OID entry to the database.
func (d *DB) AddOID(entry OIDEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ctx := context.Background()
	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO mib_oid_names (name, oid, full_path, mib_name)
		VALUES (?, ?, ?, ?)
	`, entry.Name, entry.OID, entry.FullPath, entry.MIBName)

	return err
}

// AddOIDs adds multiple OID entries in a single transaction.
func (d *DB) AddOIDs(entries []OIDEntry) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ctx := context.Background()
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO mib_oid_names (name, oid, full_path, mib_name)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		if _, err = stmt.ExecContext(ctx, entry.Name, entry.OID, entry.FullPath, entry.MIBName); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ResolveOIDName converts a name-based OID to numeric form.
// Example: "sysDescr.0" -> "1.3.6.1.2.1.1.1.0".
func (d *DB) ResolveOIDName(oid string) (string, error) {
	// If already numeric, return as-is
	if strings.HasPrefix(oid, ".") || (len(oid) > 0 && oid[0] >= '0' && oid[0] <= '9') {
		return strings.TrimPrefix(oid, "."), nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	// Extract the name part (before any dots or instance suffix)
	name := oid
	suffix := ""
	if idx := strings.Index(oid, "."); idx > 0 {
		name = oid[:idx]
		suffix = oid[idx:]
	}

	ctx := context.Background()
	var numericOID string
	err := d.db.QueryRowContext(ctx, `SELECT oid FROM mib_oid_names WHERE name = ?`, name).Scan(&numericOID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("unknown OID name: %s", name)
		}
		return "", err
	}

	return numericOID + suffix, nil
}

// GetOIDByName looks up an OID entry by name.
func (d *DB) GetOIDByName(name string) (*OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	var entry OIDEntry
	err := d.db.QueryRowContext(ctx, `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM mib_oid_names WHERE name = ?
	`, name).Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &entry, nil
}

// GetOIDByNumeric looks up an OID entry by numeric OID string.
func (d *DB) GetOIDByNumeric(oid string) (*OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Strip leading dot if present
	oid = strings.TrimPrefix(oid, ".")

	ctx := context.Background()
	var entry OIDEntry
	err := d.db.QueryRowContext(ctx, `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM mib_oid_names WHERE oid = ?
	`, oid).Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &entry, nil
}

// GetOIDsByPrefix returns all OIDs that start with the given prefix.
func (d *DB) GetOIDsByPrefix(prefix string) ([]OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	rows, err := d.db.QueryContext(ctx, `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM mib_oid_names WHERE oid LIKE ? || '%'
		ORDER BY oid
	`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []OIDEntry
	for rows.Next() {
		var entry OIDEntry
		if err = rows.Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetOIDsByMIB returns all OIDs from a specific MIB.
func (d *DB) GetOIDsByMIB(mibName string) ([]OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	rows, err := d.db.QueryContext(ctx, `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM mib_oid_names WHERE mib_name = ?
		ORDER BY oid
	`, mibName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []OIDEntry
	for rows.Next() {
		var entry OIDEntry
		if err = rows.Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// SearchOIDs searches for OIDs by name pattern (case-insensitive).
func (d *DB) SearchOIDs(pattern string) ([]OIDEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	rows, err := d.db.QueryContext(ctx, `
		SELECT name, oid, COALESCE(full_path, ''), COALESCE(mib_name, '')
		FROM mib_oid_names WHERE name LIKE ?
		ORDER BY name
		LIMIT 100
	`, "%"+pattern+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []OIDEntry
	for rows.Next() {
		var entry OIDEntry
		if err = rows.Scan(&entry.Name, &entry.OID, &entry.FullPath, &entry.MIBName); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Stats returns database statistics.
func (d *DB) Stats() (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	stats := make(map[string]int)

	var count int
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mib_oid_names`).Scan(&count); err != nil {
		return nil, err
	}
	stats["oid_entries"] = count

	// Count by MIB
	rows, err := d.db.QueryContext(ctx, `
		SELECT mib_name, COUNT(*) as cnt
		FROM mib_oid_names
		WHERE mib_name != ''
		GROUP BY mib_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mibCount := 0
	for rows.Next() {
		var mibName string
		var cnt int
		if err = rows.Scan(&mibName, &cnt); err != nil {
			return nil, err
		}
		mibCount++
	}
	stats["mib_count"] = mibCount

	return stats, rows.Err()
}

// Clear removes all OID entries from the database.
// Use with caution - typically only for testing.
func (d *DB) Clear() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ctx := context.Background()
	_, err := d.db.ExecContext(ctx, `DELETE FROM mib_oid_names`)
	return err
}
