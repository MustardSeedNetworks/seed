//go:build !windows

package iperf_test

import (
	"os"
	"syscall"
)

// processAlive reports whether a pid is still a live process. Signal 0 performs
// the permission and existence checks without delivering anything.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return proc.Signal(syscall.Signal(0)) == nil
}
