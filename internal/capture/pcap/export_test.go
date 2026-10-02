//go:build cgo || windows

package pcap

// PortError exposes portError to the external test package.
func PortError(err error) error { return portError(err) }
