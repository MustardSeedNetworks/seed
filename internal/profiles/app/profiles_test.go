package profilesapp_test

import (
	"context"
	"errors"
	"testing"

	profilesapp "github.com/MustardSeedNetworks/seed/internal/profiles/app"
)

// fakeStore is an in-memory Store for use-case tests.
type fakeStore struct {
	byID       map[string]profilesapp.Profile
	activeID   string
	createErr  error
	failGetDef bool
}

func newFakeStore() *fakeStore { return &fakeStore{byID: map[string]profilesapp.Profile{}} }

func (f *fakeStore) List(context.Context) ([]profilesapp.Profile, error) {
	out := make([]profilesapp.Profile, 0, len(f.byID))
	for _, p := range f.byID {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (profilesapp.Profile, error) {
	p, ok := f.byID[id]
	if !ok {
		return profilesapp.Profile{}, profilesapp.ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) GetByName(_ context.Context, name string) (profilesapp.Profile, bool, error) {
	for _, p := range f.byID {
		if p.Name == name {
			return p, true, nil
		}
	}
	return profilesapp.Profile{}, false, nil
}

func (f *fakeStore) GetDefault(context.Context) (profilesapp.Profile, error) {
	if f.failGetDef {
		return profilesapp.Profile{}, errors.New("db down")
	}
	for _, p := range f.byID {
		if p.IsDefault {
			return p, nil
		}
	}
	return profilesapp.Profile{}, profilesapp.ErrNotFound
}

func (f *fakeStore) Create(_ context.Context, p profilesapp.Profile) error {
	if f.createErr != nil {
		return f.createErr
	}
	for _, ex := range f.byID {
		if ex.Name == p.Name {
			return profilesapp.ErrNameExists
		}
	}
	f.byID[p.ID] = p
	return nil
}

func (f *fakeStore) Update(_ context.Context, p profilesapp.Profile, ifMatch string) error {
	existing, ok := f.byID[p.ID]
	if ifMatch != "" {
		if !ok {
			return profilesapp.ErrNotFound
		}
		if existing.ConfigJSON != ifMatch { // tests use ConfigJSON as the version token
			return profilesapp.ErrConflict
		}
	}
	f.byID[p.ID] = p
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id string) error { delete(f.byID, id); return nil }
func (f *fakeStore) Count(context.Context) (int, error)        { return len(f.byID), nil }
func (f *fakeStore) ActiveID(context.Context) (string, error)  { return f.activeID, nil }
func (f *fakeStore) SetActiveID(_ context.Context, id string) error {
	f.activeID = id
	return nil
}

func ctx() context.Context { return context.Background() }

func TestCreateValidatesNameAndStores(t *testing.T) {
	svc := profilesapp.NewService(newFakeStore())

	if _, err := svc.Create(ctx(), profilesapp.NewProfile{Name: ""}); !errors.Is(err, profilesapp.ErrNameRequired) {
		t.Fatalf("empty name should be ErrNameRequired, got %v", err)
	}

	p, err := svc.Create(ctx(), profilesapp.NewProfile{Name: "alpha", ConfigJSON: `{"a":1}`})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" || p.Name != "alpha" {
		t.Fatalf("unexpected created profile: %+v", p)
	}

	if _, dupErr := svc.Create(ctx(), profilesapp.NewProfile{Name: "alpha"}); !errors.Is(
		dupErr,
		profilesapp.ErrNameExists,
	) {
		t.Fatalf("duplicate name should be ErrNameExists, got %v", dupErr)
	}
}

func TestUpdatePartialSemantics(t *testing.T) {
	store := newFakeStore()
	store.byID["p1"] = profilesapp.Profile{ID: "p1", Name: "orig", Description: "d", ConfigJSON: "{}"}
	svc := profilesapp.NewService(store)

	// Empty name keeps existing; nil config keeps existing; description always applied.
	got, err := svc.Update(ctx(), "p1", profilesapp.ProfileUpdate{Name: "", Description: "new", ConfigJSON: nil}, "")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "orig" || got.Description != "new" || got.ConfigJSON != "{}" {
		t.Fatalf("partial update mismatch: %+v", got)
	}

	cfg := `{"x":2}`
	got, _ = svc.Update(ctx(), "p1", profilesapp.ProfileUpdate{Name: "renamed", ConfigJSON: &cfg}, "")
	if got.Name != "renamed" || got.ConfigJSON != cfg {
		t.Fatalf("full update mismatch: %+v", got)
	}

	if _, missErr := svc.Update(ctx(), "missing", profilesapp.ProfileUpdate{}, ""); !errors.Is(
		missErr,
		profilesapp.ErrNotFound,
	) {
		t.Fatalf("update missing should be ErrNotFound, got %v", missErr)
	}
}

func TestUpdateIfMatch(t *testing.T) {
	store := newFakeStore()
	// fakeStore.UpdateIfMatch treats ConfigJSON as the version token.
	store.byID["p1"] = profilesapp.Profile{ID: "p1", Name: "n", ConfigJSON: "v1"}
	svc := profilesapp.NewService(store)

	// Matching ETag writes through.
	upd := `v2`
	if _, err := svc.Update(ctx(), "p1", profilesapp.ProfileUpdate{ConfigJSON: &upd}, "v1"); err != nil {
		t.Fatalf("matching if-match should succeed: %v", err)
	}

	// Stale ETag (the row is now "v2") conflicts.
	again := `v3`
	if _, err := svc.Update(ctx(), "p1", profilesapp.ProfileUpdate{ConfigJSON: &again}, "v1"); !errors.Is(
		err,
		profilesapp.ErrConflict,
	) {
		t.Fatalf("stale if-match should be ErrConflict, got %v", err)
	}

	// Empty if-match stays unconditional.
	last := `v4`
	if _, err := svc.Update(ctx(), "p1", profilesapp.ProfileUpdate{ConfigJSON: &last}, ""); err != nil {
		t.Fatalf("unconditional update should succeed: %v", err)
	}
}

func TestDeleteGuards(t *testing.T) {
	store := newFakeStore()
	store.byID["def"] = profilesapp.Profile{ID: "def", Name: "default", IsDefault: true}
	store.byID["act"] = profilesapp.Profile{ID: "act", Name: "active"}
	store.byID["ok"] = profilesapp.Profile{ID: "ok", Name: "ok"}
	store.activeID = "act"
	svc := profilesapp.NewService(store)

	if err := svc.Delete(ctx(), "def"); !errors.Is(err, profilesapp.ErrDeleteDefault) {
		t.Fatalf("deleting default should be ErrDeleteDefault, got %v", err)
	}
	if err := svc.Delete(ctx(), "act"); !errors.Is(err, profilesapp.ErrDeleteActive) {
		t.Fatalf("deleting active should be ErrDeleteActive, got %v", err)
	}
	if err := svc.Delete(ctx(), "ok"); err != nil {
		t.Fatalf("deleting normal profile: %v", err)
	}
	if _, ok := store.byID["ok"]; ok {
		t.Fatal("profile 'ok' should have been deleted")
	}
}

func TestActiveProfileResolution(t *testing.T) {
	t.Run("explicit active", func(t *testing.T) {
		store := newFakeStore()
		store.byID["p1"] = profilesapp.Profile{ID: "p1", Name: "one"}
		store.activeID = "p1"
		got, err := profilesapp.NewService(store).ActiveProfile(ctx())
		if err != nil || got.ID != "p1" {
			t.Fatalf("want p1, got %+v err=%v", got, err)
		}
	})

	t.Run("falls back to default when unset", func(t *testing.T) {
		store := newFakeStore()
		store.byID["d"] = profilesapp.Profile{ID: "d", Name: "def", IsDefault: true}
		got, err := profilesapp.NewService(store).ActiveProfile(ctx())
		if err != nil || got.ID != "d" {
			t.Fatalf("want default d, got %+v err=%v", got, err)
		}
	})

	t.Run("no active and no default", func(t *testing.T) {
		_, err := profilesapp.NewService(newFakeStore()).ActiveProfile(ctx())
		if !errors.Is(err, profilesapp.ErrNoActiveOrDefault) {
			t.Fatalf("want ErrNoActiveOrDefault, got %v", err)
		}
	})

	t.Run("stale active self-heals to default", func(t *testing.T) {
		store := newFakeStore()
		store.byID["d"] = profilesapp.Profile{ID: "d", Name: "def", IsDefault: true}
		store.activeID = "ghost" // points at a deleted profile
		got, err := profilesapp.NewService(store).ActiveProfile(ctx())
		if err != nil || got.ID != "d" {
			t.Fatalf("want self-heal to d, got %+v err=%v", got, err)
		}
		if store.activeID != "d" {
			t.Fatalf("active id should be repointed to d, got %q", store.activeID)
		}
	})
}

func TestSetActiveProfile(t *testing.T) {
	store := newFakeStore()
	store.byID["p1"] = profilesapp.Profile{ID: "p1", Name: "one"}
	svc := profilesapp.NewService(store)

	if _, err := svc.SetActiveProfile(ctx(), ""); !errors.Is(err, profilesapp.ErrIDRequired) {
		t.Fatalf("empty id should be ErrIDRequired, got %v", err)
	}
	if _, err := svc.SetActiveProfile(ctx(), "missing"); !errors.Is(err, profilesapp.ErrNotFound) {
		t.Fatalf("missing id should be ErrNotFound, got %v", err)
	}
	p, err := svc.SetActiveProfile(ctx(), "p1")
	if err != nil || p.ID != "p1" || store.activeID != "p1" {
		t.Fatalf("set active failed: %+v err=%v active=%q", p, err, store.activeID)
	}
}

func TestDuplicateRetriesOnNameCollision(t *testing.T) {
	store := newFakeStore()
	store.byID["src"] = profilesapp.Profile{ID: "src", Name: "src", ConfigJSON: `{"k":1}`}
	store.byID["copy"] = profilesapp.Profile{ID: "copy", Name: "src (Copy)"} // forces first-attempt collision
	svc := profilesapp.NewService(store)

	dup, err := svc.Duplicate(ctx(), "src", "")
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if dup.Name == "src (Copy)" {
		t.Fatalf("expected timestamped retry name, got %q", dup.Name)
	}
	if dup.ConfigJSON != `{"k":1}` || dup.IsDefault {
		t.Fatalf("duplicate should copy config and not be default: %+v", dup)
	}
}

func TestImportCreateUpdateSkip(t *testing.T) {
	store := newFakeStore()
	store.byID["e"] = profilesapp.Profile{ID: "e", Name: "exists", ConfigJSON: "{}"}
	svc := profilesapp.NewService(store)

	res := svc.Import(ctx(), []profilesapp.ImportItem{
		{Name: "fresh", ConfigJSON: `{"a":1}`},
		{Name: ""},                              // skipped: name required
		{Name: "exists", ConfigJSON: `{"b":2}`}, // skipped: no overwrite
	}, false)

	if res.Created != 1 || res.Skipped != 2 || res.Updated != 0 {
		t.Fatalf("import(no overwrite) tally: %+v", res)
	}
	if len(res.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %v", res.Errors)
	}

	res = svc.Import(ctx(), []profilesapp.ImportItem{{Name: "exists", ConfigJSON: `{"c":3}`}}, true)
	if res.Updated != 1 || res.Created != 0 || res.Skipped != 0 {
		t.Fatalf("import(overwrite) tally: %+v", res)
	}
	if store.byID["e"].ConfigJSON != `{"c":3}` {
		t.Fatalf("overwrite should update config, got %q", store.byID["e"].ConfigJSON)
	}
}
