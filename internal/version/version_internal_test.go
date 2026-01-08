package version

import (
	"runtime/debug"
	"testing"
)

func TestExtractVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name          string
		info          *debug.BuildInfo
		wantVersion   string
		wantCommit    string
		wantBuildTime string
	}{
		{
			name:          "nil build info returns defaults",
			info:          nil,
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "empty build info returns defaults",
			info: &debug.BuildInfo{
				Main: debug.Module{},
			},
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "devel version returns dev",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "(devel)",
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "empty version returns dev",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "",
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "valid version is returned",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.2.3",
				},
			},
			wantVersion:   "v1.2.3",
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "long commit is shortened",
			info: &debug.BuildInfo{
				Main: debug.Module{},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123456789def"},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    "abc1234",
			wantBuildTime: unknownValue,
		},
		{
			name: "short commit is kept as-is",
			info: &debug.BuildInfo{
				Main: debug.Module{},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc12"},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    "abc12",
			wantBuildTime: unknownValue,
		},
		{
			name: "exact 7 char commit is kept as-is",
			info: &debug.BuildInfo{
				Main: debug.Module{},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc1234"},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    "abc1234",
			wantBuildTime: unknownValue,
		},
		{
			name: "build time is extracted",
			info: &debug.BuildInfo{
				Main: debug.Module{},
				Settings: []debug.BuildSetting{
					{Key: "vcs.time", Value: "2024-01-15T10:30:00Z"},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: "2024-01-15T10:30:00Z",
		},
		{
			name: "modified true adds dirty suffix when version is set",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.0",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion:   "v1.0.0-dirty",
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "modified true does not add dirty suffix when version is dev",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "(devel)",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "modified false does not add dirty suffix",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.0",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.modified", Value: "false"},
				},
			},
			wantVersion:   "v1.0.0",
			wantCommit:    unknownValue,
			wantBuildTime: unknownValue,
		},
		{
			name: "all settings combined",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v2.5.1",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "deadbeef12345678"},
					{Key: "vcs.time", Value: "2024-06-20T15:45:30Z"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			wantVersion:   "v2.5.1",
			wantCommit:    "deadbee",
			wantBuildTime: "2024-06-20T15:45:30Z",
		},
		{
			name: "all settings with dirty flag",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v3.0.0-rc1",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "cafebabe87654321"},
					{Key: "vcs.time", Value: "2024-12-25T00:00:00Z"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion:   "v3.0.0-rc1-dirty",
			wantCommit:    "cafebab",
			wantBuildTime: "2024-12-25T00:00:00Z",
		},
		{
			name: "unknown VCS settings are ignored",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.0",
				},
				Settings: []debug.BuildSetting{
					{Key: "GOOS", Value: "linux"},
					{Key: "GOARCH", Value: "amd64"},
					{Key: "vcs", Value: "git"},
					{Key: "vcs.revision", Value: "1234567890abcdef"},
				},
			},
			wantVersion:   "v1.0.0",
			wantCommit:    "1234567",
			wantBuildTime: unknownValue,
		},
		{
			name: "empty VCS revision value",
			info: &debug.BuildInfo{
				Main: debug.Module{},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: ""},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    "",
			wantBuildTime: unknownValue,
		},
		{
			name: "empty VCS time value",
			info: &debug.BuildInfo{
				Main: debug.Module{},
				Settings: []debug.BuildSetting{
					{Key: "vcs.time", Value: ""},
				},
			},
			wantVersion:   defaultVersion,
			wantCommit:    unknownValue,
			wantBuildTime: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotCommit, gotBuildTime := extractVersionFromBuildInfo(tt.info)

			if gotVersion != tt.wantVersion {
				t.Errorf("extractVersionFromBuildInfo() version = %q, want %q", gotVersion, tt.wantVersion)
			}
			if gotCommit != tt.wantCommit {
				t.Errorf("extractVersionFromBuildInfo() commit = %q, want %q", gotCommit, tt.wantCommit)
			}
			if gotBuildTime != tt.wantBuildTime {
				t.Errorf("extractVersionFromBuildInfo() buildTime = %q, want %q", gotBuildTime, tt.wantBuildTime)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      any
		want     any
		typeName string
	}{
		{
			name:     "shortCommitLen value",
			got:      shortCommitLen,
			want:     7,
			typeName: "shortCommitLen",
		},
		{
			name:     "defaultVersion value",
			got:      defaultVersion,
			want:     "dev",
			typeName: "defaultVersion",
		},
		{
			name:     "unknownValue value",
			got:      unknownValue,
			want:     "unknown",
			typeName: "unknownValue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.typeName, tt.got, tt.want)
			}
		})
	}
}

func TestExtractVersionFromBuildInfoSettingsOrder(t *testing.T) {
	// Test that the order of settings does not affect the result.
	info1 := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.0.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc1234567890"},
			{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	info2 := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.0.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.modified", Value: "true"},
			{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
			{Key: "vcs.revision", Value: "abc1234567890"},
		},
	}

	v1, c1, b1 := extractVersionFromBuildInfo(info1)
	v2, c2, b2 := extractVersionFromBuildInfo(info2)

	if v1 != v2 || c1 != c2 || b1 != b2 {
		t.Errorf("settings order affects results: (%q,%q,%q) != (%q,%q,%q)", v1, c1, b1, v2, c2, b2)
	}
}

func TestExtractVersionFromBuildInfoWithEmptySettings(t *testing.T) {
	info := &debug.BuildInfo{
		Main:     debug.Module{Version: "v1.0.0"},
		Settings: []debug.BuildSetting{},
	}

	version, commit, buildTime := extractVersionFromBuildInfo(info)

	if version != "v1.0.0" {
		t.Errorf("version = %q, want %q", version, "v1.0.0")
	}
	if commit != unknownValue {
		t.Errorf("commit = %q, want %q", commit, unknownValue)
	}
	if buildTime != unknownValue {
		t.Errorf("buildTime = %q, want %q", buildTime, unknownValue)
	}
}
