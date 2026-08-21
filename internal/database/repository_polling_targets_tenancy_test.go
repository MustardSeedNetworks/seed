package database_test

// repository_polling_targets_tenancy_test.go covers the seam split in #1797:
// the scheduler reads across clients on purpose, management never does, and a
// cross-client operation is indistinguishable from a missing row *and* makes
// no mutation. The second half matters as much as the first — a scoped read
// that still writes would leak through the write path.

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

const (
	clientA = database.DefaultClientID
	clientB = "other-tenant"
)

// twoTenantDB returns a DB holding one enabled and one disabled target for
// clientA, plus the enabled target owned by clientB that the cross-client
// cases reach for.
func twoTenantDB(t *testing.T) (*database.DB, *polling.Target) {
	t.Helper()

	db, cleanup := testDB(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	if err := db.Clients().Create(ctx, &database.Client{
		ID: clientB, Name: "Other", Slug: "other",
	}); err != nil {
		t.Fatalf("create client: %v", err)
	}

	mk := func(clientID, name string, enabled bool) *polling.Target {
		target := &polling.Target{
			ClientID: clientID, Name: name, IPAddress: "10.0.0.1",
			SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: enabled,
			CollectorChain: []string{"sys_info"},
		}
		if err := db.PollingTargets().Create(ctx, target); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		return target
	}

	mk(clientA, "a-enabled", true)
	mk(clientA, "a-disabled", false)
	return db, mk(clientB, "b-enabled", true)
}

func names(targets []*polling.Target) []string {
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.Name)
	}
	return out
}

func TestListEnabledCrossesClientsButHidesDisabled(t *testing.T) {
	t.Parallel()
	db, _ := twoTenantDB(t)

	got, err := db.PollingTargets().ListEnabled(context.Background())
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListEnabled returned %v, want the two enabled targets across both clients", names(got))
	}
	for _, target := range got {
		if !target.Enabled {
			t.Errorf("ListEnabled returned disabled target %q — the scheduler would poll it", target.Name)
		}
	}
}

func TestListAllIsClientScopedAndIncludesDisabled(t *testing.T) {
	t.Parallel()
	db, _ := twoTenantDB(t)

	got, err := db.PollingTargets().ListAll(context.Background(), clientA)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAll(%q) returned %v, want both of that client's targets", clientA, names(got))
	}
	for _, target := range got {
		if target.ClientID != clientA {
			t.Errorf("ListAll leaked a target owned by %q", target.ClientID)
		}
	}
	var sawDisabled bool
	for _, target := range got {
		if !target.Enabled {
			sawDisabled = true
		}
	}
	if !sawDisabled {
		t.Error("ListAll hid the disabled target — that is the row an operator came to re-enable")
	}
}

func TestListAllRefusesAnEmptyClient(t *testing.T) {
	t.Parallel()
	db, _ := twoTenantDB(t)

	if _, err := db.PollingTargets().ListAll(context.Background(), ""); err == nil {
		t.Error("ListAll(\"\") succeeded; an absent client must be an error, not the whole fleet")
	}
}

func TestCrossClientGetIsIndistinguishableFromMissing(t *testing.T) {
	t.Parallel()
	db, foreign := twoTenantDB(t)
	ctx := context.Background()

	_, crossErr := db.PollingTargets().Get(ctx, clientA, foreign.ID)
	_, missingErr := db.PollingTargets().Get(ctx, clientA, "tgt-does-not-exist")

	if crossErr == nil {
		t.Fatal("Get read a target owned by another client")
	}
	if crossErr.Error() != missingErr.Error() {
		t.Errorf("cross-client error %q differs from missing-row error %q — the difference tells a "+
			"caller the id exists", crossErr, missingErr)
	}
}

func TestCrossClientUpdateMakesNoMutation(t *testing.T) {
	t.Parallel()
	db, foreign := twoTenantDB(t)
	ctx := context.Background()

	err := db.PollingTargets().Update(ctx, clientA, &polling.Target{
		ID: foreign.ID, Name: "stolen", IPAddress: "10.9.9.9",
		SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: false,
	})
	if !errors.Is(err, polling.ErrTargetNotFound) {
		t.Fatalf("Update across clients returned %v, want ErrTargetNotFound", err)
	}

	after, getErr := db.PollingTargets().Get(ctx, clientB, foreign.ID)
	if getErr != nil {
		t.Fatalf("re-read: %v", getErr)
	}
	if after.Name != foreign.Name || after.IPAddress != foreign.IPAddress {
		t.Errorf("the foreign target was mutated: name %q ip %q", after.Name, after.IPAddress)
	}
	if !after.Enabled {
		t.Error("the foreign target was disabled by a caller from another client")
	}
}

func TestCrossClientDeleteMakesNoMutation(t *testing.T) {
	t.Parallel()
	db, foreign := twoTenantDB(t)
	ctx := context.Background()

	if err := db.PollingTargets().Delete(ctx, clientA, foreign.ID); !errors.Is(err, polling.ErrTargetNotFound) {
		t.Fatalf("Delete across clients returned %v, want ErrTargetNotFound", err)
	}
	if _, err := db.PollingTargets().Get(ctx, clientB, foreign.ID); err != nil {
		t.Errorf("the foreign target was deleted: %v", err)
	}
}

func TestCreateRefusesATargetWithNoClient(t *testing.T) {
	t.Parallel()
	db, _ := twoTenantDB(t)

	err := db.PollingTargets().Create(context.Background(), &polling.Target{
		Name: "orphan", IPAddress: "10.0.0.2", SNMPVersion: "v2c",
	})
	if err == nil {
		t.Error("Create accepted a target with no owning client; it used to substitute the default")
	}
}
