//go:build windows

package support

import "io/fs"

// Windows file ownership is an ACL, not a uid; the link and reparse-point
// checks are the guard there.
func fileOwner(fs.FileInfo) (int, bool) { return 0, false }

func invokerUID() int { return 0 }
