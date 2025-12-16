// Package constants provides common string constants used throughout the codebase.
// These constants prevent typos and enable consistent string usage across packages.
package constants

// Status strings for API responses and internal state.
const (
	StatusSuccess    = "success"
	StatusError      = "error"
	StatusWarning    = "warning"
	StatusUnknown    = "unknown"
	StatusUnknownCap = "Unknown" // Capitalized variant
)

// Protocol strings for network operations.
const (
	ProtoTCP  = "tcp"
	ProtoUDP  = "udp"
	ProtoICMP = "icmp"
)

// Operating system identifiers.
const (
	OSLinux   = "linux"
	OSWindows = "windows"
)

// Device type identifiers.
const (
	DeviceTypePrinter       = "printer"
	DeviceTypeNetworkDevice = "network-device"
	DeviceTypeServer        = "server"
)

// Network interface types.
const (
	InterfaceEthernet = "ethernet"
	InterfaceWiFi     = "wifi"
	InterfaceFiber    = "fiber"
)

// Vendor identifiers.
const (
	VendorCisco    = "cisco"
	VendorCiscoIOS = "cisco-ios"
)

// Error messages.
const (
	ErrNoIPv4Address      = "no IPv4 address found for target"
	ErrTracerouteCanceled = "traceroute canceled"
)

// Traceroute hop types.
const (
	HopTypeReply = "reply"
	HopTypeError = "error"
)

// iPerf modes.
const (
	IPerfModeBidirectional = "bidirectional"
)

// Mask strings for sensitive data.
const (
	MaskSensitive = "*****"
)

// Configuration types.
const (
	ConfigTypeStatic = "static"
)

// Web paths.
const (
	PathIndexHTML = "/index.html"
)
