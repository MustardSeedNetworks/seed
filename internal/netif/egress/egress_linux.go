//go:build linux

package egress

import (
	"net"

	"golang.org/x/sys/unix"
)

// bind sets SO_BINDTODEVICE, which needs no privilege since Linux 5.7.
func bind(fd uintptr, _ string, link *net.Interface) error {
	return unix.BindToDevice(int(fd), link.Name)
}
