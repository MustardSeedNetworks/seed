package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

var outputJSON bool

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Check setup status",
	Long: `Check if initial setup is required and display setup instructions.

This command checks whether The Seed has been configured with a secure
password. If setup is required, it provides instructions for accessing
the web-based setup wizard.

The setup wizard allows you to:
  - Set your admin password
  - Optionally auto-generate a secure password
  - Complete initial configuration

Use the --json flag to output the status in machine-readable JSON format.`,
	Run: runCredentials,
}

func initCredentialsCmd() {
	credentialsCmd.Flags().BoolVar(&outputJSON, "json", false, "output status as JSON")
	rootCmd.AddCommand(credentialsCmd)
}

func runCredentials(_ *cobra.Command, _ []string) {
	// Resolve config path using paths package
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

	// Load or create config
	cfg, result, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Determine setup status
	needsSetup := errors.Is(err, config.ErrInsecureCredentials) || result.GeneratedCreds
	protocol := "https"
	if !cfg.Server.HTTPS {
		protocol = "http"
	}

	// Prepare status output
	status := struct {
		NeedsSetup bool   `json:"needs_setup"`
		Username   string `json:"username"`
		URL        string `json:"url"`
		Message    string `json:"message"`
	}{
		NeedsSetup: needsSetup,
		Username:   cfg.Auth.DefaultUsername,
		URL:        fmt.Sprintf("%s://localhost:%d", protocol, cfg.Server.Port),
	}

	if needsSetup {
		status.Message = "Initial setup required. Visit the web UI to set your admin password."
	} else {
		status.Message = "Setup complete. Use the web UI to change your password if needed."
	}

	// Output status
	if outputJSON {
		var jsonData []byte
		jsonData, err = json.MarshalIndent(status, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal status: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, string(jsonData))
	} else {
		fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════════╗")
		fmt.Fprintln(os.Stdout, "║              THE SEED - SETUP STATUS                             ║")
		fmt.Fprintln(os.Stdout, "║              Mustard Seed Networks                               ║")
		fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════════╣")
		if needsSetup {
			fmt.Fprintln(os.Stdout, "║  Status: SETUP REQUIRED                                          ║")
			fmt.Fprintln(os.Stdout, "║                                                                  ║")
			fmt.Fprintln(os.Stdout, "║  Please open your web browser and navigate to:                   ║")
			fmt.Fprintf(os.Stdout, "║    %-62s ║\n", status.URL)
			fmt.Fprintln(os.Stdout, "║                                                                  ║")
			fmt.Fprintln(os.Stdout, "║  The setup wizard will guide you through:                        ║")
			fmt.Fprintln(os.Stdout, "║    - Setting your admin password                                 ║")
			fmt.Fprintln(os.Stdout, "║    - Optionally auto-generating a secure password               ║")
			fmt.Fprintln(os.Stdout, "║    - Initial configuration                                       ║")
		} else {
			fmt.Fprintln(os.Stdout, "║  Status: SETUP COMPLETE                                          ║")
			fmt.Fprintln(os.Stdout, "║                                                                  ║")
			fmt.Fprintf(os.Stdout, "║  Username: %-53s ║\n", status.Username)
			fmt.Fprintln(os.Stdout, "║                                                                  ║")
			fmt.Fprintln(os.Stdout, "║  To change your password, use the web UI Settings panel.         ║")
			fmt.Fprintln(os.Stdout, "║  If you've lost access, delete the config file and restart.     ║")
		}
		fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════════╝")
	}
}
