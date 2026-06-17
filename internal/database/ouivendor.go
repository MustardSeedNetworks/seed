package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// OUIVendorDB represents an OUI vendor record in the database.
type OUIVendorDB struct {
	OUIPrefix    string `json:"ouiPrefix"` // First 3 bytes of MAC (e.g., "00:50:56")
	VendorName   string `json:"vendorName"`
	VendorAlias  string `json:"vendorAlias,omitempty"`
	IsPrivate    bool   `json:"isPrivate"`
	AddressBlock string `json:"addressBlock,omitempty"` // MA-L, MA-M, MA-S
}

// CreateOUIVendor creates a new OUI vendor record.
func (r *DiscoveryRepository) CreateOUIVendor(ctx context.Context, vendor *OUIVendorDB) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO oui_vendors (oui_prefix, vendor_name, vendor_alias, is_private, address_block)
		VALUES (?, ?, ?, ?, ?)
	`, vendor.OUIPrefix, vendor.VendorName, vendor.VendorAlias,
		boolToInt(vendor.IsPrivate), vendor.AddressBlock)
	if err != nil {
		return fmt.Errorf("failed to create oui vendor: %w", err)
	}
	return nil
}

// GetOUIVendor retrieves a vendor by OUI prefix.
func (r *DiscoveryRepository) GetOUIVendor(ctx context.Context, ouiPrefix string) (*OUIVendorDB, error) {
	// Normalize the prefix (uppercase, colon-separated)
	normalized := normalizeOUI(ouiPrefix)

	row := r.db.QueryRow(ctx, `
		SELECT oui_prefix, vendor_name, vendor_alias, is_private, address_block
		FROM oui_vendors WHERE oui_prefix = ?
	`, normalized)

	var v OUIVendorDB
	var alias, addressBlock sql.NullString
	var isPrivate int

	err := row.Scan(&v.OUIPrefix, &v.VendorName, &alias, &isPrivate, &addressBlock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOUIVendorNotFound
		}
		return nil, fmt.Errorf("failed to scan oui vendor: %w", err)
	}

	v.VendorAlias = alias.String
	v.AddressBlock = addressBlock.String
	v.IsPrivate = isPrivate == 1
	return &v, nil
}

// LookupVendorByMAC looks up the vendor for a MAC address.
func (r *DiscoveryRepository) LookupVendorByMAC(ctx context.Context, mac string) (string, error) {
	// Extract OUI prefix from MAC address
	ouiPrefix := extractOUIPrefix(mac)
	if ouiPrefix == "" {
		return "", nil
	}

	vendor, err := r.GetOUIVendor(ctx, ouiPrefix)
	if err != nil {
		if errors.Is(err, ErrOUIVendorNotFound) {
			return "", nil
		}
		return "", err
	}
	return vendor.VendorName, nil
}

// BulkUpsertOUIVendors inserts or updates multiple OUI vendors efficiently.
func (r *DiscoveryRepository) BulkUpsertOUIVendors(ctx context.Context, vendors []OUIVendorDB) error {
	if len(vendors) == 0 {
		return nil
	}

	// Use INSERT OR REPLACE for SQLite
	query := `INSERT OR REPLACE INTO oui_vendors (oui_prefix, vendor_name, vendor_alias, is_private, address_block) VALUES `
	var args []any
	var placeholders []string

	for _, v := range vendors {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?)")
		args = append(args, v.OUIPrefix, v.VendorName, v.VendorAlias, boolToInt(v.IsPrivate), v.AddressBlock)
	}

	query += strings.Join(placeholders, ", ")

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert oui vendors: %w", err)
	}
	return nil
}

// Channel Utilization operations
