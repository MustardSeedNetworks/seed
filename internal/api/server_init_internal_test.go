// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/license"
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

// TestDatabaseServicesRegisterTheEnginesAndUseCases pins the wiring server.go
// builds over the database (seed#2750): the probe engine, the four topology
// reconcilers and the alert pipelines register, and the PAT and health
// use-cases reach their tables.
func TestDatabaseServicesRegisterTheEnginesAndUseCases(t *testing.T) {
	db := newTestDB(t)
	licenseDir := t.TempDir()
	mgr, err := license.NewManagerWithDir(licenseDir)
	if err != nil {
		t.Fatalf("license manager: %v", err)
	}
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}

	cfg := testutil.NewConfigBuilder().Build()
	s := &Server{
		config:     cfg,
		configPath: filepath.Join(t.TempDir(), "seed.json"),
		engines:    engine.NewRegistry(nil),
		licenseDir: licenseDir,
		dbConn:     db,
	}
	s.initDatabaseDependentServices()
	s.initHealthUseCases()
	s.initIdentityUseCases()

	var names []string
	for _, e := range s.engines.Engines() {
		names = append(names, e.Name())
	}
	for _, want := range []string{
		"probe", "probe-anomaly",
		"topology-sysinfo-reconciler", "topology-iftable-reconciler",
		"topology-edge-reconciler", "topology-arp-reconciler",
		"alert-listener-pipeline", "alert-observation-pipeline", "alert-escalation",
	} {
		if !slices.Contains(names, want) {
			t.Errorf("engine %q not registered; got %v", want, names)
		}
	}

	if err := s.identityTokens.Revoke(t.Context(), "no-such-token", "nobody"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("token revoke did not reach the api_tokens table: %v", err)
	}
	if _, err := s.healthMonitoring.Anomalies(t.Context(), ""); err != nil {
		t.Errorf("health anomalies did not reach the anomaly store: %v", err)
	}
	if err := s.healthSettings.SeedDefaults(t.Context(), config.HealthChecksConfig{}); err != nil {
		t.Errorf("health seed marker did not reach the settings table: %v", err)
	}
	if _, err := s.healthSettings.HealthCheckProbes(t.Context()); err != nil {
		t.Errorf("health probes did not reach the probes table: %v", err)
	}
}
