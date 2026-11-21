//go:build !windows

package support_test

func runtimeEnforcesModes() bool { return true }

func runtimeIsWindows() bool { return false }
