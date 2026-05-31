package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrClientNotFound is returned when a client is not found.
var ErrClientNotFound = errors.New("client not found")

// ErrClientSlugExists is returned when a client slug already exists.
var ErrClientSlugExists = errors.New("client slug already exists")

// DefaultClientID is the id of the seeded default client. Single-tenant
// deployments use this id for everything; multi-tenant deployments
// keep it as the fallback when no explicit client is selected.
const DefaultClientID = "default"

// ClientRepository provides CRUD operations for clients.
type ClientRepository struct {
	db *DB
}

// Create inserts a new client. Generates an id when Client.ID is
// empty. Returns ErrClientSlugExists when the slug collides.
func (r *ClientRepository) Create(ctx context.Context, c *Client) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	_, err := r.db.Exec(
		ctx,
		`
		INSERT INTO clients (id, name, slug, branding_json, default_retention_overrides_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		c.ID,
		c.Name,
		c.Slug,
		toNullString(c.BrandingJSON),
		toNullString(c.DefaultRetentionOverridesJSON),
		c.CreatedAt.Format(time.RFC3339Nano),
		c.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrClientSlugExists
		}
		return fmt.Errorf("create client: %w", err)
	}
	return nil
}

// Get returns the client with the given id. Returns ErrClientNotFound
// when no such client exists.
func (r *ClientRepository) Get(ctx context.Context, id string) (*Client, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, branding_json, default_retention_overrides_json, created_at, updated_at
		FROM clients WHERE id = ?
	`, id)
	return scanClient(row.Scan)
}

// GetBySlug returns the client with the given slug. Returns
// ErrClientNotFound when no such client exists.
func (r *ClientRepository) GetBySlug(ctx context.Context, slug string) (*Client, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, slug, branding_json, default_retention_overrides_json, created_at, updated_at
		FROM clients WHERE slug = ?
	`, slug)
	return scanClient(row.Scan)
}

// List returns all clients, ordered by name.
func (r *ClientRepository) List(ctx context.Context) ([]*Client, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, branding_json, default_retention_overrides_json, created_at, updated_at
		FROM clients ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*Client
	for rows.Next() {
		c, scanErr := scanClient(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, c)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("list clients iter: %w", rowsErr)
	}
	return out, nil
}

// Update modifies an existing client. UpdatedAt is set to now.
func (r *ClientRepository) Update(ctx context.Context, c *Client) error {
	c.UpdatedAt = time.Now().UTC()
	res, err := r.db.Exec(ctx, `
		UPDATE clients
		SET name = ?, slug = ?, branding_json = ?, default_retention_overrides_json = ?, updated_at = ?
		WHERE id = ?
	`,
		c.Name,
		c.Slug,
		toNullString(c.BrandingJSON),
		toNullString(c.DefaultRetentionOverridesJSON),
		c.UpdatedAt.Format(time.RFC3339Nano),
		c.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrClientSlugExists
		}
		return fmt.Errorf("update client: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrClientNotFound
	}
	return nil
}

// Delete removes a client. Refuses to delete the default client
// (the system always needs at least one client to attribute
// observations to).
func (r *ClientRepository) Delete(ctx context.Context, id string) error {
	if id == DefaultClientID {
		return errors.New("cannot delete the default client")
	}
	res, err := r.db.Exec(ctx, `DELETE FROM clients WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete client: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrClientNotFound
	}
	return nil
}

// Count returns the number of clients.
func (r *ClientRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM clients`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count clients: %w", err)
	}
	return n, nil
}

// toNullString converts "" → invalid [sql.NullString]. Used in INSERT
// and UPDATE to keep optional TEXT columns as SQL NULL rather than
// the empty string, which lets indexes / queries treat absence
// cleanly.
func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// scanClient reads a Client from a [sql.Row.Scan] or [sql.Rows.Scan]
// signature.
func scanClient(scan func(...any) error) (*Client, error) {
	var (
		c                 Client
		branding          sql.NullString
		retentionOverride sql.NullString
		createdAt         string
		updatedAt         string
	)
	err := scan(&c.ID, &c.Name, &c.Slug, &branding, &retentionOverride, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrClientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan client: %w", err)
	}
	if branding.Valid {
		c.BrandingJSON = branding.String
	}
	if retentionOverride.Valid {
		c.DefaultRetentionOverridesJSON = retentionOverride.String
	}
	if t, parseErr := time.Parse(time.RFC3339Nano, createdAt); parseErr == nil {
		c.CreatedAt = t
	}
	if t, parseErr := time.Parse(time.RFC3339Nano, updatedAt); parseErr == nil {
		c.UpdatedAt = t
	}
	return &c, nil
}
