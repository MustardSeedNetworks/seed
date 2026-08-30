//go:build iperf3

// Tests that need a genuine iperf3 binary and a genuine transfer.
//
// Behind a build tag so the default `go test ./...` never picks them up, and
// run by one CI job that installs a pinned iperf3. Nothing here skips: a
// missing binary, a version the product refuses, or a transfer that moves no
// bytes is a failure. The suite exists to prove those work, and a suite that
// quietly no-ops is the defect it replaces — the previous versions of these
// tests were gated on SKIP_IPERF_TEST (which CI set to 1) and, when they did
// run, logged the outcome instead of asserting it.
package iperf_test

import (
	"context"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// requireIperf3 fails when the environment this job exists to exercise is
// absent, rather than skipping past it.
func requireIperf3(t *testing.T) {
	t.Helper()

	if os.Getenv("SEED_IPERF3_TESTS") != "1" {
		t.Fatal("SEED_IPERF3_TESTS must be 1 for this suite; it exists so an " +
			"accidental `go test -tags iperf3` without a real iperf3 fails loudly " +
			"rather than reporting a false pass")
	}

	if err := iperf.CheckInstalled(); err != nil {
		t.Fatalf("iperf3 is not available: %v\n"+
			"this suite requires a real iperf3 — see the iperf3 job in .github/workflows/ci.yml", err)
	}
}

func TestReal_VersionIsReadableAndSupported(t *testing.T) {
	requireIperf3(t)

	version, err := iperf.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if version == "" {
		t.Fatal("GetVersion returned an empty string")
	}
	t.Logf("iperf3 version: %s", version)

	// The product refuses to run against a version below its floor, so the
	// pinned one in CI has to satisfy it. Logging this and moving on — which is
	// what the previous test did — hid a pin that no longer qualified.
	if err := iperf.ValidateVersion(); err != nil {
		t.Fatalf("ValidateVersion rejected the installed iperf3 (%s): %v", version, err)
	}
}

// freeLoopbackPort returns a port nothing is listening on.
func freeLoopbackPort(t *testing.T) int {
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

func TestReal_ServerAndClientMoveBytesOverLoopback(t *testing.T) {
	requireIperf3(t)

	manager := iperf.NewManager()
	port := freeLoopbackPort(t)

	if err := manager.StartServer(port); err != nil {
		t.Fatalf("StartServer(%d): %v", port, err)
	}
	t.Cleanup(func() { _ = manager.StopServer() })

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := manager.RunClient(ctx, &iperf.ClientConfig{
		Server:   "127.0.0.1",
		Port:     port,
		Duration: 1,
	})
	if err != nil {
		t.Fatalf("RunClient against the loopback server: %v", err)
	}
	if result == nil {
		t.Fatal("RunClient returned a nil result without an error")
	}

	// Assert the fields the API actually serves, not merely that a call
	// returned. A parser change that produced a zeroed Result would otherwise
	// pass.
	if result.BitsPerSecond <= 0 {
		t.Errorf("BitsPerSecond = %v, want a positive throughput", result.BitsPerSecond)
	}
	if result.Transfer <= 0 {
		t.Errorf("Transfer = %v MB, want a positive amount", result.Transfer)
	}
	if result.Duration <= 0 {
		t.Errorf("Duration = %v, want a positive duration", result.Duration)
	}
	if result.Port != port {
		t.Errorf("Port = %d, want %d", result.Port, port)
	}
	if result.Server != "127.0.0.1" {
		t.Errorf("Server = %q, want 127.0.0.1", result.Server)
	}
	if result.Protocol == "" {
		t.Error("Protocol is empty")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}

	status := manager.GetClientStatus()
	if status.Running {
		t.Error("client still reports Running after the run finished")
	}
}

func TestReal_ClientFailsAgainstNoServer(t *testing.T) {
	requireIperf3(t)

	manager := iperf.NewManager()
	// A port with nothing on it: reserved and released, so the connection is
	// refused rather than hanging.
	port := freeLoopbackPort(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := manager.RunClient(ctx, &iperf.ClientConfig{
		Server:   "127.0.0.1",
		Port:     port,
		Duration: 1,
	}); err == nil {
		t.Fatalf("RunClient against nothing on port %d returned nil, want an error", port)
	}

	status := manager.GetClientStatus()
	if status.Running {
		t.Error("client still reports Running after a failed run")
	}
	if status.Phase != "idle" {
		t.Errorf("Phase = %q, want idle after a failed run", status.Phase)
	}
}

func TestReal_ClientStopsOnContextCancellation(t *testing.T) {
	requireIperf3(t)

	manager := iperf.NewManager()
	port := freeLoopbackPort(t)

	if err := manager.StartServer(port); err != nil {
		t.Fatalf("StartServer(%d): %v", port, err)
	}
	t.Cleanup(func() { _ = manager.StopServer() })

	// A long run, cancelled shortly after it starts: the point is that
	// cancellation actually stops it rather than waiting out the duration.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	_, err := manager.RunClient(ctx, &iperf.ClientConfig{
		Server:   "127.0.0.1",
		Port:     port,
		Duration: 60,
	})
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("RunClient returned nil for a cancelled run")
	}
	// 60s was requested; returning anywhere near that means cancellation did
	// not reach the child process.
	if elapsed > 20*time.Second {
		t.Errorf("RunClient took %v to notice cancellation of a 60s run", elapsed)
	}
	if manager.GetClientStatus().Running {
		t.Error("client still reports Running after cancellation")
	}
}

// Deliberately not tested here: dialing the server port directly after
// StartServer returns. iperf3's server handles one control connection at a
// time, and the readiness probe inside StartServer is itself such a
// connection — a second immediate dial is intermittently refused, which made
// the assertion flaky rather than wrong-detecting. That the server is genuinely
// reachable is proven by TestReal_ServerAndClientMoveBytesOverLoopback, which
// runs a real client against it. The hermetic manager tests cover the
// readiness-wait logic itself against a stand-in that accepts freely.

func TestReal_SystemBinaryIsAbsoluteExecutableAndValid(t *testing.T) {
	requireIperf3(t)

	path, err := iperf.FindSystemIperf3()
	if err != nil {
		t.Fatalf("FindSystemIperf3: %v", err)
	}
	if path == "" {
		t.Fatal("FindSystemIperf3 returned an empty path without an error")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("path = %q, want an absolute path", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("%s is not executable (mode %v)", path, info.Mode())
	}

	// validateBinary runs the binary and checks it answers as iperf3. The
	// previous home for this assertion skipped when iperf3 was absent, so it
	// never ran in CI.
	if !iperf.ValidateBinary(path) {
		t.Errorf("ValidateBinary(%q) = false for a real iperf3", path)
	}
}

func TestReal_FindIperf3BinaryResolvesAndCaches(t *testing.T) {
	requireIperf3(t)

	original := iperf.IperfBinaryPath()
	t.Cleanup(func() { iperf.SetIperfBinaryPath(original) })
	iperf.ClearIperfBinaryPath()

	path, err := iperf.FindIperf3Binary()
	if err != nil {
		t.Fatalf("FindIperf3Binary after clearing the cache: %v", err)
	}
	if path == "" {
		t.Fatal("FindIperf3Binary returned an empty path without an error")
	}
	if got := iperf.IperfBinaryPath(); got != path {
		t.Errorf("cached path = %q, want the resolved %q", got, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("%s is not executable (mode %v)", path, info.Mode())
	}
}
