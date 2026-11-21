// SPDX-License-Identifier: BUSL-1.1

package main

import "errors"

// Process exit statuses shared by the operations commands (status, logs,
// support-bundle, backup, restore). Scripts branch on them, so they are a
// contract: 0 it worked, 1 it failed, 2 the daemon's state decided the answer
// ("not running" for status, "running" for a command that refuses to touch a
// live install). exitDaemonRunning, the status the write-refusing commands
// already used, is the same 2.
const (
	exitFailure    = 1
	exitNotRunning = 2
)

// exitError carries a specific process exit status out to the single exit
// point. A nil err means the command already printed everything it had to say.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *exitError) Unwrap() error { return e.err }

// withExitCode tags err with the exit status it should produce. A nil err
// stays nil, so a caller can wrap a path that may have succeeded.
func withExitCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return &exitError{code: code, err: err}
}

// reportedExit is an exit status for a command that has already written its
// own output, such as `status` printing NOT RUNNING.
func reportedExit(code int) error {
	return &exitError{code: code}
}

// codeOf returns the status err carries, if any.
func codeOf(err error) (int, bool) {
	if coded, ok := errors.AsType[*exitError](err); ok {
		return coded.code, true
	}
	return 0, false
}
