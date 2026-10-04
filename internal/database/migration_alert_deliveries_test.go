package database_test

// 00014 moves the webhook outcome 00013 kept on the alert row into
// alert_deliveries. An upgrade that dropped those columns without carrying
// them over would erase the one record of every receiver failure so far.

import (
	"context"
	"testing"
)

func TestMigration00014CarriesWebhookOutcomesOver(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 13)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO alerts (id, type, severity, title, message, created_at,
		                    delivery_status, delivery_attempted_at, delivery_error)
		VALUES (1, 'system', 'error', 'failed one', 'm', '2026-09-14T12:00:00Z',
		        'failed', '2026-09-14T12:00:05Z', 'receiver answered 500 for alert 1'),
		       (2, 'system', 'info', 'never offered', 'm', '2026-09-14T12:00:00Z',
		        '', NULL, '')
	`); err != nil {
		t.Fatalf("insert alerts: %v", err)
	}

	upTo(t, db, 14)

	rows, err := db.QueryContext(ctx,
		`SELECT alert_id, channel, status, attempted_at, error FROM alert_deliveries`)
	if err != nil {
		t.Fatalf("read alert_deliveries: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var id int64
		var channel, status, attemptedAt, errText string
		if scanErr := rows.Scan(&id, &channel, &status, &attemptedAt, &errText); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		got = append(got, channel+"|"+status+"|"+attemptedAt+"|"+errText)
		if id != 1 {
			t.Errorf("alert %d gained a delivery row; it was never offered to a receiver", id)
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("rows: %v", rowsErr)
	}
	want := "webhook|failed|2026-09-14T12:00:05Z|receiver answered 500 for alert 1"
	if len(got) != 1 || got[0] != want {
		t.Errorf("alert_deliveries = %v, want [%s]", got, want)
	}
}
