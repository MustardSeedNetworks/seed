package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AlertRule is one row of alert_rules. Match fields default to ""
// which the listener pipeline treats as "any value matches".
type AlertRule struct {
	ID                   int64
	Name                 string
	Enabled              bool
	MatchKind            string
	MatchSeverity        string
	MatchPayloadContains string
	AlertType            string
	AlertSeverity        string
	AlertTitle           string
	AlertMessage         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// AlertRulesRepository owns CRUD over alert_rules.
type AlertRulesRepository struct {
	db *DB
}

// ErrAlertRuleNotFound is returned when a rule lookup misses.
var ErrAlertRuleNotFound = errors.New("alert rule not found")

// Create inserts a new rule.
func (r *AlertRulesRepository) Create(ctx context.Context, rule *AlertRule) error {
	if rule.Name == "" {
		return errors.New("alert_rules: Name required")
	}
	if rule.AlertType == "" || rule.AlertSeverity == "" {
		return errors.New("alert_rules: AlertType + AlertSeverity required")
	}
	if rule.AlertTitle == "" {
		return errors.New("alert_rules: AlertTitle required")
	}
	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	res, err := r.db.Exec(ctx, `
		INSERT INTO alert_rules
		  (name, enabled, match_kind, match_severity, match_payload_contains,
		   alert_type, alert_severity, alert_title, alert_message,
		   created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rule.Name, enabled,
		toNullString(rule.MatchKind),
		toNullString(rule.MatchSeverity),
		toNullString(rule.MatchPayloadContains),
		rule.AlertType, rule.AlertSeverity,
		rule.AlertTitle, rule.AlertMessage,
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create alert_rule: %w", err)
	}
	if id, idErr := res.LastInsertId(); idErr == nil {
		rule.ID = id
	}
	return nil
}

// Get returns the rule with the given id.
func (r *AlertRulesRepository) Get(ctx context.Context, id int64) (*AlertRule, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, enabled, match_kind, match_severity, match_payload_contains,
		       alert_type, alert_severity, alert_title, alert_message,
		       created_at, updated_at
		FROM alert_rules WHERE id = ?
	`, id)
	return scanAlertRule(row.Scan)
}

// List returns every rule ordered by id ascending so the pipeline
// applies rules in a deterministic order. enabledOnly filters out
// disabled rows (the pipeline uses true; the operator UI uses false).
func (r *AlertRulesRepository) List(ctx context.Context, enabledOnly bool) ([]*AlertRule, error) {
	query := `
		SELECT id, name, enabled, match_kind, match_severity, match_payload_contains,
		       alert_type, alert_severity, alert_title, alert_message,
		       created_at, updated_at
		FROM alert_rules
	`
	if enabledOnly {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY id ASC`

	return queryRows(ctx, r.db, query, nil, scanAlertRule, "list alert_rules")
}

// Update replaces every writable field on the rule. Returns
// ErrAlertRuleNotFound when id doesn't exist.
func (r *AlertRulesRepository) Update(ctx context.Context, rule *AlertRule) error {
	if rule.ID == 0 {
		return errors.New("alert_rules: ID required for Update")
	}
	if rule.Name == "" || rule.AlertType == "" || rule.AlertSeverity == "" || rule.AlertTitle == "" {
		return errors.New("alert_rules: Name + AlertType + AlertSeverity + AlertTitle required")
	}
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	res, err := r.db.Exec(ctx, `
		UPDATE alert_rules SET
			name = ?, enabled = ?,
			match_kind = ?, match_severity = ?, match_payload_contains = ?,
			alert_type = ?, alert_severity = ?, alert_title = ?, alert_message = ?,
			updated_at = ?
		WHERE id = ?
	`,
		rule.Name, enabled,
		toNullString(rule.MatchKind),
		toNullString(rule.MatchSeverity),
		toNullString(rule.MatchPayloadContains),
		rule.AlertType, rule.AlertSeverity,
		rule.AlertTitle, rule.AlertMessage,
		time.Now().UTC().Format(time.RFC3339Nano),
		rule.ID,
	)
	if err != nil {
		return fmt.Errorf("update alert_rule: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAlertRuleNotFound
	}
	return nil
}

// Delete removes a rule by id.
func (r *AlertRulesRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.Exec(ctx, `DELETE FROM alert_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete alert_rule: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrAlertRuleNotFound
	}
	return nil
}

func scanAlertRule(scan func(...any) error) (*AlertRule, error) {
	var (
		rule         AlertRule
		enabledInt   int
		matchKind    sql.NullString
		matchSev     sql.NullString
		matchPayload sql.NullString
		createdStr   string
		updatedStr   string
	)
	err := scan(
		&rule.ID, &rule.Name, &enabledInt,
		&matchKind, &matchSev, &matchPayload,
		&rule.AlertType, &rule.AlertSeverity,
		&rule.AlertTitle, &rule.AlertMessage,
		&createdStr, &updatedStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAlertRuleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan alert_rule: %w", err)
	}
	rule.Enabled = enabledInt != 0
	if matchKind.Valid {
		rule.MatchKind = matchKind.String
	}
	if matchSev.Valid {
		rule.MatchSeverity = matchSev.String
	}
	if matchPayload.Valid {
		rule.MatchPayloadContains = matchPayload.String
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, createdStr); perr == nil {
		rule.CreatedAt = parsed
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, updatedStr); perr == nil {
		rule.UpdatedAt = parsed
	}
	return &rule, nil
}
