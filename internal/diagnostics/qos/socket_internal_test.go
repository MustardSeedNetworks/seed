package qos

import (
	"context"
	"net"
	"runtime"
	"testing"
	"time"
)

// freeUDPPort asks the kernel for a port nothing on this host is bound to.
func freeUDPPort(t *testing.T, network string) int {
	t.Helper()
	c, err := net.ListenUDP(network, &net.UDPAddr{})
	if err != nil {
		t.Skipf("ListenUDP %s: %v", network, err)
	}
	defer func() { _ = c.Close() }()
	addr, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("LocalAddr is %T", c.LocalAddr())
	}
	return addr.Port
}

// sendUntilHeard repeats a send until the listen closes, so the assertion
// never depends on guessing when the listen bound its port.
func sendUntilHeard(t *testing.T, req SendRequest, done <-chan struct{}) {
	t.Helper()
	for {
		if _, err := Send(t.Context(), req); err != nil {
			t.Errorf("Send: %v", err)
			return
		}
		select {
		case <-done:
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// End to end over loopback, which rewrites nothing: every class must arrive
// preserved, which is only true if Send marked each datagram with its own
// class and Listen read the marking off the socket.
func TestSendAndListenOverLoopback(t *testing.T) {
	for _, tc := range []struct{ family, network, target string }{
		{"ipv4", "udp4", "127.0.0.1"},
		{"ipv6", "udp6", "::1"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			port := freeUDPPort(t, tc.network)
			classes := []int{46, 34, 0}
			done := make(chan struct{})
			sent := make(chan struct{})
			go func() {
				defer close(sent)
				sendUntilHeard(t, SendRequest{Target: tc.target, Port: port, DSCP: classes, Count: 3}, done)
			}()

			res, err := Listen(t.Context(), ListenRequest{Port: port, Family: tc.family, DurationSeconds: 1})
			close(done)
			<-sent
			if err != nil {
				t.Fatalf("Listen: %v", err)
			}
			if len(res.Runs) == 0 {
				t.Fatalf("heard no run on %s port %d: %+v", tc.target, port, res)
			}

			checkLoopbackRuns(t, res, len(classes))
		})
	}
}

// checkLoopbackRuns asserts every complete run arrived as sent. Windows cannot
// read the marking, so there the result must say so rather than claim the
// classes were preserved.
func checkLoopbackRuns(t *testing.T, res *ListenResult, classes int) {
	t.Helper()
	observes := runtime.GOOS != "windows"
	if res.DSCPObserved != observes {
		t.Fatalf("DSCPObserved = %v on %s", res.DSCPObserved, runtime.GOOS)
	}
	want := VerdictPreserved
	if !observes {
		want = VerdictUnobserved
	}
	complete := 0
	for _, run := range res.Runs {
		if len(run.Classes) != classes {
			t.Fatalf("run %+v has %d classes, sent %d", run, len(run.Classes), classes)
		}
		if run.Classes[0].Received < run.Classes[0].Expected {
			continue // the listen closed mid-burst
		}
		complete++
		for _, c := range run.Classes {
			if c.Verdict != want {
				t.Fatalf("class %d arrived %+v over loopback, want %s", c.SentDSCP, c, want)
			}
		}
	}
	if complete == 0 {
		t.Fatalf("no complete run in %+v", res.Runs)
	}
}

// A probe that beats the far side's listen draws an ICMP port-unreachable.
// Linux and macOS hand that back to a connected UDP socket as the next
// write's error, so the burst must not ride one: the operator starting the
// sender a moment early would otherwise lose the run.
func TestSendToAPortNobodyListensOnStillSends(t *testing.T) {
	t.Parallel()

	res, err := Send(t.Context(), SendRequest{Target: "127.0.0.1", Port: freeUDPPort(t, "udp4"), Count: 3})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	for _, c := range res.Classes {
		if c.Sent != 3 {
			t.Fatalf("class %+v: the burst stopped early", c)
		}
	}
}

func TestListenRefusesAPortInUse(t *testing.T) {
	t.Parallel()

	c, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	defer func() { _ = c.Close() }()
	addr, _ := c.LocalAddr().(*net.UDPAddr)
	if _, err = Listen(t.Context(), ListenRequest{Port: addr.Port, DurationSeconds: 1}); err == nil {
		t.Fatal("Listen on a bound port succeeded")
	}
}

// A cancel must end a blocked read at once, not at the end of the window: the
// operator pressed stop, and the port should close with it.
func TestCancelEndsABlockedListenPromptly(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	res, err := Listen(ctx, ListenRequest{Port: freeUDPPort(t, "udp4"), DurationSeconds: 30})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("cancelled listen took %s", elapsed)
	}
	if res.ListenedMs >= 5000 || res.Probes != 0 || len(res.Runs) != 0 || res.Runs == nil {
		t.Fatalf("result %+v on a silent port", res)
	}
}
