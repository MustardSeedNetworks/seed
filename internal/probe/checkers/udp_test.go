package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

type fakeUDPProber struct {
	err     error
	gotAddr string
}

func (f *fakeUDPProber) Probe(_ context.Context, addr string, _ []byte, _ time.Duration) error {
	f.gotAddr = addr
	return f.err
}

func TestUDPChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewUDPChecker().Kind() != "udp" {
		t.Errorf("Kind() != udp")
	}
}

func TestUDPChecker_Run_Success(t *testing.T) {
	t.Parallel()
	pr := &fakeUDPProber{}
	c := checkers.NewUDPChecker().WithUDPProber(pr)
	r := c.Run(context.Background(), probe.Probe{
		Kind: "udp", Target: "ntp.example.com",
		Params: json.RawMessage(`{"port":123}`),
	})
	if !r.Success {
		t.Errorf("Success = false; want true: %s", r.Error)
	}
	if pr.gotAddr != "ntp.example.com:123" {
		t.Errorf("gotAddr = %q", pr.gotAddr)
	}
}

func TestUDPChecker_Run_ProbeError(t *testing.T) {
	t.Parallel()
	pr := &fakeUDPProber{err: errors.New("network unreachable")}
	c := checkers.NewUDPChecker().WithUDPProber(pr)
	r := c.Run(context.Background(), probe.Probe{
		Kind: "udp", Target: "x",
		Params: json.RawMessage(`{"port":53}`),
	})
	if r.Success {
		t.Error("Success = true; want false")
	}
}

func TestUDPChecker_Run_InvalidPort(t *testing.T) {
	t.Parallel()
	c := checkers.NewUDPChecker()
	r := c.Run(context.Background(), probe.Probe{Kind: "udp", Target: "x"})
	if r.Success {
		t.Error("missing port should fail")
	}
}
