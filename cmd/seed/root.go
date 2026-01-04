// Package main is the entry point for The Seed by Mustard Seed Networks.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/version"
)

var (
	cfgFile        string
	devMode        bool
	trustedProxies string
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "seed",
	Short: "The Seed - Network Diagnostics by Mustard Seed Networks",
	Long: fmt.Sprintf(`The Seed %s - Network Diagnostics by Mustard Seed Networks

A comprehensive network diagnostic tool that provides:`, version.GetVersion()) + `

  - Network device discovery and monitoring
  - WiFi site surveys and heatmaps
  - Speed testing and performance analysis
  - DHCP rogue detection
  - Vulnerability scanning
  - Cable diagnostics
  - VLAN management
  - Real-time network monitoring

The Seed runs as a web server with a modern React-based UI.`,
	// When no subcommand is specified, run the serve command
	Run: func(cmd *cobra.Command, args []string) {
		runServe(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	initCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initCommands initializes all subcommands and their flags.
// This replaces init() functions to satisfy gochecknoinits linter.
func initCommands() {
	// Set version for --version flag and help output
	rootCmd.Version = version.GetVersion()

	// Persistent flags are available to all subcommands
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file path (default: XDG config paths or configs/seed.yaml)")
	rootCmd.PersistentFlags().
		BoolVar(&devMode, "dev", false, "run in development mode (HTTP instead of HTTPS)")
	rootCmd.PersistentFlags().
		StringVar(&trustedProxies, "trusted-proxies", "", "comma-separated list of trusted proxy IPs/CIDRs for X-Forwarded-For")

	// Add completion command
	rootCmd.AddCommand(completionCmd)

	// Initialize all subcommands
	initCredentialsCmd()
	initExportCmd()
	initInstallCmd()
	initMCPCmd()
	initResetCmd()
	initServeCmd()
	initSetupCmd()
	initUninstallCmd()
	initValidateCmd()
	initVersionCmd()
}

// completionCmd represents the completion command.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:
  $ source <(seed completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ seed completion bash > /etc/bash_completion.d/seed
  # macOS:
  $ seed completion bash > $(brew --prefix)/etc/bash_completion.d/seed

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ seed completion zsh > "${fpath[1]}/_seed"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ seed completion fish | source

  # To load completions for each session, execute once:
  $ seed completion fish > ~/.config/fish/completions/seed.fish

PowerShell:
  PS> seed completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> seed completion powershell > seed.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating completion: %v\n", err)
			os.Exit(1)
		}
	},
}
