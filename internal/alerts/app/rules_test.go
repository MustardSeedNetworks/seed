package alertsapp_test

import (
	"context"
	"errors"
	"testing"

	alertsapp "github.com/MustardSeedNetworks/seed/internal/alerts/app"
)

type fakeRuleStore struct {
	byID      map[int64]alertsapp.Rule
	nextID    int64
	createErr error
	updateErr error
}

func newFakeRuleStore() *fakeRuleStore {
	return &fakeRuleStore{byID: map[int64]alertsapp.Rule{}, nextID: 1}
}

func (f *fakeRuleStore) List(_ context.Context, enabledOnly bool) ([]alertsapp.Rule, error) {
	out := make([]alertsapp.Rule, 0, len(f.byID))
	for _, r := range f.byID {
		if enabledOnly && !r.Enabled {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRuleStore) Get(_ context.Context, id int64) (alertsapp.Rule, error) {
	r, ok := f.byID[id]
	if !ok {
		return alertsapp.Rule{}, alertsapp.ErrRuleNotFound
	}
	return r, nil
}

func (f *fakeRuleStore) Create(_ context.Context, r alertsapp.Rule) (alertsapp.Rule, error) {
	if f.createErr != nil {
		return alertsapp.Rule{}, f.createErr
	}
	r.ID = f.nextID
	f.nextID++
	f.byID[r.ID] = r
	return r, nil
}

func (f *fakeRuleStore) Update(_ context.Context, r alertsapp.Rule) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.byID[r.ID]; !ok {
		return alertsapp.ErrRuleNotFound
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeRuleStore) Delete(_ context.Context, id int64) error {
	if _, ok := f.byID[id]; !ok {
		return alertsapp.ErrRuleNotFound
	}
	delete(f.byID, id)
	return nil
}

func ctx() context.Context { return context.Background() }

func TestCreateNormalizesThresholdAndIgnoresInputID(t *testing.T) {
	store := newFakeRuleStore()
	svc := alertsapp.NewRuleService(store)

	got, err := svc.Create(ctx(), alertsapp.Rule{ID: 999, Name: "n", ThresholdCount: 0})
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
	store.createErr = &alertsapp.ValidationError{Msg: "alert_rules: bad matchKind"}
	svc := alertsapp.NewRuleService(store)

	_, err := svc.Create(ctx(), alertsapp.Rule{Name: "n"})
	var ve *alertsapp.ValidationError
	if !errors.As(err, &ve) || ve.Msg != "alert_rules: bad matchKind" {
		t.Fatalf("want ValidationError carrying the message, got %v", err)
	}
}

func TestUpdateReadsBackAndMaps(t *testing.T) {
	store := newFakeRuleStore()
	created, _ := store.Create(ctx(), alertsapp.Rule{Name: "orig", ThresholdCount: 3})
	svc := alertsapp.NewRuleService(store)

	got, err := svc.Update(ctx(), created.ID, alertsapp.Rule{Name: "updated", ThresholdCount: 0})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Name != "updated" || got.ThresholdCount != 1 {
		t.Fatalf("update mismatch: %+v", got)
	}

	if _, missErr := svc.Update(ctx(), 404, alertsapp.Rule{Name: "x"}); !errors.Is(missErr, alertsapp.ErrRuleNotFound) {
		t.Fatalf("update missing should be ErrRuleNotFound, got %v", missErr)
	}
}

func TestGetDeleteNotFound(t *testing.T) {
	svc := alertsapp.NewRuleService(newFakeRuleStore())
	if _, err := svc.Get(ctx(), 1); !errors.Is(err, alertsapp.ErrRuleNotFound) {
		t.Fatalf("get missing: %v", err)
	}
	if err := svc.Delete(ctx(), 1); !errors.Is(err, alertsapp.ErrRuleNotFound) {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestListEnabledOnly(t *testing.T) {
	store := newFakeRuleStore()
	_, _ = store.Create(ctx(), alertsapp.Rule{Name: "on", Enabled: true})
	_, _ = store.Create(ctx(), alertsapp.Rule{Name: "off", Enabled: false})
	svc := alertsapp.NewRuleService(store)

	all, _ := svc.List(ctx(), false)
	enabled, _ := svc.List(ctx(), true)
	if len(all) != 2 || len(enabled) != 1 {
		t.Fatalf("enabled filter: all=%d enabled=%d", len(all), len(enabled))
	}
}
