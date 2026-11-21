package support_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/support"
)

func archiveEpoch() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }

type install struct {
	dir     string
	targets []support.Target
}

func newInstall(t *testing.T) install {
	t.Helper()
	dir := t.TempDir()
	return install{
		dir: dir,
		targets: []support.Target{
			{Entry: support.EntryConfig, Path: filepath.Join(dir, "etc", "seed.json")},
			{Entry: support.EntryKeyring, Path: filepath.Join(dir, "etc", "credential.key")},
			{Entry: support.EntryDatabase, Path: filepath.Join(dir, "work", "data", "seed.db")},
			{
				Entry:          support.EntryDatabaseWAL,
				Path:           filepath.Join(dir, "work", "data", "seed.db-wal"),
				RemoveIfAbsent: true,
			},
			{
				Entry:          support.EntryDatabaseSHM,
				Path:           filepath.Join(dir, "work", "data", "seed.db-shm"),
				RemoveIfAbsent: true,
			},
			{Entry: support.EntryCertificate, Path: filepath.Join(dir, "work", "certs", "server.crt")},
			{Entry: support.EntryKey, Path: filepath.Join(dir, "work", "certs", "server.key")},
		},
	}
}

func (i install) path(entry string) string {
	for _, target := range i.targets {
		if target.Entry == entry {
			return target.Path
		}
	}
	panic("unknown entry " + entry)
}

func (i install) write(t *testing.T, entry, content string) {
	t.Helper()
	path := i.path(entry)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (i install) read(t *testing.T, entry string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(i.path(entry))
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data), true
}

func (i install) members(t *testing.T) []support.Member {
	t.Helper()
	var members []support.Member
	for _, target := range i.targets {
		if _, ok := i.read(t, target.Entry); ok {
			members = append(members, support.Member{Entry: target.Entry, Path: target.Path})
		}
	}
	return members
}

func (i install) leftovers(t *testing.T) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(i.dir, func(path string, _ os.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".restore-tmp") {
			found = append(found, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func backupBytes(t *testing.T, members []support.Member) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := support.Backup(&buf, members, "v0.0.0-test", archiveEpoch()); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	return buf.Bytes()
}

func TestBackupRestoreRoundTrip(t *testing.T) {
	src := newInstall(t)
	contents := map[string]string{
		support.EntryConfig:      `{"server":{"port":8443}}`,
		support.EntryKeyring:     "keyring-bytes",
		support.EntryDatabase:    "sqlite-bytes",
		support.EntryDatabaseWAL: "wal-bytes",
		support.EntryCertificate: "cert-pem",
		support.EntryKey:         "key-pem",
	}
	for entry, content := range contents {
		src.write(t, entry, content)
	}
	archive := backupBytes(t, src.members(t))

	dst := newInstall(t)
	restored, err := support.Restore(bytes.NewReader(archive), dst.targets, nil)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(restored) != len(contents) {
		t.Errorf("restored %d entries, want %d: %v", len(restored), len(contents), restored)
	}
	for entry, want := range contents {
		got, ok := dst.read(t, entry)
		if !ok || got != want {
			t.Errorf("%s = %q (present %v), want %q", entry, got, ok, want)
		}
	}
	if _, ok := dst.read(t, support.EntryDatabaseSHM); ok {
		t.Error("restore invented a shared-memory file the archive did not carry")
	}
	if left := dst.leftovers(t); len(left) != 0 {
		t.Errorf("restore left temporary files behind: %v", left)
	}
	info, err := os.Stat(dst.path(support.EntryKey))
	if err != nil {
		t.Fatal(err)
	}
	if runtimeEnforcesModes() && info.Mode().Perm() != 0o600 {
		t.Errorf("restored key mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestRestoreReplacesAndRemovesStaleDatabaseSidecars(t *testing.T) {
	dst := newInstall(t)
	dst.write(t, support.EntryConfig, "old-config")
	dst.write(t, support.EntryDatabase, "old-db")
	dst.write(t, support.EntryDatabaseWAL, "old-wal")
	dst.write(t, support.EntryDatabaseSHM, "old-shm")

	src := newInstall(t)
	src.write(t, support.EntryConfig, "new-config")
	src.write(t, support.EntryDatabase, "new-db")
	archive := backupBytes(t, src.members(t))

	hookCalls := 0
	if _, err := support.Restore(bytes.NewReader(archive), dst.targets, func() error {
		hookCalls++
		if got, _ := dst.read(t, support.EntryConfig); got != "old-config" {
			t.Errorf("config was already replaced when the pre-restore hook ran: %q", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if hookCalls != 1 {
		t.Errorf("beforeInstall ran %d times, want 1", hookCalls)
	}
	if got, _ := dst.read(t, support.EntryDatabase); got != "new-db" {
		t.Errorf("database = %q, want new-db", got)
	}
	for _, entry := range []string{support.EntryDatabaseWAL, support.EntryDatabaseSHM} {
		if _, ok := dst.read(t, entry); ok {
			t.Errorf("stale %s survived a restore of a database that has none", entry)
		}
	}
}

func TestRestoreHookFailureLeavesInstallUntouched(t *testing.T) {
	dst := newInstall(t)
	dst.write(t, support.EntryConfig, "old-config")
	src := newInstall(t)
	src.write(t, support.EntryConfig, "new-config")
	archive := backupBytes(t, src.members(t))

	_, err := support.Restore(bytes.NewReader(archive), dst.targets, func() error { return errors.New("disk full") })
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("Restore error = %v, want the hook's failure", err)
	}
	if got, _ := dst.read(t, support.EntryConfig); got != "old-config" {
		t.Errorf("config changed to %q despite the failed hook", got)
	}
}

type tarEntry struct {
	name     string
	typeflag byte
	body     string
	link     string
}

func buildArchive(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		header := &tar.Header{
			Name: e.name, Typeflag: e.typeflag, Linkname: e.link, Mode: 0o600,
			Size: int64(len(e.body)),
		}
		if e.typeflag != tar.TypeReg {
			header.Size = 0
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func manifestEntry() tarEntry {
	return tarEntry{name: support.ManifestEntry, typeflag: tar.TypeReg, body: `{"version":1,"entries":[]}`}
}

func reg(name, body string) tarEntry { return tarEntry{name: name, typeflag: tar.TypeReg, body: body} }

// TestRestoreRefusesHostileArchives is the traversal table: every archive here
// is refused as a whole, before anything is written, and the install it was
// aimed at is left exactly as it was.
func TestRestoreRefusesHostileArchives(t *testing.T) {
	good := reg(support.EntryConfig, "from-archive")
	tests := []struct {
		name    string
		archive func(t *testing.T) []byte
	}{
		{"parent traversal", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), reg("../../escape.txt", "x"), good})
		}},
		{"traversal hidden in a known path", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), reg("config/../../escape.txt", "x")})
		}},
		{"absolute path", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), reg("/etc/passwd", "x")})
		}},
		{"windows separators", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), reg(`config\seed.json`, "x")})
		}},
		{"symlink to elsewhere", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{
				manifestEntry(),
				{name: support.EntryConfig, typeflag: tar.TypeSymlink, link: "/etc/shadow"},
			})
		}},
		{"hard link", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{
				manifestEntry(),
				{name: support.EntryConfig, typeflag: tar.TypeLink, link: "/etc/shadow"},
			})
		}},
		{"directory entry", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), {name: "config/", typeflag: tar.TypeDir}})
		}},
		{"entry that is not a backup role", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), good, reg("config/other.json", "x")})
		}},
		{"duplicate entry", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry(), good, good})
		}},
		{"no manifest", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{good})
		}},
		{"manifest not first", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{good, manifestEntry()})
		}},
		{"unsupported manifest version", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{reg(support.ManifestEntry, `{"version":99}`), good})
		}},
		{"unreadable manifest", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{reg(support.ManifestEntry, `not json`), good})
		}},
		{"manifest only", func(t *testing.T) []byte {
			return buildArchive(t, []tarEntry{manifestEntry()})
		}},
		{"not gzip", func(*testing.T) []byte { return []byte("plain text, not an archive") }},
		{"truncated mid-archive", func(t *testing.T) []byte {
			full := buildArchive(
				t,
				[]tarEntry{manifestEntry(), reg(support.EntryDatabase, strings.Repeat("d", 1<<16)), good},
			)
			return full[:len(full)/2]
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dst := newInstall(t)
			dst.write(t, support.EntryConfig, "precious-config")
			dst.write(t, support.EntryDatabase, "precious-db")

			hookCalls := 0
			_, err := support.Restore(bytes.NewReader(tc.archive(t)), dst.targets, func() error {
				hookCalls++
				return nil
			})
			if !errors.Is(err, support.ErrArchive) {
				t.Fatalf("Restore error = %v, want ErrArchive", err)
			}
			if hookCalls != 0 {
				t.Error("the pre-restore hook ran for an archive that was refused")
			}
			if got, _ := dst.read(t, support.EntryConfig); got != "precious-config" {
				t.Errorf("config = %q: a refused archive changed the install", got)
			}
			if got, _ := dst.read(t, support.EntryDatabase); got != "precious-db" {
				t.Errorf("database = %q: a refused archive changed the install", got)
			}
			if left := dst.leftovers(t); len(left) != 0 {
				t.Errorf("a refused archive left temporary files: %v", left)
			}
			if _, statErr := os.Stat(filepath.Join(dst.dir, "escape.txt")); statErr == nil {
				t.Error("traversal wrote outside the install")
			}
		})
	}
}

func TestBackupRefusesUnknownRole(t *testing.T) {
	src := newInstall(t)
	src.write(t, support.EntryConfig, "x")
	err := support.Backup(
		io.Discard,
		[]support.Member{{Entry: "../etc/passwd", Path: src.path(support.EntryConfig)}},
		"v",
		archiveEpoch(),
	)
	if err == nil {
		t.Fatal("Backup accepted an entry that is not a backup role")
	}
}

func TestBackupRefusesNonRegularSource(t *testing.T) {
	src := newInstall(t)
	dir := filepath.Dir(src.path(support.EntryConfig))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	err := support.Backup(io.Discard, []support.Member{{Entry: support.EntryConfig, Path: dir}}, "v", archiveEpoch())
	if err == nil {
		t.Fatal("Backup archived a directory as a file")
	}
}

func TestWriteBundleRefusesEscapingNames(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{"flat", "manifest.json", false},
		{"nested", "logs/seed.log", false},
		{"parent", "../manifest.json", true},
		{"absolute", "/manifest.json", true},
		{"backslash", `logs\seed.log`, true},
		{"empty", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := support.WriteBundle(io.Discard, []support.File{{Name: tc.file, Data: []byte("x")}}, archiveEpoch())
			if (err != nil) != tc.wantErr {
				t.Errorf("WriteBundle(%q) error = %v, wantErr %v", tc.file, err, tc.wantErr)
			}
		})
	}
}
