//go:build !windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

func TestVerifyDataDirRefusesAGroupWritableDirectory(t *testing.T) {
	dir := privateTempDir(t)
	if err := os.Chmod(dir, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := verifyDataDir(dir, securedir.Options{}); !errors.Is(err, securedir.ErrUnsafeAccess) {
		t.Fatalf("verifyDataDir error = %v, want ErrUnsafeAccess", err)
	}
}

func TestPrepareServiceDirsRefusesALinkedRoot(t *testing.T) {
	base := privateTempDir(t)
	target := filepath.Join(base, "elsewhere")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "Seed")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	_, err := prepareServiceDirs(servicePathsUnder(link), securedir.Options{})
	if !errors.Is(err, securedir.ErrReparsePoint) {
		t.Fatalf("prepareServiceDirs error = %v, want ErrReparsePoint", err)
	}
}
