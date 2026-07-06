package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	// backupSuffix is the suffix added to backup files.
	backupSuffix = ".backup"
)

// Applier handles applying updates to the running binary.
type Applier struct {
	binaryPath string
	backupPath string

	mu     sync.RWMutex
	status UpdateStatus
}

// NewApplier creates a new update applier.
func NewApplier() (*Applier, error) {
	// Get the path to the currently running binary
	binaryPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	// Resolve any symlinks
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve symlinks: %w", err)
	}

	return &Applier{
		binaryPath: binaryPath,
		backupPath: binaryPath + backupSuffix,
		status: UpdateStatus{
			State: StateIdle,
		},
	}, nil
}

// Apply applies the downloaded update.
// This will:
// 1. Backup the current binary.
// 2. Replace it with the new binary.
// 3. Set appropriate permissions.
// 4. Signal for restart.
func (a *Applier) Apply(_ context.Context, newBinaryPath string) error {
	a.mu.Lock()
	a.status = UpdateStatus{
		State:     StateApplying,
		StartedAt: time.Now(),
		Message:   "Preparing to apply update",
	}
	a.mu.Unlock()

	// Step 1: Verify the new binary exists and is executable
	if err := a.verifyNewBinary(newBinaryPath); err != nil {
		a.setError(fmt.Errorf("verify new binary: %w", err))
		return err
	}

	// Step 2: Create backup of current binary
	a.updateMessage("Creating backup of current binary")
	if err := a.createBackup(); err != nil {
		a.setError(fmt.Errorf("create backup: %w", err))
		return err
	}

	// Step 3: Replace binary with new version
	a.updateMessage("Installing new binary")
	if err := a.replaceBinary(newBinaryPath); err != nil {
		// Attempt rollback
		a.updateMessage("Installation failed, rolling back")
		if rollbackErr := a.Rollback(); rollbackErr != nil {
			a.setError(fmt.Errorf("replace failed (%w) and rollback failed (%w)", err, rollbackErr))
			return fmt.Errorf("replace failed: %w; rollback also failed: %w", err, rollbackErr)
		}
		a.setError(fmt.Errorf("replace binary: %w (rolled back)", err))
		return err
	}

	// Step 4: Set permissions
	a.updateMessage("Setting permissions")
	//nolint:gosec // G302: Executable needs 755 permissions
	if err := os.Chmod(a.binaryPath, 0o755); err != nil {
		// Non-fatal, continue
		_ = err
	}

	a.mu.Lock()
	a.status.State = StateComplete
	a.status.Message = "Update applied successfully. Restart required."
	a.status.Progress = progressComplete
	a.mu.Unlock()

	return nil
}

// verifyNewBinary verifies the new binary is valid.
func (a *Applier) verifyNewBinary(newBinaryPath string) error {
	info, err := os.Stat(newBinaryPath)
	if err != nil {
		return fmt.Errorf("stat new binary: %w", err)
	}

	if info.IsDir() {
		return errors.New("new binary is a directory")
	}

	if info.Size() == 0 {
		return errors.New("new binary is empty")
	}

	// Check if it's executable (basic check)
	if runtime.GOOS != "windows" {
		if info.Mode()&0o111 == 0 {
			// Try to make it executable
			//nolint:gosec // G302: Executable needs 755 permissions
			if chmodErr := os.Chmod(newBinaryPath, 0o755); chmodErr != nil {
				return fmt.Errorf("make executable: %w", chmodErr)
			}
		}
	}

	return nil
}

// createBackup creates a backup of the current binary.
func (a *Applier) createBackup() error {
	// Remove old backup if exists
	_ = os.Remove(a.backupPath)

	// Copy current binary to backup location
	// Note: We use copy instead of rename to ensure we can still recover if the binary is currently running
	if err := copyFile(a.binaryPath, a.backupPath); err != nil {
		return fmt.Errorf("copy to backup: %w", err)
	}

	return nil
}

// replaceBinary replaces the current binary with the new one.
func (a *Applier) replaceBinary(newBinaryPath string) error {
	// On Windows, we can't replace a running binary directly
	// Use a different approach: rename current, copy new, delete old
	if runtime.GOOS == "windows" {
		return a.replaceWindowsBinary(newBinaryPath)
	}

	// On Unix, we can atomically replace the binary
	// The running process keeps using the old binary in memory
	if renameErr := os.Rename(newBinaryPath, a.binaryPath); renameErr != nil {
		// If rename fails (cross-device), try copy
		if copyErr := copyFile(newBinaryPath, a.binaryPath); copyErr != nil {
			return fmt.Errorf("copy new binary: %w", copyErr)
		}
		// Clean up the source
		_ = os.Remove(newBinaryPath)
	}

	return nil
}

// replaceWindowsBinary handles binary replacement on Windows.
func (a *Applier) replaceWindowsBinary(newBinaryPath string) error {
	// On Windows, rename current binary to .old
	oldPath := a.binaryPath + ".old"
	_ = os.Remove(oldPath)

	if err := os.Rename(a.binaryPath, oldPath); err != nil {
		return fmt.Errorf("rename current binary: %w", err)
	}

	// Copy new binary to target location
	if err := copyFile(newBinaryPath, a.binaryPath); err != nil {
		// Rollback: restore old binary
		_ = os.Rename(oldPath, a.binaryPath)
		return fmt.Errorf("copy new binary: %w", err)
	}

	// Clean up
	_ = os.Remove(oldPath)
	_ = os.Remove(newBinaryPath)

	return nil
}

// Rollback reverts to the backup binary.
func (a *Applier) Rollback() error {
	a.mu.Lock()
	a.status.State = StateApplying
	a.status.Message = "Rolling back to previous version"
	a.mu.Unlock()

	// Check if backup exists
	if _, err := os.Stat(a.backupPath); os.IsNotExist(err) {
		return errors.New("no backup available")
	}

	// Copy backup back to original location
	if err := copyFile(a.backupPath, a.binaryPath); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	a.mu.Lock()
	a.status.State = StateRolledBack
	a.status.Message = "Rolled back to previous version"
	a.mu.Unlock()

	return nil
}

// CleanupBackup removes the backup file after successful update verification.
func (a *Applier) CleanupBackup() error {
	return os.Remove(a.backupPath)
}

// GetStatus returns the current applier status.
func (a *Applier) GetStatus() UpdateStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// GetBinaryPath returns the path to the current binary.
func (a *Applier) GetBinaryPath() string {
	return a.binaryPath
}

// setError sets an error state.
func (a *Applier) setError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.State = StateFailed
	a.status.Error = err.Error()
}

// updateMessage updates the status message.
func (a *Applier) updateMessage(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.Message = msg
}

// RestartService attempts to restart the service.
// This is platform-specific and may not work in all scenarios.
func (a *Applier) RestartService(ctx context.Context) error {
	a.mu.Lock()
	a.status.State = StateRestarting
	a.status.Message = "Restarting service"
	a.mu.Unlock()

	// Get the binary path
	binaryPath := a.binaryPath

	// Create a detached process to restart the service
	// This depends on how the service is run (systemd, launchd, direct, etc.)

	// For now, we'll just signal that a restart is needed
	// The actual restart mechanism depends on the deployment
	switch runtime.GOOS {
	case "linux":
		// Try systemd restart if available
		if err := exec.CommandContext(ctx, "systemctl", "restart", "seed").Run(); err != nil {
			// Fall back to direct restart
			return a.directRestart(ctx, binaryPath)
		}
	case "darwin":
		// Try launchctl restart if available
		launchCmd := exec.CommandContext(
			ctx, "launchctl", "kickstart", "-k", "system/com.seed.service")
		if err := launchCmd.Run(); err != nil {
			// Fall back to direct restart
			return a.directRestart(ctx, binaryPath)
		}
	default:
		return a.directRestart(ctx, binaryPath)
	}

	return nil
}

// directRestart starts a new process and exits the current one.
func (a *Applier) directRestart(ctx context.Context, binaryPath string) error {
	// Start new process
	//nolint:gosec // G204: binaryPath is the known executable path, not user input
	cmd := exec.CommandContext(ctx, binaryPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start new process: %w", err)
	}

	// Exit current process
	// Note: This is a hard exit - callers should ensure cleanup is done first
	os.Exit(0)

	return nil // Never reached
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) (err error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	// Get source file info for permissions
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	// destFile is writable: a failed Close (e.g. a delayed write-back error)
	// can mean data loss, so surface it instead of discarding it — but only
	// if ReadFrom itself didn't already fail.
	defer func() {
		if closeErr := destFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close copied file %s: %w", dst, closeErr)
		}
	}()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
