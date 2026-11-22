//go:build !windows

package main

// processElevated is always true off Windows: only the Windows system layout,
// which is closed to non-elevated users, needs the check.
func processElevated() bool { return true }
