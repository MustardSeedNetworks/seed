package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

// TestRecordInterfaceRates writes the same ifIndex on two targets and checks
// both raw rows and the hourly rollup keep them apart: metrics_hourly is
// unique on interface_name, so a shared ifName or ifIndex there would make
// one device's hour overwrite the other's.
func TestRecordInterfaceRates(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	at := time.Date(2026, 10, 4, 12, 30, 0, 0, time.UTC)
	rates := []ifrate.Rate{
		{ClientID: "default", TargetID: "sw-a", IfIndex: 7, At: at, Octets: &ifrate.Octets{In: 100}, OutDiscards: 2},
		{ClientID: "default", TargetID: "sw-b", IfIndex: 7, At: at, Octets: &ifrate.Octets{In: 300}},
	}
	if err := db.Metrics().RecordInterfaceRates(ctx, rates); err != nil {
		t.Fatalf("RecordInterfaceRates: %v", err)
	}

	rows, err := db.Query(ctx, `
		SELECT target_kind, target_id, interface_name, unit, value, timestamp
		FROM metrics WHERE metric_type = ? ORDER BY target_id`, ifrate.MetricInOctets)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	type row struct {
		kind, id, name, unit, ts string
		value                    float64
	}
	var got []row
	for rows.Next() {
		var r row
		if scanErr := rows.Scan(&r.kind, &r.id, &r.name, &r.unit, &r.value, &r.ts); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		got = append(got, r)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("rows: %v", rowsErr)
	}
	want := []row{
		{ifrate.TargetKind, "sw-a/7", "sw-a/7", ifrate.UnitOctets, "2026-10-04T12:30:00Z", 100},
		{ifrate.TargetKind, "sw-b/7", "sw-b/7", ifrate.UnitOctets, "2026-10-04T12:30:00Z", 300},
	}
	if len(got) != len(want) {
		t.Fatalf("in-octet rows = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	count, err := db.Metrics().Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 12 {
		t.Errorf("metric rows = %d, want 12 (six points per rate)", count)
	}

	hour := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	if _, rollErr := database.NewMetricsRollupSource(db).RollupHour(ctx, hour); rollErr != nil {
		t.Fatalf("RollupHour: %v", rollErr)
	}
	var hourly int
	if scanErr := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM metrics_hourly WHERE metric_type = ?`, ifrate.MetricInOctets,
	).Scan(&hourly); scanErr != nil {
		t.Fatalf("hourly count: %v", scanErr)
	}
	if hourly != 2 {
		t.Errorf("hourly in-octet rows = %d, want one per target", hourly)
	}
}
