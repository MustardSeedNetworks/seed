package orchestrator_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
func at() time.Time              { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

// openTestDB returns a freshly migrated database backed by a
// temporary file. Closed automatically when t terminates.
func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func allLicensed(string) bool { return true }

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
				Rates:         db.Metrics(),
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
			"missing Rates",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Observations:  db.SNMPObservations(),
				Scheduler:     sched,
				ClientFactory: nopClientFactory,
				Credentials:   db.DeviceCredentials(),
				Decrypter:     nopDecrypter{},
			},
		},
		{
			"missing Scheduler",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Observations:  db.SNMPObservations(),
				Rates:         db.Metrics(),
				ClientFactory: nopClientFactory,
			},
		},
		{
			"missing ClientFactory",
			orchestrator.Config{
				Targets:      db.PollingTargets(),
				Observations: db.SNMPObservations(),
				Rates:        db.Metrics(),
				Scheduler:    sched,
			},
		},
		{
			// A poller without these cannot authenticate, so Build must refuse
			// rather than produce one that polls unauthenticated.
			"missing Credentials",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Observations:  db.SNMPObservations(),
				Rates:         db.Metrics(),
				Scheduler:     sched,
				ClientFactory: nopClientFactory,
				Decrypter:     nopDecrypter{},
			},
		},
		{
			"missing Decrypter",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Observations:  db.SNMPObservations(),
				Rates:         db.Metrics(),
				Scheduler:     sched,
				ClientFactory: nopClientFactory,
				Credentials:   db.DeviceCredentials(),
			},
		},
		{
			// Which collectors are for sale is the licence's decision; a
			// poller built without asking would run every one of them.
			"missing Licensed",
			orchestrator.Config{
				Targets:       db.PollingTargets(),
				Observations:  db.SNMPObservations(),
				Scheduler:     sched,
				ClientFactory: nopClientFactory,
				Credentials:   db.DeviceCredentials(),
				Decrypter:     nopDecrypter{},
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
		Rates:         db.Metrics(),
		Scheduler:     sched,
		ClientFactory: nopClientFactory,
		Logger:        silentLogger(),
		Now:           at,
		Credentials:   db.DeviceCredentials(),
		Decrypter:     nopDecrypter{},
		Licensed:      allLicensed,
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
		Rates:         db.Metrics(),
		Scheduler:     sched,
		ClientFactory: nopClientFactory,
		Logger:        silentLogger(),
		Now:           at,
		Credentials:   db.DeviceCredentials(),
		Decrypter:     nopDecrypter{},
		Licensed:      allLicensed,
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

func TestBuild_RegistersAllTenCollectorChainKinds(t *testing.T) {
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
		Rates:         db.Metrics(),
		Scheduler:     sched,
		ClientFactory: nopClientFactory,
		Logger:        silentLogger(),
		Now:           at,
		Credentials:   db.DeviceCredentials(),
		Decrypter:     nopDecrypter{},
		Licensed:      allLicensed,
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

// nopDecrypter satisfies the decrypter seam for wiring tests; credential
// decryption itself is covered in internal/polling/snmp.
type nopDecrypter struct{}

func (nopDecrypter) DecryptValue(encrypted string) (string, error) { return encrypted, nil }

// seedHostResourcesTarget stores one credentialed target whose chain is only
// host_resources.
func seedHostResourcesTarget(t *testing.T, db *database.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(ctx, `
		INSERT INTO device_credentials
		  (id, client_id, name, kind, snmp_community_enc, created_at, updated_at)
		VALUES ('cred-1', 'default', 'lab', 'v2c', ?, ?, ?)
	`, []byte("enc:v1:community"), now, now); err != nil {
		t.Fatalf("seed credential: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO polling_targets
		  (id, name, ip_address, snmp_version, credentials_id, poll_interval_seconds, enabled,
		   collector_chain, created_at, updated_at, client_id)
		VALUES ('t-1', 'server-1', '10.0.0.1', 'v2c', 'cred-1', 3600, 1, '["host_resources"]', ?, ?, 'default')
	`, now, now); err != nil {
		t.Fatalf("seed target: %v", err)
	}
}

// firstChainStatus starts poller, waits for its first chain to complete and
// returns the status after stopping it.
func firstChainStatus(t *testing.T, poller *snmp.Poller) engine.Status {
	t.Helper()
	ctx := context.Background()
	if err := poller.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for poller.Status().LastTickAt.IsZero() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if err := poller.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	status := poller.Status()
	if status.LastTickAt.IsZero() {
		t.Fatal("the target's chain never ran")
	}
	return status
}

// A collector the licence does not cover is never run, even for a target whose
// chain names it (server_monitoring and bgp_monitoring are Pro, seed#2327).
// Running it is observable as a dial through the client factory.
func TestBuild_UnlicensedCollectorNeverRuns(t *testing.T) {
	t.Parallel()

	for _, licensed := range []bool{true, false} {
		t.Run(fmt.Sprintf("licensed=%v", licensed), func(t *testing.T) {
			t.Parallel()
			db := openTestDB(t)
			seedHostResourcesTarget(t, db)

			var dials atomic.Int32
			poller, err := orchestrator.Build(orchestrator.Config{
				Targets:      db.PollingTargets(),
				Observations: db.SNMPObservations(),
				Scheduler:    scheduler.New(5 * time.Millisecond),
				ClientFactory: func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) {
					dials.Add(1)
					return nil, errors.New("orchestrator test: no device")
				},
				Logger:      silentLogger(),
				Now:         at,
				Credentials: db.DeviceCredentials(),
				Decrypter:   nopDecrypter{},
				Licensed:    func(name string) bool { return name != hostresources.Name || licensed },
			})
			if err != nil {
				t.Fatalf("Build: %v", err)
			}

			status := firstChainStatus(t, poller)
			if ran := dials.Load() > 0; ran != licensed {
				t.Errorf("host_resources dialled = %v, want %v (last error %q)", ran, licensed, status.LastError)
			}
			if !licensed && !strings.Contains(status.LastError, "not registered") {
				t.Errorf("last error = %q, want the collector reported as not registered", status.LastError)
			}
		})
	}
}
