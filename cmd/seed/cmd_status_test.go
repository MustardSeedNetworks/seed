package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"
)

func TestStatusExitCodes(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, in *opsInstall)
		args       []string
		wantCode   int
		wantStdout []string
		wantStderr []string
	}{
		{
			name:       "not running is 1",
			wantCode:   1,
			wantStdout: []string{"Status: NOT RUNNING"},
		},
		{
			name: "running and answering is 0",
			setup: func(_ *testing.T, in *opsInstall) {
				in.holdLock(in.serveVersion())
			},
			wantCode:   0,
			wantStdout: []string{"Status: RUNNING", "Version: 9.9.9-test", "Commit: abc1234", "TLS fingerprint: AA:BB"},
		},
		{
			name: "running but not answering is 2",
			setup: func(t *testing.T, in *opsInstall) {
				in.serveVersion()
				in.holdLock(freePort(t))
			},
			wantCode:   2,
			wantStdout: []string{"Status: NOT ANSWERING"},
			wantStderr: []string{"did not answer"},
		},
		{
			name: "running with no readable certificate is 2",
			setup: func(t *testing.T, in *opsInstall) {
				in.holdLock(freePort(t))
			},
			wantCode:   2,
			wantStderr: []string{"read the daemon certificate"},
		},
		{
			name: "unreadable config is 2",
			setup: func(_ *testing.T, in *opsInstall) {
				in.writeFile(in.configPath, "{not json")
			},
			wantCode:   2,
			wantStderr: []string{"load config"},
		},
		{
			name: "an unreadable lock is 2, not not-running",
			setup: func(t *testing.T, in *opsInstall) {
				if !canBreakLock() {
					t.Skip("cannot make the lock file unreadable here")
				}
				in.holdLock(0)
				lockPath := filepath.Join(filepath.Dir(in.configPath), instance.LockFileName)
				if err := os.Chmod(lockPath, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(lockPath, 0o600) })
			},
			wantCode:   2,
			wantStderr: []string{"check whether seed serve is running"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := newOpsInstall(t, nil)
			if tc.setup != nil {
				tc.setup(t, in)
			}
			code, stdout, stderr := in.run(append([]string{"status"}, tc.args...)...)
			if code != tc.wantCode {
				t.Fatalf("exit = %d, want %d\nstdout: %s\nstderr: %s", code, tc.wantCode, stdout, stderr)
			}
			assertContains(t, "stdout", stdout, tc.wantStdout)
			assertContains(t, "stderr", stderr, tc.wantStderr)
			if strings.Contains(stderr, "Usage:") {
				t.Errorf("a failed status printed the usage text:\n%s", stderr)
			}
		})
	}
}

func assertContains(t *testing.T, label, got string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %q:\n%s", label, want, got)
		}
	}
}

func TestStatusJSON(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(in *opsInstall)
		wantCode int
		check    func(t *testing.T, got map[string]any)
	}{
		{
			name:     "not running",
			wantCode: 1,
			check: func(t *testing.T, got map[string]any) {
				if got["running"] != false {
					t.Errorf("running = %v, want false", got["running"])
				}
			},
		},
		{
			name:     "running",
			setup:    func(in *opsInstall) { in.holdLock(in.serveVersion()) },
			wantCode: 0,
			check: func(t *testing.T, got map[string]any) {
				if got["running"] != true || got["version"] != "9.9.9-test" || got["uiBuildHash"] != "uihash" {
					t.Errorf("unexpected status: %v", got)
				}
				if _, ok := got["error"]; ok {
					t.Errorf("a healthy daemon reported an error: %v", got["error"])
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := newOpsInstall(t, nil)
			if tc.setup != nil {
				tc.setup(in)
			}
			code, stdout, stderr := in.run("status", "--json")
			if code != tc.wantCode {
				t.Fatalf("exit = %d, want %d (stderr %q)", code, tc.wantCode, stderr)
			}
			var got map[string]any
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("status --json is not JSON: %v\n%s", err, stdout)
			}
			tc.check(t, got)
		})
	}
}
