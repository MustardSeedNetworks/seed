//go:build darwin

package carrier

import (
	"syscall"
	"unsafe"
)

// siocgifmedia, ifmAvalid and ifmActive are from <sys/sockio.h> and
// <net/if_media.h>.
const (
	siocgifmedia = 0xc02c6938
	ifmAvalid    = 0x1
	ifmActive    = 0x2
)

// ifmediareq mirrors struct ifmediareq, which <net/if.h> packs to four bytes:
// the trailing ifm_ulist pointer sits at offset 36 and the struct is 44 bytes.
// No media list is requested, so the pointer is left zero.
type ifmediareq struct {
	name    [16]byte
	current int32
	mask    int32
	status  int32
	active  int32
	count   int32
	ulist   [2]uint32
}

// mediaActive reads the media status of name and reports whether the link is
// active. The second result is false when the interface has no media status
// (loopback, tunnels, bridges), and the caller falls back to IFF_RUNNING.
func mediaActive(name string) (bool, bool) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return false, false
	}
	defer func() { _ = syscall.Close(fd) }()

	var req ifmediareq
	copy(req.name[:len(req.name)-1], name)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), siocgifmedia, uintptr(unsafe.Pointer(&req)))
	if errno != 0 || req.status&ifmAvalid == 0 {
		return false, false
	}
	return req.status&ifmActive != 0, true
}
