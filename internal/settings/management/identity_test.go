package management_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
)

func TestIdentityUpdate(t *testing.T) {
	saved := config.IdentityConfig{Name: "seed-idf-3b", Location: "Main Office, IDF 3B"}
	cases := []struct {
		name    string
		update  any
		want    config.IdentityConfig
		wantErr string
	}{
		{
			name:   "sets both",
			update: map[string]any{"name": "seed-rack-12", "location": "Main Datacenter, Rack 12"},
			want:   config.IdentityConfig{Name: "seed-rack-12", Location: "Main Datacenter, Rack 12"},
		},
		{
			name:   "an omitted field keeps its value",
			update: map[string]any{"location": "Lab"},
			want:   config.IdentityConfig{Name: saved.Name, Location: "Lab"},
		},
		{
			name:   "empty clears",
			update: map[string]any{"name": "", "location": ""},
			want:   config.IdentityConfig{},
		},
		{
			name:   "surrounding whitespace is dropped",
			update: map[string]any{"name": "  seed-a  ", "location": "   "},
			want:   config.IdentityConfig{Name: "seed-a"},
		},
		{
			name:    "not an object",
			update:  "seed-a",
			want:    saved,
			wantErr: "identity must be an object",
		},
		{
			name:    "wrong type",
			update:  map[string]any{"name": 12.0},
			want:    saved,
			wantErr: "identity.name must be a string",
		},
		{
			// A refused field must not let its valid sibling through: the
			// write is all or nothing.
			name: "one bad field refuses the whole identity",
			update: map[string]any{
				"name":     "seed-ok",
				"location": strings.Repeat("x", config.IdentityLocationMaxRunes+1),
			},
			want:    saved,
			wantErr: "identity.location is 129 characters; the limit is 128",
		},
		{
			name:    "control character",
			update:  map[string]any{"name": "rack\n12"},
			want:    saved,
			wantErr: "identity.name must be one line of text",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Identity = saved
			svc := management.NewService(&fakeStore{cfg: cfg}, nil, nil)

			err := svc.Update(map[string]any{"identity": tc.update}, "")
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("Update: %v", err)
			case tc.wantErr != "" && !errors.Is(err, management.ErrValidation):
				t.Fatalf("Update = %v, want ErrValidation", err)
			case tc.wantErr != "" && err.Error() != tc.wantErr:
				t.Fatalf("Update reason = %q, want %q", err.Error(), tc.wantErr)
			}
			if cfg.Identity != tc.want {
				t.Fatalf("identity = %+v, want %+v", cfg.Identity, tc.want)
			}
		})
	}
}

func TestIdentityRead(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Identity = config.IdentityConfig{Name: "seed-idf-3b", Location: "IDF 3B"}
	svc := management.NewService(&fakeStore{cfg: cfg}, nil, nil)

	settings, _ := svc.Get()
	identity, ok := settings["identity"].(map[string]any)
	if !ok {
		t.Fatalf("settings has no identity object: %+v", settings["identity"])
	}
	if identity["name"] != "seed-idf-3b" || identity["location"] != "IDF 3B" {
		t.Fatalf("identity read model = %+v", identity)
	}
}

// TestIdentityUpdateMovesETag is what lets a second tab notice the rename: the
// token a client chains its next conditional write on must change with it.
func TestIdentityUpdateMovesETag(t *testing.T) {
	svc := management.NewService(&fakeStore{cfg: config.DefaultConfig()}, nil, nil)
	before := svc.ETag()
	if err := svc.Update(map[string]any{"identity": map[string]any{"name": "seed-a"}}, ""); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if svc.ETag() == before {
		t.Fatal("ETag did not move after the identity changed")
	}
	stale := strings.Trim(before, `"`)
	err := svc.Update(map[string]any{"identity": map[string]any{"name": "seed-b"}}, stale)
	if !errors.Is(err, management.ErrConflict) {
		t.Fatalf("write on the pre-rename ETag = %v, want ErrConflict", err)
	}
}
