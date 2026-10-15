package devicegroups_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
)

// memStore keeps the last saved group so Get echoes what Save stored.
type memStore struct{ saved devicegroups.Group }

func (*memStore) List(context.Context, string) ([]devicegroups.Group, error) { return nil, nil }

func (m *memStore) Get(context.Context, string, string) (devicegroups.Group, error) {
	return m.saved, nil
}

func (m *memStore) Save(_ context.Context, g *devicegroups.Group) error {
	if g.ID == "" {
		g.ID = "grp-1"
	}
	m.saved = *g
	return nil
}

func (*memStore) Delete(context.Context, string, string) error { return nil }

func TestCreateNormalizes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		in          devicegroups.Group
		wantName    string
		wantMembers []string
		wantErr     string
	}{
		{
			name:        "trims and sorts",
			in:          devicegroups.Group{Name: " core ", Members: []string{"tgt-b", " tgt-a", "tgt-b"}},
			wantName:    "core",
			wantMembers: []string{"tgt-a", "tgt-b"},
		},
		{name: "no members", in: devicegroups.Group{Name: "empty"}, wantName: "empty", wantMembers: []string{}},
		{name: "blank name", in: devicegroups.Group{Name: " "}, wantErr: "name is required"},
		{name: "long name", in: devicegroups.Group{Name: strings.Repeat("é", 65)}, wantErr: "at most 64"},
		{
			name: "long description", in: devicegroups.Group{Name: "x", Description: strings.Repeat("d", 513)},
			wantErr: "at most 512",
		},
		{
			name:    "empty member",
			in:      devicegroups.Group{Name: "x", Members: []string{" "}},
			wantErr: "member id is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := devicegroups.NewService(&memStore{}).Create(context.Background(), tt.in)
			if tt.wantErr != "" {
				var ve *devicegroups.ValidationError
				if !errors.As(err, &ve) || !strings.Contains(ve.Msg, tt.wantErr) {
					t.Fatalf("err = %v, want a validation error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if got.Name != tt.wantName || strings.Join(got.Members, ",") != strings.Join(tt.wantMembers, ",") {
				t.Errorf("got %q %v, want %q %v", got.Name, got.Members, tt.wantName, tt.wantMembers)
			}
		})
	}
}

func TestCreateIgnoresACallerSuppliedID(t *testing.T) {
	t.Parallel()
	got, err := devicegroups.NewService(&memStore{}).Create(context.Background(),
		devicegroups.Group{ID: "grp-theirs", Name: "x"})
	if err != nil || got.ID != "grp-1" {
		t.Fatalf("Create = %+v, %v; want a fresh id", got, err)
	}
}
