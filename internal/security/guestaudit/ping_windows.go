//go:build windows

package guestaudit

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

// pingHostImpl shells out to the Windows `ping` binary (uses -n / -w).
func pingHostImpl(ctx context.Context, ip string, timeout time.Duration) bool {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ms := int(timeout / time.Millisecond)
	if ms < 100 {
		ms = 100
	}
	//#nosec G204 -- ip is validated upstream; timeout ms is int math.
	cmd := exec.CommandContext(tctx, "ping", "-n", "1", "-w", strconv.Itoa(ms), ip)
	return cmd.Run() == nil
}
