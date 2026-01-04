package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/paths"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Seed",
	Long: `Uninstall Seed and optionally remove all data.

By default, configuration and data files are preserved.
Use --purge to remove all files including configuration.`,
	Run: runUninstall,
}

func initUninstallCmd() {
	uninstallCmd.Flags().Bool("purge", false, "Remove all data and configuration")
	uninstallCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	uninstallCmd.Flags().Bool("system", false, "Uninstall system service")
	uninstallCmd.Flags().Bool("user", false, "Uninstall user service")
	rootCmd.AddCommand(uninstallCmd)
}

//nolint:gocyclo // Command handler complexity is acceptable
func runUninstall(cmd *cobra.Command, _ []string) {
	if runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "Error: uninstall command is only supported on Linux\n")
		os.Exit(1)
	}

	purge, err := cmd.Flags().GetBool("purge")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting purge flag: %v\n", err)
		os.Exit(1)
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting force flag: %v\n", err)
		os.Exit(1)
	}
	systemMode, err := cmd.Flags().GetBool("system")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting system flag: %v\n", err)
		os.Exit(1)
	}
	userMode, err := cmd.Flags().GetBool("user")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting user flag: %v\n", err)
		os.Exit(1)
	}

	// Determine mode
	var mode paths.Mode
	switch {
	case systemMode:
		mode = paths.ModeSystem
	case userMode:
		mode = paths.ModeUser
	case os.Getuid() == 0:
		mode = paths.ModeSystem
	default:
		mode = paths.ModeUser
	}

	p := paths.Resolve(mode)

	// Confirm
	if !force {
		fmt.Fprintln(os.Stdout, "This will uninstall Seed:")
		fmt.Fprintf(os.Stdout, "  - Stop and disable the systemd service\n")
		fmt.Fprintf(os.Stdout, "  - Remove the binary\n")
		if purge {
			fmt.Fprintf(os.Stdout, "  - Remove all configuration in %s\n", p.ConfigDir)
			fmt.Fprintf(os.Stdout, "  - Remove all data in %s\n", p.DataDir)
			fmt.Fprintf(os.Stdout, "  - Remove all logs in %s\n", p.LogDir)
		} else {
			fmt.Fprintf(os.Stdout, "  - Keep configuration and data (use --purge to remove)\n")
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

	// Stop service (fixes #789 - log errors instead of silently ignoring)
	fmt.Fprintln(os.Stdout, "\nStopping service...")
	ctx := context.Background()
	if mode == paths.ModeSystem {
		if stopErr := exec.CommandContext(ctx, "systemctl", "stop", "seed").Run(); stopErr != nil {
			logging.GetLogger().Warn("Failed to stop seed service", "error", stopErr)
		}
		if disableErr := exec.CommandContext(ctx, "systemctl", "disable", "seed").Run(); disableErr != nil {
			logging.GetLogger().Warn("Failed to disable seed service", "error", disableErr)
		}
	} else {
		if stopErr := exec.CommandContext(ctx, "systemctl", "--user", "stop", "seed").Run(); stopErr != nil {
			logging.GetLogger().Warn("Failed to stop seed user service", "error", stopErr)
		}
		if disableErr := exec.CommandContext(ctx, "systemctl", "--user", "disable", "seed").Run(); disableErr != nil {
			logging.GetLogger().Warn("Failed to disable seed user service", "error", disableErr)
		}
	}

	// Remove service file
	fmt.Fprintln(os.Stdout, "Removing service file...")
	var servicePath string
	if mode == paths.ModeSystem {
		servicePath = "/etc/systemd/system/seed.service"
	} else {
		userConfigDir, configErr := os.UserConfigDir()
		if configErr != nil {
			logging.GetLogger().Warn("Failed to get user config dir", "error", configErr)
			return
		}
		servicePath = filepath.Join(userConfigDir, "systemd", "user", "seed.service")
	}
	if removeErr := os.Remove(servicePath); removeErr != nil && !os.IsNotExist(removeErr) {
		logging.GetLogger().Warn("Failed to remove service file", "error", removeErr)
	}

	// Reload systemd (fixes #789 - log errors instead of silently ignoring)
	if mode == paths.ModeSystem {
		if reloadErr := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); reloadErr != nil {
			logging.GetLogger().Warn("Failed to reload systemd", "error", reloadErr)
		}
	} else {
		if reloadErr := exec.CommandContext(ctx, "systemctl", "--user", "daemon-reload").Run(); reloadErr != nil {
			logging.GetLogger().Warn("Failed to reload user systemd", "error", reloadErr)
		}
	}

	// Remove binary
	fmt.Fprintln(os.Stdout, "Removing binary...")
	binaryPath := filepath.Join(p.BinaryDir, "seed")
	if mode == paths.ModeUser {
		binaryPath = filepath.Join(os.Getenv("HOME"), ".local", "bin", "seed")
	}
	if removeBinErr := os.Remove(binaryPath); removeBinErr != nil && !os.IsNotExist(removeBinErr) {
		logging.GetLogger().Warn("Failed to remove binary", "error", removeBinErr)
	}

	// Purge data
	if purge {
		fmt.Fprintln(os.Stdout, "Removing data...")
		dirs := []string{p.ConfigDir, p.DataDir, p.LogDir, p.CacheDir}
		for _, dir := range dirs {
			if rmErr := os.RemoveAll(dir); rmErr != nil {
				logging.GetLogger().Warn("Failed to remove directory", "path", dir, "error", rmErr)
			} else {
				fmt.Fprintf(os.Stdout, "  Removed: %s\n", dir)
			}
		}

		// Remove user (system mode only) - fixes #789
		if mode == paths.ModeSystem {
			fmt.Fprintln(os.Stdout, "Removing seed user...")
			if userDelErr := exec.CommandContext(ctx, "userdel", "seed").Run(); userDelErr != nil {
				logging.GetLogger().Warn("Failed to remove seed user", "error", userDelErr)
			}
		}
	}

	fmt.Fprintln(os.Stdout, "\n✓ Uninstall complete!")
	if !purge {
		fmt.Fprintf(os.Stdout, "\nConfiguration preserved at: %s\n", p.ConfigDir)
		fmt.Fprintln(os.Stdout, "Use --purge to remove all data.")
	}
}
