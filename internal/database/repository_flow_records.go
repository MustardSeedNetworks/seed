package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

// flowTimeFormat keeps millisecond precision at a fixed width so flow
// times sort correctly as text; RFC3339Nano trims trailing zeros and
// does not.
const flowTimeFormat = "2006-01-02T15:04:05.000Z"

// FlowRecordsRepository owns flow_records, written by the flow collector
// (internal/listener/flow).
type FlowRecordsRepository struct {
	db *DB
}

// InsertFlows implements flow.Store: one batch, one transaction.
func (r *FlowRecordsRepository) InsertFlows(ctx context.Context, records []flow.Record) error {
	if len(records) == 0 {
		return nil
	}
	receivedAt := time.Now().UTC().Format(flowTimeFormat)
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO flow_records
			  (exporter, format, observation_domain, flow_start, flow_end,
			   src_addr, dst_addr, src_port, dst_port, protocol, tcp_flags,
			   bytes, packets, input_if, output_if, received_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare flow_records insert: %w", err)
		}
		defer func() { _ = stmt.Close() }()
		for i := range records {
			f := &records[i]
			if _, execErr := stmt.ExecContext(ctx,
				f.Exporter.String(), string(f.Format), f.ObservationDomain,
				f.Start.UTC().Format(flowTimeFormat), f.End.UTC().Format(flowTimeFormat),
				f.SrcAddr.String(), f.DstAddr.String(), f.SrcPort, f.DstPort,
				f.Protocol, f.TCPFlags, saturatingInt64(f.Bytes), saturatingInt64(f.Packets),
				f.InputIf, f.OutputIf, receivedAt,
			); execErr != nil {
				return fmt.Errorf("insert flow_record: %w", execErr)
			}
		}
		return nil
	})
}

// DeleteOlderThan removes flows that ended before cutoff.
func (r *FlowRecordsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `DELETE FROM flow_records WHERE flow_end < ?`,
		cutoff.UTC().Format(flowTimeFormat))
	if err != nil {
		return 0, fmt.Errorf("purge flow_records: %w", err)
	}
	if res == nil {
		return 0, errors.New("purge flow_records: nil sql.Result")
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// saturatingInt64 stores a uint64 counter in a SQLite INTEGER. No real
// flow reaches 2^63 bytes; a corrupt record that claims to is pinned at
// the maximum rather than stored negative.
func saturatingInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}
