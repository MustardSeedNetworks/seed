// Package egress pins a socket's traffic to one network interface, so a check
// made "on" an interface leaves through it rather than through whichever link
// carries the host's route to the destination.
//
// Binding a socket to the interface's address is not enough on Linux or
// macOS: both use the weak host model, and a packet sourced from one link's
// address still leaves through the route's link. Each platform has its own
// option that selects the outgoing interface itself.
package egress

import (
	"fmt"
	"net"
	"syscall"
)

// Control returns a [net.Dialer] or [net.ListenConfig] Control function that
// binds the socket to iface. With iface empty it returns nil, which leaves the
// socket on host routing.
func Control(iface string) func(network, address string, c syscall.RawConn) error {
	if iface == "" {
		return nil
	}
	return func(network, _ string, c syscall.RawConn) error {
		link, err := net.InterfaceByName(iface)
		if err != nil {
			return fmt.Errorf("bind to %s: %w", iface, err)
		}
		var bindErr error
		if ctlErr := c.Control(func(fd uintptr) { bindErr = bind(fd, network, link) }); ctlErr != nil {
			return fmt.Errorf("bind to %s: %w", iface, ctlErr)
		}
		if bindErr != nil {
			return fmt.Errorf("bind to %s: %w", iface, bindErr)
		}
		return nil
	}
}
