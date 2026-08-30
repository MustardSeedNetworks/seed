package iperf_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
)

// Manager tests that drive a real child process without requiring iperf3.
//
// These used to be gated on SKIP_IPERF_TEST (which CI sets to 1 in both jobs)
// and on `iperf.CheckInstalled()`, and the ones that did run treated a failure
// to start as an acceptable outcome — "may fail due to permissions or binary
// issues - that's OK". A test that accepts both outcomes cannot fail.
//
// findIperf3Binary returns the cached path first, so pointing the cache at a
// stand-in binary needs no production change. The stand-in is a real process
// that really binds the port it is given, so the spawn, the readiness wait, the
// PID, the Wait monitor and the Kill are all exercised for real. Only the
// iperf3 protocol is absent, and that belongs in the iperf3-required CI job.

// fakeIperf3 builds the stand-in binary into the test's own temp dir and
// returns its path. Built per test rather than cached in a package-level
// variable: Go's build cache makes the repeat builds cheap, and the alternative
// needs mutable globals that outlive the test that created them.
//
// A build failure fails the test — there is nothing to skip for, since the
// toolchain running the test is the one that builds it.
func fakeIperf3(t *testing.T) string {
	t.Helper()

	out := filepath.Join(t.TempDir(), "iperf3")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", out, "./testdata/fakeiperf3")
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the stand-in iperf3 binary: %v\n%s", err, combined)
	}

	return out
}

// useFakeIperf3 points the package's cached binary path at the stand-in for the
// duration of one test, restoring whatever was there before.
func useFakeIperf3(t *testing.T) {
	t.Helper()

	previous := iperf.IperfBinaryPath()
	iperf.SetIperfBinaryPath(fakeIperf3(t))
	t.Cleanup(func() { iperf.SetIperfBinaryPath(previous) })
}

// freePort returns a port nothing is listening on. The listener is closed
// before returning, so the port is free for the child process to bind.
func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if closeErr := listener.Close(); closeErr != nil {
		t.Fatalf("releasing the reserved port: %v", closeErr)
	}

	return port
}

// portAccepts reports whether something is listening on the port right now.
func portAccepts(port int) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()

	return true
}

// waitUntil polls until cond holds, failing with msg if it never does. Used
// instead of a fixed sleep so the test tracks observable state.
func waitUntil(t *testing.T, msg string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", msg)
}

func TestStartServer_StartsAListeningProcess(t *testing.T) {
	useFakeIperf3(t)
	manager := iperf.NewManager()
	port := freePort(t)

	if err := manager.StartServer(port); err != nil {
		t.Fatalf("StartServer(%d): %v", port, err)
	}
	t.Cleanup(func() { _ = manager.StopServer() })

	status := manager.GetServerStatus()
	if !status.Running {
		t.Error("status.Running = false after a successful start")
	}
	if status.Port != port {
		t.Errorf("status.Port = %d, want %d", status.Port, port)
	}
	if status.PID <= 0 {
		t.Errorf("status.PID = %d, want the child's pid", status.PID)
	}
	// The status is only worth anything if something is actually listening.
	if !portAccepts(port) {
		t.Errorf("nothing is listening on port %d despite status.Running", port)
	}
}

func TestStopServer_KillsTheProcessAndFreesThePort(t *testing.T) {
	useFakeIperf3(t)
	manager := iperf.NewManager()
	port := freePort(t)

	if err := manager.StartServer(port); err != nil {
		t.Fatalf("StartServer(%d): %v", port, err)
	}
	pid := manager.GetServerStatus().PID

	if err := manager.StopServer(); err != nil {
		t.Fatalf("StopServer: %v", err)
	}

	if manager.GetServerStatus().Running {
		t.Error("status.Running = true after StopServer")
	}
	// The half that matters: a stop that leaves the child alive holds the port
	// and the next start fails for a reason nobody can see. processAlive uses
	// signal 0, which also succeeds on a zombie — so this catches a child that
	// was killed but never reaped, not only one still running.
	waitUntil(t, "the port to be released", func() bool { return !portAccepts(port) })
	waitUntil(t, "the child process to be gone", func() bool { return !processAlive(pid) })
}

func TestStartServer_RefusesASecondStart(t *testing.T) {
	useFakeIperf3(t)
	manager := iperf.NewManager()
	port := freePort(t)

	if err := manager.StartServer(port); err != nil {
		t.Fatalf("StartServer(%d): %v", port, err)
	}
	t.Cleanup(func() { _ = manager.StopServer() })

	err := manager.StartServer(freePort(t))
	if err == nil {
		t.Fatal("a second StartServer returned nil, want an error")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %v, want it to say the server is already running", err)
	}
	// The first server must be untouched by the refused second start.
	if got := manager.GetServerStatus().Port; got != port {
		t.Errorf("status.Port = %d, want the original %d", got, port)
	}
}

func TestStartServer_RejectsAnInvalidPortBeforeSpawning(t *testing.T) {
	useFakeIperf3(t)
	manager := iperf.NewManager()

	for _, port := range []int{0, -1, 70000} {
		if err := manager.StartServer(port); err == nil {
			_ = manager.StopServer()
			t.Errorf("StartServer(%d) returned nil, want a validation error", port)
		}
		if manager.GetServerStatus().Running {
			t.Errorf("status.Running = true after a rejected port %d", port)
			_ = manager.StopServer()
		}
	}
}

func TestStartServer_ReportsABinaryThatExitsImmediately(t *testing.T) {
	useFakeIperf3(t)
	t.Setenv("FAKE_IPERF3_MODE", "exit-nonzero")
	manager := iperf.NewManager()
	port := freePort(t)

	err := manager.StartServer(port)
	if err == nil {
		_ = manager.StopServer()
		t.Fatal("StartServer returned nil for a binary that exits immediately")
	}
	if manager.GetServerStatus().Running {
		t.Error("status.Running = true after a failed start")
	}
}

func TestStartServer_KillsAProcessThatNeverListens(t *testing.T) {
	// The leak this guards: the child starts fine but never binds, the
	// readiness wait times out, and the process is left running for the life of
	// the daemon.
	useFakeIperf3(t)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	t.Setenv("FAKE_IPERF3_MODE", "never-listen")
	t.Setenv("FAKE_IPERF3_PIDFILE", pidFile)
	manager := iperf.NewManager()
	port := freePort(t)

	err := manager.StartServer(port)
	if err == nil {
		_ = manager.StopServer()
		t.Fatal("StartServer returned nil though nothing ever listened")
	}
	if !strings.Contains(err.Error(), "failed to start listening") {
		t.Errorf("error = %v, want it to name the port-readiness failure", err)
	}
	if manager.GetServerStatus().Running {
		t.Error("status.Running = true after the readiness wait failed")
	}

	// Asserting the error is not enough: the child was spawned, and a failed
	// start that returns without killing it leaves a process running for the
	// life of the daemon. Read the pid the child recorded and check it is gone.
	pid := readPID(t, pidFile)
	waitUntil(t, "the abandoned child to be killed", func() bool { return !processAlive(pid) })
}

// readPID reads the pid the stand-in binary recorded for itself.
func readPID(t *testing.T, path string) int {
	t.Helper()

	var raw []byte
	waitUntil(t, "the child to record its pid", func() bool {
		content, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		raw = content

		return len(raw) > 0
	})

	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("parsing recorded pid %q: %v", raw, err)
	}

	return pid
}

func TestStopServer_OnAStoppedManager(t *testing.T) {
	manager := iperf.NewManager()

	if err := manager.StopServer(); err == nil {
		t.Fatal("StopServer on a manager that never started returned nil, want an error")
	}
}

func TestStartServer_CanRestartOnTheSamePort(t *testing.T) {
	// Restart is the operator's first move after a bad run, and it only works
	// if the previous stop actually released the port.
	useFakeIperf3(t)
	manager := iperf.NewManager()
	port := freePort(t)

	// Without this, a t.Fatalf between Start and Stop leaves the child running
	// for the life of the machine, holding the port. That is not hypothetical:
	// it happened while writing these tests and produced an intermittent
	// bind failure in a later run.
	t.Cleanup(func() { _ = manager.StopServer() })

	for attempt := 1; attempt <= 3; attempt++ {
		if err := manager.StartServer(port); err != nil {
			t.Fatalf("attempt %d: StartServer(%d): %v", attempt, port, err)
		}
		if !portAccepts(port) {
			t.Fatalf("attempt %d: nothing listening on %d", attempt, port)
		}
		if err := manager.StopServer(); err != nil {
			t.Fatalf("attempt %d: StopServer: %v", attempt, err)
		}
		waitUntil(t, "the port to be released", func() bool { return !portAccepts(port) })
	}
}
