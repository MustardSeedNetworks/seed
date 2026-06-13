package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	query "github.com/MustardSeedNetworks/seed/internal/logs/query"
)

type fakeStore struct {
	available bool
	entries   []*logging.LogEntry
	err       error
	gotParams query.Params
}

func (f *fakeStore) Available() bool { return f.available }
func (f *fakeStore) List(_ context.Context, p query.Params) ([]*logging.LogEntry, error) {
	f.gotParams = p
	return f.entries, f.err
}

type fakeBuffer struct {
	available bool
	entries   []*logging.LogEntry
}

func (f *fakeBuffer) Available() bool          { return f.available }
func (f *fakeBuffer) All() []*logging.LogEntry { return f.entries }

func entry(level, layer, component, msg string) *logging.LogEntry {
	return &logging.LogEntry{Level: level, Layer: layer, Component: component, Message: msg}
}

func TestQueryPrefersStore(t *testing.T) {
	store := &fakeStore{available: true, entries: []*logging.LogEntry{entry("ERROR", "api", "auth", "boom")}}
	buf := &fakeBuffer{available: true, entries: []*logging.LogEntry{entry("INFO", "backend", "x", "noise")}}
	svc := query.NewService(store, buf)

	res, err := svc.Query(context.Background(), query.Params{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res.Entries) != 1 || res.Entries[0].Message != "boom" {
		t.Fatalf("expected store entry, got %+v", res.Entries)
	}
	if res.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", res.TotalCount)
	}
}

func TestQueryFallsBackOnStoreError(t *testing.T) {
	store := &fakeStore{available: true, err: errors.New("db down")}
	buf := &fakeBuffer{available: true, entries: []*logging.LogEntry{
		entry("ERROR", "api", "auth", "failed login"),
		entry("INFO", "backend", "poll", "ok"),
	}}
	svc := query.NewService(store, buf)

	res, err := svc.Query(context.Background(), query.Params{Levels: []string{"error"}, Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res.Entries) != 1 || res.Entries[0].Message != "failed login" {
		t.Fatalf("expected the ERROR entry from the buffer, got %+v", res.Entries)
	}
}

func TestQueryBufferFilterAndSearch(t *testing.T) {
	buf := &fakeBuffer{available: true, entries: []*logging.LogEntry{
		entry("ERROR", "api", "auth", "DNS lookup failed"),
		entry("ERROR", "backend", "poll", "timeout"),
		entry("WARN", "api", "auth", "retrying"),
	}}
	svc := query.NewService(&fakeStore{available: false}, buf)

	res, err := svc.Query(context.Background(), query.Params{
		Levels: []string{"error"}, Layers: []string{"api"}, Search: "dns", Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res.Entries) != 1 || res.Entries[0].Message != "DNS lookup failed" {
		t.Fatalf("filter+search wrong: %+v", res.Entries)
	}
}

func TestQueryBufferPagination(t *testing.T) {
	all := make([]*logging.LogEntry, 5)
	for i := range all {
		all[i] = entry("INFO", "api", "x", "m")
	}
	svc := query.NewService(&fakeStore{available: false}, &fakeBuffer{available: true, entries: all})

	res, err := svc.Query(context.Background(), query.Params{Offset: 2, Limit: 2})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Errorf("page size = %d, want 2", len(res.Entries))
	}
	if res.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5 (pre-pagination)", res.TotalCount)
	}
}

func TestQueryNoSource(t *testing.T) {
	svc := query.NewService(&fakeStore{available: false}, &fakeBuffer{available: false})
	if _, err := svc.Query(context.Background(), query.Params{}); !errors.Is(err, query.ErrNoSource) {
		t.Errorf("want ErrNoSource, got %v", err)
	}
}
