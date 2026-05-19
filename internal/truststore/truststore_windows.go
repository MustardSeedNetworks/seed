//go:build windows

package truststore

import (
	"context"
	"fmt"
	"os/exec"
)

// installPlatform installs the cert into the LocalMachine ROOT store using
// the built-in certutil.exe. Requires an elevated shell.
func installPlatform(ctx context.Context, certPath string) (Result, error) {
	cmd := exec.CommandContext(ctx, "certutil.exe", "-addstore", "Root", certPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("certutil -addstore Root: %s: %w", trimError(out), err)
	}
	return Result{
		Stores: []string{"Windows Certificate Store (LocalMachine\\Root)"},
	}, nil
}

// uninstallPlatform removes the cert from the LocalMachine ROOT store. The
// cert's serial number identifies the entry to certutil.
func uninstallPlatform(ctx context.Context, certPath string) (Result, error) {
	cert, err := ValidateCertFile(certPath)
	if err != nil {
		return Result{}, err
	}
	serial := cert.SerialNumber.Text(16)
	cmd := exec.CommandContext(ctx, "certutil.exe", "-delstore", "Root", serial)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("certutil -delstore Root %s: %s: %w", serial, trimError(out), err)
	}
	return Result{
		Stores: []string{"Windows Certificate Store (LocalMachine\\Root)"},
	}, nil
}
