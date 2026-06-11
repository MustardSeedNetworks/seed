package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
)

// fakeRepo is a test double implementing users.Repository.
type fakeRepo struct {
	available bool
	user      *database.User
	listUsers []*database.User
	err       error
}

func (f *fakeRepo) Available() bool { return f.available }

func (f *fakeRepo) List(_ context.Context) ([]*database.User, error) {
	return f.listUsers, f.err
}

func (f *fakeRepo) Create(_ context.Context, username, _, role string) (*database.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &database.User{Username: username, Role: role}, nil
}

func (f *fakeRepo) Get(_ context.Context, _ string) (*database.User, error) {
	return f.user, f.err
}

func (f *fakeRepo) UpdatePassword(_ context.Context, _, _ string) error { return f.err }
func (f *fakeRepo) UpdateRole(_ context.Context, _, _ string) error     { return f.err }
func (f *fakeRepo) Deactivate(_ context.Context, _ string) error        { return f.err }
func (f *fakeRepo) Delete(_ context.Context, _ string) error            { return f.err }

func TestService_UnavailableReturnsErrUnavailable(t *testing.T) {
	t.Parallel()
	svc := users.NewService(&fakeRepo{available: false})
	ctx := context.Background()

	if _, err := svc.List(ctx); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("List: got %v, want ErrUnavailable", err)
	}
	if _, err := svc.Create(ctx, "x", "h", database.RoleViewer); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Create: got %v, want ErrUnavailable", err)
	}
	if _, err := svc.Get(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Get: got %v, want ErrUnavailable", err)
	}
	if err := svc.UpdatePassword(ctx, "x", "h"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("UpdatePassword: got %v, want ErrUnavailable", err)
	}
	if err := svc.UpdateRole(ctx, "x", database.RoleViewer); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("UpdateRole: got %v, want ErrUnavailable", err)
	}
	if err := svc.Deactivate(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Deactivate: got %v, want ErrUnavailable", err)
	}
	if err := svc.Delete(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Delete: got %v, want ErrUnavailable", err)
	}
}

func TestService_HappyPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	admin := &database.User{Username: "admin", Role: database.RoleAdmin, IsActive: true}

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		svc := users.NewService(&fakeRepo{available: true, listUsers: []*database.User{admin}})
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].Username != "admin" {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		svc := users.NewService(&fakeRepo{available: true})
		u, err := svc.Create(ctx, "alice", "hash", database.RoleOperator)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "alice" || u.Role != database.RoleOperator {
			t.Errorf("unexpected user: %+v", u)
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		svc := users.NewService(&fakeRepo{available: true, user: admin})
		u, err := svc.Get(ctx, "admin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "admin" {
			t.Errorf("unexpected user: %+v", u)
		}
	})

	// The mutation-only methods share one shape (delegate, return nil when the
	// repo is happy) — table-drive them so each is a focused, low-complexity case.
	mutators := []struct {
		name string
		call func(*users.Service) error
	}{
		{"UpdatePassword", func(s *users.Service) error { return s.UpdatePassword(ctx, "admin", "newhash") }},
		{"UpdateRole", func(s *users.Service) error { return s.UpdateRole(ctx, "admin", database.RoleViewer) }},
		{"Deactivate", func(s *users.Service) error { return s.Deactivate(ctx, "admin") }},
		{"Delete", func(s *users.Service) error { return s.Delete(ctx, "admin") }},
	}
	for _, m := range mutators {
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()
			if err := m.call(users.NewService(&fakeRepo{available: true})); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestService_DomainSentinelsPassThrough(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"ErrUserExists", database.ErrUserExists, database.ErrUserExists},
		{"ErrUserNotFound", database.ErrUserNotFound, database.ErrUserNotFound},
		{"ErrLastAdmin", database.ErrLastAdmin, database.ErrLastAdmin},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := users.NewService(&fakeRepo{available: true, err: tc.err})
			// Create passes the error through.
			if _, err := svc.Create(ctx, "x", "h", database.RoleViewer); !errors.Is(err, tc.wantErr) {
				t.Errorf("Create: got %v, want %v", err, tc.wantErr)
			}
			// UpdateRole passes the error through.
			if err := svc.UpdateRole(ctx, "x", database.RoleViewer); !errors.Is(err, tc.wantErr) {
				t.Errorf("UpdateRole: got %v, want %v", err, tc.wantErr)
			}
			// Delete passes the error through.
			if err := svc.Delete(ctx, "x"); !errors.Is(err, tc.wantErr) {
				t.Errorf("Delete: got %v, want %v", err, tc.wantErr)
			}
		})
	}
}
