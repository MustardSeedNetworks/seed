//go:build darwin

package wifihelper_test

import (
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

// dialedPair returns the accepted end of a connected unix socket in this
// process; the dialing end is kept open for the lifetime of the test.
//
// Test names here are deliberately short: t.TempDir() embeds the name and unix
// socket paths are capped near 104 bytes.
func dialedPair(t *testing.T) *net.UnixConn {
	t.Helper()

	path := filepath.Join(t.TempDir(), "p.sock")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	type accepted struct {
		conn *net.UnixConn
		err  error
	}
	ch := make(chan accepted, 1)
	go func() {
		c, acceptErr := l.AcceptUnix()
		ch <- accepted{c, acceptErr}
	}()

	client, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	got := <-ch
	if got.err != nil {
		t.Fatalf("accept: %v", got.err)
	}
	t.Cleanup(func() { _ = got.conn.Close() })

	return got.conn
}

// The daemon must refuse Wi-Fi data from any process that is not the helper it
// expects, so verification has to fail closed against a requirement the peer
// cannot satisfy.
func TestPeerWrongID(t *testing.T) {
	t.Parallel()

	server := dialedPair(t)

	err := wifihelper.VerifyPeer(server, `identifier "net.mustardseed.definitely-not-this-process"`)
	if !errors.Is(err, wifihelper.ErrPeerRejected) {
		t.Fatalf("VerifyPeer() error = %v, want ErrPeerRejected", err)
	}
}

func TestPeerBadReq(t *testing.T) {
	t.Parallel()

	server := dialedPair(t)

	if err := wifihelper.VerifyPeer(server, "this is not a requirement"); err == nil {
		t.Fatal("VerifyPeer() accepted a malformed requirement")
	}
}
