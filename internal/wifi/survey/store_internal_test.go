package survey

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSurveyFilePathContainment verifies the path-traversal guard: a clean
// survey ID resolves to a file directly inside the storage directory, while any
// ID that would escape that directory is rejected.
func TestSurveyFilePathContainment(t *testing.T) {
	base := t.TempDir()
	m := &Manager{storagePath: base}

	t.Run("valid id stays inside the storage dir", func(t *testing.T) {
		const id = "11111111-2222-3333-4444-555555555555"
		got, err := m.surveyFilePath(id)
		if err != nil {
			t.Fatalf("surveyFilePath(%q) returned error: %v", id, err)
		}
		want := filepath.Join(base, id+".json")
		if got != want {
			t.Fatalf("surveyFilePath(%q) = %q, want %q", id, got, want)
		}
		if !strings.HasPrefix(got, filepath.Clean(base)+string(filepath.Separator)) {
			t.Fatalf("resolved path %q escaped storage dir %q", got, base)
		}
	})

	traversals := []struct {
		name string
		id   string
	}{
		{"parent escape", "../evil"},
		{"deep parent escape", "../../etc/passwd"},
		{"nested separator", "sub/dir"},
		{"dotdot only", ".."},
		{"absolute path", "/etc/shadow"},
	}
	for _, tc := range traversals {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := m.surveyFilePath(tc.id); err == nil {
				t.Fatalf("surveyFilePath(%q) = nil error, want rejection", tc.id)
			}
		})
	}
}
