package persistence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/settings/persistence"
)

type fakeStore struct {
	activeID    string
	activeErr   error
	defaultID   string
	defaultErr  error
	savedID     string
	savedConfig string
	saveErr     error
	saveCalls   int
}

func (f *fakeStore) ActiveProfileID(context.Context) (string, error) {
	return f.activeID, f.activeErr
}

func (f *fakeStore) DefaultProfileID(context.Context) (string, error) {
	return f.defaultID, f.defaultErr
}

func (f *fakeStore) SaveProfileConfig(_ context.Context, id, configJSON string) error {
	f.saveCalls++
	f.savedID = id
	f.savedConfig = configJSON
	return f.saveErr
}

type fakeConfig struct {
	json string
	err  error
}

func (f fakeConfig) ProfileJSON() (string, error) { return f.json, f.err }

func TestSaveToActiveProfileUsesActiveID(t *testing.T) {
	store := &fakeStore{activeID: "active-1"}
	uc := persistence.NewService(store, fakeConfig{json: `{"k":"v"}`})

	if err := uc.SaveToActiveProfile(context.Background()); err != nil {
		t.Fatalf("SaveToActiveProfile: %v", err)
	}
	if store.saveCalls != 1 {
		t.Fatalf("expected 1 save, got %d", store.saveCalls)
	}
	if store.savedID != "active-1" {
		t.Fatalf("saved to %q, want active-1", store.savedID)
	}
	if store.savedConfig != `{"k":"v"}` {
		t.Fatalf("saved config %q", store.savedConfig)
	}
}

func TestSaveToActiveProfileFallsBackToDefault(t *testing.T) {
	// Empty active id and a lookup error both route to the default profile.
	for _, tc := range []struct {
		name  string
		store *fakeStore
	}{
		{"empty active", &fakeStore{activeID: "", defaultID: "default-1"}},
		{"active lookup error", &fakeStore{activeErr: errors.New("boom"), defaultID: "default-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := persistence.NewService(tc.store, fakeConfig{json: "{}"})
			if err := uc.SaveToActiveProfile(context.Background()); err != nil {
				t.Fatalf("SaveToActiveProfile: %v", err)
			}
			if tc.store.savedID != "default-1" {
				t.Fatalf("saved to %q, want default-1", tc.store.savedID)
			}
		})
	}
}

func TestSaveToActiveProfileNoProfileIsNoOp(t *testing.T) {
	store := &fakeStore{activeID: "", defaultErr: persistence.ErrNoProfile}
	uc := persistence.NewService(store, fakeConfig{json: "{}"})

	if err := uc.SaveToActiveProfile(context.Background()); err != nil {
		t.Fatalf("expected nil (no-op), got %v", err)
	}
	if store.saveCalls != 0 {
		t.Fatalf("expected no save, got %d", store.saveCalls)
	}
}

func TestSaveToActiveProfileSerializeErrorPropagates(t *testing.T) {
	wantErr := errors.New("serialize failed")
	store := &fakeStore{activeID: "active-1"}
	uc := persistence.NewService(store, fakeConfig{err: wantErr})

	if err := uc.SaveToActiveProfile(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected serialize error to propagate, got %v", err)
	}
	if store.saveCalls != 0 {
		t.Fatalf("expected no save on serialize error, got %d", store.saveCalls)
	}
}

func TestSaveToActiveProfileDefaultLookupErrorPropagates(t *testing.T) {
	// A non-sentinel default-profile lookup error is real and must propagate
	// (only ErrNoProfile is the legitimate no-op).
	wantErr := errors.New("db down")
	store := &fakeStore{activeID: "", defaultErr: wantErr}
	uc := persistence.NewService(store, fakeConfig{json: "{}"})

	if err := uc.SaveToActiveProfile(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected default lookup error to propagate, got %v", err)
	}
}

func TestSaveToActiveProfileSaveErrorPropagates(t *testing.T) {
	wantErr := errors.New("update failed")
	store := &fakeStore{activeID: "active-1", saveErr: wantErr}
	uc := persistence.NewService(store, fakeConfig{json: "{}"})

	if err := uc.SaveToActiveProfile(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected save error to propagate, got %v", err)
	}
}

func TestSaveToActiveProfileNilStoreIsNoOp(t *testing.T) {
	uc := persistence.NewService(nil, fakeConfig{json: "{}"})
	if err := uc.SaveToActiveProfile(context.Background()); err != nil {
		t.Fatalf("nil store should be a no-op, got %v", err)
	}
	var nilUC *persistence.Service
	if err := nilUC.SaveToActiveProfile(context.Background()); err != nil {
		t.Fatalf("nil use-case should be a no-op, got %v", err)
	}
}
