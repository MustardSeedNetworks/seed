package targets_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/targets"
)

type fakeRepo struct {
	store     map[string]*polling.Target
	createErr error
	failIP    string
	created   int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[string]*polling.Target{}} }

func (f *fakeRepo) ListAll(context.Context, string) ([]*polling.Target, error) {
	out := make([]*polling.Target, 0, len(f.store))
	for _, t := range f.store {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, _ string, id string) (*polling.Target, error) {
	if t, ok := f.store[id]; ok {
		return t, nil
	}
	return nil, polling.ErrTargetNotFound
}

func (f *fakeRepo) Create(_ context.Context, t *polling.Target) error {
	if f.createErr != nil && (f.failIP == "" || f.failIP == t.IPAddress) {
		return f.createErr
	}
	f.created++
	if t.ID == "" {
		t.ID = fmt.Sprintf("generated-%d", f.created)
	}
	f.store[t.ID] = t
	return nil
}

func (f *fakeRepo) Update(_ context.Context, _ string, t *polling.Target) error {
	if _, ok := f.store[t.ID]; !ok {
		return polling.ErrTargetNotFound
	}
	f.store[t.ID] = t
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, _ string, id string) error {
	if _, ok := f.store[id]; !ok {
		return polling.ErrTargetNotFound
	}
	delete(f.store, id)
	return nil
}

func TestCreateClassifiesRepoValidationError(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("polling_targets: name must be unique")
	svc := targets.NewService(repo)

	err := svc.Create(context.Background(), &polling.Target{Name: "x"})
	var ve targets.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	if ve.Msg != "polling_targets: name must be unique" {
		t.Errorf("validation message not preserved: %q", ve.Msg)
	}
}

func TestGetAndDeleteMapNotFound(t *testing.T) {
	svc := targets.NewService(newFakeRepo())
	if _, err := svc.Get(context.Background(), "acme", "missing"); !errors.Is(err, targets.ErrNotFound) {
		t.Errorf("Get: want ErrNotFound, got %v", err)
	}
	if err := svc.Delete(context.Background(), "acme", "missing"); !errors.Is(err, targets.ErrNotFound) {
		t.Errorf("Delete: want ErrNotFound, got %v", err)
	}
}

func TestUpdateEchoesFreshRowAndMapsNotFound(t *testing.T) {
	repo := newFakeRepo()
	repo.store["t1"] = &polling.Target{ID: "t1", Name: "old"}
	svc := targets.NewService(repo)

	got, err := svc.Update(context.Background(), "acme", &polling.Target{ID: "t1", Name: "new"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Update did not echo the fresh row: %+v", got)
	}

	_, err = svc.Update(context.Background(), "acme", &polling.Target{ID: "nope"})
	if !errors.Is(err, targets.ErrNotFound) {
		t.Errorf("Update missing: want ErrNotFound, got %v", err)
	}
}

func target(ip, name string) *polling.Target {
	return &polling.Target{ClientID: "acme", Name: name, IPAddress: ip, SNMPVersion: "v2c"}
}

// CreateMissing is "add what is not there": discovery promotes the devices a
// sweep found answering SNMP, and the sweep runs again every minute
// (seed#2692). An address that already has a target keeps it untouched — a
// disabled target is how an operator says "do not poll this".
func TestCreateMissingAddsOnlyAbsentAddresses(t *testing.T) {
	repo := newFakeRepo()
	svc := targets.NewService(repo)
	existing := &polling.Target{
		ID: "tgt-1", ClientID: "acme", Name: "operator's own",
		IPAddress: "10.44.40.1", SNMPVersion: "v2c", Enabled: false,
	}
	if err := svc.Create(context.Background(), existing); err != nil {
		t.Fatalf("seed: %v", err)
	}

	created, err := svc.CreateMissing(context.Background(), "acme", []*polling.Target{
		target("10.44.40.1", "edge-rtr"),
		target("10.44.40.2", "core-sw"),
		nil,
	})
	if err != nil {
		t.Fatalf("CreateMissing: %v", err)
	}

	if created != 1 {
		t.Errorf("created = %d, want 1", created)
	}
	if len(repo.store) != 2 {
		t.Fatalf("store holds %d targets, want 2: %+v", len(repo.store), repo.store)
	}
	if kept := repo.store["tgt-1"]; kept.Name != "operator's own" || kept.Enabled {
		t.Errorf("existing target was rewritten: %+v", kept)
	}
}

// Two candidates for one address are one target: the table has no unique index
// on the address, so a duplicate here is a device polled twice.
func TestCreateMissingCollapsesDuplicateAddresses(t *testing.T) {
	repo := newFakeRepo()
	svc := targets.NewService(repo)

	created, err := svc.CreateMissing(context.Background(), "acme", []*polling.Target{
		target("10.44.40.2", "core-sw"),
		target("10.44.40.2", "core-sw again"),
	})
	if err != nil {
		t.Fatalf("CreateMissing: %v", err)
	}
	if created != 1 || len(repo.store) != 1 {
		t.Errorf("created %d targets (store %d), want 1", created, len(repo.store))
	}
}

// One unusable candidate must not cost the others their targets: a sweep of a
// hundred devices should not lose ninety-nine promotions to one bad row.
func TestCreateMissingReportsFailuresAndKeepsGoing(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("polling_targets: IPAddress required")
	repo.failIP = "10.44.40.1"
	svc := targets.NewService(repo)

	created, err := svc.CreateMissing(context.Background(), "acme", []*polling.Target{
		target("10.44.40.1", "edge-rtr"),
		target("10.44.40.2", "core-sw"),
	})

	if err == nil {
		t.Fatal("CreateMissing returned no error although a write failed")
	}
	if created != 1 || len(repo.store) != 1 {
		t.Errorf("created %d targets (store %d), want the one that could be written",
			created, len(repo.store))
	}
}

// Concurrent callers see one target per address. The read and the writes are
// one operation, or two sweeps that overlap each create a row for the same
// device.
func TestCreateMissingIsSerialised(t *testing.T) {
	repo := newFakeRepo()
	svc := targets.NewService(repo)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := svc.CreateMissing(context.Background(), "acme", []*polling.Target{
				target("10.44.40.1", "edge-rtr"),
				target("10.44.40.2", "core-sw"),
			}); err != nil {
				t.Errorf("CreateMissing: %v", err)
			}
		})
	}
	wg.Wait()

	if len(repo.store) != 2 {
		t.Fatalf("store holds %d targets, want 2: %+v", len(repo.store), repo.store)
	}
}
