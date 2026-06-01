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
//
// Time-windowed rules (#1379): WindowSeconds > 0 + ThresholdCount > 1
// means "fire only after N matching events within W seconds." The
// default (0/1) preserves the pre-#1379 fire-on-first-match
// behavior. Validation rejects ThresholdCount > 1 with WindowSeconds
// = 0 because a threshold without a window has no time bound and
// would silently never reset.
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
	WindowSeconds        int
	ThresholdCount       int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// AlertRulesRepository owns CRUD over alert_rules.
type AlertRulesRepository struct {
	db *DB
}

// ErrAlertRuleNotFound is returned when a rule lookup misses.
var ErrAlertRuleNotFound = errors.New("alert rule not found")

// validateRule enforces invariants both Create and Update need.
// Time-window validation (#1379): threshold > 1 requires a window;
// negative values are nonsense for either field. ThresholdCount == 0
// is normalized to 1 in-place so callers (handlers, tests) that
// don't think about windowing don't need to set the field.
func validateRule(rule *AlertRule) error {
	if rule.Name == "" {
		return errors.New("alert_rules: Name required")
	}
	if rule.AlertType == "" || rule.AlertSeverity == "" {
		return errors.New("alert_rules: AlertType + AlertSeverity required")
	}
	if rule.AlertTitle == "" {
		return errors.New("alert_rules: AlertTitle required")
	}
	if rule.WindowSeconds < 0 {
		return errors.New("alert_rules: WindowSeconds must be >= 0")
	}
	if rule.ThresholdCount == 0 {
		rule.ThresholdCount = 1
	}
	if rule.ThresholdCount < 1 {
		return errors.New("alert_rules: ThresholdCount must be >= 1")
	}
	if rule.ThresholdCount > 1 && rule.WindowSeconds <= 0 {
		return errors.New("alert_rules: ThresholdCount > 1 requires WindowSeconds > 0")
	}
	return nil
}

// Create inserts a new rule.
func (r *AlertRulesRepository) Create(ctx context.Context, rule *AlertRule) error {
	if err := validateRule(rule); err != nil {
		return err
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
		   window_seconds, threshold_count,
		   created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rule.Name, enabled,
		toNullString(rule.MatchKind),
		toNullString(rule.MatchSeverity),
		toNullString(rule.MatchPayloadContains),
		rule.AlertType, rule.AlertSeverity,
		rule.AlertTitle, rule.AlertMessage,
		rule.WindowSeconds, rule.ThresholdCount,
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
		       window_seconds, threshold_count,
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
		       window_seconds, threshold_count,
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
	if err := validateRule(rule); err != nil {
		return err
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
			window_seconds = ?, threshold_count = ?,
			updated_at = ?
		WHERE id = ?
	`,
		rule.Name, enabled,
		toNullString(rule.MatchKind),
		toNullString(rule.MatchSeverity),
		toNullString(rule.MatchPayloadContains),
		rule.AlertType, rule.AlertSeverity,
		rule.AlertTitle, rule.AlertMessage,
		rule.WindowSeconds, rule.ThresholdCount,
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
		&rule.WindowSeconds, &rule.ThresholdCount,
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
