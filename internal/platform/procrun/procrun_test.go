//go:build unix

package procrun_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/platform/procrun"
)

func TestOutput(t *testing.T) {
	out, err := procrun.Output(exec.CommandContext(t.Context(), "sh", "-c", "echo out; echo err >&2"))
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "out" {
		t.Fatalf("Output() = %q, want %q", got, "out")
	}
}

func TestCombinedOutputKeepsOutputOfAFailedHelper(t *testing.T) {
	out, err := procrun.CombinedOutput(exec.CommandContext(t.Context(), "sh", "-c", "echo out; echo err >&2; exit 3"))
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Fatalf("CombinedOutput() error = %v, want exit status 3", err)
	}
	if got := string(out); !strings.Contains(got, "out") || !strings.Contains(got, "err") {
		t.Fatalf("CombinedOutput() = %q, want both streams", got)
	}
}

func TestOutputRefusesAnAlreadySetStdout(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "true")
	cmd.Stdout = &strings.Builder{}
	if _, err := procrun.Output(cmd); err == nil {
		t.Fatal("Output() with Stdout set: want an error")
	}
}

// After exec the context on the Cmd bounds the helper, as with os/exec.
func TestRunHonoursTheCommandContextOnceStarted(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	begin := time.Now()
	err := procrun.Run(exec.CommandContext(ctx, "sleep", "10"))
	if err == nil {
		t.Fatal("Run() of a killed helper: want an error")
	}
	if elapsed := time.Since(begin); elapsed > 5*time.Second {
		t.Fatalf("Run() returned after %s, want about the 100ms deadline", elapsed)
	}
}

func TestStartReportsAMissingBinary(t *testing.T) {
	err := procrun.Start(exec.CommandContext(t.Context(), "/nonexistent/seed-helper"))
	if err == nil || errors.Is(err, procrun.ErrStartTimeout) {
		t.Fatalf("Start() error = %v, want the exec error", err)
	}
}
