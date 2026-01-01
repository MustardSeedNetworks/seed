package main

import (
	"context"
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

var installCmd = &cobra.Command{
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
	Run: runInstall,
}

func init() {
	installCmd.Flags().Bool("system", false, "Install as system service (requires root)")
	installCmd.Flags().Bool("user", false, "Install as user service (systemd --user)")
	installCmd.Flags().Bool("no-service", false, "Skip systemd service installation")
	installCmd.Flags().BoolP("force", "f", false, "Overwrite existing installation")
	rootCmd.AddCommand(installCmd)
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

//nolint:gocyclo // Command handler complexity is acceptable
func runInstall(cmd *cobra.Command, _ []string) {
	if runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "Error: install command is only supported on Linux\n")
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
	noService, err := cmd.Flags().GetBool("no-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting no-service flag: %v\n", err)
		os.Exit(1)
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting force flag: %v\n", err)
		os.Exit(1)
	}

	// Determine mode
	if systemMode && userMode {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --system and --user\n")
		os.Exit(1)
	}

	mode := paths.ModeAuto
	if systemMode {
		mode = paths.ModeSystem
	} else if userMode {
		mode = paths.ModeUser
	}

	// Check root for system mode
	if mode == paths.ModeSystem || (mode == paths.ModeAuto && os.Getuid() == 0) {
		if os.Getuid() != 0 {
			fmt.Fprintf(os.Stderr, "Error: system installation requires root privileges (use sudo)\n")
			os.Exit(1)
		}
		mode = paths.ModeSystem
	} else {
		mode = paths.ModeUser
	}

	// Detect distro
	distro := DetectDistro()
	fmt.Fprintf(os.Stdout, "Detected: %s (%s family)\n", distro.Name, distro.Family)

	// Resolve paths
	p := paths.Resolve(mode)

	// Get current binary path
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Installation mode: %s\n", modeString(mode))
	fmt.Fprintf(os.Stdout, "Config directory: %s\n", p.ConfigDir)
	fmt.Fprintf(os.Stdout, "Data directory: %s\n", p.DataDir)
	fmt.Fprintf(os.Stdout, "Log directory: %s\n", p.LogDir)

	// Create directories
	fmt.Fprintln(os.Stdout, "\nCreating directories...")
	dirs := []string{p.ConfigDir, p.DataDir, p.LogDir, p.CacheDir}
	for _, dir := range dirs {
		err = os.MkdirAll(dir, 0o750)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "  Created: %s\n", dir)
	}

	// System mode: create user/group
	if mode == paths.ModeSystem {
		fmt.Fprintln(os.Stdout, "\nCreating seed user and group...")
		createSystemUser()

		// Set ownership
		for _, dir := range dirs {
			err = runCommand("chown", "-R", "seed:seed", dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to set ownership for %s: %v\n", dir, err)
			}
		}
	}

	// Copy binary
	destBinary := filepath.Join(p.BinaryDir, "seed")
	if mode == paths.ModeUser {
		destBinary = filepath.Join(os.Getenv("HOME"), ".local", "bin", "seed")
		//nolint:gosec // G301: User binary directory needs 0755 permissions
		err = os.MkdirAll(filepath.Dir(destBinary), 0o755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating binary directory: %v\n", err)
			os.Exit(1)
		}
	}

	_, err = os.Stat(destBinary)
	if err == nil && !force {
		fmt.Fprintf(os.Stdout, "\nBinary already exists at %s\n", destBinary)
		fmt.Fprintln(os.Stdout, "Use --force to overwrite")
	} else {
		fmt.Fprintf(os.Stdout, "\nCopying binary to %s...\n", destBinary)
		err = copyFile(executable, destBinary)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error copying binary: %v\n", err)
			os.Exit(1)
		}
		//nolint:gosec // G302: Binary file needs execute permission (0755)
		err = os.Chmod(destBinary, 0o755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting binary permissions: %v\n", err)
			os.Exit(1)
		}
	}

	// Set capabilities (system mode only)
	if mode == paths.ModeSystem {
		fmt.Fprintln(os.Stdout, "\nSetting capabilities...")
		err = runCommand("setcap", "cap_net_raw,cap_net_admin=+ep", destBinary)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to set capabilities: %v\n", err)
			fmt.Fprintln(os.Stdout, "  ICMP and protocol capture features will require root")
		} else {
			fmt.Fprintln(os.Stdout, "  Set cap_net_raw,cap_net_admin for raw socket access")
		}
	}

	// Create default config
	configFile := filepath.Join(p.ConfigDir, "seed.yaml")
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stdout, "\nCreating default config at %s...\n", configFile)
		cfg := config.DefaultConfig()
		err = cfg.Save(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
		}
	}

	// Install systemd service
	if !noService {
		fmt.Fprintln(os.Stdout, "\nInstalling systemd service...")
		installSystemdService(mode, p, destBinary)
	}

	fmt.Fprintln(os.Stdout, "\n✓ Installation complete!")
	fmt.Fprintf(os.Stdout, "\nTo start the service:\n")
	if mode == paths.ModeSystem {
		fmt.Fprintln(os.Stdout, "  sudo systemctl start seed")
		fmt.Fprintln(os.Stdout, "  sudo systemctl enable seed  # Start on boot")
	} else {
		fmt.Fprintln(os.Stdout, "  systemctl --user start seed")
		fmt.Fprintln(os.Stdout, "  systemctl --user enable seed  # Start on login")
	}
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
