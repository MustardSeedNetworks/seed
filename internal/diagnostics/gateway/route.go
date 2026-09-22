package gateway

import "net"

// RouteInfo is one row of the host's forwarding table. Every field means the
// same thing whichever platform's table it was read from, which is what makes
// it usable for "which networks are routed via this interface" (seed#2765):
//
//   - Destination is the masked network address on its own — never a CIDR and
//     never the word "default" — and Prefix is its length in bits. A default
//     route is 0.0.0.0/0 or ::/0; a host route is a /32 or /128. Reading a
//     missing mask as /0 is the confusion this shape exists to prevent.
//   - Interface is the interface *name* on all three platforms, so a caller
//     can match it against net.Interface.Name.
type RouteInfo struct {
	Destination string `json:"destination"`
	Prefix      int    `json:"prefix"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Family      string `json:"family"` // "inet" or "inet6"
}

// prefixFromMask counts the leading ones of a 4- or 16-byte netmask. The
// second return is false for a mask that is not contiguous, which is not a
// prefix length at all and must not be reported as one.
func prefixFromMask(mask []byte) (int, bool) {
	ones, bits := net.IPMask(mask).Size()
	if bits == 0 {
		return 0, false
	}
	return ones, true
}
