//go:build !windows

package guestaudit

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

// pingHostImpl shells out to the system `ping` binary with -c1 so we don't
// need CAP_NET_RAW. Returns true if the host responded within timeout.
func pingHostImpl(ctx context.Context, ip string, timeout time.Duration) bool {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// -c 1: one packet. -W <sec>: timeout. Round up to >=1 second.
	secs := max(int(timeout/time.Second), 1)
	//#nosec G204 -- ip is validated upstream via validation.ValidateIP; timeout secs is int math.
	cmd := exec.CommandContext(tctx, "ping", "-c", "1", "-W", strconv.Itoa(secs), ip)
	return cmd.Run() == nil
}
