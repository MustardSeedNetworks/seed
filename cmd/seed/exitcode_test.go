package main

import (
	"errors"
	"testing"
)

func TestExitCodeFor(t *testing.T) {
	plain := errors.New("boom")
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"plain error", plain, 1},
		{"daemon running", &daemonRunningError{dir: "/etc/seed", instead: "stop it"}, exitDaemonRunning},
		{"tagged 2", withExitCode(exitNotRunning, plain), 2},
		{"tagged 1", withExitCode(exitFailure, plain), 1},
		{"tagged and wrapped", errors.Join(errors.New("ctx"), withExitCode(2, plain)), 2},
		{"reported without a message", reportedExit(2), 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := exitCodeFor(tc.err); got != tc.want {
				t.Errorf("exitCodeFor = %d, want %d", got, tc.want)
			}
		})
	}
	if withExitCode(2, nil) != nil {
		t.Error("withExitCode(nil) must stay nil so success paths can wrap unconditionally")
	}
	if msg := reportedExit(2).Error(); msg != "" {
		t.Errorf("a reported exit carries message %q; Execute would print it twice", msg)
	}
}
