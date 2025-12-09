//go:build darwin

package gateway

import (
	"net"
	"syscall"
	"unsafe"
)

// RouteInfo contains information about a route.
type RouteInfo struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Family      string `json:"family"` // "inet" or "inet6"
}

// GetAllRoutes returns all routes using the routing socket.
func GetAllRoutes() ([]RouteInfo, error) {
	// For now, return nil - full route table parsing is complex
	// The main use case (default gateway) is handled by detectGatewayPlatform
	return nil, nil
}

// GetDefaultGatewayInterface returns the interface used for the default route.
func GetDefaultGatewayInterface() (string, error) {
	// Get the default route and look up the interface
	rib, err := syscall.RouteRIB(syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", err
	}

	msgs, err := parseRoutingMessages(rib)
	if err != nil {
		return "", err
	}

	for _, msg := range msgs {
		if isDefaultRoute(msg) && msg.ifIndex > 0 {
			iface, err := net.InterfaceByIndex(msg.ifIndex)
			if err == nil {
				return iface.Name, nil
			}
		}
	}

	return "", nil
}

// isNetlinkAvailable returns false on macOS where netlink is not available.
func isNetlinkAvailable() bool {
	return false
}

// routeMessage represents a parsed routing message.
type routeMessage struct {
	destination net.IP
	gateway     net.IP
	ifIndex     int
	family      int // syscall.AF_INET or syscall.AF_INET6
}

// detectGatewayPlatform is the platform-specific gateway detection.
// On macOS, this uses the BSD routing socket directly.
func detectGatewayPlatform() (string, error) {
	rib, err := syscall.RouteRIB(syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", err
	}

	msgs, err := parseRoutingMessages(rib)
	if err != nil {
		return "", err
	}

	// Find IPv4 default route
	for _, msg := range msgs {
		if msg.family == syscall.AF_INET && isDefaultRoute(msg) && msg.gateway != nil {
			return msg.gateway.String(), nil
		}
	}

	return "", nil
}

// detectGatewayIPv6Platform is the platform-specific IPv6 gateway detection.
func detectGatewayIPv6Platform() (string, error) {
	rib, err := syscall.RouteRIB(syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", err
	}

	msgs, err := parseRoutingMessages(rib)
	if err != nil {
		return "", err
	}

	// Find IPv6 default route
	for _, msg := range msgs {
		if msg.family == syscall.AF_INET6 && isDefaultRoute(msg) && msg.gateway != nil {
			return msg.gateway.String(), nil
		}
	}

	return "", nil
}

// isDefaultRoute checks if a route is the default route.
func isDefaultRoute(msg routeMessage) bool {
	if msg.destination == nil {
		return true // nil destination means default
	}
	// Check for 0.0.0.0 (IPv4) or :: (IPv6)
	return msg.destination.IsUnspecified()
}

// parseRoutingMessages parses the routing information base.
func parseRoutingMessages(rib []byte) ([]routeMessage, error) {
	var msgs []routeMessage

	for len(rib) > 0 {
		// Route message header
		if len(rib) < 4 {
			break
		}

		msgLen := int(nativeEndian.Uint16(rib[0:2]))
		if msgLen == 0 || msgLen > len(rib) {
			break
		}

		msgType := rib[3]

		// Only process RTM_GET type messages (route entries)
		if msgType == syscall.RTM_GET || msgType == syscall.RTM_ADD {
			msg := parseRouteMessage(rib[:msgLen])
			if msg != nil {
				msgs = append(msgs, *msg)
			}
		}

		rib = rib[msgLen:]
	}

	return msgs, nil
}

// rt_msghdr structure offsets (macOS specific)
const (
	rtMsgHdrSize = 92 // Size of rt_msghdr on macOS (64-bit)
)

// parseRouteMessage parses a single route message.
func parseRouteMessage(data []byte) *routeMessage {
	if len(data) < rtMsgHdrSize {
		return nil
	}

	// rt_msghdr fields:
	// rtm_msglen (2), rtm_version (1), rtm_type (1), rtm_index (2), ...
	// rtm_flags at offset 8 (4 bytes)
	// rtm_addrs at offset 12 (4 bytes)
	// rtm_index at offset 4 (2 bytes)

	addrs := int(nativeEndian.Uint32(data[12:16]))
	ifIndex := int(nativeEndian.Uint16(data[4:6]))

	msg := &routeMessage{
		ifIndex: ifIndex,
	}

	// Parse socket addresses after the header
	pos := rtMsgHdrSize

	// RTA_DST (1), RTA_GATEWAY (2), RTA_NETMASK (4), RTA_IFP (16), RTA_IFA (32)
	for i := 0; i < 8 && pos < len(data); i++ {
		bit := 1 << i
		if addrs&bit == 0 {
			continue
		}

		if pos >= len(data) {
			break
		}

		saLen := int(data[pos])
		if saLen == 0 {
			saLen = 4 // Minimum sockaddr length with padding
		}

		// Round up to 4-byte alignment
		saLen = (saLen + 3) &^ 3

		if pos+saLen > len(data) {
			break
		}

		saFamily := int(data[pos+1])

		switch bit {
		case 1: // RTA_DST
			msg.destination = parseSockaddr(data[pos:pos+saLen], saFamily)
			msg.family = saFamily
		case 2: // RTA_GATEWAY
			msg.gateway = parseSockaddr(data[pos:pos+saLen], saFamily)
			if msg.family == 0 {
				msg.family = saFamily
			}
		}

		pos += saLen
	}

	return msg
}

// parseSockaddr extracts an IP address from a sockaddr structure.
func parseSockaddr(data []byte, family int) net.IP {
	if len(data) < 4 {
		return nil
	}

	switch family {
	case syscall.AF_INET:
		if len(data) >= 8 {
			// sockaddr_in: family (1), len (1), port (2), addr (4)
			return net.IP(data[4:8])
		}
	case syscall.AF_INET6:
		if len(data) >= 24 {
			// sockaddr_in6: len (1), family (1), port (2), flowinfo (4), addr (16)
			return net.IP(data[8:24])
		}
	case syscall.AF_LINK:
		// Link-layer address, no IP
		return nil
	}

	return nil
}

// nativeEndian is the byte order of the current system.
var nativeEndian = getNativeEndian()

func getNativeEndian() interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
} {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = 0x0102
	if buf[0] == 0x01 {
		return bigEndian{}
	}
	return littleEndian{}
}

type littleEndian struct{}

func (littleEndian) Uint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleEndian) Uint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

type bigEndian struct{}

func (bigEndian) Uint16(b []byte) uint16 {
	return uint16(b[1]) | uint16(b[0])<<8
}

func (bigEndian) Uint32(b []byte) uint32 {
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}
