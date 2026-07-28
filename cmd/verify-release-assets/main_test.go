package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testChecksum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestVerifyReleaseCoverage(t *testing.T) {
	manifest, catalog, checksums, files := completeRelease(t)

	subjects, err := verifyRelease(manifest, catalog, checksums, files)
	if err != nil {
		t.Fatalf("verifyRelease() error = %v", err)
	}
	if len(subjects) != len(manifest.PrimaryAssets) {
		t.Fatalf(
			"verifyRelease() subjects = %d, want %d",
			len(subjects),
			len(manifest.PrimaryAssets),
		)
	}
	for index := 1; index < len(subjects); index++ {
		if subjects[index-1].Name >= subjects[index].Name {
			t.Fatalf(
				"verifyRelease() subjects are not sorted: %q before %q",
				subjects[index-1].Name,
				subjects[index].Name,
			)
		}
	}
}

func TestReleaseManifestLocksNinePrimaryAssets(t *testing.T) {
	manifest, err := readManifest(os.DirFS("../.."), "release/asset-manifest.json")
	if err != nil {
		t.Fatalf("readManifest() error = %v", err)
	}
	if len(manifest.PrimaryAssets) != 9 {
		t.Fatalf("primary asset count = %d, want 9", len(manifest.PrimaryAssets))
	}
}

func TestRunWritesValidatedProvenanceSubjects(t *testing.T) {
	manifest, catalog, checksums, files := completeRelease(t)
	rootPath := t.TempDir()
	dist := filepath.Join(rootPath, "dist")
	if err := os.Mkdir(dist, 0o700); err != nil {
		t.Fatalf("create dist fixture: %v", err)
	}
	manifestPath := "asset-manifest.json"
	artifactsPath := "dist/artifacts.json"
	checksumsPath := "dist/checksums.txt"
	subjectsPath := "dist/provenance-subjects.txt"

	writeJSON(t, filepath.Join(rootPath, manifestPath), manifest)
	writeJSON(t, filepath.Join(rootPath, artifactsPath), catalog)
	var checksumText strings.Builder
	for name, checksum := range checksums {
		fmt.Fprintf(&checksumText, "%s  %s\n", checksum, name)
	}
	if err := os.WriteFile(filepath.Join(rootPath, checksumsPath), []byte(checksumText.String()), 0o600); err != nil {
		t.Fatalf("write checksums: %v", err)
	}
	for name := range files {
		if err := os.WriteFile(filepath.Join(dist, name), []byte("fixture"), 0o600); err != nil {
			t.Fatalf("write fixture %q: %v", name, err)
		}
	}

	if err := run(rootPath, manifestPath, artifactsPath, checksumsPath, "dist", subjectsPath); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	subjectData, err := os.ReadFile(filepath.Join(rootPath, subjectsPath))
	if err != nil {
		t.Fatalf("read subjects: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(subjectData)), "\n")
	if len(lines) != 9 {
		t.Fatalf("subject lines = %d, want 9", len(lines))
	}
	if !strings.HasSuffix(lines[0], "seed-1.2.3-1.aarch64.rpm") {
		t.Fatalf("subjects are not sorted: first line = %q", lines[0])
	}
}

func TestValidateManifestRejectsInvalidContracts(t *testing.T) {
	valid := primaryAsset{
		ID:           "archive",
		Type:         "Archive",
		GOOS:         "linux",
		GOARCH:       "amd64",
		Format:       "tar.gz",
		NameTemplate: "seed-{version}.tar.gz",
	}
	tests := []struct {
		name     string
		manifest releaseManifest
	}{
		{name: "empty", manifest: releaseManifest{}},
		{
			name: "empty identity",
			manifest: releaseManifest{
				PrimaryAssets: []primaryAsset{{NameTemplate: "seed-{version}.tar.gz"}},
			},
		},
		{
			name: "missing placeholder",
			manifest: releaseManifest{
				PrimaryAssets: []primaryAsset{
					{
						ID:           valid.ID,
						Type:         valid.Type,
						GOOS:         valid.GOOS,
						GOARCH:       valid.GOARCH,
						Format:       valid.Format,
						NameTemplate: "seed.tar.gz",
					},
				},
			},
		},
		{name: "duplicate", manifest: releaseManifest{PrimaryAssets: []primaryAsset{valid, valid}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateManifest(test.manifest); err == nil {
				t.Fatal("validateManifest() error = nil, want error")
			}
		})
	}
}

func TestReadChecksumsRejectsInvalidFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed", content: "not-a-checksum artifact.tar.gz\n"},
		{
			name:    "duplicate",
			content: testChecksum + "  artifact.tar.gz\n" + testChecksum + "  artifact.tar.gz\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "checksums.txt")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatalf("write checksum fixture: %v", err)
			}
			if _, err := readChecksums(os.DirFS(filepath.Dir(path)), filepath.Base(path)); err == nil {
				t.Fatal("readChecksums() error = nil, want error")
			}
		})
	}
}

func TestFileReadersRejectInvalidInputs(t *testing.T) {
	t.Run("trailing manifest JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "manifest.json")
		if err := os.WriteFile(path, []byte(`{"primaryAssets":[]} {}`), 0o600); err != nil {
			t.Fatalf("write manifest fixture: %v", err)
		}
		if _, err := readManifest(os.DirFS(filepath.Dir(path)), filepath.Base(path)); err == nil {
			t.Fatal("readManifest() error = nil, want error")
		}
	})

	t.Run("invalid artifact JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "artifacts.json")
		if err := os.WriteFile(path, []byte(`{"not":"an array"}`), 0o600); err != nil {
			t.Fatalf("write artifact fixture: %v", err)
		}
		if _, err := readArtifacts(os.DirFS(filepath.Dir(path)), filepath.Base(path)); err == nil {
			t.Fatal("readArtifacts() error = nil, want error")
		}
	})

	missingRoot := t.TempDir()
	if _, err := listFiles(os.DirFS(missingRoot), "missing"); err == nil {
		t.Fatal("listFiles() error = nil, want error")
	}
	root, err := os.OpenRoot(missingRoot)
	if err != nil {
		t.Fatalf("open test root: %v", err)
	}
	defer root.Close()
	writeErr := writeSubjects(root, "missing/subjects.txt", nil)
	if writeErr == nil {
		t.Fatal("writeSubjects() error = nil, want error")
	}
}

func TestVerifyReleaseCoverageRejectsIncompleteCoverage(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*releaseManifest, *[]artifact, map[string]string, map[string]struct{})
		wantError string
	}{
		{
			name: "missing primary asset",
			mutate: func(_ *releaseManifest, catalog *[]artifact, _ map[string]string, _ map[string]struct{}) {
				*catalog = (*catalog)[1:]
			},
			wantError: "missing primary asset",
		},
		{
			name: "orphan primary asset",
			mutate: func(_ *releaseManifest, catalog *[]artifact, _ map[string]string, _ map[string]struct{}) {
				*catalog = append(*catalog, artifact{
					Name:   "seed-1.2.3-freebsd-amd64.tar.gz",
					Type:   "Archive",
					GOOS:   "freebsd",
					GOARCH: "amd64",
					Extra: artifactExtra{
						Checksum: "sha256:" + testChecksum,
						Format:   "tar.gz",
						ID:       "freebsd-archive",
					},
				})
			},
			wantError: "orphan primary asset",
		},
		{
			name: "missing primary asset file",
			mutate: func(_ *releaseManifest, _ *[]artifact, _ map[string]string, files map[string]struct{}) {
				delete(files, "seed-1.2.3-darwin-arm64.tar.gz")
			},
			wantError: "missing primary asset file",
		},
		{
			name: "missing checksum",
			mutate: func(_ *releaseManifest, _ *[]artifact, checksums map[string]string, _ map[string]struct{}) {
				delete(checksums, "seed-1.2.3-darwin-arm64.tar.gz")
			},
			wantError: "missing checksum",
		},
		{
			name: "checksum mismatch",
			mutate: func(_ *releaseManifest, _ *[]artifact, checksums map[string]string, _ map[string]struct{}) {
				checksums["seed-1.2.3-darwin-arm64.tar.gz"] = strings.Repeat("f", 64)
			},
			wantError: "checksum mismatch",
		},
		{
			name: "invalid artifact checksum",
			mutate: func(_ *releaseManifest, catalog *[]artifact, _ map[string]string, _ map[string]struct{}) {
				(*catalog)[0].Extra.Checksum = "sha256:invalid"
			},
			wantError: "invalid artifact checksum",
		},
		{
			name: "wrong artifact format",
			mutate: func(_ *releaseManifest, catalog *[]artifact, _ map[string]string, _ map[string]struct{}) {
				(*catalog)[0].Extra.Format = "zip"
			},
			wantError: "missing primary asset",
		},
		{
			name: "missing signature",
			mutate: func(_ *releaseManifest, _ *[]artifact, _ map[string]string, files map[string]struct{}) {
				delete(files, "seed-1.2.3-darwin-arm64.tar.gz.cosign.bundle")
			},
			wantError: "missing Cosign bundle",
		},
		{
			name: "missing package SBOM",
			mutate: func(_ *releaseManifest, _ *[]artifact, _ map[string]string, files map[string]struct{}) {
				delete(files, "seed_1.2.3_amd64.deb.sbom.json")
			},
			wantError: "missing SBOM",
		},
		{
			name: "orphan SBOM",
			mutate: func(_ *releaseManifest, _ *[]artifact, _ map[string]string, files map[string]struct{}) {
				files["seed-0.9.0-linux-amd64.tar.gz.sbom.json"] = struct{}{}
			},
			wantError: "orphan SBOM",
		},
		{
			name: "orphan signature",
			mutate: func(_ *releaseManifest, _ *[]artifact, _ map[string]string, files map[string]struct{}) {
				files["seed-0.9.0-linux-amd64.tar.gz.cosign.bundle"] = struct{}{}
			},
			wantError: "orphan Cosign bundle",
		},
		{
			name: "inconsistent version",
			mutate: func(_ *releaseManifest, catalog *[]artifact, _ map[string]string, _ map[string]struct{}) {
				(*catalog)[0].Name = strings.Replace((*catalog)[0].Name, "1.2.3", "2.0.0", 1)
			},
			wantError: "release version",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, catalog, checksums, files := completeRelease(t)
			test.mutate(&manifest, &catalog, checksums, files)

			_, err := verifyRelease(manifest, catalog, checksums, files)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("verifyRelease() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func completeRelease(
	t *testing.T,
) (releaseManifest, []artifact, map[string]string, map[string]struct{}) {
	t.Helper()

	manifest, err := readManifest(os.DirFS("../.."), "release/asset-manifest.json")
	if err != nil {
		t.Fatalf("read manifest fixture: %v", err)
	}

	catalog := make([]artifact, 0, len(manifest.PrimaryAssets))
	checksums := make(map[string]string, len(manifest.PrimaryAssets))
	files := make(map[string]struct{}, len(manifest.PrimaryAssets)*2)
	for _, expected := range manifest.PrimaryAssets {
		name := strings.Replace(expected.NameTemplate, "{version}", "1.2.3", 1)
		catalog = append(catalog, artifact{
			Name:   name,
			Type:   expected.Type,
			GOOS:   expected.GOOS,
			GOARCH: expected.GOARCH,
			Extra: artifactExtra{
				Checksum: "sha256:" + testChecksum,
				Format:   expected.Format,
				ID:       expected.ID,
			},
		})
		checksums[name] = testChecksum
		files[name] = struct{}{}
		files[name+".cosign.bundle"] = struct{}{}
		files[name+".sbom.json"] = struct{}{}
	}

	return manifest, catalog, checksums, files
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON fixture: %v", err)
	}
	writeErr := os.WriteFile(path, data, 0o600)
	if writeErr != nil {
		t.Fatalf("write JSON fixture: %v", writeErr)
	}
}
