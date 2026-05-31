package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/probe/checkers"
)

// fakePingDialer is a programmable dialer for tests. attempts[i] is
// the (conn, err) tuple returned by the i-th Dial call.
type fakePingDialer struct {
	attempts []dialOutcome
	called   int
	gotAddrs []string
}

type dialOutcome struct {
	conn net.Conn
	err  error
}

// fakeConn is a no-op [net.Conn] used to satisfy the interface in
// tests.
type fakeConn struct{ net.Conn }

func (fakeConn) Close() error                     { return nil }
func (fakeConn) Read([]byte) (int, error)         { return 0, nil }
func (fakeConn) Write(b []byte) (int, error)      { return len(b), nil }
func (fakeConn) LocalAddr() net.Addr              { return nil }
func (fakeConn) RemoteAddr() net.Addr             { return nil }
func (fakeConn) SetDeadline(time.Time) error      { return nil }
func (fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (fakeConn) SetWriteDeadline(time.Time) error { return nil }

func (f *fakePingDialer) Dial(_ context.Context, _, addr string) (net.Conn, error) {
	f.gotAddrs = append(f.gotAddrs, addr)
	idx := f.called
	f.called++
	if idx >= len(f.attempts) {
		return nil, errors.New("no more outcomes configured")
	}
	o := f.attempts[idx]
	return o.conn, o.err
}

func TestPingChecker_Kind(t *testing.T) {
	t.Parallel()
	c := checkers.NewPingChecker()
	if c.Kind() != "ping" {
		t.Errorf("Kind() = %q, want %q", c.Kind(), "ping")
	}
}

func TestPingChecker_RequiredCapabilities(t *testing.T) {
	t.Parallel()
	c := checkers.NewPingChecker()
	if caps := c.RequiredCapabilities(); len(caps) != 0 {
		t.Errorf("RequiredCapabilities() = %v, want empty", caps)
	}
}

func TestPingChecker_Run_PrimaryPortSucceeds(t *testing.T) {
	t.Parallel()
	dialer := &fakePingDialer{attempts: []dialOutcome{{conn: fakeConn{}}}}
	c := checkers.NewPingChecker().WithPingDialer(dialer)

	r := c.Run(context.Background(), probe.Probe{Kind: "ping", Target: "google.com"})

	if !r.Success {
		t.Errorf("Result.Success = false, want true; error=%q", r.Error)
	}
	if dialer.called != 1 {
		t.Errorf("dialer.called = %d, want 1 (no fallback needed)", dialer.called)
	}
	if dialer.gotAddrs[0] != "google.com:80" {
		t.Errorf("dialer addr = %q, want google.com:80", dialer.gotAddrs[0])
	}

	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["fell_back"] != false {
		t.Errorf("metadata.fell_back = %v, want false", meta["fell_back"])
	}
}

func TestPingChecker_Run_FallsBackToHTTPS(t *testing.T) {
	t.Parallel()
	dialer := &fakePingDialer{attempts: []dialOutcome{
		{err: errors.New("connection refused")},
		{conn: fakeConn{}},
	}}
	c := checkers.NewPingChecker().WithPingDialer(dialer)

	r := c.Run(context.Background(), probe.Probe{Kind: "ping", Target: "secure-only.example.com"})

	if !r.Success {
		t.Errorf("Result.Success = false, want true after fallback")
	}
	if dialer.called != 2 {
		t.Errorf("dialer.called = %d, want 2 (primary + fallback)", dialer.called)
	}
	if dialer.gotAddrs[1] != "secure-only.example.com:443" {
		t.Errorf("fallback addr = %q, want secure-only.example.com:443", dialer.gotAddrs[1])
	}

	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["fell_back"] != true {
		t.Errorf("metadata.fell_back = %v, want true", meta["fell_back"])
	}
}

func TestPingChecker_Run_BothPortsFail(t *testing.T) {
	t.Parallel()
	dialer := &fakePingDialer{attempts: []dialOutcome{
		{err: errors.New("primary refused")},
		{err: errors.New("fallback refused")},
	}}
	c := checkers.NewPingChecker().WithPingDialer(dialer)

	r := c.Run(context.Background(), probe.Probe{Kind: "ping", Target: "down.example.com"})

	if r.Success {
		t.Error("Result.Success = true, want false")
	}
	if r.Error == "" {
		t.Error("Result.Error should describe failure")
	}
	if dialer.called != 2 {
		t.Errorf("dialer.called = %d, want 2", dialer.called)
	}
}

func TestPingChecker_Run_CustomPort_NoFallback(t *testing.T) {
	t.Parallel()
	dialer := &fakePingDialer{attempts: []dialOutcome{{err: errors.New("refused")}}}
	c := checkers.NewPingChecker().WithPingDialer(dialer)

	p := probe.Probe{
		Kind:   "ping",
		Target: "mqtt.example.com",
		Params: json.RawMessage(`{"port":"1883"}`),
	}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false")
	}
	if dialer.called != 1 {
		t.Errorf("dialer.called = %d, want 1 (operator-pinned port, no fallback)", dialer.called)
	}
	if dialer.gotAddrs[0] != "mqtt.example.com:1883" {
		t.Errorf("dialed addr = %q, want mqtt.example.com:1883", dialer.gotAddrs[0])
	}
}

func TestPingChecker_Run_CustomTimeout(t *testing.T) {
	t.Parallel()
	// Verify Params.TimeoutMs is honored by the context deadline.
	// We can't easily assert on the deadline value through the
	// dialer interface, but a 1ms timeout against a slow dialer
	// proves the context cancellation reaches the dial.
	dialer := &slowDialer{delay: 50 * time.Millisecond}
	c := checkers.NewPingChecker().WithPingDialer(dialer)

	p := probe.Probe{
		Kind:   "ping",
		Target: "slow.example.com",
		Params: json.RawMessage(`{"timeout_ms":1}`),
	}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false (timeout)")
	}
}

// slowDialer sleeps for delay before checking ctx and returning.
type slowDialer struct {
	delay time.Duration
}

func (s *slowDialer) Dial(ctx context.Context, _, _ string) (net.Conn, error) {
	select {
	case <-time.After(s.delay):
		return fakeConn{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
