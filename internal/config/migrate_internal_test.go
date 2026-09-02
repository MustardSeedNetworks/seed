package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #377 renamed the persisted key additional_subnets to target_networks.
// Config.UnmarshalJSON rejects unknown fields by design, and seed.service is
// Restart=on-failure, so every existing install upgrading past that rename
// enters a crash-loop:
//
//	Fatal: Failed to load configuration: failed to load config:
//	parse config JSON: json: unknown field "additional_subnets"
//
// Observed on the lab probe (CT313) by installing the new binary over 0.212.11.
// Pre-v1 forbids keeping the old spelling working; it requires migrating every
// caller, and a config file on disk is a caller.
func TestLoadMigratesRenamedKeys(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed.json")
	previous := `{
  "version": 1,
  "networkDiscovery": {
    "additional_subnets": [{"cidr": "192.0.2.0/24", "name": "lab", "enabled": true}]
  }
}`
	if err := os.WriteFile(path, []byte(previous), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s) = %v, want a migrated config", path, err)
	}

	if len(cfg.NetworkDiscovery.TargetNetworks) != 1 {
		t.Fatalf("TargetNetworks = %#v, want the one entry from additional_subnets",
			cfg.NetworkDiscovery.TargetNetworks)
	}
	if got := cfg.NetworkDiscovery.TargetNetworks[0].CIDR; got != "192.0.2.0/24" {
		t.Errorf("CIDR = %q, want %q", got, "192.0.2.0/24")
	}
	if cfg.Version != ConfigVersion {
		t.Errorf("Version = %d, want it stamped to %d", cfg.Version, ConfigVersion)
	}
}

// Load does not rewrite the file it was asked to read. The migration is in
// memory; the operator's config is left as they wrote it until something
// actually changes a setting, and the Save that follows emits the new spelling.
func TestLoadLeavesTheFileAloneAndSaveEmitsTheNewKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed.json")
	previous := `{"version": 1, "networkDiscovery": {"additional_subnets": []}}`
	if err := os.WriteFile(path, []byte(previous), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	untouched, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("re-reading the config: %v", readErr)
	}
	if string(untouched) != previous {
		t.Errorf("Load rewrote the file:\n got %s\nwant %s", untouched, previous)
	}

	if saveErr := cfg.Save(path); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	saved, savedErr := os.ReadFile(path)
	if savedErr != nil {
		t.Fatalf("reading the saved config: %v", savedErr)
	}
	if strings.Contains(string(saved), "additional_subnets") {
		t.Error("Save wrote the old key")
	}
	if !strings.Contains(string(saved), "target_networks") {
		t.Error("Save did not write the new key")
	}

	var stamped struct {
		Version int `json:"version"`
	}
	if parseErr := json.Unmarshal(saved, &stamped); parseErr != nil {
		t.Fatalf("re-parsing the saved config: %v", parseErr)
	}
	if stamped.Version != ConfigVersion {
		t.Errorf("saved version = %d, want %d", stamped.Version, ConfigVersion)
	}
}

// Loading twice must give the same answer — the migration cannot depend on
// having rewritten anything.
func TestLoadIsIdempotent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed.json")
	previous := `{"version": 1, "networkDiscovery": {"additional_subnets": [{"cidr": "192.0.2.0/24"}]}}`
	if err := os.WriteFile(path, []byte(previous), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	first, err := Load(path)
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	second, secondErr := Load(path)
	if secondErr != nil {
		t.Fatalf("second Load: %v", secondErr)
	}

	if len(first.NetworkDiscovery.TargetNetworks) != len(second.NetworkDiscovery.TargetNetworks) {
		t.Errorf("two loads disagreed: %d then %d target networks",
			len(first.NetworkDiscovery.TargetNetworks),
			len(second.NetworkDiscovery.TargetNetworks))
	}
}

// The migration must not become a general amnesty: a key that is simply wrong
// is still rejected, which is the whole point of the strict decoder.
func TestLoadStillRejectsAnUnknownKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed.json")
	if err := os.WriteFile(path, []byte(`{"version": 1, "nonsense_key": 1}`), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load accepted an unknown key; the strict decoder is no longer strict")
	}
}

// A config already at the current version needs no migration at all.
func TestLoadLeavesACurrentConfigAlone(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed.json")
	current := `{"version": 2, "networkDiscovery": {"target_networks": []}}`
	if err := os.WriteFile(path, []byte(current), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	before, fixtureErr := os.ReadFile(path)
	if fixtureErr != nil {
		t.Fatalf("reading the fixture: %v", fixtureErr)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("re-reading the config: %v", readErr)
	}
	if string(before) != string(after) {
		t.Error("a current config was rewritten; migration should be a no-op here")
	}
}
