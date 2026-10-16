//go:build windows

package egress

import (
	"fmt"
	"math"
	"math/bits"
	"net"

	"golang.org/x/sys/windows"
)

// The ws2ipdef.h socket options that select a socket's outgoing interface.
// x/sys/windows does not define them.
const (
	ipUnicastIf   = 31
	ipv6UnicastIf = 31
)

// bind sets IP_UNICAST_IF or IPV6_UNICAST_IF. The IPv4 option takes the
// interface index in network byte order, which on a little-endian host is the
// index byte-swapped; the IPv6 one takes it in host order.
func bind(fd uintptr, network string, link *net.Interface) error {
	h := windows.Handle(fd)
	if isIPv6(network) {
		return windows.SetsockoptInt(h, windows.IPPROTO_IPV6, ipv6UnicastIf, link.Index)
	}
	if link.Index < 0 || link.Index > math.MaxUint32 {
		return fmt.Errorf("interface index %d out of range", link.Index)
	}
	networkOrder := int(bits.ReverseBytes32(uint32(link.Index)))
	return windows.SetsockoptInt(h, windows.IPPROTO_IP, ipUnicastIf, networkOrder)
}
