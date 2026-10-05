package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener/voip"
)

// VoIPStreamsRepository owns voip_streams, written by the VoIP analyser
// (internal/listener/voip).
type VoIPStreamsRepository struct {
	db *DB
}

// InsertVoIPReports implements voip.Store: one batch, one transaction.
func (r *VoIPStreamsRepository) InsertVoIPReports(ctx context.Context, reports []voip.Report) error {
	if len(reports) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO voip_streams
			  (interface_name, src_addr, dst_addr, ssrc, codec, started_at, ended_at,
			   packets_expected, packets_received, loss_pct, jitter_ms, max_jitter_ms,
			   delay_ms, r_factor, mos)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare voip_streams insert: %w", err)
		}
		defer func() { _ = stmt.Close() }()
		for i := range reports {
			rep := &reports[i]
			if _, execErr := stmt.ExecContext(ctx,
				rep.Interface, rep.Src.String(), rep.Dst.String(), int64(rep.SSRC), rep.Codec,
				rep.Start.UTC().Format(burstTimeFormat), rep.End.UTC().Format(burstTimeFormat),
				rep.Expected, rep.Received, rep.LossPct, rep.JitterMs, rep.MaxJitterMs,
				rep.DelayMs, rep.RFactor, rep.MOS,
			); execErr != nil {
				return fmt.Errorf("insert voip_stream: %w", execErr)
			}
		}
		return nil
	})
}

// DeleteOlderThan removes windows that started before cutoff.
func (r *VoIPStreamsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `DELETE FROM voip_streams WHERE started_at < ?`,
		cutoff.UTC().Format(burstTimeFormat))
	if err != nil {
		return 0, fmt.Errorf("purge voip_streams: %w", err)
	}
	if res == nil {
		return 0, errors.New("purge voip_streams: nil sql.Result")
	}
	return res.RowsAffected()
}
