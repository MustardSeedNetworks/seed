package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

// resetFlags holds the parsed flags for the reset command.
type resetFlags struct {
	preserveAuth bool
	preserveJWT  bool
	backup       bool
	force        bool
}

func initResetCmd(state *cliState) {
	resetCmd := &cobra.Command{
		Use:   "reset-config",
		Short: "Reset configuration to defaults",
		Long: `Reset configuration to defaults.

By default, this will create a backup of the current config and replace it
with a fresh default configuration. Authentication credentials can optionally
be preserved.`,
		Run: func(cmd *cobra.Command, args []string) {
			runReset(cmd, args, state)
		},
	}
	resetCmd.Flags().Bool("preserve-auth", false, "Preserve authentication credentials")
	resetCmd.Flags().Bool("preserve-jwt", false, "Preserve JWT secret")
	resetCmd.Flags().Bool("backup", true, "Create backup before reset")
	resetCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	state.rootCmd.AddCommand(resetCmd)
}

// parseResetFlags extracts and validates all reset command flags.
func parseResetFlags(cmd *cobra.Command) (resetFlags, error) {
	var flags resetFlags
	var err error

	flags.preserveAuth, err = cmd.Flags().GetBool("preserve-auth")
	if err != nil {
		return flags, fmt.Errorf("getting preserve-auth flag: %w", err)
	}
	flags.preserveJWT, err = cmd.Flags().GetBool("preserve-jwt")
	if err != nil {
		return flags, fmt.Errorf("getting preserve-jwt flag: %w", err)
	}
	flags.backup, err = cmd.Flags().GetBool("backup")
	if err != nil {
		return flags, fmt.Errorf("getting backup flag: %w", err)
	}
	flags.force, err = cmd.Flags().GetBool("force")
	if err != nil {
		return flags, fmt.Errorf("getting force flag: %w", err)
	}
	return flags, nil
}

// confirmReset prompts the user for confirmation before resetting.
// Returns true if the user confirms, false otherwise.
func confirmReset(configPath string, preserveAuth bool) bool {
	fmt.Fprintf(os.Stdout, "This will reset the configuration at:\n  %s\n\n", configPath)
	if preserveAuth {
		fmt.Fprintln(os.Stdout, "Authentication credentials WILL be preserved.")
	} else {
		fmt.Fprintln(os.Stdout, "WARNING: Authentication credentials will be LOST!")
		fmt.Fprintln(os.Stdout, "Use --preserve-auth to keep your username and password.")
	}
	fmt.Fprint(os.Stdout, "\nContinue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, readErr := reader.ReadString('\n')
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", readErr)
		os.Exit(1)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// createConfigBackup creates a backup of the existing configuration.
func createConfigBackup(configPath string) {
	const maxBackups = 10
	backupMgr := config.NewBackupManager(configPath, "", maxBackups)
	backupInfo, backupErr := backupMgr.CreateBackup()
	if backupErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create backup: %v\n", backupErr)
	} else {
		fmt.Fprintf(os.Stdout, "Backup created: %s\n", backupInfo.Path)
	}
}

// preserveExistingCredentials copies credentials from existing config to new config.
func preserveExistingCredentials(newCfg, existingCfg *config.Config, flags resetFlags) {
	if existingCfg == nil {
		return
	}
	if flags.preserveAuth {
		newCfg.Auth.DefaultUsername = existingCfg.Auth.DefaultUsername
		newCfg.Auth.DefaultPasswordHash = existingCfg.Auth.DefaultPasswordHash
	}
	if flags.preserveJWT {
		newCfg.Auth.JWTSecret = existingCfg.Auth.JWTSecret
	}
}

func runReset(cmd *cobra.Command, _ []string, state *cliState) {
	flags, err := parseResetFlags(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	configPath := paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto)

	// Load existing config if it exists (for preservation)
	var existingCfg *config.Config
	if _, statErr := os.Stat(configPath); statErr == nil {
		existingCfg, _ = config.Load(configPath)
	}

	// Confirm unless --force
	if !flags.force && !confirmReset(configPath, flags.preserveAuth) {
		fmt.Fprintln(os.Stdout, "Aborted.")
		return
	}

	// Create backup
	if flags.backup && existingCfg != nil {
		createConfigBackup(configPath)
	}

	// Create new default config and preserve credentials if requested
	newCfg := config.DefaultConfig()
	preserveExistingCredentials(newCfg, existingCfg, flags)

	// Ensure config directory exists
	if dir := filepath.Dir(configPath); dir != "" && dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", mkdirErr)
			os.Exit(1)
		}
	}

	// Save new config
	if saveErr := newCfg.Save(configPath); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", saveErr)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Configuration reset to defaults at: %s\n", configPath)
	if !flags.preserveAuth {
		fmt.Fprintln(
			os.Stdout,
			"\nNOTE: You will need to re-run the setup wizard to set your password.",
		)
		fmt.Fprintln(os.Stdout, "Start the server and visit the web UI to complete setup.")
	}
}
