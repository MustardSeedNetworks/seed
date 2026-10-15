// SPDX-License-Identifier: BUSL-1.1

package api_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// TestNewServerRejectsPartialDeps pins seed#2751: a Deps without a required
// field is refused with ErrMissingDependency naming that field, not a panic.
func TestNewServerRejectsPartialDeps(t *testing.T) {
	cfg := testutil.NewConfigBuilder().Build()
	configPath := filepath.Join(t.TempDir(), "seed.json")

	tests := []struct {
		name    string
		deps    api.Deps
		missing string
	}{
		{name: "empty", deps: api.Deps{}, missing: "Config"},
		{name: "no config", deps: api.Deps{ConfigPath: configPath}, missing: "Config"},
		{name: "no config path", deps: api.Deps{Config: cfg}, missing: "ConfigPath"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := api.NewServer(tt.deps)
			if !errors.Is(err, api.ErrMissingDependency) {
				t.Fatalf("NewServer error = %v, want ErrMissingDependency", err)
			}
			if !strings.HasSuffix(err.Error(), ": "+tt.missing) {
				t.Errorf("error %q does not name %s", err, tt.missing)
			}
			if s != nil {
				t.Error("NewServer returned a server alongside the error")
			}
		})
	}

	s, err := api.NewServer(api.Deps{Config: cfg, ConfigPath: configPath})
	if err != nil {
		t.Fatalf("NewServer with the required fields: %v", err)
	}
	s.Close()
}
