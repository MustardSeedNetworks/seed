// SPDX-License-Identifier: BUSL-1.1

package main

import "errors"

// Process exit statuses. `status` matches niac's so scripts behave the same
// across products: 0 running, 1 not running, 2 could not tell (an error, or a
// daemon that holds the lock but does not answer). The other operations
// commands (logs, support-bundle, backup, restore) exit 1 on failure, as niac's
// do, and 2 (exitRefused) when they refuse to touch an install: a running daemon
// owns it, or on Windows it needs an elevated shell or failed verification.
const (
	exitFailure       = 1
	exitStatusDown    = 1
	exitStatusUnknown = 2
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
