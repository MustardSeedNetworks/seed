package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

var setupCmd = &cobra.Command{
	Use:   "setup-wizard",
	Short: "Re-run the setup wizard",
	Long: `Re-run the first-time setup to reset or regenerate credentials.

This command allows you to regenerate authentication credentials without
going through the web UI. Use --generate-password to auto-generate a
secure password, or start the server and use the web wizard for
interactive setup.`,
	Run: runSetup,
}

func initSetupCmd() {
	setupCmd.Flags().Bool("generate-password", false, "Auto-generate a secure password")
	setupCmd.Flags().Bool("json", false, "Output credentials as JSON")
	setupCmd.Flags().Bool("reset-jwt", false, "Also regenerate the JWT secret")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, _ []string) {
	generatePwd, err := cmd.Flags().GetBool("generate-password")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting generate-password flag: %v\n", err)
		os.Exit(1)
	}
	outputAsJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting json flag: %v\n", err)
		os.Exit(1)
	}
	resetJWT, err := cmd.Flags().GetBool("reset-jwt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting reset-jwt flag: %v\n", err)
		os.Exit(1)
	}

	// Resolve config path
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

	// Load or create config
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if generatePwd {
		// Generate new credentials
		password, genErr := auth.GenerateSecurePassword(20)
		if genErr != nil {
			fmt.Fprintf(os.Stderr, "Error generating password: %v\n", genErr)
			os.Exit(1)
		}

		passwordHash, hashErr := auth.HashPassword(password)
		if hashErr != nil {
			fmt.Fprintf(os.Stderr, "Error hashing password: %v\n", hashErr)
			os.Exit(1)
		}

		// Update config
		cfg.Auth.DefaultPasswordHash = passwordHash
		if resetJWT || cfg.Auth.JWTSecret == "" {
			cfg.Auth.JWTSecret = auth.GenerateJWTSecret()
		}

		// Ensure config directory exists
		if dir := filepath.Dir(configPath); dir != "" && dir != "." {
			if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
				fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", mkdirErr)
				os.Exit(1)
			}
		}

		// Save config
		if saveErr := cfg.Save(configPath); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", saveErr)
			os.Exit(1)
		}

		// Output credentials
		creds := struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Config   string `json:"config_path"`
		}{
			Username: cfg.Auth.DefaultUsername,
			Password: password,
			Config:   configPath,
		}

		if outputAsJSON {
			data, marshalErr := json.MarshalIndent(creds, "", "  ")
			if marshalErr != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling credentials: %v\n", marshalErr)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stdout, string(data))
		} else {
			fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════════╗")
			fmt.Fprintln(os.Stdout, "║              THE SEED - CREDENTIALS GENERATED                    ║")
			fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════════╣")
			fmt.Fprintf(os.Stdout, "║  Username: %-53s ║\n", creds.Username)
			fmt.Fprintf(os.Stdout, "║  Password: %-53s ║\n", creds.Password)
			fmt.Fprintln(os.Stdout, "║                                                                  ║")
			fmt.Fprintln(os.Stdout, "║  IMPORTANT: Save this password securely!                         ║")
			fmt.Fprintln(os.Stdout, "║  It will not be shown again.                                     ║")
			fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════════╝")
		}
	} else {
		// Clear password to trigger web wizard
		cfg.Auth.DefaultPasswordHash = ""
		if resetJWT {
			cfg.Auth.JWTSecret = ""
		}

		if saveErr := cfg.Save(configPath); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", saveErr)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stdout, "Setup wizard has been reset.")
		fmt.Fprintln(os.Stdout, "Start the server and visit the web UI to set your password.")
		fmt.Fprintf(os.Stdout, "\nConfig: %s\n", configPath)
	}
}
