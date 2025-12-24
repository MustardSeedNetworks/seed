package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

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

func init() {
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
		fmt.Println("This will uninstall Seed:")
		fmt.Printf("  - Stop and disable the systemd service\n")
		fmt.Printf("  - Remove the binary\n")
		if purge {
			fmt.Printf("  - Remove all configuration in %s\n", p.ConfigDir)
			fmt.Printf("  - Remove all data in %s\n", p.DataDir)
			fmt.Printf("  - Remove all logs in %s\n", p.LogDir)
		} else {
			fmt.Printf("  - Keep configuration and data (use --purge to remove)\n")
		}
		fmt.Print("\nContinue? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	// Stop service (fixes #789 - log errors instead of silently ignoring)
	fmt.Println("\nStopping service...")
	if mode == paths.ModeSystem {
		if err := exec.Command("systemctl", "stop", "seed").Run(); err != nil {
			slog.Warn("Failed to stop seed service", "error", err)
		}
		if err := exec.Command("systemctl", "disable", "seed").Run(); err != nil {
			slog.Warn("Failed to disable seed service", "error", err)
		}
	} else {
		if err := exec.Command("systemctl", "--user", "stop", "seed").Run(); err != nil {
			slog.Warn("Failed to stop seed user service", "error", err)
		}
		if err := exec.Command("systemctl", "--user", "disable", "seed").Run(); err != nil {
			slog.Warn("Failed to disable seed user service", "error", err)
		}
	}

	// Remove service file
	fmt.Println("Removing service file...")
	var servicePath string
	if mode == paths.ModeSystem {
		servicePath = "/etc/systemd/system/seed.service"
	} else {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			slog.Warn("Failed to get user config dir", "error", err)
			return
		}
		servicePath = filepath.Join(userConfigDir, "systemd", "user", "seed.service")
	}
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove service file", "error", err)
	}

	// Reload systemd (fixes #789 - log errors instead of silently ignoring)
	if mode == paths.ModeSystem {
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			slog.Warn("Failed to reload systemd", "error", err)
		}
	} else {
		if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
			slog.Warn("Failed to reload user systemd", "error", err)
		}
	}

	// Remove binary
	fmt.Println("Removing binary...")
	binaryPath := filepath.Join(p.BinaryDir, "seed")
	if mode == paths.ModeUser {
		binaryPath = filepath.Join(os.Getenv("HOME"), ".local", "bin", "seed")
	}
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove binary", "error", err)
	}

	// Purge data
	if purge {
		fmt.Println("Removing data...")
		dirs := []string{p.ConfigDir, p.DataDir, p.LogDir, p.CacheDir}
		for _, dir := range dirs {
			if err := os.RemoveAll(dir); err != nil {
				slog.Warn("Failed to remove directory", "path", dir, "error", err)
			} else {
				fmt.Printf("  Removed: %s\n", dir)
			}
		}

		// Remove user (system mode only) - fixes #789
		if mode == paths.ModeSystem {
			fmt.Println("Removing seed user...")
			if err := exec.Command("userdel", "seed").Run(); err != nil {
				slog.Warn("Failed to remove seed user", "error", err)
			}
		}
	}

	fmt.Println("\n✓ Uninstall complete!")
	if !purge {
		fmt.Printf("\nConfiguration preserved at: %s\n", p.ConfigDir)
		fmt.Println("Use --purge to remove all data.")
	}
}
