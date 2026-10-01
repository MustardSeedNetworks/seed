package api

import (
	"context"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"
)

// pingerReceivers counts live ICMP receiver goroutines. Each one holds an open
// ICMP socket for as long as it runs.
func pingerReceivers() int {
	buf := make([]byte, 1<<22)
	return strings.Count(string(buf[:runtime.Stack(buf, true)]), "enumerate.(*ICMPPinger).receiver")
}

// waitForPingerReceivers fails the test unless the receiver count falls back
// to want: a receiver exits once its socket closes, which is not instant.
func waitForPingerReceivers(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for pingerReceivers() > want {
		if time.Now().After(deadline) {
			t.Fatalf("%d ICMP receiver goroutine(s) still running", pingerReceivers()-want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// The gateway tester opens an ICMP socket when the server is built, so closing
// the server must close it: before this, every server left its socket and
// receiver goroutine behind, the live goroutine in seed#2719's teardown dump.
// Not parallel, so no other test's pinger moves the count.
func TestServerCloseStopsTheGatewayPinger(t *testing.T) {
	before := pingerReceivers()
	s := NewTestServer()
	if pingerReceivers() == before {
		s.Close()
		t.Skip("this host refuses both ICMP sockets, so the tester opened no pinger")
	}

	s.Close()
	waitForPingerReceivers(t, before)
}

// The production path: Shutdown's teardown closes the gateway tester too.
func TestShutdownStopsTheGatewayPinger(t *testing.T) {
	before := pingerReceivers()
	s := NewTestServer()
	t.Cleanup(s.Close)
	if pingerReceivers() == before {
		t.Skip("this host refuses both ICMP sockets, so the tester opened no pinger")
	}
	serveOnLoopback(t, s, http.NewServeMux())

	ctx, cancel := context.WithTimeout(context.Background(), shutdownDrainTimeout)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	waitForPingerReceivers(t, before)
}
