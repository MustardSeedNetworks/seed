package testutil

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/platform/procrun"
)

// totalLoss matches the summary line every supported ping prints when it sent
// echoes and got none back: "100% packet loss" (Linux), "100.0% packet loss"
// (macOS) and "(100% loss)" (Windows).
var totalLoss = regexp.MustCompile(`100(\.0)?% (packet )?loss`)

// pingDeadline bounds the system ping: one echo with a two-second wait, plus
// process start-up.
const pingDeadline = 10 * time.Second

// SkipIfHostDropsLoopbackICMP skips the test when the operating system's own
// ping sends an echo to 127.0.0.1 and gets no reply. A host firewall can drop
// loopback ICMP (macOS stealth mode was the suspect on seed#2905); Seed is not
// required to work there, so a loopback round-trip test has nothing to prove.
//
// Call it only after the code under test has failed its loopback round trip.
// It never skips when ping answers, is missing, or fails for another reason:
// then the failure belongs to Seed, not to the host.
func SkipIfHostDropsLoopbackICMP(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), pingDeadline)
	defer cancel()

	var args []string
	switch runtime.GOOS {
	case "windows":
		args = []string{"-n", "1", "-w", "2000", "127.0.0.1"}
	case "darwin":
		args = []string{"-c", "1", "-t", "2", "127.0.0.1"}
	default:
		args = []string{"-c", "1", "-W", "2", "127.0.0.1"}
	}
	out, err := procrun.CombinedOutput(exec.CommandContext(ctx, "ping", args...))
	if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) && totalLoss.Match(out) {
		t.Skipf("loopback ICMP dropped by host firewall: the system ping got no reply from 127.0.0.1\n%s", out)
	}
}
