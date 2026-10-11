package procrun

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// A wedged child never lets Cmd.Start return; start must give up anyway.
func TestStartGivesUpOnAChildThatNeverExecs(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	wedged := func() error {
		<-release
		return errors.New("released")
	}

	begin := time.Now()
	err := start(exec.Command("wedged-helper"), wedged, 50*time.Millisecond)
	if !errors.Is(err, ErrStartTimeout) {
		t.Fatalf("start() error = %v, want ErrStartTimeout", err)
	}
	if elapsed := time.Since(begin); elapsed > 5*time.Second {
		t.Fatalf("start() returned after %s, want about the 50ms bound", elapsed)
	}
}

func TestStartReturnsTheStartError(t *testing.T) {
	want := errors.New("exec failed")
	err := start(exec.Command("helper"), func() error { return want }, time.Minute)
	if !errors.Is(err, want) {
		t.Fatalf("start() error = %v, want %v", err, want)
	}
}

func TestCollectDropsOutputAfterStartTimeout(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantOut string
	}{
		{name: "finished", err: nil, wantOut: "out"},
		{name: "exited non-zero", err: errors.New("exit status 1"), wantOut: "out"},
		{name: "never started", err: ErrStartTimeout, wantOut: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			buf.WriteString("out")
			got, err := collect(&buf, tt.err)
			if !errors.Is(err, tt.err) {
				t.Fatalf("collect() error = %v, want %v", err, tt.err)
			}
			if string(got) != tt.wantOut {
				t.Fatalf("collect() = %q, want %q", got, tt.wantOut)
			}
		})
	}
}
