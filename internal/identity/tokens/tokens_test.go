package tokens_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
)

// fakeStore is a test double implementing tokens.Store.
type fakeStore struct {
	available bool
	records   []tokens.Record
	insertErr error
	revokeErr error
	touchErr  error
	touched   []string
}

func (f *fakeStore) Available() bool { return f.available }

func (f *fakeStore) Insert(_ context.Context, t tokens.Record) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.records = append(f.records, t)
	return nil
}

func (f *fakeStore) ListByOwner(_ context.Context, owner string) ([]tokens.Record, error) {
	var out []tokens.Record
	for _, r := range f.records {
		if r.OwnerUsername == owner {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeStore) Revoke(_ context.Context, _, _ string) error { return f.revokeErr }

func (f *fakeStore) FindActiveByHash(_ context.Context, hash string) (tokens.Record, error) {
	for _, r := range f.records {
		if r.TokenHash == hash && r.RevokedAt.IsZero() {
			return r, nil
		}
	}
	return tokens.Record{}, sql.ErrNoRows
}

func (f *fakeStore) TouchLastUsed(_ context.Context, id string) error {
	f.touched = append(f.touched, id)
	return f.touchErr
}

// fakeGate is a test double implementing tokens.LicenseGate.
type fakeGate struct{ allowed bool }

func (g fakeGate) AllowsMinting() bool { return g.allowed }

func TestService_UnavailableReturnsErrUnavailable(t *testing.T) {
	t.Parallel()
	svc := tokens.NewService(&fakeStore{available: false}, fakeGate{allowed: true})
	ctx := context.Background()

	rec := tokens.Record{ID: "x", OwnerUsername: "alice", Name: "test", CreatedAt: time.Now()}
	if err := svc.Mint(ctx, rec); !errors.Is(err, tokens.ErrUnavailable) {
		t.Errorf("Mint unavailable: got %v, want ErrUnavailable", err)
	}
	if err := svc.Revoke(ctx, "x", "alice"); !errors.Is(err, tokens.ErrUnavailable) {
		t.Errorf("Revoke unavailable: got %v, want ErrUnavailable", err)
	}
}

func TestService_ListReturnsEmptyWhenUnavailable(t *testing.T) {
	t.Parallel()
	svc := tokens.NewService(&fakeStore{available: false}, fakeGate{allowed: true})
	got, err := svc.List(context.Background(), "alice")
	if err != nil {
		t.Fatalf("List unavailable: unexpected error %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List unavailable: expected empty slice, got %v", got)
	}
}

func TestService_MintingNotAllowed(t *testing.T) {
	t.Parallel()
	svc := tokens.NewService(&fakeStore{available: true}, fakeGate{allowed: false})
	rec := tokens.Record{ID: "x", OwnerUsername: "alice", Name: "test", CreatedAt: time.Now()}
	if err := svc.Mint(context.Background(), rec); !errors.Is(err, tokens.ErrMintingNotAllowed) {
		t.Errorf("Mint not allowed: got %v, want ErrMintingNotAllowed", err)
	}
}

func TestService_HappyMintAndList(t *testing.T) {
	t.Parallel()
	store := &fakeStore{available: true}
	svc := tokens.NewService(store, fakeGate{allowed: true})
	ctx := context.Background()

	rec := tokens.Record{
		ID:            "abc123",
		OwnerUsername: "alice",
		Name:          "ci-token",
		CreatedAt:     time.Now().UTC(),
	}
	if err := svc.Mint(ctx, rec); err != nil {
		t.Fatalf("Mint: unexpected error %v", err)
	}

	rows, err := svc.List(ctx, "alice")
	if err != nil {
		t.Fatalf("List: unexpected error %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "abc123" {
		t.Errorf("List: unexpected result %v", rows)
	}
}

func TestService_RevokePassesThroughSQLErrNoRows(t *testing.T) {
	t.Parallel()
	store := &fakeStore{available: true, revokeErr: sql.ErrNoRows}
	svc := tokens.NewService(store, fakeGate{allowed: true})
	err := svc.Revoke(context.Background(), "nosuchid", "alice")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Revoke: got %v, want sql.ErrNoRows", err)
	}
}

func TestService_HappyRevoke(t *testing.T) {
	t.Parallel()
	store := &fakeStore{available: true}
	svc := tokens.NewService(store, fakeGate{allowed: true})
	if err := svc.Revoke(context.Background(), "id1", "alice"); err != nil {
		t.Errorf("Revoke: unexpected error %v", err)
	}
}

func TestService_Resolve(t *testing.T) {
	t.Parallel()
	active := tokens.Record{ID: "live", OwnerUsername: "alice", TokenHash: "h-live"}
	revoked := tokens.Record{ID: "dead", OwnerUsername: "alice", TokenHash: "h-dead", RevokedAt: time.Now()}

	tests := []struct {
		name        string
		store       *fakeStore
		hash        string
		wantOK      bool
		wantTouched []string
	}{
		{
			"active token is returned and stamped",
			&fakeStore{available: true, records: []tokens.Record{active}},
			"h-live",
			true,
			[]string{"live"},
		},
		{
			"failed stamp still admits the token",
			&fakeStore{available: true, records: []tokens.Record{active}, touchErr: errors.New("disk full")},
			"h-live",
			true,
			[]string{"live"},
		},
		{
			"revoked token is refused",
			&fakeStore{available: true, records: []tokens.Record{revoked}},
			"h-dead",
			false,
			nil,
		},
		{
			"unknown hash is refused",
			&fakeStore{available: true, records: []tokens.Record{active}},
			"h-none",
			false,
			nil,
		},
		{
			"no store refuses without a lookup",
			&fakeStore{available: false, records: []tokens.Record{active}},
			"h-live",
			false,
			nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := tokens.NewService(tc.store, fakeGate{allowed: true})
			rec, ok := svc.Resolve(context.Background(), tc.hash)
			if ok != tc.wantOK {
				t.Fatalf("Resolve ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && rec.ID != "live" {
				t.Errorf("Resolve record = %q, want %q", rec.ID, "live")
			}
			if !ok && rec != (tokens.Record{}) {
				t.Errorf("Resolve returned %+v with ok=false, want the zero record", rec)
			}
			if len(tc.store.touched) != len(tc.wantTouched) {
				t.Errorf("touched = %v, want %v", tc.store.touched, tc.wantTouched)
			}
		})
	}
}
