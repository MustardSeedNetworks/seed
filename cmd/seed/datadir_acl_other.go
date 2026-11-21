//go:build !windows

package main

// verifyDirSecurity has nothing to check off Windows: the directory modes are
// the package manager's and the unit's.
func verifyDirSecurity(string) error { return nil }
