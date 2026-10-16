//go:build darwin || windows

package egress

import "strings"

// isIPv6 reports whether the socket Control is called for is IPv6. Go
// passes Control the resolved family ("udp4", "udp6", "ip6:ipv6-icmp"), never
// the dual-stack name the caller dialled.
func isIPv6(network string) bool {
	return strings.HasSuffix(network, "6") || strings.HasPrefix(network, "ip6")
}
