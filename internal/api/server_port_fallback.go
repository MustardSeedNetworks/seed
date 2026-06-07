package api

// server_port_fallback.go provides bindWithFallback, a helper that opens a
// TCP listener on a desired port and walks +1..+9 if the canonical port
// is already in use. This keeps `seed` runnable for developers who have
// another service squatting on 8443 without changing the documented
// default port (see #69).

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// portFallbackMaxOffset is the maximum offset above the requested port that
// bindWithFallback will probe. Probes are requested..requested+portFallbackMaxOffset.
const portFallbackMaxOffset = 9

// bindWithFallback opens a TCP listener on host:port. If that port is in
// use it walks ports+1..+portFallbackMaxOffset and returns the first
// listener that binds, logging a WARN with the requested and actual port.
//
// Non-EADDRINUSE errors are returned immediately — the caller must treat
// them as fatal (permission denied, invalid address, etc.).
//
// The caller is responsible for closing the returned listener (typically
// by passing it to [http.Server.Serve] / [http.Server.ServeTLS], which
// close on shutdown).
func bindWithFallback(ctx context.Context, host string, port int) (net.Listener, int, error) {
	var lc net.ListenConfig
	for offset := 0; offset <= portFallbackMaxOffset; offset++ {
		actual := port + offset
		addr := net.JoinHostPort(host, strconv.Itoa(actual))
		ln, err := lc.Listen(ctx, "tcp", addr)
		if err == nil {
			if offset > 0 {
				logging.WarnContext(
					ctx,
					"requested port is in use, bound fallback port instead",
					"requested", port,
					"bound", actual,
				)
			}
			return ln, actual, nil
		}
		if !isAddrInUse(err) {
			return nil, 0, fmt.Errorf("bind %s: %w", addr, err)
		}
	}
	return nil, 0, fmt.Errorf(
		"bind %s:%d and +1..+%d all in use",
		host, port, portFallbackMaxOffset,
	)
}

// isAddrInUse reports whether err indicates the address-in-use condition.
// It checks [syscall.EADDRINUSE] via [errors.Is] (works on Linux/macOS) and
// falls back to a string match for platforms whose listener wrapping does
// not unwrap to the syscall errno.
func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		return containsAddrInUse(opErr.Err.Error())
	}
	return false
}

// containsAddrInUse looks for the canonical address-in-use substring.
// Split out so it can be unit-tested independently of platform errno
// behaviour.
func containsAddrInUse(msg string) bool {
	const needle = "address already in use"
	if len(msg) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(msg); i++ {
		if msg[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
