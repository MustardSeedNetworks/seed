//go:build darwin

package wifihelper

/*
#cgo CFLAGS: -fobjc-arc -Wall
#cgo LDFLAGS: -framework Foundation -framework Security -lbsm
#include <stdlib.h>
#include "peer_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// ErrPeerRejected means the process on the other end of the socket does not
// satisfy the required code signature.
var ErrPeerRejected = errors.New("wifihelper: peer rejected")

// VerifyPeer checks that the process connected on conn satisfies a code-signing
// requirement, so the daemon accepts Wi-Fi data only from the helper it expects
// and not from any other process running as the same user.
func VerifyPeer(conn *net.UnixConn, requirement string) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("wifihelper: peer socket: %w", err)
	}

	cReq := C.CString(requirement)
	defer C.free(unsafe.Pointer(cReq))

	var verifyErr error
	ctrlErr := raw.Control(func(fd uintptr) {
		out := C.peer_verify(C.int(fd), cReq)
		if out == nil {
			return
		}
		defer C.peer_free(out)
		verifyErr = fmt.Errorf("%w: %s", ErrPeerRejected, C.GoString(out))
	})
	if ctrlErr != nil {
		return fmt.Errorf("wifihelper: peer socket control: %w", ctrlErr)
	}
	return verifyErr
}

// socketUmask keeps group and other from writing to the socket the daemon
// creates; the helper only needs to connect, which requires no write bit.
const socketUmask = 0o111

// listenExclusive replaces any stale socket and creates the listener with a
// umask that keeps group and other from writing to it.
func listenExclusive(path string) (*net.UnixListener, error) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("wifihelper: clear stale socket: %w", err)
	}

	// The helper runs as the console user, so it must be able to connect; the
	// containing directory is what restricts who can reach the path at all.
	old := syscall.Umask(socketUmask)
	defer syscall.Umask(old)

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("wifihelper: listen: %w", err)
	}
	return l, nil
}
