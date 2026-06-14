package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// PollingTargetRepository owns CRUD + status-update access to
// polling_targets.
type PollingTargetRepository struct {
	db *DB
}

// List returns all enabled polling targets for a client. Empty
// clientID returns enabled targets across every client (useful for
// the system-wide poller loop).
func (r *PollingTargetRepository) List(ctx context.Context, clientID string) ([]*polling.Target, error) {
	query := `
		SELECT id, client_id, name, ip_address, snmp_version,
			credentials_id, poll_interval_seconds, enabled, collector_chain,
			last_polled_at, last_status, last_error, created_at, updated_at
		FROM polling_targets
		WHERE enabled = 1
	`
	args := []any{}
	if clientID != "" {
		query += " AND client_id = ?"
		args = append(args, clientID)
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list polling_targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*polling.Target
	for rows.Next() {
		t, scanErr := scanPollingTarget(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, t)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("list polling_targets iter: %w", rowsErr)
	}
	return out, nil
}

// Get returns one polling target by id.
func (r *PollingTargetRepository) Get(ctx context.Context, id string) (*polling.Target, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, client_id, name, ip_address, snmp_version,
			credentials_id, poll_interval_seconds, enabled, collector_chain,
			last_polled_at, last_status, last_error, created_at, updated_at
		FROM polling_targets WHERE id = ?
	`, id)
	return scanPollingTarget(row.Scan)
}

// Create inserts a new polling target. ID + timestamps are stamped
// here if zero; the caller is expected to populate the remaining
// fields (Name, IPAddress, SNMPVersion, CollectorChain, etc.).
// Default poll interval is 300s when caller passes 0.
func (r *PollingTargetRepository) Create(ctx context.Context, t *polling.Target) error {
	if t.IPAddress == "" {
		return errors.New("polling_targets: IPAddress required")
	}
	if t.Name == "" {
		return errors.New("polling_targets: Name required")
	}
	if t.ID == "" {
		t.ID = "tgt-" + randomID()
	}
	if t.ClientID == "" {
		t.ClientID = "default"
	}
	if t.SNMPVersion == "" {
		t.SNMPVersion = "v2c"
	}
	if t.PollIntervalSec == 0 {
		t.PollIntervalSec = 300
	}
	chainJSON, _ := json.Marshal(t.CollectorChain)
	if len(t.CollectorChain) == 0 {
		// Default chain matches the migration default so a freshly
		// added target picks up the same set the migration would
		// have populated.
		chainJSON = []byte(`["sys_info","if_table","lldp","arp","fdb"]`)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO polling_targets
		  (id, client_id, name, ip_address, snmp_version, credentials_id,
		   poll_interval_seconds, enabled, collector_chain,
		   created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.ClientID, t.Name, t.IPAddress, t.SNMPVersion,
		toNullString(t.CredentialsID),
		t.PollIntervalSec, enabled, string(chainJSON),
		now, now,
	)
	if err != nil {
		return fmt.Errorf("create polling_target: %w", err)
	}
	return nil
}

// Update modifies the writable fields (name, ip, snmp version, poll
// interval, enabled, collector chain, credentials_id). Read-only
// audit columns (created_at, last_polled_at, last_status, last_error)
// stay untouched. Returns [polling.ErrTargetNotFound] when id is
// absent — distinguishes a missing target from a SQL-level
// [sql.ErrNoRows] surface so handlers can map it to HTTP 404.
func (r *PollingTargetRepository) Update(ctx context.Context, t *polling.Target) error {
	if t.ID == "" {
		return errors.New("polling_targets: ID required for Update")
	}
	chainJSON, _ := json.Marshal(t.CollectorChain)
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	res, err := r.db.Exec(ctx, `
		UPDATE polling_targets SET
			name = ?, ip_address = ?, snmp_version = ?,
			credentials_id = ?, poll_interval_seconds = ?, enabled = ?,
			collector_chain = ?, updated_at = ?
		WHERE id = ?
	`,
		t.Name, t.IPAddress, t.SNMPVersion,
		toNullString(t.CredentialsID),
		t.PollIntervalSec, enabled, string(chainJSON),
		time.Now().UTC().Format(time.RFC3339Nano),
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("update polling_target: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return polling.ErrTargetNotFound
	}
	return nil
}

// Delete removes a polling target by id. Cascades via foreign-key
// chain on dependent rows (none today). Returns
// polling.ErrTargetNotFound when no row matched.
func (r *PollingTargetRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("polling_targets: ID required for Delete")
	}
	res, err := r.db.Exec(ctx, `DELETE FROM polling_targets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete polling_target: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return polling.ErrTargetNotFound
	}
	return nil
}

// randomID returns a 12-char hex string suitable for non-secret IDs.
// crypto/rand is overkill for primary keys; we keep it for the
// uniform-distribution guarantee under contention.
func randomID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// UpdateLastPoll records the outcome of the most recent poll attempt.
// errMsg may be empty on success.
func (r *PollingTargetRepository) UpdateLastPoll(ctx context.Context, id, status, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE polling_targets
		SET last_polled_at = ?, last_status = ?, last_error = ?, updated_at = ?
		WHERE id = ?
	`,
		time.Now().UTC().Format(time.RFC3339Nano),
		status,
		toNullString(errMsg),
		time.Now().UTC().Format(time.RFC3339Nano),
		id,
	)
	if err != nil {
		return fmt.Errorf("update polling_target last poll: %w", err)
	}
	return nil
}

// scanPollingTarget reads a polling.Target row via a [sql.Row.Scan]
// or [sql.Rows.Scan] signature.
func scanPollingTarget(scan func(...any) error) (*polling.Target, error) {
	var (
		t             polling.Target
		credentialsID sql.NullString
		chainJSON     sql.NullString
		lastPolledAt  sql.NullString
		lastStatus    sql.NullString
		lastError     sql.NullString
		enabledInt    int
		createdAtStr  string
		updatedAtStr  string
	)
	err := scan(
		&t.ID, &t.ClientID, &t.Name, &t.IPAddress, &t.SNMPVersion,
		&credentialsID, &t.PollIntervalSec, &enabledInt, &chainJSON,
		&lastPolledAt, &lastStatus, &lastError, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, polling.ErrTargetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan polling_target: %w", err)
	}

	t.Enabled = enabledInt != 0
	if credentialsID.Valid {
		t.CredentialsID = credentialsID.String
	}
	if chainJSON.Valid && chainJSON.String != "" {
		var chain []string
		if jsonErr := json.Unmarshal([]byte(chainJSON.String), &chain); jsonErr == nil {
			t.CollectorChain = chain
		}
	}
	if lastPolledAt.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, lastPolledAt.String); parseErr == nil {
			t.LastPolledAt = parsed
		}
	}
	if lastStatus.Valid {
		t.LastStatus = lastStatus.String
	}
	if lastError.Valid {
		t.LastError = lastError.String
	}
	if parsed, parseErr := time.Parse(time.RFC3339Nano, createdAtStr); parseErr == nil {
		t.CreatedAt = parsed
	}
	if parsed, parseErr := time.Parse(time.RFC3339Nano, updatedAtStr); parseErr == nil {
		t.UpdatedAt = parsed
	}
	return &t, nil
}
