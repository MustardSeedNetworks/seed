package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrPollingTargetNotFound is returned when a polling target lookup
// misses.
var ErrPollingTargetNotFound = errors.New("polling target not found")

// PollingTarget mirrors a polling_targets row. CollectorChain is
// decoded from the JSON column. Last* fields record the most
// recent poll's outcome and feed the operator-facing target status.
type PollingTarget struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"clientId"`
	Name            string    `json:"name"`
	IPAddress       string    `json:"ipAddress"`
	SNMPVersion     string    `json:"snmpVersion"`
	CredentialsID   string    `json:"credentialsId,omitempty"`
	PollIntervalSec int       `json:"pollIntervalSeconds"`
	Enabled         bool      `json:"enabled"`
	CollectorChain  []string  `json:"collectorChain"`
	LastPolledAt    time.Time `json:"lastPolledAt,omitzero"`
	LastStatus      string    `json:"lastStatus,omitempty"`
	LastError       string    `json:"lastError,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// PollingTargetRepository owns CRUD + status-update access to
// polling_targets.
type PollingTargetRepository struct {
	db *DB
}

// List returns all enabled polling targets for a client. Empty
// clientID returns enabled targets across every client (useful for
// the system-wide poller loop).
func (r *PollingTargetRepository) List(ctx context.Context, clientID string) ([]*PollingTarget, error) {
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

	var out []*PollingTarget
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
func (r *PollingTargetRepository) Get(ctx context.Context, id string) (*PollingTarget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, client_id, name, ip_address, snmp_version,
			credentials_id, poll_interval_seconds, enabled, collector_chain,
			last_polled_at, last_status, last_error, created_at, updated_at
		FROM polling_targets WHERE id = ?
	`, id)
	return scanPollingTarget(row.Scan)
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

// scanPollingTarget reads a PollingTarget row via a [sql.Row.Scan]
// or [sql.Rows.Scan] signature.
func scanPollingTarget(scan func(...any) error) (*PollingTarget, error) {
	var (
		t             PollingTarget
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
		return nil, ErrPollingTargetNotFound
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
