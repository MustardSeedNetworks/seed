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

func init() {
	setupCmd.Flags().Bool("generate-password", false, "Auto-generate a secure password")
	setupCmd.Flags().Bool("json", false, "Output credentials as JSON")
	setupCmd.Flags().Bool("reset-jwt", false, "Also regenerate the JWT secret")
	rootCmd.AddCommand(setupCmd)
}

//nolint:gocyclo // Command handler complexity is acceptable
func runSetup(cmd *cobra.Command, _ []string) {
	generatePwd, err := cmd.Flags().GetBool("generate-password")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting generate-password flag: %v\n", err)
		os.Exit(1)
	}
	outputJSON, err := cmd.Flags().GetBool("json")
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
		password, err := auth.GenerateSecurePassword(20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating password: %v\n", err)
			os.Exit(1)
		}

		passwordHash, err := auth.HashPassword(password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error hashing password: %v\n", err)
			os.Exit(1)
		}

		// Update config
		cfg.Auth.DefaultPasswordHash = passwordHash
		if resetJWT || cfg.Auth.JWTSecret == "" {
			cfg.Auth.JWTSecret = auth.GenerateJWTSecret()
		}

		// Ensure config directory exists
		if dir := filepath.Dir(configPath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Save config
		if err := cfg.Save(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
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

		if outputJSON {
			data, err := json.MarshalIndent(creds, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling credentials: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(data))
		} else {
			fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
			fmt.Println("║              THE SEED - CREDENTIALS GENERATED                    ║")
			fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
			fmt.Printf("║  Username: %-53s ║\n", creds.Username)
			fmt.Printf("║  Password: %-53s ║\n", creds.Password)
			fmt.Println("║                                                                  ║")
			fmt.Println("║  IMPORTANT: Save this password securely!                         ║")
			fmt.Println("║  It will not be shown again.                                     ║")
			fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
		}
	} else {
		// Clear password to trigger web wizard
		cfg.Auth.DefaultPasswordHash = ""
		if resetJWT {
			cfg.Auth.JWTSecret = ""
		}

		if err := cfg.Save(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Setup wizard has been reset.")
		fmt.Println("Start the server and visit the web UI to set your password.")
		fmt.Printf("\nConfig: %s\n", configPath)
	}
}
