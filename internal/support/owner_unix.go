//go:build !windows

package support

import (
	"io/fs"
	"os"
	"syscall"
)

// fileOwner returns the owning user of info, and whether the platform reports
// one.
func fileOwner(info fs.FileInfo) (int, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(stat.Uid), true
}

func invokerUID() int { return os.Geteuid() }
