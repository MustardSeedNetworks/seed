package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"
)

// isolateConfigDir points every path resolver at a temp directory so the test
// neither reads nor writes the developer's own ~/.config/seed (seed#2688), and
// returns the config path the CLI will resolve.
func isolateConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	configPath := filepath.Join(dir, "seed.json")
	t.Setenv("SEED_CONFIG_PATH", configPath)
	if err := os.WriteFile(configPath, []byte(`{"server":{"port":18443}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

// TestDaemonOwnedCommandsRefuseWhileServeHoldsTheLock is the row's table test:
// every CLI verb that writes daemon-owned state refuses while seed serve holds
// the instance lock, and says what to use instead.
func TestDaemonOwnedCommandsRefuseWhileServeHoldsTheLock(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantsRoute string
	}{
		{"reset-config", []string{"reset-config", "--force"}, "seed serve"},
		{"credentials", []string{"credentials"}, "/api/v1/setup/status"},
		{"setup-wizard", []string{"setup-wizard", "--generate-password"}, "/api/v1/setup/complete"},
		{"license activate", []string{"license", "activate", "-k", "AAAA"}, "seed serve"},
		{"license trial", []string{"license", "trial"}, "seed serve"},
		{"license deactivate", []string{"license", "deactivate"}, "seed serve"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := isolateConfigDir(t)

			lock, err := instance.Acquire(filepath.Dir(configPath))
			if err != nil {
				t.Fatalf("acquire lock: %v", err)
			}
			t.Cleanup(func() { _ = lock.Release() })
			if setErr := lock.SetPort(18443); setErr != nil {
				t.Fatalf("set port: %v", setErr)
			}

			state := newCLIState()
			initCommands(state)
			state.rootCmd.SetArgs(tc.args)
			state.rootCmd.SetOut(&bytes.Buffer{})
			state.rootCmd.SetErr(&bytes.Buffer{})

			execErr := state.rootCmd.Execute()
			if execErr == nil {
				t.Fatalf("%v ran while the daemon held the lock; want a refusal", tc.args)
			}

			var held *daemonRunningError
			if !errors.As(execErr, &held) {
				t.Fatalf("error is %T (%v); want *daemonRunningError", execErr, execErr)
			}
			if got := exitCodeFor(execErr); got != exitDaemonRunning {
				t.Errorf("exit code = %d, want %d", got, exitDaemonRunning)
			}
			msg := execErr.Error()
			if !strings.Contains(msg, tc.wantsRoute) {
				t.Errorf("message %q does not name %q", msg, tc.wantsRoute)
			}
			if !strings.Contains(msg, "18443") {
				t.Errorf("message %q does not name the running daemon's port", msg)
			}
		})
	}
}

// TestLicenseStatusIsNotRefused pins the read/write split: the guard exists so
// no CLI quietly *writes* daemon-owned state, so a read stays available.
func TestLicenseStatusIsNotRefused(t *testing.T) {
	configPath := isolateConfigDir(t)

	lock, err := instance.Acquire(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	t.Cleanup(func() { _ = lock.Release() })

	state := newCLIState()
	initCommands(state)
	state.rootCmd.SetArgs([]string{"license", "status"})
	state.rootCmd.SetOut(&bytes.Buffer{})
	state.rootCmd.SetErr(&bytes.Buffer{})

	if execErr := state.rootCmd.Execute(); execErr != nil {
		t.Fatalf("license status refused while the daemon runs: %v", execErr)
	}
}

// TestCommandsRunWithNoDaemon is the other half: with nothing holding the lock
// the guard is invisible, so a refusal cannot be a constant.
func TestCommandsRunWithNoDaemon(t *testing.T) {
	configPath := isolateConfigDir(t)
	if _, held, err := instance.Probe(filepath.Dir(configPath)); err != nil || held {
		t.Fatalf("probe on a free directory: held=%v err=%v", held, err)
	}

	state := newCLIState()
	initCommands(state)
	state.rootCmd.SetArgs([]string{"credentials"})
	state.rootCmd.SetOut(&bytes.Buffer{})
	state.rootCmd.SetErr(&bytes.Buffer{})

	if execErr := state.rootCmd.Execute(); execErr != nil {
		t.Fatalf("credentials refused with no daemon running: %v", execErr)
	}
}

// TestServeHoldsTheLockAndPublishesItsPort starts a real seed serve in another
// process and reads its PID and port back through the lock.
//
// It cannot be done in one process. A test that takes the lock itself and
// calls SetPort is asserting the number it just wrote: deleting SetPort from
// runServe leaves such a test green, and the PID it reports is its own. Only a
// second process can show that the daemon published them.
func TestServeHoldsTheLockAndPublishesItsPort(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a real daemon; -short skips it")
	}
	bin := buildSeedBinary(t)

	// The daemon writes certs/ and data/ relative to its working directory, so
	// it gets its own; HOME and XDG keep its licence state out of the
	// developer's ~/.config/seed (seed#2688).
	runDir := t.TempDir()
	configPath := filepath.Join(runDir, "seed.json")
	writeQuietConfig(t, configPath, 19443)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "serve", "--config", configPath)
	cmd.Dir = runDir
	cmd.Env = append(os.Environ(),
		"HOME="+runDir,
		"XDG_CONFIG_HOME="+filepath.Join(runDir, "config"),
		"XDG_DATA_HOME="+filepath.Join(runDir, "data"),
		"XDG_STATE_HOME="+filepath.Join(runDir, "state"),
		"XDG_CACHE_HOME="+filepath.Join(runDir, "cache"),
	)
	var daemonLog bytes.Buffer
	cmd.Stdout = &daemonLog
	cmd.Stderr = &daemonLog
	if err := cmd.Start(); err != nil {
		t.Fatalf("start seed serve: %v", err)
	}
	t.Cleanup(cancel)

	// The first start generates a 4096-bit self-signed certificate, so the port
	// is published seconds after the lock is taken, not with it.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	info := awaitPublishedPort(t, runDir, &daemonLog, exited)

	if info.PID != cmd.Process.Pid {
		t.Errorf("lock reports pid %d; the daemon is %d", info.PID, cmd.Process.Pid)
	}
	if info.Port != 19443 {
		t.Errorf("lock reports port %d; want 19443\n%s", info.Port, daemonLog.String())
	}

	// And the guard reads what that daemon published, from this process.
	err := refuseIfDaemonOwns(configPath, "use the API")
	var held *daemonRunningError
	if !errors.As(err, &held) {
		t.Fatalf("guard returned %v; want a refusal while the daemon runs", err)
	}
	if !strings.Contains(held.Error(), "19443") || !strings.Contains(held.Error(), strconv.Itoa(cmd.Process.Pid)) {
		t.Errorf("refusal %q does not name the running daemon's pid and port", held.Error())
	}
}

// awaitPublishedPort polls the lock until the daemon has bound and published,
// rather than sleeping a guessed interval.
func awaitPublishedPort(t *testing.T, dir string, daemonLog *bytes.Buffer, exited <-chan error) instance.Info {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		info, held, err := instance.Probe(dir)
		if err != nil {
			t.Fatalf("probe: %v", err)
		}
		if held && info.Port != 0 {
			return info
		}
		select {
		case waitErr := <-exited:
			t.Fatalf("daemon exited before publishing a port (%v)\n%s", waitErr, daemonLog.String())
		case <-time.After(200 * time.Millisecond):
		}
	}
	t.Fatalf("daemon did not publish a port within the deadline\n%s", daemonLog.String())
	return instance.Info{}
}

// buildSeedBinary builds the binary under test once for this test.
func buildSeedBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "seed")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// writeQuietConfig writes a config whose daemon binds a high port and puts no
// packets on the wire, so a unit test never sweeps the developer's LAN.
func writeQuietConfig(t *testing.T, path string, port int) {
	t.Helper()
	cfg := fmt.Sprintf(`{
  "server": {"port": %d},
  "networkDiscovery": {"enabled": false},
  "database": {"path": "seed.db"}
}`, port)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
