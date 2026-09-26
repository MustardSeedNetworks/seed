// Package fsutil holds small filesystem helpers shared across features that
// need to open or write a single administrator- or config-supplied path
// (an OUI database file, a config file) with no wider directory of its own
// to confine writes to. A feature that owns a whole directory (a backup
// directory, a CA trust-anchor directory) should scope an [os.Root] to that
// directory directly rather than using this package — see
// internal/config's BackupManager and internal/truststore for that pattern.
package fsutil

import (
	"os"
	"path/filepath"
)

// RootAt opens an [os.Root] scoped to path's parent directory and returns it
// alongside path's base name, so the caller addresses exactly the file path
// names through the returned root's own methods, without a raw call on the
// [os] package whose path argument crossed a function boundary tripping
// gosec's G703 taint check. Joining the parent directory back with the base
// name always reconstructs the original (cleaned) path, so this resolves to
// the identical file a direct call would have; the benefit is [os.Root]'s
// symlink-escape protection, not a change in which file is addressed. The
// caller closes the returned root once it is done with it.
func RootAt(path string) (*os.Root, string, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, "", err
	}
	return root, filepath.Base(path), nil
}
