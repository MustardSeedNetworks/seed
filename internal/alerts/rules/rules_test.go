package rules_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts/rules"
)

type fakeRuleStore struct {
	byID      map[int64]rules.Rule
	nextID    int64
	createErr error
	updateErr error
}

func newFakeRuleStore() *fakeRuleStore {
	return &fakeRuleStore{byID: map[int64]rules.Rule{}, nextID: 1}
}

func (f *fakeRuleStore) List(_ context.Context, enabledOnly bool) ([]rules.Rule, error) {
	out := make([]rules.Rule, 0, len(f.byID))
	for _, r := range f.byID {
		if enabledOnly && !r.Enabled {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRuleStore) Get(_ context.Context, id int64) (rules.Rule, error) {
	r, ok := f.byID[id]
	if !ok {
		return rules.Rule{}, rules.ErrNotFound
	}
	return r, nil
}

func (f *fakeRuleStore) Create(_ context.Context, r rules.Rule) (rules.Rule, error) {
	if f.createErr != nil {
		return rules.Rule{}, f.createErr
	}
	r.ID = f.nextID
	f.nextID++
	f.byID[r.ID] = r
	return r, nil
}

func (f *fakeRuleStore) Update(_ context.Context, r rules.Rule) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.byID[r.ID]; !ok {
		return rules.ErrNotFound
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeRuleStore) Delete(_ context.Context, id int64) error {
	if _, ok := f.byID[id]; !ok {
		return rules.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func ctx() context.Context { return context.Background() }

func TestCreateNormalizesThresholdAndIgnoresInputID(t *testing.T) {
	store := newFakeRuleStore()
	svc := rules.NewService(store)

	got, err := svc.Create(ctx(), rules.Rule{ID: 999, Name: "n", ThresholdCount: 0})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ID != 1 {
		t.Fatalf("store should assign id, got %d", got.ID)
	}
	if got.ThresholdCount != 1 {
		t.Fatalf("threshold should normalize to 1, got %d", got.ThresholdCount)
	}
}

func TestCreateErrorPropagates(t *testing.T) {
	store := newFakeRuleStore()
	store.createErr = &rules.ValidationError{Msg: "alert_rules: bad matchKind"}
	svc := rules.NewService(store)

	_, err := svc.Create(ctx(), rules.Rule{Name: "n"})
	var ve *rules.ValidationError
	if !errors.As(err, &ve) || ve.Msg != "alert_rules: bad matchKind" {
		t.Fatalf("want ValidationError carrying the message, got %v", err)
	}
}

func TestUpdateReadsBackAndMaps(t *testing.T) {
	store := newFakeRuleStore()
	created, _ := store.Create(ctx(), rules.Rule{Name: "orig", ThresholdCount: 3})
	svc := rules.NewService(store)

	got, err := svc.Update(ctx(), created.ID, rules.Rule{Name: "updated", ThresholdCount: 0})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "updated" || got.ThresholdCount != 1 {
		t.Fatalf("update mismatch: %+v", got)
	}

	if _, missErr := svc.Update(ctx(), 404, rules.Rule{Name: "x"}); !errors.Is(missErr, rules.ErrNotFound) {
		t.Fatalf("update missing should be ErrNotFound, got %v", missErr)
	}
}

func TestGetDeleteNotFound(t *testing.T) {
	svc := rules.NewService(newFakeRuleStore())
	if _, err := svc.Get(ctx(), 1); !errors.Is(err, rules.ErrNotFound) {
		t.Fatalf("get missing: %v", err)
	}
	if err := svc.Delete(ctx(), 1); !errors.Is(err, rules.ErrNotFound) {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestListEnabledOnly(t *testing.T) {
	store := newFakeRuleStore()
	_, _ = store.Create(ctx(), rules.Rule{Name: "on", Enabled: true})
	_, _ = store.Create(ctx(), rules.Rule{Name: "off", Enabled: false})
	svc := rules.NewService(store)

	all, _ := svc.List(ctx(), false)
	enabled, _ := svc.List(ctx(), true)
	if len(all) != 2 || len(enabled) != 1 {
		t.Fatalf("enabled filter: all=%d enabled=%d", len(all), len(enabled))
	}
}
