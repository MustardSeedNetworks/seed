//go:build darwin

package gateway

import (
	"net"
	"syscall"

	"golang.org/x/net/route"
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
	rib, err := route.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return nil, err
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return nil, err
	}

	var routes []RouteInfo
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		ri := RouteInfo{}

		// Get destination
		if len(rm.Addrs) > syscall.RTAX_DST {
			if dst := rm.Addrs[syscall.RTAX_DST]; dst != nil {
				switch a := dst.(type) {
				case *route.Inet4Addr:
					ri.Destination = net.IP(a.IP[:]).String()
					ri.Family = "inet"
				case *route.Inet6Addr:
					ri.Destination = net.IP(a.IP[:]).String()
					ri.Family = "inet6"
				}
			}
		}

		// Get gateway
		if len(rm.Addrs) > syscall.RTAX_GATEWAY {
			if gw := rm.Addrs[syscall.RTAX_GATEWAY]; gw != nil {
				switch a := gw.(type) {
				case *route.Inet4Addr:
					ri.Gateway = net.IP(a.IP[:]).String()
				case *route.Inet6Addr:
					ri.Gateway = net.IP(a.IP[:]).String()
				}
			}
		}

		// Get interface
		if rm.Index > 0 {
			if iface, err := net.InterfaceByIndex(rm.Index); err == nil {
				ri.Interface = iface.Name
			}
		}

		if ri.Destination != "" || ri.Gateway != "" {
			routes = append(routes, ri)
		}
	}

	return routes, nil
}

// GetDefaultGatewayInterface returns the interface used for the default route.
func GetDefaultGatewayInterface() (string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", err
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", err
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		if isDefaultRouteMsg(rm, syscall.AF_INET) && rm.Index > 0 {
			if iface, err := net.InterfaceByIndex(rm.Index); err == nil {
				return iface.Name, nil
			}
		}
	}

	return "", nil
}

// detectGatewayPlatform is the platform-specific gateway detection.
// On macOS, this uses golang.org/x/net/route for reliable parsing.
func detectGatewayPlatform() (string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", err
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", err
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		if !isDefaultRouteMsg(rm, syscall.AF_INET) {
			continue
		}

		// Get gateway address
		if len(rm.Addrs) > syscall.RTAX_GATEWAY {
			if gw := rm.Addrs[syscall.RTAX_GATEWAY]; gw != nil {
				if a, ok := gw.(*route.Inet4Addr); ok {
					return net.IP(a.IP[:]).String(), nil
				}
			}
		}
	}

	return "", nil
}

// detectGatewayIPv6Platform is the platform-specific IPv6 gateway detection.
func detectGatewayIPv6Platform() (string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET6, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", err
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", err
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		if !isDefaultRouteMsg(rm, syscall.AF_INET6) {
			continue
		}

		// Get gateway address
		if len(rm.Addrs) > syscall.RTAX_GATEWAY {
			if gw := rm.Addrs[syscall.RTAX_GATEWAY]; gw != nil {
				if a, ok := gw.(*route.Inet6Addr); ok {
					ip := net.IP(a.IP[:])
					// Skip link-local addresses if not the only option
					if !ip.IsLinkLocalUnicast() {
						return ip.String(), nil
					}
				}
			}
		}
	}

	// Second pass: accept link-local addresses
	msgs, err = route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", nil
	}
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		if !isDefaultRouteMsg(rm, syscall.AF_INET6) {
			continue
		}

		if len(rm.Addrs) > syscall.RTAX_GATEWAY {
			if gw := rm.Addrs[syscall.RTAX_GATEWAY]; gw != nil {
				if a, ok := gw.(*route.Inet6Addr); ok {
					return net.IP(a.IP[:]).String(), nil
				}
			}
		}
	}

	return "", nil
}

// isDefaultRouteMsg checks if a route message represents the default route.
func isDefaultRouteMsg(rm *route.RouteMessage, family int) bool {
	if len(rm.Addrs) <= syscall.RTAX_DST {
		return false
	}

	dst := rm.Addrs[syscall.RTAX_DST]
	if dst == nil {
		// No destination means default route
		return true
	}

	switch family {
	case syscall.AF_INET:
		if a, ok := dst.(*route.Inet4Addr); ok {
			// Check for 0.0.0.0
			return a.IP == [4]byte{0, 0, 0, 0}
		}
	case syscall.AF_INET6:
		if a, ok := dst.(*route.Inet6Addr); ok {
			// Check for ::
			return a.IP == [16]byte{}
		}
	}

	return false
}
