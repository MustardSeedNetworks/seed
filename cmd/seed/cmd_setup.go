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

const (
	// defaultPasswordLength is the number of characters for auto-generated secure passwords.
	defaultPasswordLength = 20
)

func initSetupCmd(state *cliState) {
	setupCmd := &cobra.Command{
		Use:   "setup-wizard",
		Short: "Re-run the setup wizard",
		Long: `Re-run the first-time setup to reset or regenerate credentials.

This command allows you to regenerate authentication credentials without
going through the web UI. Use --generate-password to auto-generate a
secure password, or start the server and use the web wizard for
interactive setup.`,
		Example: `  # Generate a strong password and print it once
  seed setup-wizard --generate-password

  # Generate a password and emit machine-readable output
  seed setup-wizard --generate-password --json

  # Reset back to the first-run web wizard
  seed setup-wizard

  # Also rotate the JWT secret (forces all sessions to log in again)
  seed setup-wizard --generate-password --reset-jwt`,
		Run: func(cmd *cobra.Command, args []string) {
			runSetup(cmd, args, state)
		},
	}
	setupCmd.Flags().Bool("generate-password", false, "Auto-generate a secure password")
	setupCmd.Flags().Bool("json", false, "Output credentials as JSON")
	setupCmd.Flags().Bool("reset-jwt", false, "Also regenerate the JWT secret")
	state.rootCmd.AddCommand(setupCmd)
}

// setupCredentials holds the generated credentials for output.
//
// The Password field is intentionally included in JSON output: this struct
// is only used by `seed setup --json`, which is the bootstrap CLI invocation
// that GENERATES the initial admin password and shows it to the operator
// once. After setup runs, only the bcrypt hash is persisted; this struct
// is never serialized to disk or transmitted over a network.
type setupCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Config   string `json:"config_path"`
}

// generatePasswordAndHash creates a secure password and returns it with its hash.
func generatePasswordAndHash() (string, string, error) {
	password, err := auth.GenerateSecurePassword(defaultPasswordLength)
	if err != nil {
		return "", "", fmt.Errorf("generating password: %w", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", "", fmt.Errorf("hashing password: %w", err)
	}
	return password, hash, nil
}

// ensureConfigDir creates the config directory if needed.
func ensureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return nil
}

// outputCredentials writes credentials to stdout in the requested format.
func outputCredentials(creds setupCredentials, asJSON bool) error {
	if asJSON {
		//nolint:gosec // G117: bootstrap-only output to operator on first-run setup; Password is intentionally shown once and never persisted in this form
		data, err := json.MarshalIndent(creds, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling credentials: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stdout, "║              THE SEED - CREDENTIALS GENERATED                    ║")
	fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════════╣")
	fmt.Fprintf(os.Stdout, "║  Username: %-53s ║\n", creds.Username)
	fmt.Fprintf(os.Stdout, "║  Password: %-53s ║\n", creds.Password)
	fmt.Fprintln(os.Stdout, "║                                                                  ║")
	fmt.Fprintln(os.Stdout, "║  IMPORTANT: Save this password securely!                         ║")
	fmt.Fprintln(os.Stdout, "║  It will not be shown again.                                     ║")
	fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════════╝")
	return nil
}

// runSetupWithGeneratedPassword handles the --generate-password flow.
func runSetupWithGeneratedPassword(cfg *config.Config, configPath string, resetJWT, outputAsJSON bool) {
	password, passwordHash, genErr := generatePasswordAndHash()
	if genErr != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", genErr)
		os.Exit(1)
	}

	cfg.Auth.DefaultPasswordHash = passwordHash
	if resetJWT || cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = auth.GenerateJWTSecret()
	}

	if dirErr := ensureConfigDir(configPath); dirErr != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", dirErr)
		os.Exit(1)
	}

	if saveErr := cfg.Save(configPath); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", saveErr)
		os.Exit(1)
	}

	creds := setupCredentials{
		Username: cfg.Auth.DefaultUsername,
		Password: password,
		Config:   configPath,
	}
	if outErr := outputCredentials(creds, outputAsJSON); outErr != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", outErr)
		os.Exit(1)
	}
}

// runSetupWebWizard handles the web wizard reset flow.
func runSetupWebWizard(cfg *config.Config, configPath string, resetJWT bool) {
	cfg.Auth.DefaultPasswordHash = ""
	if resetJWT {
		cfg.Auth.JWTSecret = ""
	}

	if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "Setup wizard has been reset.")
	fmt.Fprintln(os.Stdout, "Start the server and visit the web UI to set your password.")
	fmt.Fprintf(os.Stdout, "\nConfig: %s\n", configPath)
}

func runSetup(cmd *cobra.Command, _ []string, state *cliState) {
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

	configPath := paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto)

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if generatePwd {
		runSetupWithGeneratedPassword(cfg, configPath, resetJWT, outputAsJSON)
		return
	}
	runSetupWebWizard(cfg, configPath, resetJWT)
}
