package main

import (
	"bufio"
	"context"
	"errors"
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

const systemServicePath = "/etc/systemd/system/seed.service"

// uninstallFlags holds parsed command flags for uninstall.
type uninstallFlags struct {
	purge      bool
	force      bool
	systemMode bool
	userMode   bool
}

func initUninstallCmd(state *cliState) {
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Seed",
		Long: `Uninstall Seed and optionally remove all data.

By default, configuration and data files are preserved.
Use --purge to remove all files including configuration.`,
		Run: func(cmd *cobra.Command, args []string) {
			runUninstall(cmd, args, state)
		},
	}
	uninstallCmd.Flags().Bool("purge", false, "Remove all data and configuration")
	uninstallCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	uninstallCmd.Flags().Bool("system", false, "Uninstall system service")
	uninstallCmd.Flags().Bool("user", false, "Uninstall user service")
	state.rootCmd.AddCommand(uninstallCmd)
}

// parseUninstallFlags extracts flags from the command.
func parseUninstallFlags(cmd *cobra.Command) (uninstallFlags, error) {
	var flags uninstallFlags
	var err error

	flags.purge, err = cmd.Flags().GetBool("purge")
	if err != nil {
		return flags, fmt.Errorf("getting purge flag: %w", err)
	}
	flags.force, err = cmd.Flags().GetBool("force")
	if err != nil {
		return flags, fmt.Errorf("getting force flag: %w", err)
	}
	flags.systemMode, err = cmd.Flags().GetBool("system")
	if err != nil {
		return flags, fmt.Errorf("getting system flag: %w", err)
	}
	flags.userMode, err = cmd.Flags().GetBool("user")
	if err != nil {
		return flags, fmt.Errorf("getting user flag: %w", err)
	}
	return flags, nil
}

// determineUninstallMode returns the appropriate paths.Mode based on flags.
func determineUninstallMode(flags uninstallFlags) paths.Mode {
	switch {
	case flags.systemMode:
		return paths.ModeSystem
	case flags.userMode:
		return paths.ModeUser
	case os.Getuid() == 0:
		return paths.ModeSystem
	default:
		return paths.ModeUser
	}
}

// confirmUninstall prompts the user for confirmation and returns true if confirmed.
func confirmUninstall(p *paths.Paths, purge bool) bool {
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
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// stopAndDisableService stops and disables the systemd service.
func stopAndDisableService(ctx context.Context, mode paths.Mode) {
	fmt.Fprintln(os.Stdout, "\nStopping service...")
	logger := logging.GetLogger()

	if mode == paths.ModeSystem {
		if err := exec.CommandContext(ctx, "systemctl", "stop", "seed").Run(); err != nil {
			logger.WarnContext(ctx, "Failed to stop seed service", "error", err)
		}
		if err := exec.CommandContext(ctx, "systemctl", "disable", "seed").Run(); err != nil {
			logger.WarnContext(ctx, "Failed to disable seed service", "error", err)
		}
		return
	}

	if err := exec.CommandContext(ctx, "systemctl", "--user", "stop", "seed").Run(); err != nil {
		logger.WarnContext(ctx, "Failed to stop seed user service", "error", err)
	}
	if err := exec.CommandContext(ctx, "systemctl", "--user", "disable", "seed").Run(); err != nil {
		logger.WarnContext(ctx, "Failed to disable seed user service", "error", err)
	}
}

// removeServiceFile removes the systemd service file.
func removeServiceFile(ctx context.Context, mode paths.Mode) error {
	fmt.Fprintln(os.Stdout, "Removing service file...")

	servicePath := getServiceFilePath(ctx, mode)
	if servicePath == "" {
		return errors.New("failed to determine service file path")
	}

	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		logging.GetLogger().WarnContext(ctx, "Failed to remove service file", "error", err)
	}
	return nil
}

// getServiceFilePath returns the path to the systemd service file.
func getServiceFilePath(ctx context.Context, mode paths.Mode) string {
	if mode == paths.ModeSystem {
		return systemServicePath
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		logging.GetLogger().WarnContext(ctx, "Failed to get user config dir", "error", err)
		return ""
	}
	return filepath.Join(userConfigDir, "systemd", "user", "seed.service")
}

// reloadSystemd reloads the systemd daemon.
func reloadSystemd(ctx context.Context, mode paths.Mode) {
	logger := logging.GetLogger()

	if mode == paths.ModeSystem {
		if err := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); err != nil {
			logger.WarnContext(ctx, "Failed to reload systemd", "error", err)
		}
		return
	}

	if err := exec.CommandContext(ctx, "systemctl", "--user", "daemon-reload").Run(); err != nil {
		logger.WarnContext(ctx, "Failed to reload user systemd", "error", err)
	}
}

// removeBinary removes the seed binary.
func removeBinary(ctx context.Context, p *paths.Paths, mode paths.Mode) {
	fmt.Fprintln(os.Stdout, "Removing binary...")

	binaryPath := filepath.Join(p.BinaryDir, "seed")
	if mode == paths.ModeUser {
		binaryPath = filepath.Join(os.Getenv("HOME"), ".local", "bin", "seed")
	}

	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		logging.GetLogger().WarnContext(ctx, "Failed to remove binary", "error", err)
	}
}

// purgeData removes all configuration, data, log, and cache directories.
func purgeData(ctx context.Context, p *paths.Paths, mode paths.Mode) {
	fmt.Fprintln(os.Stdout, "Removing data...")
	logger := logging.GetLogger()

	dirs := []string{p.ConfigDir, p.DataDir, p.LogDir, p.CacheDir}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			logger.WarnContext(ctx, "Failed to remove directory", "path", dir, "error", err)
		} else {
			fmt.Fprintf(os.Stdout, "  Removed: %s\n", dir)
		}
	}

	if mode == paths.ModeSystem {
		removeSeedUser(ctx)
	}
}

// removeSeedUser removes the seed system user.
func removeSeedUser(ctx context.Context) {
	fmt.Fprintln(os.Stdout, "Removing seed user...")
	if err := exec.CommandContext(ctx, "userdel", "seed").Run(); err != nil {
		logging.GetLogger().WarnContext(ctx, "Failed to remove seed user", "error", err)
	}
}

// printUninstallComplete prints the completion message.
func printUninstallComplete(p *paths.Paths, purge bool) {
	fmt.Fprintln(os.Stdout, "\n✓ Uninstall complete!")
	if !purge {
		fmt.Fprintf(os.Stdout, "\nConfiguration preserved at: %s\n", p.ConfigDir)
		fmt.Fprintln(os.Stdout, "Use --purge to remove all data.")
	}
}

func runUninstall(cmd *cobra.Command, _ []string, _ *cliState) {
	if runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "Error: uninstall command is only supported on Linux\n")
		os.Exit(1)
	}

	flags, err := parseUninstallFlags(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mode := determineUninstallMode(flags)
	p := paths.Resolve(mode)

	if !flags.force && !confirmUninstall(p, flags.purge) {
		fmt.Fprintln(os.Stdout, "Aborted.")
		return
	}

	ctx := context.Background()

	stopAndDisableService(ctx, mode)

	if removeErr := removeServiceFile(ctx, mode); removeErr != nil {
		return
	}

	reloadSystemd(ctx, mode)
	removeBinary(ctx, p, mode)

	if flags.purge {
		purgeData(ctx, p, mode)
	}

	printUninstallComplete(p, flags.purge)
}
