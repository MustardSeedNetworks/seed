//go:build windows

package iperf_test

import (
	"os/exec"
	"strconv"
	"strings"
)

// processAlive reports whether a pid is still a live process. Windows has no
// signal-0 equivalent, so ask the task list.
func processAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), strconv.Itoa(pid))
}
