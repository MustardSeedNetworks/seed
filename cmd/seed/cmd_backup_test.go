package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/seed/internal/support"
)

type fixture map[string]string

func (in *opsInstall) fixtureFiles() fixture {
	return fixture{
		in.keyringPath():   "keyring-v1",
		in.dbPath:          "db-v1",
		in.dbPath + "-wal": "wal-v1",
		in.certFile:        "cert-v1",
		in.keyFile:         "key-v1",
	}
}

func (in *opsInstall) plant(f fixture) {
	in.t.Helper()
	for path, content := range f {
		in.writeFile(path, content)
	}
}

func (in *opsInstall) assert(f fixture) {
	in.t.Helper()
	for path, want := range f {
		if got, ok := in.read(path); !ok || got != want {
			in.t.Errorf("%s = %q (present %v), want %q", path, got, ok, want)
		}
	}
}

func TestBackupRestoreRoundTripWithPreRestoreCopy(t *testing.T) {
	in := newOpsInstall(t, nil)
	original := in.fixtureFiles()
	in.plant(original)
	configV1, _ := in.read(in.configPath)

	archive := filepath.Join(t.TempDir(), "seed-backup.tar.gz")
	code, stdout, stderr := in.run("backup", "-o", archive)
	if code != 0 {
		t.Fatalf("backup exit = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	files := readTarGz(t, archive)
	for _, entry := range []string{
		support.ManifestEntry, support.EntryConfig, support.EntryKeyring,
		support.EntryDatabase, support.EntryDatabaseWAL, support.EntryCertificate, support.EntryKey,
	} {
		if _, ok := files[entry]; !ok {
			t.Errorf("backup lacks %s; has %v", entry, keys(files))
		}
	}
	if _, ok := files[support.EntryDatabaseSHM]; ok {
		t.Error("backup invented a shared-memory file")
	}

	// The install drifts after the backup was taken and grows a shared-memory
	// file the archive never had, which the restored database must not inherit.
	in.plant(fixture{
		in.keyringPath():   "keyring-v2",
		in.dbPath:          "db-v2",
		in.dbPath + "-wal": "wal-v2",
		in.dbPath + "-shm": "shm-v2",
		in.certFile:        "cert-v2",
		in.keyFile:         "key-v2",
	})
	in.writeFile(in.configPath, `{"server":{"port":9999}}`)

	code, stdout, stderr = in.run("restore", archive)
	if code != 0 {
		t.Fatalf("restore exit = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	in.assert(original)
	if got, _ := in.read(in.configPath); got != configV1 {
		t.Errorf("config not restored: %q", got)
	}
	if _, ok := in.read(in.dbPath + "-shm"); ok {
		t.Error("a stale shared-memory file survived the restore")
	}

	preRestore, err := filepath.Glob(strings.TrimSuffix(archive, ".tar.gz") + ".pre-restore-*.tar.gz")
	if err != nil || len(preRestore) != 1 {
		t.Fatalf("pre-restore copies = %v (err %v), want exactly one", preRestore, err)
	}
	if !strings.Contains(stdout, preRestore[0]) {
		t.Errorf("restore did not say where the pre-restore copy is:\n%s", stdout)
	}
	kept := readTarGz(t, preRestore[0])
	if kept[support.EntryDatabase] != "db-v2" || kept[support.EntryConfig] != `{"server":{"port":9999}}` {
		t.Errorf("the pre-restore copy does not hold the state that was replaced: %v", keys(kept))
	}

	// Restoring the pre-restore copy undoes the restore.
	if code, _, stderr = in.run("restore", preRestore[0]); code != 0 {
		t.Fatalf("undo restore exit = %d (stderr %q)", code, stderr)
	}
	if got, _ := in.read(in.dbPath); got != "db-v2" {
		t.Errorf("database after undo = %q, want db-v2", got)
	}
}

func TestBackupAndRestoreRefuseWhileTheDaemonRuns(t *testing.T) {
	in := newOpsInstall(t, nil)
	original := in.fixtureFiles()
	in.plant(original)

	archive := filepath.Join(t.TempDir(), "seed-backup.tar.gz")
	if code, _, stderr := in.run("backup", "-o", archive); code != 0 {
		t.Fatalf("backup exit = %d (stderr %q)", code, stderr)
	}
	in.plant(fixture{in.dbPath: "db-live"})
	in.holdLock(8443)

	tests := []struct {
		name string
		args []string
	}{
		{"backup", []string{"backup", "-o", filepath.Join(t.TempDir(), "live.tar.gz")}},
		{"restore", []string{"restore", archive}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := in.run(tc.args...)
			if code != exitRefused {
				t.Fatalf("exit = %d, want %d (stderr %q)", code, exitRefused, stderr)
			}
			if !strings.Contains(stderr, "seed serve is running") || !strings.Contains(stderr, "Stop seed serve") {
				t.Errorf("refusal does not say what is running or what to do:\n%s", stderr)
			}
		})
	}
	if got, _ := in.read(in.dbPath); got != "db-live" {
		t.Errorf("a refused restore changed the database: %q", got)
	}
}

func TestBackupAndRestoreFailClosedWhenTheLockCannotBeRead(t *testing.T) {
	if !canBreakLock() {
		t.Skip("cannot make the lock file unreadable here")
	}
	in := newOpsInstall(t, nil)
	in.plant(in.fixtureFiles())
	archive := filepath.Join(t.TempDir(), "seed-backup.tar.gz")
	if code, _, stderr := in.run("backup", "-o", archive); code != 0 {
		t.Fatalf("backup exit = %d (stderr %q)", code, stderr)
	}
	in.plant(fixture{in.dbPath: "db-live"})

	in.holdLock(0)
	lockPath := filepath.Join(filepath.Dir(in.configPath), instance.LockFileName)
	if err := os.Chmod(lockPath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(lockPath, 0o600) })

	for _, args := range [][]string{
		{"backup", "-o", filepath.Join(t.TempDir(), "x.tar.gz")},
		{"restore", archive},
	} {
		code, _, stderr := in.run(args...)
		if code != 1 || !strings.Contains(stderr, "check whether seed serve is running") {
			t.Errorf("%s: exit %d, stderr %q; want a refusal to proceed on an unreadable lock", args[0], code, stderr)
		}
	}
	if got, _ := in.read(in.dbPath); got != "db-live" {
		t.Errorf("restore touched the database despite an unreadable lock: %q", got)
	}
}

func TestBackupErrors(t *testing.T) {
	t.Run("nothing to back up", func(t *testing.T) {
		in := newOpsInstall(t, nil)
		if err := os.Remove(in.configPath); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "x.tar.gz")
		code, _, stderr := in.run("backup", "-o", out)
		if code != 1 || !strings.Contains(stderr, "nothing to back up") {
			t.Errorf("exit %d, stderr %q", code, stderr)
		}
		if _, err := os.Stat(out); err == nil {
			t.Error("an empty backup left an archive behind")
		}
	})
	t.Run("existing output is never overwritten", func(t *testing.T) {
		in := newOpsInstall(t, nil)
		out := filepath.Join(t.TempDir(), "x.tar.gz")
		in.writeFile(out, "do not clobber")
		code, _, _ := in.run("backup", "-o", out)
		if code != 1 {
			t.Errorf("exit = %d, want 1", code)
		}
		if got, _ := in.read(out); got != "do not clobber" {
			t.Errorf("backup overwrote the existing file: %q", got)
		}
	})
}

// TestRestoreRefusesTraversalThroughTheCommand drives a hostile archive through
// the real command: exit 1, nothing changed, no pre-restore copy made for an
// archive that was never accepted.
func TestRestoreRefusesTraversalThroughTheCommand(t *testing.T) {
	in := newOpsInstall(t, nil)
	in.plant(in.fixtureFiles())
	outside := filepath.Join(filepath.Dir(in.workDir), "escaped.txt")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range []struct{ name, body string }{
		{support.ManifestEntry, `{"version":1}`},
		{"../escaped.txt", "pwned"},
		{"../../escaped.txt", "pwned"},
		{support.EntryDatabase, "attacker-db"},
	} {
		header := &tar.Header{Name: e.name, Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(e.body))}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archiveDir := t.TempDir()
	archive := filepath.Join(archiveDir, "hostile.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := in.run("restore", archive)
	if code != 1 || !strings.Contains(stderr, "nothing was changed") {
		t.Fatalf("exit %d, stderr %q; want exit 1 and nothing changed", code, stderr)
	}
	if got, _ := in.read(in.dbPath); got != "db-v1" {
		t.Errorf("database = %q: a hostile archive changed the install", got)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Error("the archive wrote outside the install")
	}
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("a refused archive produced a pre-restore copy: %v", entries)
	}
}

func TestRestoreOnAFreshHostKeepsNoPreRestoreCopy(t *testing.T) {
	src := newOpsInstall(t, nil)
	src.plant(src.fixtureFiles())
	archive := filepath.Join(t.TempDir(), "seed-backup.tar.gz")
	if code, _, stderr := src.run("backup", "-o", archive); code != 0 {
		t.Fatalf("backup exit = %d (stderr %q)", code, stderr)
	}

	dst := newOpsInstall(t, nil)
	if err := os.Remove(dst.configPath); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := dst.run("restore", archive)
	if code != 0 {
		t.Fatalf("exit = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "pre-restore") {
		t.Errorf("restore claimed to keep a copy of nothing:\n%s", stdout)
	}
	if got, _ := dst.read(dst.dbPath); got != "db-v1" {
		t.Errorf("database not restored on a fresh host: %q", got)
	}
}
