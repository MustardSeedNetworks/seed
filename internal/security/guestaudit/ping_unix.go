//go:build !windows

package guestaudit

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/platform/procrun"
)

// pingHostImpl shells out to the system `ping` binary with -c1 so we don't
// need CAP_NET_RAW. Returns true if the host responded within timeout.
func pingHostImpl(ctx context.Context, ip string, timeout time.Duration) bool {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	//#nosec G204 -- ip is validated upstream via validation.ValidateIP; the wait is int math.
	cmd := exec.CommandContext(tctx, "ping", pingArgs(runtime.GOOS, ip, timeout)...)
	return procrun.Run(cmd) == nil
}

// pingArgs sends one echo. The context in pingHostImpl enforces the budget, so
// -W only has to wait at least that long: macOS reads it as milliseconds and
// Linux as whole seconds, rounded up (seed#3310).
func pingArgs(goos, ip string, timeout time.Duration) []string {
	unit := time.Second
	if goos == "darwin" {
		unit = time.Millisecond
	}
	wait := max(int((timeout+unit-1)/unit), 1)
	return []string{"-c", "1", "-W", strconv.Itoa(wait), ip}
}
