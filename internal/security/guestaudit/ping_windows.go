//go:build windows

package guestaudit

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

// minPingTimeoutMs is the floor for the ping -w argument: the Windows
// ping utility needs a nonzero millisecond timeout, and a caller-supplied
// timeout shorter than this would round to something ping cannot use.
const minPingTimeoutMs = 100

// pingHostImpl shells out to the Windows `ping` binary (uses -n / -w).
func pingHostImpl(ctx context.Context, ip string, timeout time.Duration) bool {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ms := max(int(timeout/time.Millisecond), minPingTimeoutMs)
	//#nosec G204 -- ip is validated upstream; timeout ms is int math.
	cmd := exec.CommandContext(tctx, "ping", "-n", "1", "-w", strconv.Itoa(ms), ip)
	return cmd.Run() == nil
}
