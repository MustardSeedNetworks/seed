//go:build !windows

package main

// initServiceCmd is a no-op on non-Windows platforms.
// Windows service management is only available on Windows.
func initServiceCmd(_ *cliState) {}
