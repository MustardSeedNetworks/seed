// Package fsutil holds small filesystem helpers shared across features that
// need to open or write a single administrator- or config-supplied path
// (an OUI database file, a config file) with no wider directory of its own
// to confine writes to. A feature that owns a whole directory (a backup
// directory, a CA trust-anchor directory) should scope an [os.Root] to that
// directory directly rather than using this package — see
// internal/config's BackupManager and internal/truststore for that pattern.
package fsutil

import (
	"errors"
	"os"
	"path/filepath"
)

// RootAt resolves path's symlinks, then opens an [os.Root] scoped to the
// resolved parent directory and returns it alongside the resolved base
// name, so the caller addresses exactly the file path names today through
// the returned root's own methods, without a raw call on the [os] package
// whose path argument crossed a function boundary tripping gosec's G703
// taint check. Resolving symlinks first (rather than handing [os.Root] the
// unresolved directory) matters because [os.Root] refuses to follow a
// symlink that leaves its scope: an operator-supplied path that is itself
// a symlink to elsewhere (an alternatives-style config layout, for
// example) would otherwise silently stop being addressable through a bare
// [os.Open] or [os.WriteFile] call. If path does not exist yet — a config
// being restored after the operator deleted it, for instance — its parent
// directory is resolved instead, so the file is still created exactly
// where path names; this matches what a direct [os.WriteFile] call would
// have done. A dangling symlink at path (one whose target lies outside
// path's own directory) is the one case this can't preserve: [os.Root]
// refuses to create through it, where [os.WriteFile] would have followed
// it and written the target anyway. The caller closes the returned root
// once it is done with it.
func RootAt(path string) (*os.Root, string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		dir, dirErr := filepath.EvalSymlinks(filepath.Dir(path))
		if dirErr != nil {
			return nil, "", dirErr
		}
		resolved = filepath.Join(dir, filepath.Base(path))
	case err != nil:
		return nil, "", err
	}
	root, err := os.OpenRoot(filepath.Dir(resolved))
	if err != nil {
		return nil, "", err
	}
	return root, filepath.Base(resolved), nil
}
