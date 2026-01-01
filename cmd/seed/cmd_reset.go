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

var resetCmd = &cobra.Command{
	Use:   "reset-config",
	Short: "Reset configuration to defaults",
	Long: `Reset configuration to defaults.

By default, this will create a backup of the current config and replace it
with a fresh default configuration. Authentication credentials can optionally
be preserved.`,
	Run: runReset,
}

func init() {
	resetCmd.Flags().Bool("preserve-auth", false, "Preserve authentication credentials")
	resetCmd.Flags().Bool("preserve-jwt", false, "Preserve JWT secret")
	resetCmd.Flags().Bool("backup", true, "Create backup before reset")
	resetCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, _ []string) {
	preserveAuth, err := cmd.Flags().GetBool("preserve-auth")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting preserve-auth flag: %v\n", err)
		os.Exit(1)
	}
	preserveJWT, err := cmd.Flags().GetBool("preserve-jwt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting preserve-jwt flag: %v\n", err)
		os.Exit(1)
	}
	backup, err := cmd.Flags().GetBool("backup")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting backup flag: %v\n", err)
		os.Exit(1)
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting force flag: %v\n", err)
		os.Exit(1)
	}

	// Resolve config path
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

	// Load existing config if it exists (for preservation)
	var existingCfg *config.Config
	if _, statErr := os.Stat(configPath); statErr == nil {
		// Errors loading existing config are not fatal during reset
		existingCfg, _ = config.Load(configPath)
	}

	// Confirm unless --force
	if !force {
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
		if response != "y" && response != "yes" {
			fmt.Fprintln(os.Stdout, "Aborted.")
			return
		}
	}

	// Create backup
	if backup && existingCfg != nil {
		backupMgr := config.NewBackupManager(configPath, "", 10)
		backupInfo, backupErr := backupMgr.CreateBackup()
		if backupErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create backup: %v\n", backupErr)
		} else {
			fmt.Fprintf(os.Stdout, "Backup created: %s\n", backupInfo.Path)
		}
	}

	// Create new default config
	newCfg := config.DefaultConfig()

	// Preserve credentials if requested
	if preserveAuth && existingCfg != nil {
		newCfg.Auth.DefaultUsername = existingCfg.Auth.DefaultUsername
		newCfg.Auth.DefaultPasswordHash = existingCfg.Auth.DefaultPasswordHash
	}
	if preserveJWT && existingCfg != nil {
		newCfg.Auth.JWTSecret = existingCfg.Auth.JWTSecret
	}

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
	if !preserveAuth {
		fmt.Fprintln(os.Stdout, "\nNOTE: You will need to re-run the setup wizard to set your password.")
		fmt.Fprintln(os.Stdout, "Start the server and visit the web UI to complete setup.")
	}
}
