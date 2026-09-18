package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestSaveIsNeverObservedPartial is the reproduction for #2747: a reader that
// opens the config while Save is running must never see a truncated document.
// A plain [os.WriteFile] truncates the file before it writes, so a concurrent
// Load lands in that window and fails to parse.
func TestSaveIsNeverObservedPartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.json")

	cfg := config.DefaultConfig()
	// Make the document large enough that the write is not a single atomic
	// page store, which is what opens the window in the first place.
	for i := range 400 {
		cfg.NetworkDiscovery.TargetNetworks = append(
			cfg.NetworkDiscovery.TargetNetworks,
			config.SubnetConfig{
				CIDR: "10." + itoa(i/256) + "." + itoa(i%256) + ".0/24",
				Name: "generated network " + itoa(i),
			},
		)
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	partial := make(chan string, 1)

	wg.Go(func() {
		readUntilPartial(path, stop, partial)
	})

	for range 200 {
		if err := cfg.Save(path); err != nil {
			close(stop)
			wg.Wait()
			t.Fatalf("save: %v", err)
		}
	}
	close(stop)
	wg.Wait()

	select {
	case msg := <-partial:
		t.Fatalf("a concurrent reader observed a partial config: %s", msg)
	default:
	}
}

// readUntilPartial re-reads path until stop closes or one read fails to parse,
// and reports the first such read.
func readUntilPartial(path string, stop <-chan struct{}, partial chan<- string) {
	report := func(msg string) {
		select {
		case partial <- msg:
		default:
		}
	}
	for {
		select {
		case <-stop:
			return
		default:
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report("read: " + err.Error())
			return
		}
		var doc map[string]any
		if parseErr := json.Unmarshal(data, &doc); parseErr != nil {
			report("parse at " + itoa(len(data)) + " bytes: " + parseErr.Error())
			return
		}
	}
}

// TestSaveFailureLeavesPreviousConfigIntact covers the row's acceptance: a save
// that cannot complete must leave the file that was there, and must not leave
// its scratch file behind.
func TestSaveFailureLeavesPreviousConfigIntact(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.json")

	cfg := config.DefaultConfig()
	cfg.Server.Port = 8443
	if err := cfg.Save(path); err != nil {
		t.Fatalf("seed save: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read seed config: %v", err)
	}

	if chmodErr := os.Chmod(dir, 0o500); chmodErr != nil {
		t.Fatalf("chmod dir: %v", chmodErr)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	cfg.Server.Port = 9999
	if saveErr := cfg.Save(path); saveErr == nil {
		t.Fatal("expected Save to fail in a read-only directory")
	}

	if restoreErr := os.Chmod(dir, 0o700); restoreErr != nil {
		t.Fatalf("restore dir: %v", restoreErr)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after failed save: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("failed save changed the config on disk:\nbefore %d bytes\nafter  %d bytes", len(before), len(after))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Errorf("failed save left %q behind", e.Name())
		}
	}
}

// TestSaveCleansUpAfterAFailedRename covers the other half of a failed save:
// the scratch file exists by the time the replace fails, and must not be left
// in the operator's config directory.
func TestSaveCleansUpAfterAFailedRename(t *testing.T) {
	dir := t.TempDir()
	// A directory cannot be replaced by a rename from a regular file, so the
	// save fails after the scratch file has been written.
	path := filepath.Join(dir, "seed.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := config.DefaultConfig().Save(path); err == nil {
		t.Fatal("expected Save to fail when the target is a directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Errorf("failed save left %q behind", e.Name())
		}
	}
}

// TestSaveKeepsOwnerOnlyMode pins that the config still lands at 0600 — the
// temp file [os.CreateTemp] hands back is 0600 too, but only by its own default.
func TestSaveKeepsOwnerOnlyMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.json")

	cfg := config.DefaultConfig()
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("config mode = %o, want 600", mode)
	}
}

// TestSaveThroughSymlinkWritesTheTarget pins the behaviour [os.WriteFile] had and
// a naive rename loses: an operator who symlinks the config to a managed file
// keeps the symlink, and the bytes land on its target.
func TestSaveThroughSymlinkWritesTheTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "seed.json")

	cfg := config.DefaultConfig()
	if err := cfg.Save(target); err != nil {
		t.Fatalf("seed save: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	cfg.Server.Port = 9443
	if err := cfg.Save(link); err != nil {
		t.Fatalf("save through symlink: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("save replaced the symlink with a regular file")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.Contains(string(data), "9443") {
		t.Error("save through the symlink did not write the target")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
