//go:build darwin

package egress

import (
	"net"

	"golang.org/x/sys/unix"
)

// bind sets IP_BOUND_IF or IPV6_BOUND_IF, macOS's per-socket outgoing
// interface.
func bind(fd uintptr, network string, link *net.Interface) error {
	if isIPv6(network) {
		return unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, link.Index)
	}
	return unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, link.Index)
}
