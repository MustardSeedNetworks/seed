// Package carrier reports whether an interface has a physical link.
//
// Linux and Windows set IFF_RUNNING from the link's operational state, so the
// flag is the answer there. macOS sets IFF_RUNNING on an Ethernet port with no
// cable plugged in; its carrier is the media status that `ifconfig` prints as
// "status: active" or "status: inactive".
package carrier

import "net"

// Present reports whether iface has carrier.
func Present(iface net.Interface) bool {
	if active, ok := mediaActive(iface.Name); ok {
		return active
	}
	return iface.Flags&net.FlagRunning != 0
}
