//go:build linux

package truststore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// pathExists reports whether a filesystem path exists. Used by
// detectLinuxStore to pick the trust-store layout the host uses.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// linuxStore describes how a particular Linux distribution stores its
// system-wide trust anchors and how to refresh the bundle afterwards.
type linuxStore struct {
	// AnchorDir is the directory CA-trust source PEMs go into.
	AnchorDir string
	// Suffix is the file extension expected by the distribution
	// (".crt" for Debian, ".pem" for RHEL/SUSE).
	Suffix string
	// RefreshCommand is the command to run after writing/removing a file
	// to update the system trust bundle.
	RefreshCommand []string
	// Label is a human-readable name used in Result.Stores.
	Label string
}

// detectLinuxStore inspects the filesystem and returns the trust-store
// layout the current host uses, or false if none is recognized.
func detectLinuxStore() (linuxStore, bool) {
	candidates := []linuxStore{
		{
			AnchorDir:      "/usr/local/share/ca-certificates",
			Suffix:         ".crt",
			RefreshCommand: []string{"update-ca-certificates"},
			Label:          "Debian/Ubuntu system CA bundle",
		},
		{
			AnchorDir:      "/etc/pki/ca-trust/source/anchors",
			Suffix:         ".pem",
			RefreshCommand: []string{"update-ca-trust", "extract"},
			Label:          "RHEL/Fedora system CA bundle",
		},
		{
			AnchorDir:      "/etc/ca-certificates/trust-source/anchors",
			Suffix:         ".crt",
			RefreshCommand: []string{"trust", "extract-compat"},
			Label:          "Arch system CA bundle",
		},
		{
			AnchorDir:      "/usr/share/pki/trust/anchors",
			Suffix:         ".pem",
			RefreshCommand: []string{"update-ca-certificates"},
			Label:          "openSUSE system CA bundle",
		},
	}
	for _, c := range candidates {
		if pathExists(c.AnchorDir) {
			return c, true
		}
	}
	return linuxStore{}, false
}

// anchorPath is the destination path for the seed CA inside the host's
// anchor directory.
func (s linuxStore) anchorPath() string {
	return filepath.Join(s.AnchorDir, "seed-root"+s.Suffix)
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
	pemBytes, err := os.ReadFile(certPath)
	if err != nil {
		return Result{}, fmt.Errorf("read certificate: %w", err)
	}

	dst := store.anchorPath()
	// 0o644 matches the permissions update-ca-certificates expects for
	// world-readable anchor files; the file contains a public cert only.
	if err := os.WriteFile(dst, pemBytes, 0o644); err != nil { //nolint:gosec // public cert; world-readable by design
		return Result{}, fmt.Errorf("write %s: %w", dst, err)
	}

	cmd := exec.CommandContext(ctx, store.RefreshCommand[0], store.RefreshCommand[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf(
			"%s: %s: %w", strings.Join(store.RefreshCommand, " "), trimError(out), err)
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
	res := Result{}
	if pathExists(dst) {
		if err := os.Remove(dst); err != nil {
			return Result{}, fmt.Errorf("remove %s: %w", dst, err)
		}
		res.Stores = append(res.Stores, store.Label+" ("+dst+")")
	} else {
		res.Skipped = append(res.Skipped, store.Label+": "+dst+" not present")
	}

	cmd := exec.CommandContext(ctx, store.RefreshCommand[0], store.RefreshCommand[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return res, fmt.Errorf(
			"%s: %s: %w", strings.Join(store.RefreshCommand, " "), trimError(out), err)
	}
	return res, nil
}
