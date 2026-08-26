//go:build !windows

package api_test

import (
	"net"
	"syscall"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

// TestIsAddrInUse_RecognisesSyscall confirms isAddrInUse matches a wrapped
// EADDRINUSE, the errno bind returns on Linux/macOS.
func TestIsAddrInUse_RecognisesSyscall(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: syscall.EADDRINUSE}
	if !api.ExportIsAddrInUse(wrapped) {
		t.Fatalf("expected isAddrInUse to match EADDRINUSE")
	}
}
