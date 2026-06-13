package catalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
)

// fakeStore is an in-memory Store for use-case tests.
type fakeStore struct {
	byID       map[string]catalog.Profile
	activeID   string
	createErr  error
	failGetDef bool
}

func newFakeStore() *fakeStore { return &fakeStore{byID: map[string]catalog.Profile{}} }

func (f *fakeStore) Available() bool { return true }

func (f *fakeStore) List(context.Context) ([]catalog.Profile, error) {
	out := make([]catalog.Profile, 0, len(f.byID))
	for _, p := range f.byID {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (catalog.Profile, error) {
	p, ok := f.byID[id]
	if !ok {
		return catalog.Profile{}, catalog.ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) GetByName(_ context.Context, name string) (catalog.Profile, bool, error) {
	for _, p := range f.byID {
		if p.Name == name {
			return p, true, nil
		}
	}
	return catalog.Profile{}, false, nil
}

func (f *fakeStore) GetDefault(context.Context) (catalog.Profile, error) {
	if f.failGetDef {
		return catalog.Profile{}, errors.New("db down")
	}
	for _, p := range f.byID {
		if p.IsDefault {
			return p, nil
		}
	}
	return catalog.Profile{}, catalog.ErrNotFound
}

func (f *fakeStore) Create(_ context.Context, p catalog.Profile) error {
	if f.createErr != nil {
		return f.createErr
	}
	for _, ex := range f.byID {
		if ex.Name == p.Name {
			return catalog.ErrNameExists
		}
	}
	f.byID[p.ID] = p
	return nil
}

func (f *fakeStore) Update(_ context.Context, p catalog.Profile, ifMatch string) error {
	existing, ok := f.byID[p.ID]
	if ifMatch != "" {
		if !ok {
			return catalog.ErrNotFound
		}
		if existing.ConfigJSON != ifMatch { // tests use ConfigJSON as the version token
			return catalog.ErrConflict
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
	svc := catalog.NewService(newFakeStore(), nil)

	if _, err := svc.Create(ctx(), catalog.NewProfile{Name: ""}); !errors.Is(err, catalog.ErrNameRequired) {
		t.Fatalf("empty name should be ErrNameRequired, got %v", err)
	}

	p, err := svc.Create(ctx(), catalog.NewProfile{Name: "alpha", ConfigJSON: `{"a":1}`})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" || p.Name != "alpha" {
		t.Fatalf("unexpected created profile: %+v", p)
	}

	if _, dupErr := svc.Create(ctx(), catalog.NewProfile{Name: "alpha"}); !errors.Is(
		dupErr,
		catalog.ErrNameExists,
	) {
		t.Fatalf("duplicate name should be ErrNameExists, got %v", dupErr)
	}
}

func TestUpdatePartialSemantics(t *testing.T) {
	store := newFakeStore()
	store.byID["p1"] = catalog.Profile{ID: "p1", Name: "orig", Description: "d", ConfigJSON: "{}"}
	svc := catalog.NewService(store, nil)

	// Empty name keeps existing; nil config keeps existing; description always applied.
	got, err := svc.Update(ctx(), "p1", catalog.ProfileUpdate{Name: "", Description: "new", ConfigJSON: nil}, "")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "orig" || got.Description != "new" || got.ConfigJSON != "{}" {
		t.Fatalf("partial update mismatch: %+v", got)
	}

	cfg := `{"x":2}`
	got, _ = svc.Update(ctx(), "p1", catalog.ProfileUpdate{Name: "renamed", ConfigJSON: &cfg}, "")
	if got.Name != "renamed" || got.ConfigJSON != cfg {
		t.Fatalf("full update mismatch: %+v", got)
	}

	if _, missErr := svc.Update(ctx(), "missing", catalog.ProfileUpdate{}, ""); !errors.Is(
		missErr,
		catalog.ErrNotFound,
	) {
		t.Fatalf("update missing should be ErrNotFound, got %v", missErr)
	}
}

func TestUpdateIfMatch(t *testing.T) {
	store := newFakeStore()
	// fakeStore.UpdateIfMatch treats ConfigJSON as the version token.
	store.byID["p1"] = catalog.Profile{ID: "p1", Name: "n", ConfigJSON: "v1"}
	svc := catalog.NewService(store, nil)

	// Matching ETag writes through.
	upd := `v2`
	if _, err := svc.Update(ctx(), "p1", catalog.ProfileUpdate{ConfigJSON: &upd}, "v1"); err != nil {
		t.Fatalf("matching if-match should succeed: %v", err)
	}

	// Stale ETag (the row is now "v2") conflicts.
	again := `v3`
	if _, err := svc.Update(ctx(), "p1", catalog.ProfileUpdate{ConfigJSON: &again}, "v1"); !errors.Is(
		err,
		catalog.ErrConflict,
	) {
		t.Fatalf("stale if-match should be ErrConflict, got %v", err)
	}

	// Empty if-match stays unconditional.
	last := `v4`
	if _, err := svc.Update(ctx(), "p1", catalog.ProfileUpdate{ConfigJSON: &last}, ""); err != nil {
		t.Fatalf("unconditional update should succeed: %v", err)
	}
}

func TestDeleteGuards(t *testing.T) {
	store := newFakeStore()
	store.byID["def"] = catalog.Profile{ID: "def", Name: "default", IsDefault: true}
	store.byID["act"] = catalog.Profile{ID: "act", Name: "active"}
	store.byID["ok"] = catalog.Profile{ID: "ok", Name: "ok"}
	store.activeID = "act"
	svc := catalog.NewService(store, nil)

	if err := svc.Delete(ctx(), "def"); !errors.Is(err, catalog.ErrDeleteDefault) {
		t.Fatalf("deleting default should be ErrDeleteDefault, got %v", err)
	}
	if err := svc.Delete(ctx(), "act"); !errors.Is(err, catalog.ErrDeleteActive) {
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
		store.byID["p1"] = catalog.Profile{ID: "p1", Name: "one"}
		store.activeID = "p1"
		got, err := catalog.NewService(store, nil).ActiveProfile(ctx())
		if err != nil || got.ID != "p1" {
			t.Fatalf("want p1, got %+v err=%v", got, err)
		}
	})

	t.Run("falls back to default when unset", func(t *testing.T) {
		store := newFakeStore()
		store.byID["d"] = catalog.Profile{ID: "d", Name: "def", IsDefault: true}
		got, err := catalog.NewService(store, nil).ActiveProfile(ctx())
		if err != nil || got.ID != "d" {
			t.Fatalf("want default d, got %+v err=%v", got, err)
		}
	})

	t.Run("no active and no default", func(t *testing.T) {
		_, err := catalog.NewService(newFakeStore(), nil).ActiveProfile(ctx())
		if !errors.Is(err, catalog.ErrNoActiveOrDefault) {
			t.Fatalf("want ErrNoActiveOrDefault, got %v", err)
		}
	})

	t.Run("stale active self-heals to default", func(t *testing.T) {
		store := newFakeStore()
		store.byID["d"] = catalog.Profile{ID: "d", Name: "def", IsDefault: true}
		store.activeID = "ghost" // points at a deleted profile
		got, err := catalog.NewService(store, nil).ActiveProfile(ctx())
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
	store.byID["p1"] = catalog.Profile{ID: "p1", Name: "one"}
	svc := catalog.NewService(store, nil)

	if _, err := svc.SetActiveProfile(ctx(), ""); !errors.Is(err, catalog.ErrIDRequired) {
		t.Fatalf("empty id should be ErrIDRequired, got %v", err)
	}
	if _, err := svc.SetActiveProfile(ctx(), "missing"); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("missing id should be ErrNotFound, got %v", err)
	}
	p, err := svc.SetActiveProfile(ctx(), "p1")
	if err != nil || p.ID != "p1" || store.activeID != "p1" {
		t.Fatalf("set active failed: %+v err=%v active=%q", p, err, store.activeID)
	}
}

// fakeLive records applied profile JSON and can be made to fail the persist.
type fakeLive struct {
	applied string
	err     error
}

func (f *fakeLive) Apply(_ context.Context, profileJSON string) error {
	f.applied = profileJSON
	return f.err
}

func TestSetActiveProfileAppliesConfig(t *testing.T) {
	store := newFakeStore()
	store.byID["p1"] = catalog.Profile{ID: "p1", Name: "one", ConfigJSON: `{"k":1}`}
	live := &fakeLive{}
	svc := catalog.NewService(store, live)

	if _, err := svc.SetActiveProfile(ctx(), "p1"); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if live.applied != `{"k":1}` {
		t.Errorf("expected the profile config to be applied, got %q", live.applied)
	}
}

func TestSetActiveProfileSurfacesApplyError(t *testing.T) {
	store := newFakeStore()
	store.byID["p1"] = catalog.Profile{ID: "p1", Name: "one", ConfigJSON: `{"k":1}`}
	svc := catalog.NewService(store, &fakeLive{err: catalog.ErrConfigApply})

	_, err := svc.SetActiveProfile(ctx(), "p1")
	if !errors.Is(err, catalog.ErrConfigApply) {
		t.Errorf("want ErrConfigApply, got %v", err)
	}
	// The profile is still activated even though the apply failed.
	if store.activeID != "p1" {
		t.Errorf("active id should be set despite apply error, got %q", store.activeID)
	}
}

func TestSetActiveProfileSkipsApplyWhenConfigEmpty(t *testing.T) {
	store := newFakeStore()
	store.byID["p1"] = catalog.Profile{ID: "p1", Name: "one"} // no ConfigJSON
	live := &fakeLive{err: errors.New("should not be called")}
	svc := catalog.NewService(store, live)

	if _, err := svc.SetActiveProfile(ctx(), "p1"); err != nil {
		t.Fatalf("set active with empty config should not invoke apply: %v", err)
	}
	if live.applied != "" {
		t.Errorf("apply should be skipped for an empty config, got %q", live.applied)
	}
}

func TestDuplicateRetriesOnNameCollision(t *testing.T) {
	store := newFakeStore()
	store.byID["src"] = catalog.Profile{ID: "src", Name: "src", ConfigJSON: `{"k":1}`}
	store.byID["copy"] = catalog.Profile{ID: "copy", Name: "src (Copy)"} // forces first-attempt collision
	svc := catalog.NewService(store, nil)

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
	store.byID["e"] = catalog.Profile{ID: "e", Name: "exists", ConfigJSON: "{}"}
	svc := catalog.NewService(store, nil)

	res := svc.Import(ctx(), []catalog.ImportItem{
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

	res = svc.Import(ctx(), []catalog.ImportItem{{Name: "exists", ConfigJSON: `{"c":3}`}}, true)
	if res.Updated != 1 || res.Created != 0 || res.Skipped != 0 {
		t.Fatalf("import(overwrite) tally: %+v", res)
	}
	if store.byID["e"].ConfigJSON != `{"c":3}` {
		t.Fatalf("overwrite should update config, got %q", store.byID["e"].ConfigJSON)
	}
}
