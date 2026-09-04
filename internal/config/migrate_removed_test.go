package config_test

// migrate_removed_test.go pins the upgrade path: a config written by a real
// released version must still start the daemon.
//
// Config.UnmarshalJSON disallows unknown fields on purpose, and seed.service is
// Restart=on-failure, so a key the current schema has dropped does not degrade —
// it crash-loops an install the operator cannot reach, because the daemon never
// comes up. #377 was one instance of this; every schema removal is another.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// v0200Fixture is generated from the v0.200.0 tag, not written by hand — a
// hand-written one would only prove that the keys someone remembered are
// handled. Regenerate with:
//
//	git worktree add /tmp/seed-v0200 --detach v0.200.0
//	# a main() that prints json.MarshalIndent(config.DefaultConfig())
//
// then blank auth.default_password_hash and every *_secret / *_api_key before
// committing it.
const v0200Fixture = "testdata/config-v0.200.0.json"

func TestLoadConfigFromV0200(t *testing.T) {
	cfg, err := config.Load(v0200Fixture)
	if err != nil {
		t.Fatalf("a config written by v0.200.0 must still load: %v", err)
	}

	// The fixture is version 1; Load brings it to the current schema.
	if cfg.Version != config.ConfigVersion {
		t.Errorf("Version = %d after load, want %d", cfg.Version, config.ConfigVersion)
	}

	// Settings the release did carry must survive the migration rather than
	// being reset to defaults along with the removed ones.
	var raw map[string]any
	data, readErr := os.ReadFile(v0200Fixture)
	if readErr != nil {
		t.Fatalf("read fixture: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
		t.Fatalf("parse fixture: %v", unmarshalErr)
	}
	server, _ := raw["server"].(map[string]any)
	wantPort, _ := server["port"].(float64)
	if cfg.Server.Port != int(wantPort) {
		t.Errorf("Server.Port = %d, want %d from the fixture", cfg.Server.Port, int(wantPort))
	}
}

// TestLoadConfigRejectsUnknownKey is the other half: the removal table must not
// turn the strict decode into a shrug. A key nobody removed is still fatal.
func TestLoadConfigRejectsUnknownKey(t *testing.T) {
	data, err := os.ReadFile(v0200Fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var document map[string]any
	if unmarshalErr := json.Unmarshal(data, &document); unmarshalErr != nil {
		t.Fatalf("parse fixture: %v", unmarshalErr)
	}

	// A plausible misspelling of a real key, not a removed one.
	document["netwrokDiscovery"] = map[string]any{"enabled": true}

	corrupted, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatalf("re-encode fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "typo.json")
	if writeErr := os.WriteFile(path, corrupted, 0o600); writeErr != nil {
		t.Fatalf("write fixture: %v", writeErr)
	}

	_, err = config.Load(path)
	if err == nil {
		t.Fatal("a misspelled key loaded without error; the strict decode is no longer strict")
	}
	if !strings.Contains(err.Error(), "netwrokDiscovery") {
		t.Errorf("error does not name the offending key: %v", err)
	}
}

// TestRemovedKeysAreStripped names the two keys that block a v0.200.0 upgrade,
// so a future change that drops the table fails here rather than in the field.
func TestRemovedKeysAreStripped(t *testing.T) {
	for _, tc := range []struct {
		name     string
		document map[string]any
	}{
		{
			name:     "top-level pipeline",
			document: map[string]any{"pipeline": map[string]any{"phases": map[string]any{}}},
		},
		{
			name:     "server.https",
			document: map[string]any{"server": map[string]any{"https": true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.MarshalIndent(tc.document, "", "  ")
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			path := filepath.Join(t.TempDir(), "legacy.json")
			if writeErr := os.WriteFile(path, encoded, 0o600); writeErr != nil {
				t.Fatalf("write: %v", writeErr)
			}
			if _, loadErr := config.Load(path); loadErr != nil {
				t.Errorf("Load with a removed key present: %v", loadErr)
			}
		})
	}
}
