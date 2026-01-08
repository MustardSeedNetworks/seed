package main

import (
	"bytes"
	"io"
	"os"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
	"github.com/spf13/cobra"
)

// ExportedCLIState exposes cliState for testing.
type ExportedCLIState = cliState

// ExportedInstallFlags exposes installFlags for testing.
type ExportedInstallFlags = installFlags

// ExportedResetFlags exposes resetFlags for testing.
type ExportedResetFlags = resetFlags

// ExportedUninstallFlags exposes uninstallFlags for testing.
type ExportedUninstallFlags = uninstallFlags

// ExportedServiceConfig exposes serviceConfig for testing.
type ExportedServiceConfig = serviceConfig

// ExportedDistro exposes Distro for testing.
type ExportedDistro = Distro

// ExportedSetupCredentials exposes setupCredentials for testing.
type ExportedSetupCredentials = setupCredentials

// ExportedValidationResult exposes ValidationResult for testing.
type ExportedValidationResult = ValidationResult

// NewTestCLIState creates a new CLI state for testing.
func NewTestCLIState() *cliState {
	return newCLIState()
}

// InitTestCommands initializes commands on a CLI state for testing.
func InitTestCommands(state *cliState) {
	initCommands(state)
}

// ExportModeString exposes modeString for testing.
func ExportModeString(mode paths.Mode) string {
	return modeString(mode)
}

// ExportParseOSRelease exposes parseOSRelease for testing.
func ExportParseOSRelease(content string) *Distro {
	return parseOSRelease(content)
}

// ExportCheckConfigWarnings exposes checkConfigWarnings for testing.
func ExportCheckConfigWarnings(cfg *config.Config) []string {
	return checkConfigWarnings(cfg)
}

// ExportRedactSecrets exposes redactSecrets for testing.
func ExportRedactSecrets(cfg *config.Config) *config.Config {
	return redactSecrets(cfg)
}

// ExportPreserveExistingCredentials exposes preserveExistingCredentials for testing.
func ExportPreserveExistingCredentials(newCfg, existingCfg *config.Config, flags resetFlags) {
	preserveExistingCredentials(newCfg, existingCfg, flags)
}

// ExportDetermineUninstallMode exposes determineUninstallMode for testing.
func ExportDetermineUninstallMode(flags uninstallFlags) paths.Mode {
	return determineUninstallMode(flags)
}

// ExportEnsureConfigDir exposes ensureConfigDir for testing.
func ExportEnsureConfigDir(configPath string) error {
	return ensureConfigDir(configPath)
}

// ExportOutputCredentials exposes outputCredentials for testing.
func ExportOutputCredentials(creds setupCredentials, asJSON bool) error {
	return outputCredentials(creds, asJSON)
}

// ExportOutputResult exposes outputResult for testing.
func ExportOutputResult(result ValidationResult, asJSON bool) {
	outputResult(result, asJSON)
}

// FindCommand finds a subcommand by name for testing.
func FindCommand(state *cliState, name string) *cobra.Command {
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == name || hasPrefix(cmd.Use, name+" ") {
			return cmd
		}
	}
	return nil
}

// hasPrefix is a helper to check if s starts with prefix.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// CaptureStdout captures stdout output during function execution.
func CaptureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// CaptureStderr captures stderr output during function execution.
func CaptureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// GetLogBroadcasterBufferSize returns the log broadcaster buffer size constant.
func GetLogBroadcasterBufferSize() int {
	return logBroadcasterBufferSize
}

// GetSignalChannelBufferSize returns the signal channel buffer size constant.
func GetSignalChannelBufferSize() int {
	return signalChannelBufferSize
}

// GetShutdownTimeoutSeconds returns the shutdown timeout constant.
func GetShutdownTimeoutSeconds() int {
	return shutdownTimeoutSeconds
}

// GetUserCheckTimeoutSeconds returns the user check timeout constant.
func GetUserCheckTimeoutSeconds() int {
	return userCheckTimeoutSeconds
}

// GetCommandTimeoutSeconds returns the command timeout constant.
func GetCommandTimeoutSeconds() int {
	return commandTimeoutSeconds
}

// GetDefaultPasswordLength returns the default password length constant.
func GetDefaultPasswordLength() int {
	return defaultPasswordLength
}

// GetRedactedValue returns the redacted value constant.
func GetRedactedValue() string {
	return redactedValue
}

// GetSystemdServiceTemplate returns the systemd service template.
func GetSystemdServiceTemplate() string {
	return systemdServiceTemplate
}

// GetUserServiceTemplate returns the user service template.
func GetUserServiceTemplate() string {
	return userServiceTemplate
}

// GetExpectedLinuxReleaseParts returns the expected Linux release parts constant.
func GetExpectedLinuxReleaseParts() int {
	return expectedLinuxReleaseParts
}
