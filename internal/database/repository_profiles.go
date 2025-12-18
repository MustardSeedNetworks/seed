// Package database provides profile repository for managing configuration profiles.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrProfileNotFound is returned when a profile is not found.
var ErrProfileNotFound = errors.New("profile not found")

// ErrProfileNameExists is returned when a profile name already exists.
var ErrProfileNameExists = errors.New("profile name already exists")

// ProfileRepository provides CRUD operations for profiles.
type ProfileRepository struct {
	db *DB
}

// Create creates a new profile.
func (r *ProfileRepository) Create(ctx context.Context, profile *Profile) error {
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO profiles (id, name, description, config_json, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Name, profile.Description, profile.ConfigJSON,
		boolToInt(profile.IsDefault), profile.CreatedAt.Format(time.RFC3339), profile.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrProfileNameExists
		}
		return fmt.Errorf("failed to create profile: %w", err)
	}

	return nil
}

// Get retrieves a profile by ID.
func (r *ProfileRepository) Get(ctx context.Context, id string) (*Profile, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, description, config_json, is_default, created_at, updated_at
		FROM profiles WHERE id = ?
	`, id)

	return r.scanProfile(row)
}

// GetByName retrieves a profile by name.
func (r *ProfileRepository) GetByName(ctx context.Context, name string) (*Profile, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, description, config_json, is_default, created_at, updated_at
		FROM profiles WHERE name = ?
	`, name)

	return r.scanProfile(row)
}

// GetDefault retrieves the default profile.
func (r *ProfileRepository) GetDefault(ctx context.Context) (*Profile, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, description, config_json, is_default, created_at, updated_at
		FROM profiles WHERE is_default = 1 LIMIT 1
	`)

	return r.scanProfile(row)
}

// List retrieves all profiles.
func (r *ProfileRepository) List(ctx context.Context) ([]*Profile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, description, config_json, is_default, created_at, updated_at
		FROM profiles ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*Profile
	for rows.Next() {
		profile, err := r.scanProfileFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

// Update updates a profile.
func (r *ProfileRepository) Update(ctx context.Context, profile *Profile) error {
	profile.UpdatedAt = time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET name = ?, description = ?, config_json = ?, is_default = ?, updated_at = ?
		WHERE id = ?
	`, profile.Name, profile.Description, profile.ConfigJSON,
		boolToInt(profile.IsDefault), profile.UpdatedAt.Format(time.RFC3339), profile.ID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrProfileNameExists
		}
		return fmt.Errorf("failed to update profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProfileNotFound
	}

	return nil
}

// Delete deletes a profile by ID.
func (r *ProfileRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProfileNotFound
	}

	return nil
}

// SetDefault sets a profile as the default, clearing the default flag from all other profiles.
func (r *ProfileRepository) SetDefault(ctx context.Context, id string) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// Clear existing default
		if _, err := tx.ExecContext(ctx, `UPDATE profiles SET is_default = 0`); err != nil {
			return fmt.Errorf("failed to clear existing default: %w", err)
		}

		// Set new default
		result, err := tx.ExecContext(ctx, `UPDATE profiles SET is_default = 1 WHERE id = ?`, id)
		if err != nil {
			return fmt.Errorf("failed to set default: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return ErrProfileNotFound
		}

		return nil
	})
}

// Count returns the total number of profiles.
func (r *ProfileRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM profiles`)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count profiles: %w", err)
	}
	return count, nil
}

// scanProfile scans a single profile from a row.
func (r *ProfileRepository) scanProfile(row *sql.Row) (*Profile, error) {
	var p Profile
	var createdAt, updatedAt string
	var isDefault int

	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.ConfigJSON, &isDefault, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to scan profile: %w", err)
	}

	p.IsDefault = isDefault == 1
	if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		p.CreatedAt = t
	}
	if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
		p.UpdatedAt = t
	}

	return &p, nil
}

// scanProfileFromRows scans a profile from rows.
func (r *ProfileRepository) scanProfileFromRows(rows *sql.Rows) (*Profile, error) {
	var p Profile
	var createdAt, updatedAt string
	var isDefault int

	err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ConfigJSON, &isDefault, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan profile: %w", err)
	}

	p.IsDefault = isDefault == 1
	if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		p.CreatedAt = t
	}
	if t, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
		p.UpdatedAt = t
	}

	return &p, nil
}

// boolToInt converts a bool to an int for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isUniqueConstraintError checks if the error is a unique constraint violation.
func isUniqueConstraintError(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") ||
		contains(err.Error(), "duplicate key"))
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

// containsRune is a helper for contains.
func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
