package main

import (
	"fmt"
	"os"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/paths"
)

const (
	privateDirMode  = 0o750
	privateFileMode = 0o600
)

// The service reaches its data directory only through the three seams below,
// so adopting foundation's pkg/securedir (protected DACLs, fail-closed on
// Windows) is a change to these bodies and nothing else.
//
// On Windows today they are no stronger than their Unix mode bits: MkdirAll
// and a file mode set no DACL, so the directory is as private as its parent
// under %ProgramData% makes it. deploy/windows/install.ps1 creates the root
// with a protected ACL, which is what keeps this safe until the swap.

// ensureDataDir creates path and any missing parents.
//
// TODO(#3848): use securedir.Ensure.
func ensureDataDir(path string) error {
	return os.MkdirAll(path, privateDirMode)
}

// verifyDataDir fails unless path is a real directory (not a link) and every
// file in files that exists is a regular file.
//
// TODO(#3848): use securedir.Verify.
func verifyDataDir(path string, files ...string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("data directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("data directory %s is not a directory (mode %s)", path, info.Mode().Type())
	}
	for _, file := range files {
		fileInfo, statErr := os.Lstat(file)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return fmt.Errorf("data file %s: %w", file, statErr)
		}
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("data file %s is not a regular file (mode %s)", file, fileInfo.Mode().Type())
		}
	}
	return nil
}

// writePrivateFile replaces path with data, readable by its owner only. The
// replacement is atomic: a crash mid-write leaves the previous contents.
//
// TODO(#3848): use securedir.WriteFile.
func writePrivateFile(path string, data []byte) error {
	return config.WriteFileAtomically(path, data, privateFileMode)
}

// savePrivateConfig is cfg.Save through writePrivateFile.
func savePrivateConfig(cfg *config.Config, path string) error {
	data, err := cfg.Marshal()
	if err != nil {
		return err
	}
	return writePrivateFile(path, data)
}

// prepareServiceDirs creates the service's directory tree and checks its root.
// An SCM-started service has C:\Windows\System32 as its working directory, so
// nothing the service writes may rely on one.
func prepareServiceDirs(sp *paths.Paths) error {
	for _, dir := range []string{sp.Root, sp.ConfigDir, sp.DataDir, sp.LogDir, sp.CertDir} {
		if err := ensureDataDir(dir); err != nil {
			return fmt.Errorf("create service directory: %w", err)
		}
	}
	return verifyDataDir(sp.Root, sp.ConfigFile)
}
