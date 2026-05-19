//go:build darwin

package truststore

import (
	"context"
	"fmt"
	"os/exec"
)

// systemKeychain is the macOS system-wide keychain that ships trusted roots
// for the whole machine. Adding a cert here is equivalent to a system admin
// dropping it into Keychain Access → System → Certificates and marking it
// "Always Trust" for SSL.
const systemKeychain = "/Library/Keychains/System.keychain"

func installPlatform(ctx context.Context, certPath string) (Result, error) {
	// `security add-trusted-cert -d -k <keychain> <cert>` installs the cert
	// and applies SSL trust. Requires root (the binary is meant to be run
	// under sudo) — security(1) will prompt otherwise.
	cmd := exec.CommandContext(ctx, "security",
		"add-trusted-cert", "-d", "-r", "trustRoot",
		"-k", systemKeychain, certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("security add-trusted-cert: %s: %w", trimError(out), err)
	}
	return Result{
		Stores: []string{"macOS System Keychain (" + systemKeychain + ")"},
	}, nil
}

func uninstallPlatform(ctx context.Context, certPath string) (Result, error) {
	cmd := exec.CommandContext(ctx, "security",
		"remove-trusted-cert", "-d", certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("security remove-trusted-cert: %s: %w", trimError(out), err)
	}
	return Result{
		Stores: []string{"macOS System Keychain (" + systemKeychain + ")"},
	}, nil
}
