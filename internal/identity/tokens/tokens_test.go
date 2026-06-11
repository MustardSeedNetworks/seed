package tokens_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
)

// fakeStore is a test double implementing tokens.Store.
type fakeStore struct {
	available bool
	records   []database.APITokenRecord
	insertErr error
	revokeErr error
}

func (f *fakeStore) Available() bool { return f.available }

func (f *fakeStore) Insert(_ context.Context, t database.APITokenRecord) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.records = append(f.records, t)
	return nil
}

func (f *fakeStore) ListByOwner(_ context.Context, owner string) ([]database.APITokenRecord, error) {
	var out []database.APITokenRecord
	for _, r := range f.records {
		if r.OwnerUsername == owner {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeStore) Revoke(_ context.Context, _, _ string) error { return f.revokeErr }

// fakeGate is a test double implementing tokens.LicenseGate.
type fakeGate struct{ allowed bool }

func (g fakeGate) AllowsMinting() bool { return g.allowed }

func TestService_UnavailableReturnsErrUnavailable(t *testing.T) {
	t.Parallel()
	svc := tokens.NewService(&fakeStore{available: false}, fakeGate{allowed: true})
	ctx := context.Background()

	rec := database.APITokenRecord{ID: "x", OwnerUsername: "alice", Name: "test", CreatedAt: time.Now()}
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
	rec := database.APITokenRecord{ID: "x", OwnerUsername: "alice", Name: "test", CreatedAt: time.Now()}
	if err := svc.Mint(context.Background(), rec); !errors.Is(err, tokens.ErrMintingNotAllowed) {
		t.Errorf("Mint not allowed: got %v, want ErrMintingNotAllowed", err)
	}
}

func TestService_HappyMintAndList(t *testing.T) {
	t.Parallel()
	store := &fakeStore{available: true}
	svc := tokens.NewService(store, fakeGate{allowed: true})
	ctx := context.Background()

	rec := database.APITokenRecord{
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
