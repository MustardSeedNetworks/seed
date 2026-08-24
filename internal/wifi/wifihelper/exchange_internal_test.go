//go:build darwin

package wifihelper

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"path/filepath"
	"testing"
)

// Test names here are deliberately short: t.TempDir() embeds the name and unix
// socket paths are capped near 104 bytes.
func connectedPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()

	addr := &net.UnixAddr{Name: filepath.Join(t.TempDir(), "p.sock"), Net: "unix"}
	l, err := net.ListenUnix("unix", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()

	type accepted struct {
		conn *net.UnixConn
		err  error
	}
	ch := make(chan accepted, 1)
	go func() {
		c, acceptErr := l.AcceptUnix()
		ch <- accepted{c, acceptErr}
	}()

	client, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	got := <-ch
	if got.err != nil {
		t.Fatalf("accept: %v", got.err)
	}
	t.Cleanup(func() { _ = got.conn.Close() })

	return got.conn, client
}

// replyOnce stands in for the helper: it reads one request and writes the reply
// the test wants, so the daemon's side of the exchange can be driven directly.
func replyOnce(t *testing.T, conn *net.UnixConn, reply Response) chan Op {
	t.Helper()

	seen := make(chan Op, 1)
	go func() {
		line, err := bufio.NewReader(conn).ReadBytes('\n')
		if err != nil {
			close(seen)
			return
		}
		req, decodeErr := DecodeRequest(line)
		if decodeErr != nil {
			close(seen)
			return
		}
		seen <- req.Op

		payload, _ := json.Marshal(reply)
		_, _ = conn.Write(append(payload, '\n'))
	}()
	return seen
}

func TestScanTrip(t *testing.T) {
	t.Parallel()

	server, client := connectedPair(t)
	seen := replyOnce(t, client, Response{Networks: []Network{{
		SSID: "Neuroplasticity", BSSID: "24:5a:4c:6b:b5:c9", RSSI: -54, Channel: 149, Band: 5,
	}}})

	s := newServerWithConn(server, slog.New(slog.DiscardHandler))

	networks, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() unexpected error: %v", err)
	}
	if op := <-seen; op != OpScan {
		t.Errorf("helper saw op %q, want %q", op, OpScan)
	}
	if len(networks) != 1 || networks[0].BSSID != "24:5a:4c:6b:b5:c9" {
		t.Fatalf("Scan() = %+v, want the helper's network", networks)
	}
}

// A helper that cannot see network identifiers must surface that reason to the
// daemon, not report an empty airspace.
func TestScanErrProp(t *testing.T) {
	t.Parallel()

	server, client := connectedPair(t)
	replyOnce(t, client, Response{Error: "corewlan: Location Services authorization required"})

	s := newServerWithConn(server, slog.New(slog.DiscardHandler))

	_, err := s.Scan()
	if err == nil {
		t.Fatal("Scan() succeeded despite the helper reporting a failure")
	}
	if err.Error() != "corewlan: Location Services authorization required" {
		t.Errorf("Scan() error = %q, want the helper's reason", err)
	}
}

func TestCurNotAssoc(t *testing.T) {
	t.Parallel()

	server, client := connectedPair(t)
	replyOnce(t, client, Response{})

	s := newServerWithConn(server, slog.New(slog.DiscardHandler))

	if _, err := s.Current(); !errors.Is(err, ErrNotAssociated) {
		t.Fatalf("Current() error = %v, want ErrNotAssociated", err)
	}
}

// With no helper connected the daemon must say so, which is different from a
// scan that legitimately found nothing.
func TestNoHelper(t *testing.T) {
	t.Parallel()

	s := &Server{log: slog.New(slog.DiscardHandler)}

	if _, err := s.Scan(); !errors.Is(err, ErrNoHelper) {
		t.Fatalf("Scan() error = %v, want ErrNoHelper", err)
	}
}
