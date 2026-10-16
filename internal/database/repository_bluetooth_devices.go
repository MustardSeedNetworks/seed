package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// bluetoothTimeFormat keeps millisecond precision at a fixed width so the
// seen times sort correctly as text, the same reason as flowTimeFormat.
const bluetoothTimeFormat = "2006-01-02T15:04:05.000Z"

// BluetoothDevicesRepository owns bluetooth_devices, written by every
// Bluetooth scan (P-E3).
type BluetoothDevicesRepository struct {
	db *DB
}

// RememberBluetoothDevices upserts every device by address in one
// transaction and sets each one's ID and FirstSeen to the stored values, so a
// device heard before keeps the identity and first-seen time of its first
// scan. A name or vendor this scan did not hear keeps the stored one.
func (r *BluetoothDevicesRepository) RememberBluetoothDevices(
	ctx context.Context, devices []discovery.BluetoothDevice,
) error {
	if len(devices) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO bluetooth_devices
			  (address, id, address_type, name, vendor, type, device_class, appearance,
			   class_of_device, manufacturer_id, rssi, tx_power, first_seen, last_seen)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (address) DO UPDATE SET
			  address_type    = excluded.address_type,
			  name            = COALESCE(NULLIF(excluded.name, ''), name),
			  vendor          = COALESCE(NULLIF(excluded.vendor, ''), vendor),
			  type            = excluded.type,
			  device_class    = excluded.device_class,
			  appearance      = excluded.appearance,
			  class_of_device = excluded.class_of_device,
			  manufacturer_id = excluded.manufacturer_id,
			  rssi            = excluded.rssi,
			  tx_power        = excluded.tx_power,
			  last_seen       = excluded.last_seen
			RETURNING id, first_seen
		`)
		if err != nil {
			return fmt.Errorf("prepare bluetooth_devices upsert: %w", err)
		}
		defer func() { _ = stmt.Close() }()
		for i := range devices {
			d := &devices[i]
			var firstSeen string
			if scanErr := stmt.QueryRowContext(ctx,
				d.Address, d.ID, string(d.AddressType), d.Name, d.Vendor, string(d.Type),
				string(d.DeviceClass), d.Appearance, d.ClassOfDev, d.ManufacturerID,
				d.RSSI, d.TxPower,
				d.FirstSeen.UTC().Format(bluetoothTimeFormat), d.LastSeen.UTC().Format(bluetoothTimeFormat),
			).Scan(&d.ID, &firstSeen); scanErr != nil {
				return fmt.Errorf("upsert bluetooth device %s: %w", d.Address, scanErr)
			}
			seen, parseErr := time.Parse(bluetoothTimeFormat, firstSeen)
			if parseErr != nil {
				return fmt.Errorf("bluetooth device %s first_seen %q: %w", d.Address, firstSeen, parseErr)
			}
			d.FirstSeen = seen
		}
		return nil
	})
}

// DeleteNotSeenSince removes devices last heard before cutoff.
func (r *BluetoothDevicesRepository) DeleteNotSeenSince(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `DELETE FROM bluetooth_devices WHERE last_seen < ?`,
		cutoff.UTC().Format(bluetoothTimeFormat))
	if err != nil {
		return 0, fmt.Errorf("purge bluetooth_devices: %w", err)
	}
	if res == nil {
		return 0, errors.New("purge bluetooth_devices: nil sql.Result")
	}
	return res.RowsAffected()
}
