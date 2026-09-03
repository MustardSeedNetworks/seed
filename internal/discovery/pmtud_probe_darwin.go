//go:build darwin

package discovery

// macOS spells the don't-fragment bit IP_DONTFRAG, and has no equivalent of
// Linux's IP_PMTUDISC_PROBE -- there is no path-MTU cache to bypass, so the
// plain option is the whole of it.

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

// setDontFragment sets DF on the socket.
func setDontFragment(conn *net.IPConn) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var setErr error
	if controlErr := raw.Control(func(fd uintptr) {
		setErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_DONTFRAG, 1)
	}); controlErr != nil {
		return controlErr
	}
	return setErr
}

// oversizedLocally reports whether the kernel refused the packet because it
// exceeds the local interface MTU.
func oversizedLocally(err error) bool {
	return errors.Is(err, unix.EMSGSIZE)
}
