//go:build windows

package support_test

func runtimeEnforcesModes() bool { return false }

func runtimeIsWindows() bool { return true }
