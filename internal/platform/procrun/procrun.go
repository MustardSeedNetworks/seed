// Package procrun starts system helpers so that a child which never reaches
// execve cannot stall its caller.
//
// On macOS a fork out of the daemon can leave the child spinning in
// Network.framework's atfork handler (nw_settings_child_has_forked) before it
// execs (seed#2530). [exec.CommandContext] does not help: Cmd.Start blocks in
// the parent until the child execs or fails, and the context is only watched
// after Start returns, so a deadline on the context never fires. Every helper
// started here gets StartTimeout to reach execve; past that, the call returns
// ErrStartTimeout and the child is killed and reaped if it ever does start.
//
// Once the child has exec'd, the context on the Cmd bounds it as usual.
package procrun

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// StartTimeout is how long a child gets to reach execve. A healthy fork+exec
// takes milliseconds; the margin covers a heavily loaded host, where seed#2530
// was measured at load averages of 50 to 300.
const StartTimeout = 15 * time.Second

// ErrStartTimeout reports a child that did not reach execve within
// StartTimeout.
var ErrStartTimeout = errors.New("helper did not start")

// Start starts cmd. See the package documentation for the bound it applies.
func Start(cmd *exec.Cmd) error {
	return start(cmd, cmd.Start, StartTimeout)
}

// Run starts cmd and waits for it to finish.
func Run(cmd *exec.Cmd) error {
	if err := Start(cmd); err != nil {
		return err
	}
	return cmd.Wait()
}

// Output runs cmd and returns its standard output.
func Output(cmd *exec.Cmd) ([]byte, error) {
	if cmd.Stdout != nil {
		return nil, errors.New("procrun: Stdout already set")
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	return collect(&stdout, Run(cmd))
}

// CombinedOutput runs cmd and returns its standard output and standard error
// interleaved.
func CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	if cmd.Stdout != nil || cmd.Stderr != nil {
		return nil, errors.New("procrun: Stdout or Stderr already set")
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	return collect(&out, Run(cmd))
}

// collect returns what a finished command wrote. After ErrStartTimeout the
// buffer still belongs to the abandoned child, so none of it is returned.
func collect(buf *bytes.Buffer, err error) ([]byte, error) {
	if errors.Is(err, ErrStartTimeout) {
		return nil, err
	}
	return buf.Bytes(), err
}

// start runs startFn off the caller's goroutine and gives up after timeout.
// The abandoned goroutine stays blocked for as long as the child is wedged;
// if the child ever execs, it is killed and reaped there.
func start(cmd *exec.Cmd, startFn func() error, timeout time.Duration) error {
	started := make(chan error, 1)
	go func() { started <- startFn() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-started:
		return err
	case <-timer.C:
		go func() {
			if <-started == nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}()
		return fmt.Errorf("%w: %s after %s", ErrStartTimeout, cmd.Path, timeout)
	}
}
