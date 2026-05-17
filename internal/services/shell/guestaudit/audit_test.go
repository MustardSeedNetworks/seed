package guestaudit_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/services/shell/guestaudit"
)

func TestRun_RejectsEmptyTargets(t *testing.T) {
	_, err := guestaudit.Run(context.Background(), guestaudit.Options{})
	if err == nil {
		t.Fatal("expected error for empty target list")
	}
}

func TestRun_RejectsInvalidIP(t *testing.T) {
	_, err := guestaudit.Run(context.Background(), guestaudit.Options{
		Targets: []guestaudit.Target{{IP: "not-an-ip"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

// TestRun_ReachableLoopback exercises the happy-path detection logic by
// pointing the audit at a TCP listener bound to 127.0.0.1. The listener is
// hit on its actual port, so the report should flag isolation as failed and
// list that port as open.
func TestRun_ReachableLoopback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind loopback listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	report, err := guestaudit.Run(context.Background(), guestaudit.Options{
		Targets:    []guestaudit.Target{{IP: "127.0.0.1", Label: "lo"}},
		Ports:      []int{port},
		TCPTimeout: 500 * time.Millisecond,
		Workers:    2,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !report.IsolationFailed {
		t.Fatalf("expected isolation to fail for loopback listener, got report: %+v", report)
	}
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("expected %d results, got %d", want, got)
	}
	if !report.Results[0].Reachable {
		t.Fatal("expected loopback target marked reachable")
	}
	if len(report.Results[0].OpenPorts) != 1 || report.Results[0].OpenPorts[0] != port {
		t.Fatalf("expected open port %d to be reported, got %v", port, report.Results[0].OpenPorts)
	}
}

// TestRun_UnreachableUnusedPort confirms a closed-port probe doesn't trigger
// a false positive. We bind+release a port to get a value that's almost
// certainly unused, then probe it.
func TestRun_UnreachableUnusedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close() // release so the port is closed when we probe.

	report, err := guestaudit.Run(context.Background(), guestaudit.Options{
		Targets:    []guestaudit.Target{{IP: "127.0.0.1"}},
		Ports:      []int{port},
		TCPTimeout: 300 * time.Millisecond,
		Workers:    1,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// Loopback ICMP almost always succeeds on dev hosts, so we don't assert
	// IsolationFailed=false here. We only assert that the closed-port probe
	// itself reported open=false.
	if len(report.Results) != 1 || len(report.Results[0].PortResults) != 1 {
		t.Fatalf("expected 1 result with 1 port probe, got %+v", report)
	}
	if report.Results[0].PortResults[0].Open {
		t.Fatalf("expected closed port to report open=false; got %+v", report.Results[0].PortResults[0])
	}
}
