package main

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/paths"
)

func TestParseInstallFlagsValidation(t *testing.T) {

	tests := []struct {
		name       string
		systemMode bool
		userMode   bool
		wantErr    bool
	}{
		{
			name:       "both false is valid",
			systemMode: false,
			userMode:   false,
			wantErr:    false,
		},
		{
			name:       "system only is valid",
			systemMode: true,
			userMode:   false,
			wantErr:    false,
		},
		{
			name:       "user only is valid",
			systemMode: false,
			userMode:   true,
			wantErr:    false,
		},
		{
			name:       "both true is invalid",
			systemMode: true,
			userMode:   true,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			// Create a test command with the flags
			cmd := &cobra.Command{}
			cmd.Flags().Bool("system", tc.systemMode, "")
			cmd.Flags().Bool("user", tc.userMode, "")
			cmd.Flags().Bool("no-service", false, "")
			cmd.Flags().Bool("force", false, "")

			// Set the flag values
			_ = cmd.Flags().Set("system", boolToString(tc.systemMode))
			_ = cmd.Flags().Set("user", boolToString(tc.userMode))

			flags, err := parseInstallFlags(cmd)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseInstallFlags() error = %v, wantErr %v", err, tc.wantErr)
			}

			if err == nil {
				if flags.systemMode != tc.systemMode {
					t.Errorf("systemMode: got %v, want %v", flags.systemMode, tc.systemMode)
				}
				if flags.userMode != tc.userMode {
					t.Errorf("userMode: got %v, want %v", flags.userMode, tc.userMode)
				}
			}
		})
	}
}

func TestParseResetFlagsValidation(t *testing.T) {

	tests := []struct {
		name         string
		preserveAuth bool
		preserveJWT  bool
		backup       bool
		force        bool
	}{
		{
			name:         "all false",
			preserveAuth: false,
			preserveJWT:  false,
			backup:       false,
			force:        false,
		},
		{
			name:         "preserve auth",
			preserveAuth: true,
			preserveJWT:  false,
			backup:       false,
			force:        false,
		},
		{
			name:         "preserve jwt",
			preserveAuth: false,
			preserveJWT:  true,
			backup:       false,
			force:        false,
		},
		{
			name:         "backup enabled",
			preserveAuth: false,
			preserveJWT:  false,
			backup:       true,
			force:        false,
		},
		{
			name:         "force enabled",
			preserveAuth: false,
			preserveJWT:  false,
			backup:       false,
			force:        true,
		},
		{
			name:         "all enabled",
			preserveAuth: true,
			preserveJWT:  true,
			backup:       true,
			force:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			cmd := &cobra.Command{}
			cmd.Flags().Bool("preserve-auth", tc.preserveAuth, "")
			cmd.Flags().Bool("preserve-jwt", tc.preserveJWT, "")
			cmd.Flags().Bool("backup", tc.backup, "")
			cmd.Flags().Bool("force", tc.force, "")

			_ = cmd.Flags().Set("preserve-auth", boolToString(tc.preserveAuth))
			_ = cmd.Flags().Set("preserve-jwt", boolToString(tc.preserveJWT))
			_ = cmd.Flags().Set("backup", boolToString(tc.backup))
			_ = cmd.Flags().Set("force", boolToString(tc.force))

			flags, err := parseResetFlags(cmd)
			if err != nil {
				t.Fatalf("parseResetFlags() unexpected error: %v", err)
			}

			if flags.preserveAuth != tc.preserveAuth {
				t.Errorf("preserveAuth: got %v, want %v", flags.preserveAuth, tc.preserveAuth)
			}
			if flags.preserveJWT != tc.preserveJWT {
				t.Errorf("preserveJWT: got %v, want %v", flags.preserveJWT, tc.preserveJWT)
			}
			if flags.backup != tc.backup {
				t.Errorf("backup: got %v, want %v", flags.backup, tc.backup)
			}
			if flags.force != tc.force {
				t.Errorf("force: got %v, want %v", flags.force, tc.force)
			}
		})
	}
}

func TestParseUninstallFlagsValidation(t *testing.T) {

	tests := []struct {
		name       string
		purge      bool
		force      bool
		systemMode bool
		userMode   bool
	}{
		{
			name:       "all false",
			purge:      false,
			force:      false,
			systemMode: false,
			userMode:   false,
		},
		{
			name:       "purge enabled",
			purge:      true,
			force:      false,
			systemMode: false,
			userMode:   false,
		},
		{
			name:       "force enabled",
			purge:      false,
			force:      true,
			systemMode: false,
			userMode:   false,
		},
		{
			name:       "system mode",
			purge:      false,
			force:      false,
			systemMode: true,
			userMode:   false,
		},
		{
			name:       "user mode",
			purge:      false,
			force:      false,
			systemMode: false,
			userMode:   true,
		},
		{
			name:       "purge and force",
			purge:      true,
			force:      true,
			systemMode: false,
			userMode:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			cmd := &cobra.Command{}
			cmd.Flags().Bool("purge", tc.purge, "")
			cmd.Flags().Bool("force", tc.force, "")
			cmd.Flags().Bool("system", tc.systemMode, "")
			cmd.Flags().Bool("user", tc.userMode, "")

			_ = cmd.Flags().Set("purge", boolToString(tc.purge))
			_ = cmd.Flags().Set("force", boolToString(tc.force))
			_ = cmd.Flags().Set("system", boolToString(tc.systemMode))
			_ = cmd.Flags().Set("user", boolToString(tc.userMode))

			flags, err := parseUninstallFlags(cmd)
			if err != nil {
				t.Fatalf("parseUninstallFlags() unexpected error: %v", err)
			}

			if flags.purge != tc.purge {
				t.Errorf("purge: got %v, want %v", flags.purge, tc.purge)
			}
			if flags.force != tc.force {
				t.Errorf("force: got %v, want %v", flags.force, tc.force)
			}
			if flags.systemMode != tc.systemMode {
				t.Errorf("systemMode: got %v, want %v", flags.systemMode, tc.systemMode)
			}
			if flags.userMode != tc.userMode {
				t.Errorf("userMode: got %v, want %v", flags.userMode, tc.userMode)
			}
		})
	}
}

func TestResolveInstallModeLogic(t *testing.T) {

	tests := []struct {
		name       string
		systemMode bool
		userMode   bool
		wantMode   paths.Mode
		wantErr    bool
	}{
		{
			name:       "explicit system mode",
			systemMode: true,
			userMode:   false,
			wantMode:   paths.ModeSystem,
			wantErr:    true, // Will fail if not root
		},
		{
			name:       "explicit user mode",
			systemMode: false,
			userMode:   true,
			wantMode:   paths.ModeUser,
			wantErr:    false,
		},
		{
			name:       "auto mode (non-root)",
			systemMode: false,
			userMode:   false,
			wantMode:   paths.ModeUser,
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			flags := installFlags{
				systemMode: tc.systemMode,
				userMode:   tc.userMode,
			}

			mode, err := resolveInstallMode(flags)

			// Skip system mode test if we're not root
			if tc.systemMode && err != nil {
				// Expected error when not running as root
				return
			}

			if tc.wantErr && err == nil && tc.systemMode {
				t.Error("Expected error for system mode when not root")
				return
			}

			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if err == nil && mode != tc.wantMode {
				t.Errorf("mode: got %v, want %v", mode, tc.wantMode)
			}
		})
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestInstallCmdFlagsRegistration(t *testing.T) {

	state := newCLIState()
	initInstallCmd(state)

	var installCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "install" {
			installCmd = cmd
			break
		}
	}

	if installCmd == nil {
		t.Fatal("install command not found")
	}

	expectedFlags := []string{"system", "user", "no-service", "force"}
	for _, flag := range expectedFlags {
		f := installCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("install command should have --%s flag", flag)
		}
	}

	// Check shorthand for force
	forceFlag := installCmd.Flags().ShorthandLookup("f")
	if forceFlag == nil {
		t.Error("install command should have -f shorthand for --force")
	}
}

func TestResetCmdFlagsRegistration(t *testing.T) {

	state := newCLIState()
	initResetCmd(state)

	var resetCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "reset-config" {
			resetCmd = cmd
			break
		}
	}

	if resetCmd == nil {
		t.Fatal("reset-config command not found")
	}

	expectedFlags := []string{"preserve-auth", "preserve-jwt", "backup", "force"}
	for _, flag := range expectedFlags {
		f := resetCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("reset-config command should have --%s flag", flag)
		}
	}

	// Check shorthand for force
	forceFlag := resetCmd.Flags().ShorthandLookup("f")
	if forceFlag == nil {
		t.Error("reset-config command should have -f shorthand for --force")
	}

	// Verify backup defaults to true
	backupFlag := resetCmd.Flags().Lookup("backup")
	if backupFlag != nil && backupFlag.DefValue != "true" {
		t.Error("--backup flag should default to true")
	}
}

func TestUninstallCmdFlagsRegistration(t *testing.T) {

	state := newCLIState()
	initUninstallCmd(state)

	var uninstallCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "uninstall" {
			uninstallCmd = cmd
			break
		}
	}

	if uninstallCmd == nil {
		t.Fatal("uninstall command not found")
	}

	expectedFlags := []string{"purge", "force", "system", "user"}
	for _, flag := range expectedFlags {
		f := uninstallCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("uninstall command should have --%s flag", flag)
		}
	}

	// Check shorthand for force
	forceFlag := uninstallCmd.Flags().ShorthandLookup("f")
	if forceFlag == nil {
		t.Error("uninstall command should have -f shorthand for --force")
	}
}

func TestSetupCmdFlagsRegistration(t *testing.T) {

	state := newCLIState()
	initSetupCmd(state)

	var setupCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "setup-wizard" {
			setupCmd = cmd
			break
		}
	}

	if setupCmd == nil {
		t.Fatal("setup-wizard command not found")
	}

	expectedFlags := []string{"generate-password", "json", "reset-jwt"}
	for _, flag := range expectedFlags {
		f := setupCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("setup-wizard command should have --%s flag", flag)
		}
	}
}

func TestExportCmdFlagsRegistration(t *testing.T) {

	state := newCLIState()
	initExportCmd(state)

	var exportCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "export-config" {
			exportCmd = cmd
			break
		}
	}

	if exportCmd == nil {
		t.Fatal("export-config command not found")
	}

	expectedFlags := []string{"output", "no-redact"}
	for _, flag := range expectedFlags {
		f := exportCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("export-config command should have --%s flag", flag)
		}
	}

	// Check shorthands
	outputFlag := exportCmd.Flags().ShorthandLookup("o")
	if outputFlag == nil {
		t.Error("export-config command should have -o shorthand for --output")
	}

	// Verify defaults
	outFlag := exportCmd.Flags().Lookup("output")
	if outFlag != nil && outFlag.DefValue != "-" {
		t.Error("--output flag should default to '-'")
	}
}

func TestValidateCmdFlagsRegistration(t *testing.T) {

	state := newCLIState()
	initValidateCmd(state)

	var validateCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "validate-config" {
			validateCmd = cmd
			break
		}
	}

	if validateCmd == nil {
		t.Fatal("validate-config command not found")
	}

	expectedFlags := []string{"strict", "json"}
	for _, flag := range expectedFlags {
		f := validateCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("validate-config command should have --%s flag", flag)
		}
	}
}
