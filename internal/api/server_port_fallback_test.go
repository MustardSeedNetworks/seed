package api_test

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

// TestBindWithFallback_FreePort confirms a free port is bound at offset 0.
func TestBindWithFallback_FreePort(t *testing.T) {
	// Grab an ephemeral port, close immediately so it's free again.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	_ = probe.Close()

	ln, bound, err := api.ExportBindWithFallback(context.Background(), "127.0.0.1", port)
	if err != nil {
		t.Fatalf("bindWithFallback returned error: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if bound != port {
		t.Fatalf("expected bound port %d, got %d", port, bound)
	}
	if !strings.HasSuffix(ln.Addr().String(), ":"+strconv.Itoa(port)) {
		t.Fatalf("listener bound to unexpected address %s", ln.Addr().String())
	}
}

// TestBindWithFallback_FallsBackOneStep grabs a port, holds it open, then
// expects bindWithFallback to land on requested+1.
func TestBindWithFallback_FallsBackOneStep(t *testing.T) {
	hold, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("hold listen: %v", err)
	}
	defer func() { _ = hold.Close() }()

	taken := hold.Addr().(*net.TCPAddr).Port
	// Ask for the same port — should fall back to taken+1.
	ln, bound, err := api.ExportBindWithFallback(context.Background(), "127.0.0.1", taken)
	if err != nil {
		t.Fatalf("bindWithFallback fell through: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if bound != taken+1 {
		t.Fatalf("expected fallback to port %d, got %d", taken+1, bound)
	}
}

// TestIsAddrInUse_RecognisesSyscall confirms isAddrInUse matches a wrapped
// EADDRINUSE.
func TestIsAddrInUse_RecognisesSyscall(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: syscall.EADDRINUSE}
	if !api.ExportIsAddrInUse(wrapped) {
		t.Fatalf("expected isAddrInUse to match EADDRINUSE")
	}
}

// TestIsAddrInUse_RejectsOtherErrors ensures unrelated errors don't match.
func TestIsAddrInUse_RejectsOtherErrors(t *testing.T) {
	if api.ExportIsAddrInUse(errors.New("some unrelated failure")) {
		t.Fatalf("expected isAddrInUse to reject unrelated error")
	}
}
