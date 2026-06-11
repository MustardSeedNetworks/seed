package ssosync_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	ssosync "github.com/MustardSeedNetworks/seed/internal/identity/oauth"
)

// fakeRepo is a test double implementing ssosync.Repository.
type fakeRepo struct {
	available bool
	user      *database.User
	err       error
}

func (f *fakeRepo) Available() bool { return f.available }

func (f *fakeRepo) SyncUser(_ context.Context, _ database.SSOUserInput) (*database.User, error) {
	return f.user, f.err
}

func TestService_UnavailableReturnsErrUnavailable(t *testing.T) {
	t.Parallel()
	svc := ssosync.NewService(&fakeRepo{available: false})
	_, err := svc.SyncUser(context.Background(), database.SSOUserInput{
		Provider:   database.AuthProviderGoogle,
		ExternalID: "sub-123",
		Email:      "alice@example.com",
	})
	if !errors.Is(err, ssosync.ErrUnavailable) {
		t.Errorf("SyncUser unavailable: got %v, want ErrUnavailable", err)
	}
}

func TestService_HappySyncUser(t *testing.T) {
	t.Parallel()
	want := &database.User{Username: "google:sub-123", Role: database.RoleAdmin}
	svc := ssosync.NewService(&fakeRepo{available: true, user: want})
	got, err := svc.SyncUser(context.Background(), database.SSOUserInput{
		Provider:   database.AuthProviderGoogle,
		ExternalID: "sub-123",
		Email:      "alice@example.com",
	})
	if err != nil {
		t.Fatalf("SyncUser: unexpected error %v", err)
	}
	if got.Username != want.Username || got.Role != want.Role {
		t.Errorf("SyncUser: got %+v, want %+v", got, want)
	}
}

func TestService_ErrorPassesThrough(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("db exploded")
	svc := ssosync.NewService(&fakeRepo{available: true, err: sentinel})
	_, err := svc.SyncUser(context.Background(), database.SSOUserInput{})
	if !errors.Is(err, sentinel) {
		t.Errorf("SyncUser: got %v, want %v", err, sentinel)
	}
}
