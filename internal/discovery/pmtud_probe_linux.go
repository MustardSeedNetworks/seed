//go:build linux

package discovery

// On Linux the don't-fragment bit is IP_MTU_DISCOVER set to IP_PMTUDISC_PROBE.
// IP_PMTUDISC_DO would also set DF, but it lets the kernel answer from its own
// path-MTU cache -- we want to measure the path, not read back what the kernel
// already believes about it.

import (
	"errors"
	"net"

	"golang.org/x/sys/unix"
)

// setDontFragment sets DF on the socket and stops the kernel short-circuiting
// the probe against its path-MTU cache.
func setDontFragment(conn *net.IPConn) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var setErr error
	if controlErr := raw.Control(func(fd uintptr) {
		setErr = unix.SetsockoptInt(
			int(fd), unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, unix.IP_PMTUDISC_PROBE,
		)
	}); controlErr != nil {
		return controlErr
	}
	return setErr
}

// oversizedLocally reports whether the kernel refused the packet because it
// exceeds the local interface MTU. The packet never reached the wire, so it
// says nothing about the path -- but for the search it means the same thing as
// a router refusing it, and treating it as an error would abandon the run.
func oversizedLocally(err error) bool {
	return errors.Is(err, unix.EMSGSIZE)
}
