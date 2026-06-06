//go:build linux

package wificapture

import (
	"context"
	"fmt"
	"os/exec"
)

// DefaultEnabler returns the Linux monitor-mode enabler, which switches the
// interface with the iw/ip tools (nl80211). It requires CAP_NET_ADMIN/root; if
// the tools are missing or the call is unprivileged, Enable errors and capture
// falls back to opening the interface as-is (bring-your-own monitor).
func DefaultEnabler() Enabler { return newIWEnabler(execRunner) }

// execRunner runs name+args, surfacing combined output on failure for diagnosis.
func execRunner(ctx context.Context, name string, args ...string) error {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, out)
	}
	return nil
}
