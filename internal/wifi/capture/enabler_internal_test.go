package wificapture

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// recordingRunner captures the commands a cmdEnabler would run.
type recordingRunner struct {
	cmds   []string
	failOn string // substring; matching command returns an error
	calls  int
}

func (r *recordingRunner) run(_ context.Context, name string, args ...string) error {
	cmd := name + " " + strings.Join(args, " ")
	r.cmds = append(r.cmds, cmd)
	r.calls++
	if r.failOn != "" && strings.Contains(cmd, r.failOn) {
		return errors.New("boom")
	}
	return nil
}

func TestCmdEnablerSetsMonitorAndRestores(t *testing.T) {
	rr := &recordingRunner{}
	restore, err := newIWEnabler(rr.run).Enable(context.Background(), "wlan1")
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if len(rr.cmds) < 2 || !strings.Contains(rr.cmds[0], "iw dev wlan1 set type monitor") {
		t.Fatalf("expected monitor switch first, got %v", rr.cmds)
	}
	if rerr := restore(); rerr != nil {
		t.Fatalf("restore: %v", rerr)
	}
	joined := strings.Join(rr.cmds, " | ")
	if !strings.Contains(joined, "set type managed") {
		t.Errorf("restore should revert to managed, got %v", rr.cmds)
	}
}

func TestCmdEnablerRejectsUnsafeIface(t *testing.T) {
	rr := &recordingRunner{}
	if _, err := newIWEnabler(rr.run).Enable(context.Background(), "wlan1; rm -rf /"); err == nil {
		t.Error("unsafe interface name must be rejected")
	}
	if rr.calls != 0 {
		t.Error("no command should run for an unsafe interface name")
	}
}

func TestCmdEnablerPropagatesSwitchError(t *testing.T) {
	rr := &recordingRunner{failOn: "set type monitor"}
	if _, err := newIWEnabler(rr.run).Enable(context.Background(), "wlan1"); err == nil {
		t.Error("a failed monitor switch must surface an error")
	}
}

func TestNopEnabler(t *testing.T) {
	restore, err := nopEnabler{}.Enable(context.Background(), "anything")
	if err != nil {
		t.Fatalf("nop Enable: %v", err)
	}
	if rerr := restore(); rerr != nil {
		t.Errorf("nop restore: %v", rerr)
	}
}
