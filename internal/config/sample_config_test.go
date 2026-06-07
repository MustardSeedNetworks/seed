package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestShippedSampleConfigLoads guards finding #1: the sample config that ships
// in configs/ must exist, parse through the JSON loader, and be structurally
// schema-valid. Before the JSON unification the sample was real YAML that the
// JSON loader could not parse, so copying it and starting the appliance failed.
//
// The sample is a pristine template: it intentionally omits runtime credentials
// (the first-run setup wizard fills default_password_hash / jwt_secret), so it
// is checked against the schema (structure + value constraints) rather than the
// full business Validate(), which requires those setup-time fields.
func TestShippedSampleConfigLoads(t *testing.T) {
	// internal/config/<this file> -> repo root is two levels up.
	samplePath := filepath.Join("..", "..", "configs", "seed.json")

	// Must exist on disk. config.Load silently falls back to defaults when the
	// file is absent, so read the bytes directly to catch a missing sample.
	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("shipped sample config not readable at %q: %v", samplePath, err)
	}

	// Must parse as JSON (the loader is encoding/json).
	var cfg config.Config
	if unmarshalErr := json.Unmarshal(data, &cfg); unmarshalErr != nil {
		t.Fatalf("shipped sample config is not valid JSON: %v", unmarshalErr)
	}

	// And it must round-trip through the real loader without error.
	loaded, loadErr := config.Load(samplePath)
	if loadErr != nil {
		t.Fatalf("config.Load(%q) failed: %v", samplePath, loadErr)
	}

	// Structurally schema-valid (shape + value constraints), credentials aside.
	if schemaErrs := config.ValidateWithSchema(loaded); len(schemaErrs) != 0 {
		t.Fatalf("shipped sample config failed schema validation: %v", schemaErrs)
	}
}
