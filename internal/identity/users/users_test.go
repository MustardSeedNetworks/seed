package users_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
)

// fakeRepo is a test double implementing users.Repository.
type fakeRepo struct {
	available bool
	user      *users.User
	listUsers []*users.User
	err       error
	// stored is the last layout SetDashboard accepted; nil until one is.
	stored []string
}

func (f *fakeRepo) Available() bool { return f.available }

func (f *fakeRepo) List(_ context.Context) ([]*users.User, error) {
	return f.listUsers, f.err
}

func (f *fakeRepo) Create(_ context.Context, username, _, role string) (*users.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &users.User{Username: username, Role: role}, nil
}

func (f *fakeRepo) Get(_ context.Context, _ string) (*users.User, error) {
	return f.user, f.err
}

func (f *fakeRepo) UpdatePassword(_ context.Context, _, _ string) error { return f.err }
func (f *fakeRepo) UpdateRole(_ context.Context, _, _ string) error     { return f.err }
func (f *fakeRepo) Deactivate(_ context.Context, _ string) error        { return f.err }
func (f *fakeRepo) Delete(_ context.Context, _ string) error            { return f.err }

func (f *fakeRepo) GetDashboard(_ context.Context, _ string) ([]string, bool, error) {
	return f.stored, f.stored != nil, f.err
}

func (f *fakeRepo) SetDashboard(_ context.Context, _ string, widgets []string) error {
	if f.err != nil {
		return f.err
	}
	f.stored = widgets
	return nil
}

func TestService_UnavailableReturnsErrUnavailable(t *testing.T) {
	t.Parallel()
	svc := users.NewService(&fakeRepo{available: false})
	ctx := context.Background()

	if _, err := svc.List(ctx); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("List: got %v, want ErrUnavailable", err)
	}
	if _, err := svc.Create(ctx, "x", "h", roles.Viewer); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Create: got %v, want ErrUnavailable", err)
	}
	if _, err := svc.Get(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Get: got %v, want ErrUnavailable", err)
	}
	if err := svc.UpdatePassword(ctx, "x", "h"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("UpdatePassword: got %v, want ErrUnavailable", err)
	}
	if err := svc.UpdateRole(ctx, "x", roles.Viewer); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("UpdateRole: got %v, want ErrUnavailable", err)
	}
	if err := svc.Deactivate(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Deactivate: got %v, want ErrUnavailable", err)
	}
	if err := svc.Delete(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("Delete: got %v, want ErrUnavailable", err)
	}
	if _, _, err := svc.GetDashboard(ctx, "x"); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("GetDashboard: got %v, want ErrUnavailable", err)
	}
	if err := svc.SetDashboard(ctx, "x", []string{"link"}); !errors.Is(err, users.ErrUnavailable) {
		t.Errorf("SetDashboard: got %v, want ErrUnavailable", err)
	}
}

func TestService_HappyPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	admin := &users.User{Username: "admin", Role: roles.Admin, IsActive: true}

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		svc := users.NewService(&fakeRepo{available: true, listUsers: []*users.User{admin}})
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
		u, err := svc.Create(ctx, "alice", "hash", roles.Operator)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "alice" || u.Role != roles.Operator {
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
		{"UpdateRole", func(s *users.Service) error { return s.UpdateRole(ctx, "admin", roles.Viewer) }},
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
		{"users.ErrUserExists", users.ErrUserExists, users.ErrUserExists},
		{"users.ErrUserNotFound", users.ErrUserNotFound, users.ErrUserNotFound},
		{"users.ErrLastAdmin", users.ErrLastAdmin, users.ErrLastAdmin},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := users.NewService(&fakeRepo{available: true, err: tc.err})
			// Create passes the error through.
			if _, err := svc.Create(ctx, "x", "h", roles.Viewer); !errors.Is(err, tc.wantErr) {
				t.Errorf("Create: got %v, want %v", err, tc.wantErr)
			}
			// UpdateRole passes the error through.
			if err := svc.UpdateRole(ctx, "x", roles.Viewer); !errors.Is(err, tc.wantErr) {
				t.Errorf("UpdateRole: got %v, want %v", err, tc.wantErr)
			}
			// Delete passes the error through.
			if err := svc.Delete(ctx, "x"); !errors.Is(err, tc.wantErr) {
				t.Errorf("Delete: got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateDashboard(t *testing.T) {
	t.Parallel()

	tooMany := make([]string, users.MaxDashboardWidgets+1)
	for i := range tooMany {
		tooMany[i] = "w" + strconv.Itoa(i)
	}

	tests := []struct {
		name    string
		widgets []string
		valid   bool
	}{
		{"empty dashboard", []string{}, true},
		{"ordered ids", []string{"gateway", "link", "publicIp"}, true},
		{"at the bound", tooMany[:users.MaxDashboardWidgets], true},
		{"over the bound", tooMany, false},
		{"repeated id", []string{"link", "dns", "link"}, false},
		{"empty id", []string{""}, false},
		{"leading digit", []string{"1link"}, false},
		{"markup", []string{"<script>"}, false},
		{"path", []string{"../link"}, false},
		{"too long", []string{"a" + strings.Repeat("b", 32)}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := users.ValidateDashboard(tc.widgets)
			if tc.valid && err != nil {
				t.Errorf("ValidateDashboard(%q) = %v, want nil", tc.widgets, err)
			}
			if !tc.valid && !errors.Is(err, users.ErrInvalidDashboard) {
				t.Errorf("ValidateDashboard(%q) = %v, want ErrInvalidDashboard", tc.widgets, err)
			}
		})
	}
}

// A refused layout never reaches the store, and an accepted one reads back in
// the order it was saved.
func TestService_Dashboard(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := &fakeRepo{available: true}
	svc := users.NewService(repo)

	if _, saved, err := svc.GetDashboard(ctx, "viewer"); err != nil || saved {
		t.Fatalf("GetDashboard before save: saved=%v err=%v, want unsaved", saved, err)
	}
	if err := svc.SetDashboard(ctx, "viewer", []string{"dns", "dns"}); !errors.Is(err, users.ErrInvalidDashboard) {
		t.Fatalf("SetDashboard(repeat): got %v, want ErrInvalidDashboard", err)
	}
	if repo.stored != nil {
		t.Fatalf("refused layout reached the store: %q", repo.stored)
	}
	if err := svc.SetDashboard(ctx, "viewer", []string{"gateway", "link"}); err != nil {
		t.Fatalf("SetDashboard: %v", err)
	}
	got, saved, err := svc.GetDashboard(ctx, "viewer")
	if err != nil || !saved || !slices.Equal(got, []string{"gateway", "link"}) {
		t.Errorf("GetDashboard = %q saved=%v err=%v, want [gateway link] saved", got, saved, err)
	}
}
