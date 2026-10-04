package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/appid"
)

// repository_flow_apps.go is application identification over the flow store
// (P-C4): the signature table the collector names flows with, and the top
// applications over a window. The table lives in settings; an absent key
// means the table Seed ships with.

// SettingKeyFlowAppSignatures holds the operator's signature table as JSON.
const SettingKeyFlowAppSignatures = "flow_app_signatures"

// AppSignatures is the table in effect and whether the operator replaced
// the builtin one.
type AppSignatures struct {
	Table  *appid.Table
	Custom bool
}

// FlowApplication is one application's traffic over a window.
type FlowApplication struct {
	Name    string `json:"name"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
}

// AppSignatures returns the signature table in effect. A stored table that
// no longer parses is an error, not a silent return to the builtin: the
// operator's naming would otherwise change without anyone saying so.
func (r *FlowRecordsRepository) AppSignatures(ctx context.Context) (*AppSignatures, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.signatures != nil {
		return r.signatures, nil
	}
	stored, err := r.db.Settings().GetValue(ctx, SettingKeyFlowAppSignatures)
	if err != nil {
		return nil, fmt.Errorf("read application signatures: %w", err)
	}
	if stored == "" {
		r.signatures = &AppSignatures{Table: appid.Builtin()}
		return r.signatures, nil
	}
	table, err := appid.Parse([]byte(stored))
	if err != nil {
		return nil, fmt.Errorf("stored application signatures: %w", err)
	}
	r.signatures = &AppSignatures{Table: table, Custom: true}
	return r.signatures, nil
}

// SetAppSignatures stores the operator's table; flows stored from now on
// are named by it.
func (r *FlowRecordsRepository) SetAppSignatures(ctx context.Context, table *appid.Table) error {
	data, err := json.Marshal(table)
	if err != nil {
		return fmt.Errorf("encode application signatures: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if setErr := r.db.Settings().Set(ctx, SettingKeyFlowAppSignatures, string(data)); setErr != nil {
		return setErr
	}
	r.signatures = &AppSignatures{Table: table, Custom: true}
	return nil
}

// ResetAppSignatures returns to the builtin table.
func (r *FlowRecordsRepository) ResetAppSignatures(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.db.Settings().Delete(ctx, SettingKeyFlowAppSignatures); err != nil {
		return err
	}
	r.signatures = &AppSignatures{Table: appid.Builtin()}
	return nil
}

// flowApplicationWindowSQL is flowWindowSQL for the per-application tables.
func flowApplicationWindowSQL(tier FlowTier, from, to time.Time) (string, string, string, error) {
	switch tier {
	case FlowTierRaw:
		return `SELECT application, bytes, packets FROM flow_records
			WHERE client_id = ? AND flow_end >= ? AND flow_end < ?`,
			from.UTC().Format(flowTimeFormat), to.UTC().Format(flowTimeFormat), nil
	case FlowTierHourly:
		return `SELECT application, bytes, packets FROM flow_applications_hourly
			WHERE client_id = ? AND hour_bucket >= ? AND hour_bucket <= ?`,
			from.UTC().Format(hourFormat), to.UTC().Format(hourFormat), nil
	case FlowTierDaily:
		return `SELECT application, bytes, packets FROM flow_applications_daily
			WHERE client_id = ? AND day_bucket >= ? AND day_bucket <= ?`,
			from.UTC().Format(dayFormat), to.UTC().Format(dayFormat), nil
	}
	return "", "", "", fmt.Errorf("unknown flow tier %d", tier)
}

// TopApplications returns the limit applications that carried the most over
// [from, to). Unidentified traffic is one entry, appid.Unknown, ranked with
// the rest so its share is visible rather than hidden.
func (r *FlowRecordsRepository) TopApplications(
	ctx context.Context, clientID string, tier FlowTier, from, to time.Time, by FlowRank, limit int,
) ([]FlowApplication, error) {
	return queryFlowTop(ctx, r.db, "top applications", flowTopRead{clientID, tier, from, to, by, limit},
		flowApplicationWindowSQL, `
		SELECT application, CAST(TOTAL(bytes) AS INTEGER) AS bytes, CAST(TOTAL(packets) AS INTEGER) AS packets
		FROM f
		GROUP BY application`, "application",
		func(rows *sql.Rows, a *FlowApplication) error { return rows.Scan(&a.Name, &a.Bytes, &a.Packets) })
}
