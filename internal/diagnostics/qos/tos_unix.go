//go:build linux || darwin

package qos

import (
	"encoding/binary"
	"math"
	"net"

	"golang.org/x/sys/unix"
)

// enableTOS asks the kernel to attach each datagram's TOS byte (IPv4) or
// traffic class (IPv6) as a control message.
func enableTOS(conn *net.UDPConn, v6 bool) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var sockErr error
	err = raw.Control(func(fd uintptr) {
		if v6 {
			sockErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_RECVTCLASS, 1)
			return
		}
		sockErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_RECVTOS, 1)
	})
	if err != nil {
		return err
	}
	return sockErr
}

// parseTOS finds the TOS byte or traffic class among a datagram's control
// messages. Linux labels the IPv4 message IP_TOS and macOS IP_RECVTOS, each
// one byte; the IPv6 traffic class is a host-order int on both.
func parseTOS(oob []byte, v6 bool) (int, bool) {
	msgs, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return 0, false
	}
	for _, m := range msgs {
		switch {
		case v6 && m.Header.Level == unix.IPPROTO_IPV6 && m.Header.Type == unix.IPV6_TCLASS && len(m.Data) >= 4:
			return int(binary.NativeEndian.Uint32(m.Data) & math.MaxUint8), true
		case !v6 && m.Header.Level == unix.IPPROTO_IP &&
			(m.Header.Type == unix.IP_TOS || m.Header.Type == unix.IP_RECVTOS) && len(m.Data) >= 1:
			return int(m.Data[0]), true
		}
	}
	return 0, false
}
