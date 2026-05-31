package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/probe/checkers"
)

func TestTCPChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewTCPChecker().Kind() != "tcp" {
		t.Errorf("Kind() = %q, want tcp", checkers.NewTCPChecker().Kind())
	}
}

func TestTCPChecker_Run_Success(t *testing.T) {
	t.Parallel()
	dialer := &fakePingDialer{attempts: []dialOutcome{{conn: fakeConn{}}}}
	c := checkers.NewTCPChecker().WithTCPDialer(dialer)
	r := c.Run(context.Background(), probe.Probe{
		Kind: "tcp", Target: "db.example.com",
		Params: json.RawMessage(`{"port":5432}`),
	})
	if !r.Success {
		t.Errorf("Result.Success = false, want true; err=%q", r.Error)
	}
	if dialer.gotAddrs[0] != "db.example.com:5432" {
		t.Errorf("dialed = %q", dialer.gotAddrs[0])
	}
}

func TestTCPChecker_Run_MissingPort(t *testing.T) {
	t.Parallel()
	c := checkers.NewTCPChecker()
	r := c.Run(context.Background(), probe.Probe{Kind: "tcp", Target: "x"})
	if r.Success {
		t.Error("Success = true; want false for missing port")
	}
}

func TestTCPChecker_Run_OutOfRangePort(t *testing.T) {
	t.Parallel()
	c := checkers.NewTCPChecker()
	r := c.Run(context.Background(), probe.Probe{
		Kind: "tcp", Target: "x",
		Params: json.RawMessage(`{"port":99999}`),
	})
	if r.Success {
		t.Error("Success = true; want false for invalid port")
	}
}

func TestTCPChecker_Run_DialFails(t *testing.T) {
	t.Parallel()
	dialer := &fakePingDialer{attempts: []dialOutcome{{err: errors.New("refused")}}}
	c := checkers.NewTCPChecker().WithTCPDialer(dialer)
	r := c.Run(context.Background(), probe.Probe{
		Kind: "tcp", Target: "down",
		Params: json.RawMessage(`{"port":80}`),
	})
	if r.Success {
		t.Error("Success = true; want false")
	}
	if r.Error != "refused" {
		t.Errorf("Error = %q", r.Error)
	}
}
