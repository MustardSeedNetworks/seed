package database_test

// 00021 replaces flow_records.version with format, because sFlow v5 and
// NetFlow v5 share version 5. Every stored row keeps its values and gains
// the name of the protocol its version number stood for.

import (
	"context"
	"testing"
)

func TestMigration00021NamesFlowFormat(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 20)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO flow_records (exporter, version, observation_domain, flow_start, flow_end,
		  src_addr, dst_addr, src_port, dst_port, protocol, tcp_flags, bytes, packets,
		  input_if, output_if, received_at)
		VALUES ('192.0.2.1', 5, 1, 's', 'e', '10.0.0.1', '10.0.0.2', 1, 2, 6, 24, 1500, 3, 4, 5, 'r'),
		       ('192.0.2.1', 9, 2, 's', 'e', '10.0.0.1', '10.0.0.2', 1, 2, 6, 24, 1500, 3, 4, 5, 'r'),
		       ('192.0.2.1', 10, 3, 's', 'e', '10.0.0.1', '10.0.0.2', 1, 2, 6, 24, 1500, 3, 4, 5, 'r');
	`); err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	upTo(t, db, 21)

	var got string
	if err := db.QueryRowContext(ctx, `
		SELECT group_concat(row, ';') FROM (
		  SELECT format || ',' || observation_domain || ',' || bytes || ',' || packets || ',' ||
		         input_if || ',' || output_if || ',' || tcp_flags AS row
		  FROM flow_records ORDER BY id)`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "netflow5,1,1500,3,4,5,24;netflow9,2,1500,3,4,5,24;ipfix,3,1500,3,4,5,24"; got != want {
		t.Errorf("rows = %q, want %q", got, want)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO flow_records (exporter, format, observation_domain, flow_start, flow_end,
		  src_addr, dst_addr, src_port, dst_port, protocol, tcp_flags, bytes, packets,
		  input_if, output_if, received_at)
		VALUES ('192.0.2.1', 'netflow7', 1, 's', 'e', '10.0.0.1', '10.0.0.2', 1, 2, 6, 0, 1, 1, 0, 0, 'r')`); err == nil {
		t.Error("an unknown format was stored")
	}
}
