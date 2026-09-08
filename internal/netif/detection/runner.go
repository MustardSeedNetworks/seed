package detection

import (
	"context"
	"os/exec"
)

// commandRunner runs a system helper and returns its stdout. Platform speed and
// chipset detection go through a Detector's runner rather than calling exec
// directly, so a test can drive the parsers without starting a process.
//
// That is not tidiness. DetectAll calls getInterfaceSpeed once per interface,
// and on darwin each call forks `networksetup` and often `ifconfig`;
// TestDetectAll, TestDetectBest and TestScoreInterface between them spent ~9.6s
// of this package's 10.8s doing that against whatever interfaces the
// developer's machine happens to have. A fork out of a cgo/ObjC process is also
// the mechanism behind seed#2420: a child stuck before execve carries the
// parent's argv, so `pgrep -f 'seed --config'` counts it as another daemon.
type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// execRunner is the production runner: it actually starts the helper.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
