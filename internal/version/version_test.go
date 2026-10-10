package version_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/version"
)

func TestInfo(t *testing.T) {
	info := version.Info()

	// Without ldflags every field still answers: a fallback, never empty.
	for name, value := range map[string]string{
		"Version":      info.Version,
		"Commit":       info.Commit,
		"BuildTime":    info.BuildTime,
		"ReleaseTrain": info.ReleaseTrain,
		"UIBuildHash":  info.UIBuildHash,
	} {
		if value == "" {
			t.Errorf("Info().%s is empty", name)
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
		{
			name:     "GetReleaseTrain returns value",
			getter:   version.GetReleaseTrain,
			notEmpty: true,
		},
		{
			name:     "GetUIBuildHash returns value",
			getter:   version.GetUIBuildHash,
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
