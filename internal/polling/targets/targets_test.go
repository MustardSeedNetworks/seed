package targets_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/targets"
)

type fakeRepo struct {
	store     map[string]*database.PollingTarget
	createErr error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[string]*database.PollingTarget{}} }

func (f *fakeRepo) List(context.Context, string) ([]*database.PollingTarget, error) {
	out := make([]*database.PollingTarget, 0, len(f.store))
	for _, t := range f.store {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*database.PollingTarget, error) {
	if t, ok := f.store[id]; ok {
		return t, nil
	}
	return nil, database.ErrPollingTargetNotFound
}

func (f *fakeRepo) Create(_ context.Context, t *database.PollingTarget) error {
	if f.createErr != nil {
		return f.createErr
	}
	if t.ID == "" {
		t.ID = "generated"
	}
	f.store[t.ID] = t
	return nil
}

func (f *fakeRepo) Update(_ context.Context, t *database.PollingTarget) error {
	if _, ok := f.store[t.ID]; !ok {
		return database.ErrPollingTargetNotFound
	}
	f.store[t.ID] = t
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.store[id]; !ok {
		return database.ErrPollingTargetNotFound
	}
	delete(f.store, id)
	return nil
}

func TestCreateClassifiesRepoValidationError(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("polling_targets: name must be unique")
	svc := targets.NewService(repo)

	err := svc.Create(context.Background(), &database.PollingTarget{Name: "x"})
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
	if _, err := svc.Get(context.Background(), "missing"); !errors.Is(err, targets.ErrNotFound) {
		t.Errorf("Get: want ErrNotFound, got %v", err)
	}
	if err := svc.Delete(context.Background(), "missing"); !errors.Is(err, targets.ErrNotFound) {
		t.Errorf("Delete: want ErrNotFound, got %v", err)
	}
}

func TestUpdateEchoesFreshRowAndMapsNotFound(t *testing.T) {
	repo := newFakeRepo()
	repo.store["t1"] = &database.PollingTarget{ID: "t1", Name: "old"}
	svc := targets.NewService(repo)

	got, err := svc.Update(context.Background(), &database.PollingTarget{ID: "t1", Name: "new"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Update did not echo the fresh row: %+v", got)
	}

	_, err = svc.Update(context.Background(), &database.PollingTarget{ID: "nope"})
	if !errors.Is(err, targets.ErrNotFound) {
		t.Errorf("Update missing: want ErrNotFound, got %v", err)
	}
}
