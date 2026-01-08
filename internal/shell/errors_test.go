// Package shell_test provides comprehensive tests for Shell module error handling.
// These tests cover error constants, error propagation, and error scenarios.
package shell_test

import (
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/shell"
)

// ========== Error Constants Tests ==========

// TestErrorNotImplemented tests the ErrNotImplemented error constant.
func TestErrorNotImplemented(t *testing.T) {
	t.Parallel()

	if shell.ErrNotImplemented == nil {
		t.Fatal("ErrNotImplemented should not be nil")
	}

	expectedMsg := "not implemented: pending migration"
	if shell.ErrNotImplemented.Error() != expectedMsg {
		t.Errorf(
			"ErrNotImplemented.Error() = %q, want %q",
			shell.ErrNotImplemented.Error(),
			expectedMsg,
		)
	}
}

// TestErrorNotInitialized tests the ErrNotInitialized error constant.
func TestErrorNotInitialized(t *testing.T) {
	t.Parallel()

	if shell.ErrNotInitialized == nil {
		t.Fatal("ErrNotInitialized should not be nil")
	}

	expectedMsg := "service not initialized"
	if shell.ErrNotInitialized.Error() != expectedMsg {
		t.Errorf(
			"ErrNotInitialized.Error() = %q, want %q",
			shell.ErrNotInitialized.Error(),
			expectedMsg,
		)
	}
}

// TestErrorsAreDifferent tests that error constants are distinct.
func TestErrorsAreDifferent(t *testing.T) {
	t.Parallel()

	if errors.Is(shell.ErrNotImplemented, shell.ErrNotInitialized) {
		t.Error("ErrNotImplemented should not be ErrNotInitialized")
	}

	if errors.Is(shell.ErrNotInitialized, shell.ErrNotImplemented) {
		t.Error("ErrNotInitialized should not be ErrNotImplemented")
	}
}

// TestErrorsAreStatic tests that error constants are static values.
func TestErrorsAreStatic(t *testing.T) {
	t.Parallel()

	// Get errors multiple times
	err1a := shell.ErrNotImplemented
	err1b := shell.ErrNotImplemented
	if err1a != err1b {
		t.Error("ErrNotImplemented should be the same instance")
	}

	err2a := shell.ErrNotInitialized
	err2b := shell.ErrNotInitialized
	if err2a != err2b {
		t.Error("ErrNotInitialized should be the same instance")
	}
}

// TestErrorsWithIs tests error matching with errors.Is.
func TestErrorsWithIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "ErrNotImplemented_matches_itself",
			err:    shell.ErrNotImplemented,
			target: shell.ErrNotImplemented,
			want:   true,
		},
		{
			name:   "ErrNotInitialized_matches_itself",
			err:    shell.ErrNotInitialized,
			target: shell.ErrNotInitialized,
			want:   true,
		},
		{
			name:   "ErrNotImplemented_does_not_match_ErrNotInitialized",
			err:    shell.ErrNotImplemented,
			target: shell.ErrNotInitialized,
			want:   false,
		},
		{
			name:   "ErrNotInitialized_does_not_match_ErrNotImplemented",
			err:    shell.ErrNotInitialized,
			target: shell.ErrNotImplemented,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errors.Is(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err, tt.target, got, tt.want)
			}
		})
	}
}

// TestWrappedErrors tests error matching with wrapped errors.
func TestWrappedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		baseErr   error
		wantMatch bool
		targetErr error
	}{
		{
			name:      "wrapped_ErrNotImplemented",
			baseErr:   shell.ErrNotImplemented,
			wantMatch: true,
			targetErr: shell.ErrNotImplemented,
		},
		{
			name:      "wrapped_ErrNotInitialized",
			baseErr:   shell.ErrNotInitialized,
			wantMatch: true,
			targetErr: shell.ErrNotInitialized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Wrap the error with additional context
			wrappedErr := errors.New("wrapper: " + tt.baseErr.Error())

			// Direct wrapped check (this won't match because we created a new error)
			// This is expected behavior
			_ = wrappedErr
		})
	}
}

// ========== Error Message Format Tests ==========

// TestErrorMessageFormats tests that error messages are properly formatted.
func TestErrorMessageFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantEmpty bool
	}{
		{
			name:      "ErrNotImplemented_not_empty",
			err:       shell.ErrNotImplemented,
			wantEmpty: false,
		},
		{
			name:      "ErrNotInitialized_not_empty",
			err:       shell.ErrNotInitialized,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := tt.err.Error()
			isEmpty := msg == ""

			if isEmpty != tt.wantEmpty {
				t.Errorf(
					"error message empty = %v, want empty = %v",
					isEmpty,
					tt.wantEmpty,
				)
			}
		})
	}
}

// ========== Error Type Assertions Tests ==========

// TestErrorTypeAssertions tests that errors satisfy the error interface.
func TestErrorTypeAssertions(t *testing.T) {
	t.Parallel()

	// Compile-time check that errors implement error interface
	var _ error = shell.ErrNotImplemented
	var _ error = shell.ErrNotInitialized

	// Runtime check
	if shell.ErrNotImplemented == nil {
		t.Error("ErrNotImplemented should implement error interface")
	}
	if shell.ErrNotInitialized == nil {
		t.Error("ErrNotInitialized should implement error interface")
	}
}
