package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// opcuaFakeConn delivers canned response bytes and records the bytes
// the checker wrote (to verify the Hello prefix was sent).
type opcuaFakeConn struct {
	mu       sync.Mutex
	response []byte
	written  []byte
	pos      int
	readErr  error
}

func (o *opcuaFakeConn) Read(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.readErr != nil {
		return 0, o.readErr
	}
	if o.pos >= len(o.response) {
		return 0, errors.New("opcuaFakeConn: response exhausted")
	}
	n := copy(b, o.response[o.pos:])
	o.pos += n
	return n, nil
}

func (o *opcuaFakeConn) Write(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.written = append(o.written, b...)
	return len(b), nil
}

func (*opcuaFakeConn) Close() error                     { return nil }
func (*opcuaFakeConn) LocalAddr() net.Addr              { return nil }
func (*opcuaFakeConn) RemoteAddr() net.Addr             { return nil }
func (*opcuaFakeConn) SetDeadline(time.Time) error      { return nil }
func (*opcuaFakeConn) SetReadDeadline(time.Time) error  { return nil }
func (*opcuaFakeConn) SetWriteDeadline(time.Time) error { return nil }

type opcuaDialer struct {
	conn net.Conn
	err  error
}

func (d *opcuaDialer) Dial(_ context.Context, _, _ string) (net.Conn, error) {
	return d.conn, d.err
}

// makeOPCUAResponse builds a padded response starting with the given
// 3-byte message type, long enough to satisfy opcuaMinResponseLen.
func makeOPCUAResponse(msgType string) []byte {
	resp := make([]byte, 8)
	copy(resp, msgType)
	return resp
}

// --- Tests ---

func TestOPCUAChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewOPCUAChecker().Kind() != "opcua" {
		t.Errorf("Kind() != opcua")
	}
}

func TestOPCUAChecker_Run_SuccessACK(t *testing.T) {
	t.Parallel()

	conn := &opcuaFakeConn{response: makeOPCUAResponse("ACK")}
	c := checkers.NewOPCUAChecker().WithOPCUADialer(&opcuaDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "opcua",
		Target: "opc.tcp://plc.example.com:4840/server",
	})

	if !r.Success {
		t.Fatalf("Success = false on ACK response: %s", r.Error)
	}

	// Verify the checker wrote the 4-byte Hello prefix.
	conn.mu.Lock()
	written := conn.written
	conn.mu.Unlock()
	if len(written) < 4 || string(written[:4]) != "HELF" {
		t.Errorf("expected HELF written; got %q", written)
	}

	// Verify metadata contains acknowledged server_info.
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	serverInfo, _ := meta["server_info"].(string)
	if serverInfo == "" || serverInfo == "TCP connection successful, server may require full handshake" {
		t.Errorf("server_info %q should mention acknowledged; got %q", serverInfo, serverInfo)
	}
	if mt, ok := meta["msg_type"].(string); !ok || mt != "ACK" {
		t.Errorf("msg_type = %v, want ACK", meta["msg_type"])
	}
}

func TestOPCUAChecker_Run_SuccessERR(t *testing.T) {
	t.Parallel()

	// ERR response = still a TCP-level success (auth may be required).
	conn := &opcuaFakeConn{response: makeOPCUAResponse("ERR")}
	c := checkers.NewOPCUAChecker().WithOPCUADialer(&opcuaDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "opcua",
		Target: "opc.tcp://plc.example.com:4840/server",
	})

	if !r.Success {
		t.Fatalf("Success = false on ERR response: %s", r.Error)
	}
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	if mt, ok := meta["msg_type"].(string); !ok || mt != "ERR" {
		t.Errorf("msg_type = %v, want ERR", meta["msg_type"])
	}
}

func TestOPCUAChecker_Run_SuccessReadError(t *testing.T) {
	t.Parallel()

	// Read returns an error (e.g. timeout or connection close). The TCP
	// connect + Hello write succeeded, so Success must still be true.
	conn := &opcuaFakeConn{readErr: errors.New("i/o timeout")}
	c := checkers.NewOPCUAChecker().WithOPCUADialer(&opcuaDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "opcua",
		Target: "opc.tcp://192.0.2.1:4840",
	})

	if !r.Success {
		t.Fatalf("Success = false on read error; TCP-only path should still succeed: %s", r.Error)
	}
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	// msg_type should be absent when there was no readable response.
	if _, present := meta["msg_type"]; present {
		t.Errorf("msg_type present when no response was read; want absent")
	}
}

func TestOPCUAChecker_Run_DialError(t *testing.T) {
	t.Parallel()

	c := checkers.NewOPCUAChecker().WithOPCUADialer(&opcuaDialer{err: errors.New("connection refused")})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "opcua",
		Target: "opc.tcp://down.example.com:4840",
	})

	if r.Success {
		t.Error("Success = true; dial error should fail the probe")
	}
	if r.Error == "" {
		t.Error("expected a non-empty Error on dial failure")
	}
}

func TestOPCUAChecker_Run_MalformedTarget(t *testing.T) {
	t.Parallel()

	// A colon-only target breaks url.Parse Hostname/Port extraction.
	// The checker should fail cleanly without panicking.
	c := checkers.NewOPCUAChecker().WithOPCUADialer(&opcuaDialer{conn: &opcuaFakeConn{}})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "opcua",
		Target: "://\x00invalid",
	})

	if r.Success {
		t.Error("Success = true on malformed URL; want failure")
	}
	if r.Error == "" {
		t.Error("expected a non-empty Error for malformed URL")
	}
}

func TestOPCUAChecker_Run_DefaultPort(t *testing.T) {
	t.Parallel()

	// A target without an explicit port should dial on 4840.
	var dialedAddr string
	fakeDialer := &opcuaDialerFunc{fn: func(_ context.Context, _, addr string) (net.Conn, error) {
		dialedAddr = addr
		return &opcuaFakeConn{readErr: errors.New("timeout")}, nil
	}}

	c := checkers.NewOPCUAChecker().WithOPCUADialer(fakeDialer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "opcua",
		Target: "opc.tcp://plc.factory.local/sensor",
	})

	if !r.Success {
		t.Fatalf("Success = false: %s", r.Error)
	}
	if dialedAddr != "plc.factory.local:4840" {
		t.Errorf("dialed %q; want plc.factory.local:4840", dialedAddr)
	}
}

// opcuaDialerFunc allows an inline func as a PingDialer.
type opcuaDialerFunc struct {
	fn func(ctx context.Context, network, addr string) (net.Conn, error)
}

func (d *opcuaDialerFunc) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.fn(ctx, network, addr)
}
