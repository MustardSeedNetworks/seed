package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/identity/users"
)

// GetUserDashboard returns the widget ids the user saved, in order, and
// whether they have ever saved a layout. Returns users.ErrUserNotFound
// when no such user exists.
func (db *DB) GetUserDashboard(ctx context.Context, username string) ([]string, bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, false, errors.New("database is closed")
	}

	var raw sql.NullString
	err := db.readConn.QueryRowContext(ctx, `
		SELECT d.widgets
		FROM users u
		LEFT JOIN user_dashboards d ON d.user_id = u.id
		WHERE u.username = ?
	`, username).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, users.ErrUserNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get dashboard: %w", err)
	}
	if !raw.Valid {
		return nil, false, nil
	}
	var widgets []string
	if err = json.Unmarshal([]byte(raw.String), &widgets); err != nil {
		return nil, false, fmt.Errorf("failed to decode dashboard: %w", err)
	}
	return widgets, true, nil
}

// SetUserDashboard replaces the user's layout. Returns users.ErrUserNotFound
// when no such user exists.
func (db *DB) SetUserDashboard(ctx context.Context, username string, widgets []string) error {
	if widgets == nil {
		widgets = []string{}
	}
	encoded, err := json.Marshal(widgets)
	if err != nil {
		return fmt.Errorf("failed to encode dashboard: %w", err)
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("database is closed")
	}

	result, err := db.writeConn.ExecContext(ctx, `
		INSERT INTO user_dashboards (user_id, widgets, updated_at)
		SELECT id, ?, ? FROM users WHERE username = ?
		ON CONFLICT (user_id) DO UPDATE SET widgets = excluded.widgets, updated_at = excluded.updated_at
	`, string(encoded), time.Now().UTC().Format(time.RFC3339), username)
	if err != nil {
		return fmt.Errorf("failed to save dashboard: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return users.ErrUserNotFound
	}
	return nil
}
