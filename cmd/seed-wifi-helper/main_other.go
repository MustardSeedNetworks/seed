//go:build !darwin

// Command seed-wifi-helper exists only on macOS, where Location Services
// authorization is required to read Wi-Fi network names and cannot be held by
// a root daemon. Every other platform scans in-process and needs no helper.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "seed-wifi-helper is only used on macOS; other platforms scan in-process")
	os.Exit(1)
}
