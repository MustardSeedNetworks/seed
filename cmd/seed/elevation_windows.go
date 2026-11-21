//go:build windows

package main

import "golang.org/x/sys/windows"

// processElevated reports whether this process holds an elevated token.
func processElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
