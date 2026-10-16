//go:build darwin

package carrier

import (
	"net"
	"testing"
	"unsafe"
)

func TestIfmediareqLayout(t *testing.T) {
	t.Parallel()

	var req ifmediareq
	if got := unsafe.Sizeof(req); got != 44 {
		t.Errorf("sizeof(ifmediareq) = %d, want 44 (the size SIOCGIFMEDIA encodes)", got)
	}
	if got := unsafe.Offsetof(req.status); got != 24 {
		t.Errorf("offsetof(ifm_status) = %d, want 24", got)
	}
}

func TestLoopbackFallsBackToRunning(t *testing.T) {
	t.Parallel()

	lo, err := net.InterfaceByName("lo0")
	if err != nil {
		t.Skipf("no lo0: %v", err)
	}
	if _, ok := mediaActive(lo.Name); ok {
		t.Error("lo0 reported a media status; loopback has none")
	}
	if got, want := Present(*lo), lo.Flags&net.FlagRunning != 0; got != want {
		t.Errorf("Present(lo0) = %v, want IFF_RUNNING %v", got, want)
	}
}
