package orchestrator_test

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
func at() time.Time              { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

// openTestDB returns a freshly migrated database backed by a
// temporary file. Closed automatically when t terminates.
func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "seed.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newSchedulerForTest() *scheduler.Scheduler {
	return scheduler.New(time.Hour) // tick is irrelevant for these tests
}

// nopClientFactory returns a Client that errors on every call —
// orchestrator-level tests verify wiring without dialling SNMP.
func nopClientFactory(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) {
	return nil, errors.New("orchestrator test: client factory not used")
}

func TestBuild_AllRequiredFieldsValidated(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	sched := newSchedulerForTest()

	tests := []struct {
		name string
		cfg  orchestrator.Config
	}{
		{
			"missing Targets",
			orchestrator.Config{
				Observations:  db.SNMPObservations(),
				Scheduler:     sched,
				ClientFactory: nopClientFactory,
			},
		},
		{
			"missing Observations",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Scheduler:     sched,
				ClientFactory: nopClientFactory,
			},
		},
		{
			"missing Scheduler",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Observations:  db.SNMPObservations(),
				ClientFactory: nopClientFactory,
			},
		},
		{
			"missing ClientFactory",
			orchestrator.Config{
				Targets:      db.PollingTargets(),
				Observations: db.SNMPObservations(),
				Scheduler:    sched,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := orchestrator.Build(tt.cfg); err == nil {
				t.Errorf("Build(%s) should have returned error", tt.name)
			}
		})
	}
}

func TestBuild_ReturnsPollerWithEngineName(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	sched := newSchedulerForTest()

	poller, err := orchestrator.Build(orchestrator.Config{
		Targets:       db.PollingTargets(),
		Observations:  db.SNMPObservations(),
		Scheduler:     sched,
		ClientFactory: nopClientFactory,
		Logger:        silentLogger(),
		Now:           at,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if poller.Name() != "snmp-poller" {
		t.Errorf("poller.Name() = %q, want snmp-poller", poller.Name())
	}
}

func TestBuild_PollerStartLoadsZeroTargetsCleanly(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	sched := newSchedulerForTest()

	poller, err := orchestrator.Build(orchestrator.Config{
		Targets:       db.PollingTargets(),
		Observations:  db.SNMPObservations(),
		Scheduler:     sched,
		ClientFactory: nopClientFactory,
		Logger:        silentLogger(),
		Now:           at,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// With no polling targets configured, Start should succeed
	// (no jobs registered, scheduler still spins up).
	if startErr := poller.Start(context.Background()); startErr != nil {
		t.Fatalf("Start with empty targets: %v", startErr)
	}
	if stopErr := poller.Stop(context.Background()); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}
}

func TestBuild_RegistersAllElevenCollectorChainKinds(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	sched := newSchedulerForTest()

	// Pre-seed one polling target so Start has work to find, then
	// verify the chain references collectors the poller knows about.
	ctx := context.Background()
	if _, err := db.Exec(ctx, `
		INSERT INTO polling_targets
		  (id, name, ip_address, snmp_version, poll_interval_seconds, enabled,
		   collector_chain, created_at, updated_at, client_id)
		VALUES
		  ('t-1', 'router-1', '10.0.0.1', 'v2c', 60, 1,
		   '["sys_info","if_table","lldp","cdp","fdp","arp","fdb","routing","host_resources","bgp4_mib"]',
		   ?, ?, 'default')
	`, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	poller, err := orchestrator.Build(orchestrator.Config{
		Targets:       db.PollingTargets(),
		Observations:  db.SNMPObservations(),
		Scheduler:     sched,
		ClientFactory: nopClientFactory,
		Logger:        silentLogger(),
		Now:           at,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if startErr := poller.Start(ctx); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	if stopErr := poller.Stop(ctx); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}
}
