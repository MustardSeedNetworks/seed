package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

const (
	// userCheckTimeoutSeconds is the timeout for checking if the seed user exists.
	userCheckTimeoutSeconds = 5

	// commandTimeoutSeconds is the default timeout for executing shell commands during installation.
	commandTimeoutSeconds = 30
)

func initInstallCmd(state *cliState) {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Seed as a system service",
		Long: `Install Seed as a system service with proper permissions.

This command will:
1. Create the seed user and group (system mode)
2. Create necessary directories with proper permissions
3. Copy the binary to the install location
4. Set capabilities for raw socket access
5. Install and enable the systemd service
6. Create a default configuration file if needed`,
		Run: func(cmd *cobra.Command, args []string) {
			runInstall(cmd, args, state)
		},
	}
	installCmd.Flags().Bool("system", false, "Install as system service (requires root)")
	installCmd.Flags().Bool("user", false, "Install as user service (systemd --user)")
	installCmd.Flags().Bool("no-service", false, "Skip systemd service installation")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite existing installation")
	state.rootCmd.AddCommand(installCmd)
}

const systemdServiceTemplate = `[Unit]
Description=The Seed - Network Diagnostics by Mustard Seed Networks
After=network-online.target
Wants=network-online.target
Documentation=https://github.com/krisarmstrong/seed

[Service]
Type=simple
User={{.User}}
Group={{.Group}}
WorkingDirectory={{.DataDir}}
ExecStartPre=/sbin/setcap cap_net_raw,cap_net_admin=+ep {{.BinaryPath}}
ExecStart={{.BinaryPath}} serve
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=no
ProtectSystem=strict
ProtectHome=true
ReadWritePaths={{.ConfigDir}} {{.DataDir}} {{.LogDir}} {{.CacheDir}}
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

const userServiceTemplate = `[Unit]
Description=The Seed - Network Diagnostics
After=network-online.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

type serviceConfig struct {
	User       string
	Group      string
	BinaryPath string
	ConfigDir  string
	DataDir    string
	LogDir     string
	CacheDir   string
}

// installFlags holds the parsed command-line flags for installation.
type installFlags struct {
	systemMode bool
	userMode   bool
	noService  bool
	force      bool
}

// parseInstallFlags extracts and validates command-line flags.
func parseInstallFlags(cmd *cobra.Command) (installFlags, error) {
	var flags installFlags
	var err error

	flags.systemMode, err = cmd.Flags().GetBool("system")
	if err != nil {
		return flags, fmt.Errorf("getting system flag: %w", err)
	}

	flags.userMode, err = cmd.Flags().GetBool("user")
	if err != nil {
		return flags, fmt.Errorf("getting user flag: %w", err)
	}

	flags.noService, err = cmd.Flags().GetBool("no-service")
	if err != nil {
		return flags, fmt.Errorf("getting no-service flag: %w", err)
	}

	flags.force, err = cmd.Flags().GetBool("force")
	if err != nil {
		return flags, fmt.Errorf("getting force flag: %w", err)
	}

	if flags.systemMode && flags.userMode {
		return flags, errors.New("cannot specify both --system and --user")
	}

	return flags, nil
}

// resolveInstallMode determines the installation mode based on flags and privileges.
func resolveInstallMode(flags installFlags) (paths.Mode, error) {
	mode := paths.ModeAuto
	if flags.systemMode {
		mode = paths.ModeSystem
	} else if flags.userMode {
		mode = paths.ModeUser
	}

	if mode == paths.ModeSystem || (mode == paths.ModeAuto && os.Getuid() == 0) {
		if os.Getuid() != 0 {
			return mode, errors.New("system installation requires root privileges (use sudo)")
		}
		return paths.ModeSystem, nil
	}

	return paths.ModeUser, nil
}

// createInstallDirectories creates all required directories for installation.
func createInstallDirectories(dirs []string) error {
	fmt.Fprintln(os.Stdout, "\nCreating directories...")
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
		fmt.Fprintf(os.Stdout, "  Created: %s\n", dir)
	}
	return nil
}

// setupSystemModeUser creates the system user and sets directory ownership.
func setupSystemModeUser(dirs []string) {
	fmt.Fprintln(os.Stdout, "\nCreating seed user and group...")
	createSystemUser()

	for _, dir := range dirs {
		if err := runCommand("chown", "-R", "seed:seed", dir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to set ownership for %s: %v\n", dir, err)
		}
	}
}

// resolveBinaryDestination determines where to install the binary.
func resolveBinaryDestination(mode paths.Mode, p *paths.Paths) (string, error) {
	if mode == paths.ModeUser {
		destBinary := filepath.Join(os.Getenv("HOME"), ".local", "bin", "seed")
		//nolint:gosec // G301: User binary directory needs 0755 permissions
		if err := os.MkdirAll(filepath.Dir(destBinary), 0o755); err != nil {
			return "", fmt.Errorf("creating binary directory: %w", err)
		}
		return destBinary, nil
	}
	return filepath.Join(p.BinaryDir, "seed"), nil
}

// installBinary copies the executable to the destination path.
func installBinary(executable, destBinary string, force bool) error {
	if _, err := os.Stat(destBinary); err == nil && !force {
		fmt.Fprintf(os.Stdout, "\nBinary already exists at %s\n", destBinary)
		fmt.Fprintln(os.Stdout, "Use --force to overwrite")
		return nil
	}

	fmt.Fprintf(os.Stdout, "\nCopying binary to %s...\n", destBinary)
	if err := copyFile(executable, destBinary); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}

	//nolint:gosec // G302: Binary file needs execute permission (0755)
	if err := os.Chmod(destBinary, 0o755); err != nil {
		return fmt.Errorf("setting binary permissions: %w", err)
	}

	return nil
}

// setSystemCapabilities sets network capabilities on the binary for system mode.
func setSystemCapabilities(destBinary string) {
	fmt.Fprintln(os.Stdout, "\nSetting capabilities...")
	if err := runCommand("setcap", "cap_net_raw,cap_net_admin=+ep", destBinary); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set capabilities: %v\n", err)
		fmt.Fprintln(os.Stdout, "  ICMP and protocol capture features will require root")
	} else {
		fmt.Fprintln(os.Stdout, "  Set cap_net_raw,cap_net_admin for raw socket access")
	}
}

// createDefaultConfig creates a default configuration file if one does not exist.
func createDefaultConfig(configDir string) {
	configFile := filepath.Join(configDir, "seed.json")
	_, statErr := os.Stat(configFile)
	if os.IsNotExist(statErr) {
		fmt.Fprintf(os.Stdout, "\nCreating default config at %s...\n", configFile)
		cfg := config.DefaultConfig()
		if saveErr := cfg.Save(configFile); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", saveErr)
		}
	}
}

// printCompletionMessage displays post-installation instructions.
func printCompletionMessage(mode paths.Mode) {
	fmt.Fprintln(os.Stdout, "\n[ok] Installation complete!")
	fmt.Fprintf(os.Stdout, "\nTo start the service:\n")
	if mode == paths.ModeSystem {
		fmt.Fprintln(os.Stdout, "  sudo systemctl start seed")
		fmt.Fprintln(os.Stdout, "  sudo systemctl enable seed  # Start on boot")
	} else {
		fmt.Fprintln(os.Stdout, "  systemctl --user start seed")
		fmt.Fprintln(os.Stdout, "  systemctl --user enable seed  # Start on login")
	}
}

func runInstall(cmd *cobra.Command, _ []string, _ *cliState) {
	if runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "Error: install command is only supported on Linux\n")
		os.Exit(1)
	}

	flags, err := parseInstallFlags(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mode, err := resolveInstallMode(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	distro := DetectDistro()
	fmt.Fprintf(os.Stdout, "Detected: %s (%s family)\n", distro.Name, distro.Family)

	p := paths.Resolve(mode)

	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Installation mode: %s\n", modeString(mode))
	fmt.Fprintf(os.Stdout, "Config directory: %s\n", p.ConfigDir)
	fmt.Fprintf(os.Stdout, "Data directory: %s\n", p.DataDir)
	fmt.Fprintf(os.Stdout, "Log directory: %s\n", p.LogDir)

	dirs := []string{p.ConfigDir, p.DataDir, p.LogDir, p.CacheDir}
	err = createInstallDirectories(dirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if mode == paths.ModeSystem {
		setupSystemModeUser(dirs)
	}

	destBinary, err := resolveBinaryDestination(mode, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = installBinary(executable, destBinary, flags.force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if mode == paths.ModeSystem {
		setSystemCapabilities(destBinary)
	}

	createDefaultConfig(p.ConfigDir)

	if !flags.noService {
		fmt.Fprintln(os.Stdout, "\nInstalling systemd service...")
		installSystemdService(mode, p, destBinary)
	}

	printCompletionMessage(mode)
}

func modeString(mode paths.Mode) string {
	switch mode {
	case paths.ModeSystem:
		return "system"
	case paths.ModeUser:
		return "user"
	case paths.ModeAuto:
		return "auto"
	default:
		return "unknown"
	}
}

func createSystemUser() {
	// Check if user exists
	if _, err := exec.LookPath("id"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), userCheckTimeoutSeconds*time.Second)
		defer cancel()
		if exec.CommandContext(ctx, "id", "seed").Run() == nil {
			fmt.Fprintln(os.Stdout, "  User 'seed' already exists")
			return
		}
	}

	// Create user
	if err := runCommand("useradd", "-r", "-s", "/usr/sbin/nologin", "-d", "/var/lib/seed", "-m", "seed"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create user: %v\n", err)
	} else {
		fmt.Fprintln(os.Stdout, "  Created user 'seed'")
	}
}

func installSystemdService(mode paths.Mode, p *paths.Paths, binaryPath string) {
	var servicePath, tmpl string
	var svcCfg serviceConfig
	var err error

	if mode == paths.ModeSystem {
		servicePath = "/etc/systemd/system/seed.service"
		tmpl = systemdServiceTemplate
		svcCfg = serviceConfig{
			User:       "seed",
			Group:      "seed",
			BinaryPath: binaryPath,
			ConfigDir:  p.ConfigDir,
			DataDir:    p.DataDir,
			LogDir:     p.LogDir,
			CacheDir:   p.CacheDir,
		}
	} else {
		var userConfigDir string
		userConfigDir, err = os.UserConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting user config dir: %v\n", err)
			return
		}
		servicePath = filepath.Join(userConfigDir, "systemd", "user", "seed.service")
		//nolint:gosec // G301: User systemd directory needs 0755 permissions
		err = os.MkdirAll(filepath.Dir(servicePath), 0o755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating service directory: %v\n", err)
			return
		}
		tmpl = userServiceTemplate
		svcCfg = serviceConfig{
			BinaryPath: binaryPath,
		}
	}

	// Generate service file
	var t *template.Template
	t, err = template.New("service").Parse(tmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
		return
	}

	var f *os.File
	f, err = os.Create(servicePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating service file: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()

	err = t.Execute(f, svcCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing service file: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "  Created: %s\n", servicePath)

	// Reload systemd
	if mode == paths.ModeSystem {
		err = runCommand("systemctl", "daemon-reload")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to reload systemd: %v\n", err)
		}
	} else {
		err = runCommand("systemctl", "--user", "daemon-reload")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to reload systemd: %v\n", err)
		}
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	//nolint:gosec // G306: Binary file needs execute permission
	return os.WriteFile(dst, data, 0o755)
}

func runCommand(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeoutSeconds*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
