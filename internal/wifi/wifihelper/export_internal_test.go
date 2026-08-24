//go:build darwin

package wifihelper

import (
	"bufio"
	"log/slog"
	"net"
)

// newServerWithConn builds a Server around an already-connected socket so the
// request/reply exchange can be tested without a code-signed peer. Peer
// verification is covered separately in TestVerifyPeerRejectsWrongIdentity.
func newServerWithConn(conn *net.UnixConn, log *slog.Logger) *Server {
	return &Server{
		requirement: "unused in tests",
		log:         log,
		conn:        conn,
		rd:          bufio.NewReader(conn),
	}
}
