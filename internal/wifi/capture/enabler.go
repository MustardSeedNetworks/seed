package wificapture

import (
	"context"
	"fmt"
	"regexp"
)

// Enabler optionally puts an interface into monitor mode before capture opens it
// and restores the prior state afterwards. It is opt-in and invasive (it changes
// the interface's operating mode), so the default build wires it only when the
// operator asks for auto-enablement; otherwise capture is bring-your-own monitor.
type Enabler interface {
	// Enable puts iface into monitor mode and returns a restore function that
	// undoes the change (never nil on success). It errors if it cannot switch the
	// interface; the caller then falls back to opening the interface as-is.
	Enable(ctx context.Context, iface string) (func() error, error)
}

// ifaceName guards interface names passed to the shell-out enabler against
// injection: real interface names are short and alphanumeric with a few
// separators.
var ifaceName = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,32}$`)

// nopEnabler does nothing — used on platforms without a monitor-mode path. Its
// restore is a no-op so callers need no special-casing.
type nopEnabler struct{}

func (nopEnabler) Enable(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}

// runner executes a command, returning its error. Injected so the iw/ip
// orchestration is unit-testable without touching the host.
type runner func(ctx context.Context, name string, args ...string) error

// cmdEnabler switches monitor mode via the iw/ip command-line tools (Linux). The
// command construction is OS-agnostic and tested with a fake runner; only the
// real exec runner is platform-bound.
type cmdEnabler struct{ run runner }

// newIWEnabler builds a cmdEnabler over run.
func newIWEnabler(run runner) cmdEnabler { return cmdEnabler{run: run} }

// Enable sets iface to monitor mode and brings it up, returning a restore that
// reverts it to managed mode.
func (e cmdEnabler) Enable(ctx context.Context, iface string) (func() error, error) {
	if !ifaceName.MatchString(iface) {
		return nil, fmt.Errorf("wificapture: unsafe interface name %q", iface)
	}
	if err := e.run(ctx, "iw", "dev", iface, "set", "type", "monitor"); err != nil {
		return nil, fmt.Errorf("wificapture: set monitor on %s: %w", iface, err)
	}
	_ = e.run(ctx, "ip", "link", "set", iface, "up")

	restore := func() error {
		// Restore on a fresh context — the capture context is already cancelled
		// by the time Stop runs the restore.
		_ = e.run(context.Background(), "ip", "link", "set", iface, "down")
		return e.run(context.Background(), "iw", "dev", iface, "set", "type", "managed")
	}
	return restore, nil
}
