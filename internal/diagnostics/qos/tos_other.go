//go:build !linux && !darwin

package qos

import (
	"errors"
	"net"
)

// errTOSUnsupported marks a platform whose sockets cannot report the marking
// a datagram arrived with. The listen still counts probes; the result says
// it could not read them.
var errTOSUnsupported = errors.New("reading a datagram's TOS is not supported on this platform")

func enableTOS(*net.UDPConn, bool) error { return errTOSUnsupported }

func parseTOS([]byte, bool) (int, bool) { return 0, false }
