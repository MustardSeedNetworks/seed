//go:build windows

package api_test

import (
	"net"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

// TestIsAddrInUse_RecognisesWSAEADDRINUSE confirms isAddrInUse matches a
// wrapped WSAEADDRINUSE, the errno Winsock returns for bind contention.
// [syscall.EADDRINUSE] (the Linux/macOS errno) does NOT carry this value on
// Windows, so this must not be conflated with the unix predicate's test.
func TestIsAddrInUse_RecognisesWSAEADDRINUSE(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: windows.WSAEADDRINUSE}
	if !api.ExportIsAddrInUse(wrapped) {
		t.Fatalf("expected isAddrInUse to match WSAEADDRINUSE")
	}
}
