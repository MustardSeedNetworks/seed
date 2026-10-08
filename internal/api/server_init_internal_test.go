// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// TestNewServerWiresTheDatabaseServices pins what NewServer builds over a live
// database (seed#2750): the config admin is migrated into the users table, the
// MIB database is loaded, discovery reads SNMP credentials from the vault, and
// the retention engine and SNMP poller exist.
func TestNewServerWiresTheDatabaseServices(t *testing.T) {
	db := newTestDB(t)
	const adminHash = "$2a$10$abcdefghijklmnopqrstuuSq0wbGu2yQn6E0pE0Qb3PcJ2nqfhG5q"
	cfg := testutil.NewConfigBuilder().WithAuth("admin", adminHash).Build()
	cfg.Database.RetentionDays = 0

	s := NewServer(cfg, filepath.Join(t.TempDir(), "seed.json"), "",
		netif.NewMockManager(netif.DefaultMockConfig()), false, nil, db, nil)
	t.Cleanup(s.Close)

	if _, err := db.GetUser(t.Context(), "admin"); err != nil {
		t.Fatalf("config admin not migrated into the users table: %v", err)
	}
	stats, err := s.MibDB().Stats()
	if err != nil {
		t.Fatalf("MIB stats: %v", err)
	}
	if stats["oid_entries"] == 0 {
		t.Fatal("MIB database has no built-in OIDs")
	}
	if s.snmpCreds == nil {
		t.Fatal("discovery has no vault-backed SNMP credentials")
	}
	if s.retentionEngine == nil {
		t.Fatal("retention engine not built")
	}
	if s.snmpPoller == nil {
		t.Fatal("SNMP poller not built")
	}
}

// TestNewServerWithoutDatabaseSkipsTheDatabaseServices pins the no-database
// path: nothing database-backed is built and discovery probes no SNMP.
func TestNewServerWithoutDatabaseSkipsTheDatabaseServices(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()

	s := NewServer(cfg, filepath.Join(t.TempDir(), "seed.json"), "",
		netif.NewMockManager(netif.DefaultMockConfig()), false, nil, nil, nil)
	t.Cleanup(s.Close)

	if s.MibDB() != nil || s.snmpCreds != nil || s.retentionEngine != nil || s.snmpPoller != nil {
		t.Fatalf("database services built without a database: mib=%v creds=%v retention=%v poller=%v",
			s.MibDB() != nil, s.snmpCreds != nil, s.retentionEngine != nil, s.snmpPoller != nil)
	}
}
