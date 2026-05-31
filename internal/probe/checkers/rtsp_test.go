package checkers_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/probe/checkers"
)

// scriptedConn delivers canned bytes on Read; ignores Write.
type scriptedConn struct {
	mu       sync.Mutex
	response string
	written  []byte
	pos      int
}

func (s *scriptedConn) Read(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pos >= len(s.response) {
		return 0, errors.New("scriptedConn: response exhausted")
	}
	n := copy(b, s.response[s.pos:])
	s.pos += n
	return n, nil
}

func (s *scriptedConn) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.written = append(s.written, b...)
	return len(b), nil
}

func (*scriptedConn) Close() error                     { return nil }
func (*scriptedConn) LocalAddr() net.Addr              { return nil }
func (*scriptedConn) RemoteAddr() net.Addr             { return nil }
func (*scriptedConn) SetDeadline(time.Time) error      { return nil }
func (*scriptedConn) SetReadDeadline(time.Time) error  { return nil }
func (*scriptedConn) SetWriteDeadline(time.Time) error { return nil }

type rtspDialer struct {
	conn net.Conn
	err  error
}

func (r *rtspDialer) Dial(_ context.Context, _, _ string) (net.Conn, error) {
	return r.conn, r.err
}

func TestRTSPChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewRTSPChecker().Kind() != "rtsp" {
		t.Errorf("Kind != rtsp")
	}
}

func TestRTSPChecker_Run_200OK(t *testing.T) {
	t.Parallel()
	conn := &scriptedConn{response: "RTSP/1.0 200 OK\r\nCSeq: 1\r\n\r\n"}
	c := checkers.NewRTSPChecker().WithRTSPDialer(&rtspDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{Kind: "rtsp", Target: "camera.example.com"})
	if !r.Success {
		t.Errorf("Success = false; want true: %s", r.Error)
	}
	if !strings.Contains(string(conn.written), "OPTIONS") {
		t.Errorf("written request lacks OPTIONS: %q", conn.written)
	}
}

func TestRTSPChecker_Run_4xx(t *testing.T) {
	t.Parallel()
	conn := &scriptedConn{response: "RTSP/1.0 401 Unauthorized\r\n\r\n"}
	c := checkers.NewRTSPChecker().WithRTSPDialer(&rtspDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{Kind: "rtsp", Target: "camera.example.com"})
	if r.Success {
		t.Error("Success = true; want false on 401")
	}
}

func TestRTSPChecker_Run_DialError(t *testing.T) {
	t.Parallel()
	c := checkers.NewRTSPChecker().WithRTSPDialer(&rtspDialer{err: errors.New("refused")})
	r := c.Run(context.Background(), probe.Probe{Kind: "rtsp", Target: "down"})
	if r.Success {
		t.Error("Success = true; dial error should fail")
	}
}

func TestRTSPChecker_Run_TargetAsFullURL(t *testing.T) {
	t.Parallel()
	conn := &scriptedConn{response: "RTSP/1.0 200 OK\r\n\r\n"}
	c := checkers.NewRTSPChecker().WithRTSPDialer(&rtspDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "rtsp",
		Target: "rtsp://camera.example.com/stream1",
	})
	if !r.Success {
		t.Errorf("Success = false: %s", r.Error)
	}
	if !strings.Contains(string(conn.written), "/stream1") {
		t.Errorf("written request lacks path /stream1: %q", conn.written)
	}
}
