package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrDeviceNotFound is returned when a device is not found.
var ErrDeviceNotFound = errors.New("device not found")

// DeviceRepository provides operations for discovered devices.
type DeviceRepository struct {
	db *DB
}

// Create creates a new device record.
func (r *DeviceRepository) Create(ctx context.Context, device *Device) error {
	if device.ID == "" {
		device.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if device.FirstSeen.IsZero() {
		device.FirstSeen = now
	}
	device.LastSeen = now
	device.IsActive = true

	_, err := r.db.Exec(ctx, `
		INSERT INTO devices
		(id, ip_address, mac_address, hostname, vendor, device_type, os_family,
		 first_seen, last_seen, is_active, ports_json, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.IPAddress, device.MACAddress, device.Hostname,
		device.Vendor, device.DeviceType, device.OSFamily,
		device.FirstSeen.Format(time.RFC3339), device.LastSeen.Format(time.RFC3339),
		boolToInt(device.IsActive), device.PortsJSON, device.Metadata)
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	return nil
}

// Get retrieves a device by ID.
func (r *DeviceRepository) Get(ctx context.Context, id string) (*Device, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, ip_address, mac_address, hostname, vendor, device_type, os_family,
		       first_seen, last_seen, is_active, ports_json, metadata_json
		FROM devices WHERE id = ?
	`, id)

	return r.scanDevice(row)
}

// GetByIP retrieves a device by IP address.
func (r *DeviceRepository) GetByIP(ctx context.Context, ip string) (*Device, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, ip_address, mac_address, hostname, vendor, device_type, os_family,
		       first_seen, last_seen, is_active, ports_json, metadata_json
		FROM devices WHERE ip_address = ?
	`, ip)

	return r.scanDevice(row)
}

// GetByMAC retrieves a device by MAC address.
func (r *DeviceRepository) GetByMAC(ctx context.Context, mac string) (*Device, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, ip_address, mac_address, hostname, vendor, device_type, os_family,
		       first_seen, last_seen, is_active, ports_json, metadata_json
		FROM devices WHERE mac_address = ?
	`, mac)

	return r.scanDevice(row)
}

// List retrieves all devices matching the criteria.
//

func (r *DeviceRepository) List(ctx context.Context, opts DeviceListOptions) ([]*Device, error) {
	query := `
		SELECT id, ip_address, mac_address, hostname, vendor, device_type, os_family,
		       first_seen, last_seen, is_active, ports_json, metadata_json
		FROM devices
		WHERE 1=1
	`
	var args []any

	if opts.ActiveOnly {
		query += " AND is_active = 1"
	}

	if opts.DeviceType != "" {
		query += " AND device_type = ?"
		args = append(args, opts.DeviceType)
	}

	if opts.Vendor != "" {
		query += " AND vendor = ?"
		args = append(args, opts.Vendor)
	}

	if !opts.SeenAfter.IsZero() {
		query += " AND last_seen >= ?"
		args = append(args, opts.SeenAfter.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY last_seen DESC"

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += sqlOffset
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var devices []*Device
	for rows.Next() {
		d, scanErr := r.scanDeviceFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		devices = append(devices, d)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating devices: %w", rowsErr)
	}

	return devices, nil
}

// DeviceListOptions specifies criteria for listing devices.
type DeviceListOptions struct {
	ActiveOnly bool
	DeviceType string
	Vendor     string
	SeenAfter  time.Time
	Limit      int
	Offset     int
}

// Update updates a device record.
func (r *DeviceRepository) Update(ctx context.Context, device *Device) error {
	device.LastSeen = time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE devices
		SET ip_address = ?, mac_address = ?, hostname = ?, vendor = ?,
		    device_type = ?, os_family = ?, last_seen = ?, is_active = ?,
		    ports_json = ?, metadata_json = ?
		WHERE id = ?
	`, device.IPAddress, device.MACAddress, device.Hostname, device.Vendor,
		device.DeviceType, device.OSFamily, device.LastSeen.Format(time.RFC3339),
		boolToInt(device.IsActive), device.PortsJSON, device.Metadata, device.ID)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// Upsert creates or updates a device based on IP address.
func (r *DeviceRepository) Upsert(ctx context.Context, device *Device) error {
	existing, err := r.GetByIP(ctx, device.IPAddress)
	if err != nil && !errors.Is(err, ErrDeviceNotFound) {
		return fmt.Errorf("failed to check existing device: %w", err)
	}

	if existing != nil {
		// Update existing device
		device.ID = existing.ID
		device.FirstSeen = existing.FirstSeen
		return r.Update(ctx, device)
	}

	// Create new device
	return r.Create(ctx, device)
}

// UpsertByMAC creates or updates a device based on MAC address.
func (r *DeviceRepository) UpsertByMAC(ctx context.Context, device *Device) error {
	if device.MACAddress == "" {
		return r.Upsert(ctx, device)
	}

	existing, err := r.GetByMAC(ctx, device.MACAddress)
	if err != nil && !errors.Is(err, ErrDeviceNotFound) {
		return fmt.Errorf("failed to check existing device: %w", err)
	}

	if existing != nil {
		// Update existing device
		device.ID = existing.ID
		device.FirstSeen = existing.FirstSeen
		return r.Update(ctx, device)
	}

	// Create new device
	return r.Create(ctx, device)
}

// Delete removes a device by ID.
func (r *DeviceRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// MarkInactive marks devices as inactive if not seen since the given time.
func (r *DeviceRepository) MarkInactive(ctx context.Context, since time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE devices SET is_active = 0 WHERE last_seen < ? AND is_active = 1
	`, since.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to mark devices inactive: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return count, nil
}

// DeleteInactive removes inactive devices older than the given time.
func (r *DeviceRepository) DeleteInactive(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM devices WHERE is_active = 0 AND last_seen < ?
	`, olderThan.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to delete inactive devices: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return count, nil
}

// Count returns the total number of devices.
func (r *DeviceRepository) Count(ctx context.Context, activeOnly bool) (int64, error) {
	query := "SELECT COUNT(*) FROM devices"
	if activeOnly {
		query += " WHERE is_active = 1"
	}

	var count int64
	row := r.db.QueryRow(ctx, query)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count devices: %w", err)
	}
	return count, nil
}

// GetDistinctVendors returns all unique vendors.
func (r *DeviceRepository) GetDistinctVendors(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT vendor FROM devices WHERE vendor IS NOT NULL AND vendor != '' ORDER BY vendor
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct vendors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var vendors []string
	for rows.Next() {
		var v string
		if scanErr := rows.Scan(&v); scanErr != nil {
			return nil, fmt.Errorf("scanning vendor: %w", scanErr)
		}
		vendors = append(vendors, v)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating vendors: %w", rowsErr)
	}

	return vendors, nil
}

// GetDistinctTypes returns all unique device types.
func (r *DeviceRepository) GetDistinctTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT device_type FROM devices WHERE device_type IS NOT NULL AND device_type != '' ORDER BY device_type
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct types: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var types []string
	for rows.Next() {
		var t string
		if scanErr := rows.Scan(&t); scanErr != nil {
			return nil, fmt.Errorf("scanning type: %w", scanErr)
		}
		types = append(types, t)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating types: %w", rowsErr)
	}

	return types, nil
}

// scanDevice scans a device from a row.
func (r *DeviceRepository) scanDevice(row *sql.Row) (*Device, error) {
	var d Device
	var firstSeen, lastSeen string
	var isActive int
	var mac, hostname, vendor, deviceType, osFamily, ports, metadata sql.NullString

	err := row.Scan(&d.ID, &d.IPAddress, &mac, &hostname, &vendor, &deviceType, &osFamily,
		&firstSeen, &lastSeen, &isActive, &ports, &metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to scan device: %w", err)
	}

	d.MACAddress = mac.String
	d.Hostname = hostname.String
	d.Vendor = vendor.String
	d.DeviceType = deviceType.String
	d.OSFamily = osFamily.String
	d.PortsJSON = ports.String
	d.Metadata = metadata.String
	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		d.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		d.LastSeen = t
	}
	d.IsActive = isActive == 1

	return &d, nil
}

// scanDeviceFromRows scans a device from rows.
func (r *DeviceRepository) scanDeviceFromRows(rows *sql.Rows) (*Device, error) {
	var d Device
	var firstSeen, lastSeen string
	var isActive int
	var mac, hostname, vendor, deviceType, osFamily, ports, metadata sql.NullString

	err := rows.Scan(&d.ID, &d.IPAddress, &mac, &hostname, &vendor, &deviceType, &osFamily,
		&firstSeen, &lastSeen, &isActive, &ports, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to scan device: %w", err)
	}

	d.MACAddress = mac.String
	d.Hostname = hostname.String
	d.Vendor = vendor.String
	d.DeviceType = deviceType.String
	d.OSFamily = osFamily.String
	d.PortsJSON = ports.String
	d.Metadata = metadata.String
	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		d.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		d.LastSeen = t
	}
	d.IsActive = isActive == 1

	return &d, nil
}
