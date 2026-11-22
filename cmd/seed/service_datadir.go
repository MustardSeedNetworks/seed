package main

import (
	"fmt"
	"path/filepath"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/paths"
)

// The Windows service keeps its state in one tree under %ProgramData%, and
// reaches it only through foundation's securedir, which fails closed on a
// directory a user could have planted, a link anywhere in the path, an owner
// or ACL outside SYSTEM and Administrators, and a hard link or FIFO inside.
// securedir judges who can change a path, not who can read it; Ensure is what
// closes reads, by replacing the ACL of the tree it takes over.

// canonicalServicePaths returns sp with every path under Root rebased onto
// Root's securedir.Canonical form. securedir never follows a link in the paths
// it is given, so everything downstream uses these and not the raw layout.
func canonicalServicePaths(sp *paths.Paths, opts securedir.Options) (*paths.Paths, error) {
	root, err := securedir.Canonical(sp.Root, opts)
	if err != nil {
		return nil, fmt.Errorf("resolve service directory %s: %w", sp.Root, err)
	}
	resolved := *sp
	for _, field := range []*string{
		&resolved.Root, &resolved.ConfigDir, &resolved.ConfigFile,
		&resolved.DataDir, &resolved.LogDir, &resolved.CacheDir, &resolved.CertDir,
	} {
		if *field == "" {
			continue
		}
		rel, relErr := filepath.Rel(sp.Root, *field)
		if relErr != nil {
			return nil, fmt.Errorf("place %s under %s: %w", *field, sp.Root, relErr)
		}
		*field = filepath.Join(root, rel)
	}
	return &resolved, nil
}

// prepareServiceDirs makes the service's directory tree private and returns the
// canonical layout to use from here on. It runs at service start as SYSTEM,
// which may take over a directory install.ps1 created (the descriptor Ensure
// writes is the installer's: SYSTEM and Administrators, protected) or create
// the tree itself. An SCM-started service has C:\Windows\System32 as its
// working directory, so nothing the service writes may rely on one.
func prepareServiceDirs(sp *paths.Paths, opts securedir.Options) (*paths.Paths, error) {
	resolved, err := canonicalServicePaths(sp, opts)
	if err != nil {
		return nil, err
	}
	for _, dir := range []string{
		resolved.Root, resolved.ConfigDir, resolved.DataDir, resolved.LogDir, resolved.CertDir,
	} {
		if ensureErr := securedir.Ensure(dir, opts); ensureErr != nil {
			return nil, fmt.Errorf("secure service directory %s: %w", dir, ensureErr)
		}
	}
	if verifyErr := securedir.Verify(resolved.Root, opts); verifyErr != nil {
		return nil, fmt.Errorf("verify service directory %s: %w", resolved.Root, verifyErr)
	}
	return resolved, nil
}

// verifyDataDir fails unless root is a private tree securedir can show nobody
// untrusted can change. Run it before an elevated process reads or writes the
// tree: anything less trusts a directory a user may have made.
func verifyDataDir(root string, opts securedir.Options) error {
	canonical, err := securedir.Canonical(root, opts)
	if err != nil {
		return fmt.Errorf("data directory %s: %w", root, err)
	}
	if verifyErr := securedir.Verify(canonical, opts); verifyErr != nil {
		return fmt.Errorf("data directory %s: %w", root, verifyErr)
	}
	return nil
}

// savePrivateConfig is cfg.Save through securedir.WriteFile: atomic, owner
// only from the first byte, and refused unless path sits in a private
// directory. path must be absolute and canonical.
func savePrivateConfig(cfg *config.Config, path string, opts securedir.Options) error {
	data, err := cfg.Marshal()
	if err != nil {
		return err
	}
	return securedir.WriteFile(path, data, opts)
}
