package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener/microburst"
)

// burstTimeFormat keeps millisecond precision at a fixed width so burst
// times sort correctly as text, the same reason as flowTimeFormat.
const burstTimeFormat = "2006-01-02T15:04:05.000Z"

// MicroburstsRepository owns microburst_events, written by the
// microburst listener (internal/listener/microburst).
type MicroburstsRepository struct {
	db *DB
}

// InsertMicrobursts implements microburst.Store: one batch, one transaction.
func (r *MicroburstsRepository) InsertMicrobursts(ctx context.Context, events []microburst.Event) error {
	if len(events) == 0 {
		return nil
	}
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO microburst_events
			  (timestamp, interface_name, direction, peak_utilization_pct,
			   duration_ms, sampling_mode, link_speed_mbps)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare microburst_events insert: %w", err)
		}
		defer func() { _ = stmt.Close() }()
		for i := range events {
			e := &events[i]
			if _, execErr := stmt.ExecContext(ctx,
				e.Start.UTC().Format(burstTimeFormat), e.Interface, string(e.Direction),
				e.PeakUtilization, e.Duration.Milliseconds(), microburst.SamplingMode,
				e.LinkSpeedMbps,
			); execErr != nil {
				return fmt.Errorf("insert microburst_event: %w", execErr)
			}
		}
		return nil
	})
}

// DeleteOlderThan removes bursts that started before cutoff.
func (r *MicroburstsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `DELETE FROM microburst_events WHERE timestamp < ?`,
		cutoff.UTC().Format(burstTimeFormat))
	if err != nil {
		return 0, fmt.Errorf("purge microburst_events: %w", err)
	}
	if res == nil {
		return 0, errors.New("purge microburst_events: nil sql.Result")
	}
	return res.RowsAffected()
}
