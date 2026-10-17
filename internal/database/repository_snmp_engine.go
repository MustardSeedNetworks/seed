package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// maxSNMPEngineBoots is snmpEngineBoots' latched maximum (RFC 3414 2.2.2).
const maxSNMPEngineBoots uint32 = 2147483647

// SNMPEngineRepository owns snmp_engine, the identity Seed answers SNMPv3
// informs with (P-B4).
type SNMPEngineRepository struct {
	db *DB
}

// BootSNMPEngine records one engine start and returns the engine ID and the
// new boots count. The first call stores newID with boots 1; every later
// call keeps the stored ID and increments boots, latching at the RFC 3414
// maximum.
func (r *SNMPEngineRepository) BootSNMPEngine(ctx context.Context, newID []byte) ([]byte, uint32, error) {
	var (
		id    []byte
		boots uint32
	)
	err := r.db.WithTx(ctx, func(tx *sql.Tx) error {
		scanErr := tx.QueryRowContext(ctx, `SELECT engine_id, boots FROM snmp_engine WHERE id = 1`).
			Scan(&id, &boots)
		switch {
		case errors.Is(scanErr, sql.ErrNoRows):
			id, boots = newID, 1
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO snmp_engine (id, engine_id, boots) VALUES (1, ?, 1)`, newID); err != nil {
				return fmt.Errorf("insert snmp_engine: %w", err)
			}
			return nil
		case scanErr != nil:
			return fmt.Errorf("read snmp_engine: %w", scanErr)
		}
		boots = min(boots+1, maxSNMPEngineBoots)
		if _, err := tx.ExecContext(ctx, `UPDATE snmp_engine SET boots = ? WHERE id = 1`, boots); err != nil {
			return fmt.Errorf("update snmp_engine boots: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return id, boots, nil
}
