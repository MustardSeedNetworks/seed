//go:build !windows

package main

import "testing"

// protectDirForTest is a no-op off Windows, where verifyDirSecurity checks
// nothing.
func protectDirForTest(testing.TB, string) {}
