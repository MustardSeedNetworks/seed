//go:build linux

package truststore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// pathExists reports whether a filesystem path exists. Used by
// detectLinuxStore to pick the trust-store layout the host uses.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// linuxStore describes how a particular Linux distribution stores its
// system-wide trust anchors and how to refresh the bundle afterwards.
//
// Refresh is a closure rather than a string slice so the gosec G204
// "exec from variable" rule sees literal argument lists at the call
// site. Each detectLinuxStore branch wires in its distribution-specific
// command directly.
type linuxStore struct {
	// AnchorDir is the directory CA-trust source PEMs go into.
	AnchorDir string
	// Suffix is the file extension expected by the distribution
	// (".crt" for Debian, ".pem" for RHEL/SUSE).
	Suffix string
	// Label is a human-readable name used in Result.Stores.
	Label string
	// RefreshLabel is the command string shown in error messages
	// (e.g. "update-ca-trust extract").
	RefreshLabel string
	// Refresh runs the post-write bundle refresh and returns the
	// combined stdout/stderr for diagnostics.
	Refresh func(context.Context) ([]byte, error)
}

// detectLinuxStore inspects the filesystem and returns the trust-store
// layout the current host uses, or false if none is recognized.
func detectLinuxStore() (linuxStore, bool) {
	candidates := []linuxStore{
		{
			AnchorDir:    "/usr/local/share/ca-certificates",
			Suffix:       ".crt",
			Label:        "Debian/Ubuntu system CA bundle",
			RefreshLabel: "update-ca-certificates",
			Refresh: func(ctx context.Context) ([]byte, error) {
				return exec.CommandContext(ctx, "update-ca-certificates").CombinedOutput()
			},
		},
		{
			AnchorDir:    "/etc/pki/ca-trust/source/anchors",
			Suffix:       ".pem",
			Label:        "RHEL/Fedora system CA bundle",
			RefreshLabel: "update-ca-trust extract",
			Refresh: func(ctx context.Context) ([]byte, error) {
				return exec.CommandContext(ctx, "update-ca-trust", "extract").CombinedOutput()
			},
		},
		{
			AnchorDir:    "/etc/ca-certificates/trust-source/anchors",
			Suffix:       ".crt",
			Label:        "Arch system CA bundle",
			RefreshLabel: "trust extract-compat",
			Refresh: func(ctx context.Context) ([]byte, error) {
				return exec.CommandContext(ctx, "trust", "extract-compat").CombinedOutput()
			},
		},
		{
			AnchorDir:    "/usr/share/pki/trust/anchors",
			Suffix:       ".pem",
			Label:        "openSUSE system CA bundle",
			RefreshLabel: "update-ca-certificates",
			Refresh: func(ctx context.Context) ([]byte, error) {
				return exec.CommandContext(ctx, "update-ca-certificates").CombinedOutput()
			},
		},
	}
	for _, c := range candidates {
		if pathExists(c.AnchorDir) {
			return c, true
		}
	}
	return linuxStore{}, false
}

// anchorName is the destination filename for the seed CA inside the host's
// anchor directory.
func (s linuxStore) anchorName() string {
	return "seed-root" + s.Suffix
}

// anchorPath is the destination path for the seed CA inside the host's
// anchor directory, for display in Result messages.
func (s linuxStore) anchorPath() string {
	return filepath.Join(s.AnchorDir, s.anchorName())
}

// trustAnchorMode is the on-disk permission set the system bundle
// refresh tooling expects: world-readable, owner-writable. The file
// contents are a public certificate and contain no secret material.
const trustAnchorMode os.FileMode = 0o644

// anchorWriteMode is the permission the anchor file is created at before
// the chmod to trustAnchorMode below (owner read/write only, so the file
// does not exist world-readable while its contents are being written).
// mnd's ignored-functions list covers [os.WriteFile] but not the
// identically-named [os.Root] method used here, so this is named rather
// than repeated as a literal.
const anchorWriteMode os.FileMode = 0o600

// writeAnchorFile writes pem to name inside root at trustAnchorMode, via
// the two-step create-then-chmod described by anchorWriteMode above.
//
// name is opened through an [os.Root] scoped to root, rather than a bare
// [os.WriteFile] on a joined path, so it cannot resolve outside root even
// via a symlink.
func writeAnchorFile(root, name string, pem []byte) (string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", root, err)
	}
	defer func() { _ = r.Close() }()

	dst := filepath.Join(root, name)
	if writeErr := r.WriteFile(name, pem, anchorWriteMode); writeErr != nil {
		return "", fmt.Errorf("write %s: %w", dst, writeErr)
	}
	if chmodErr := r.Chmod(name, trustAnchorMode); chmodErr != nil {
		return "", fmt.Errorf("chmod %s: %w", dst, chmodErr)
	}
	return dst, nil
}

func installPlatform(ctx context.Context, certPath string) (Result, error) {
	store, ok := detectLinuxStore()
	if !ok {
		return Result{}, errors.New(
			"no supported system CA directory found " +
				"(tried /usr/local/share/ca-certificates, /etc/pki/ca-trust/source/anchors, " +
				"/etc/ca-certificates/trust-source/anchors, /usr/share/pki/trust/anchors)")
	}

	// #nosec G304 -- certPath is operator-supplied (validated by
	// ValidateCertFile in the caller) and the destination is a fixed
	// system directory.
	pemBytes, readErr := os.ReadFile(certPath)
	if readErr != nil {
		return Result{}, fmt.Errorf("read certificate: %w", readErr)
	}

	dst, writeErr := writeAnchorFile(store.AnchorDir, store.anchorName(), pemBytes)
	if writeErr != nil {
		return Result{}, writeErr
	}

	out, refreshErr := store.Refresh(ctx)
	if refreshErr != nil {
		return Result{}, fmt.Errorf(
			"%s: %s: %w", store.RefreshLabel, trimError(out), refreshErr)
	}

	return Result{
		Stores: []string{store.Label + " (" + dst + ")"},
	}, nil
}

func uninstallPlatform(ctx context.Context, _ string) (Result, error) {
	store, ok := detectLinuxStore()
	if !ok {
		return Result{}, errors.New("no supported system CA directory found")
	}
	dst := store.anchorPath()
	name := store.anchorName()
	res := Result{}

	root, err := os.OpenRoot(store.AnchorDir)
	if err != nil {
		return Result{}, fmt.Errorf("open %s: %w", store.AnchorDir, err)
	}
	defer func() { _ = root.Close() }()

	if _, statErr := root.Stat(name); statErr == nil {
		if removeErr := root.Remove(name); removeErr != nil {
			return Result{}, fmt.Errorf("remove %s: %w", dst, removeErr)
		}
		res.Stores = append(res.Stores, store.Label+" ("+dst+")")
	} else {
		res.Skipped = append(res.Skipped, store.Label+": "+dst+" not present")
	}

	out, refreshErr := store.Refresh(ctx)
	if refreshErr != nil {
		return res, fmt.Errorf(
			"%s: %s: %w", store.RefreshLabel, trimError(out), refreshErr)
	}
	return res, nil
}
