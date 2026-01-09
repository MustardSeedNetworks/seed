package version_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/version"
)

func TestInfo(t *testing.T) {
	tests := []struct {
		name     string
		wantKeys []string
	}{
		{
			name:     "returns all version fields",
			wantKeys: []string{"version", "commit", "buildTime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := version.Info()

			// Verify all expected keys are present
			for _, key := range tt.wantKeys {
				if _, exists := info[key]; !exists {
					t.Errorf("Info() missing key %q", key)
				}
			}

			// Verify no extra keys
			if len(info) != len(tt.wantKeys) {
				t.Errorf("Info() returned %d keys, want %d", len(info), len(tt.wantKeys))
			}
		})
	}
}

func TestInfoDefaultValues(t *testing.T) {
	info := version.Info()

	// Test default values (when not set via ldflags)
	if info["version"] == "" {
		t.Error("Info() version should not be empty")
	}

	if info["commit"] == "" {
		t.Error("Info() commit should not be empty")
	}

	if info["buildTime"] == "" {
		t.Error("Info() buildTime should not be empty")
	}
}

func TestInfoMapStructure(t *testing.T) {
	info := version.Info()

	// Verify it returns a non-nil map
	if info == nil {
		t.Fatal("Info() returned nil map")
	}

	// Verify all values are non-empty strings
	for key, value := range info {
		if value == "" {
			t.Errorf("Info()[%q] should not be empty", key)
		}
	}
}

func TestGetterFunctions(t *testing.T) {
	tests := []struct {
		name     string
		getter   func() string
		notEmpty bool
	}{
		{
			name:     "GetVersion returns value",
			getter:   version.GetVersion,
			notEmpty: true,
		},
		{
			name:     "GetCommit returns value",
			getter:   version.GetCommit,
			notEmpty: true,
		},
		{
			name:     "GetBuildTime returns value",
			getter:   version.GetBuildTime,
			notEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := tt.getter()

			if tt.notEmpty && value == "" {
				t.Error("getter should not return empty string")
			}
		})
	}
}
