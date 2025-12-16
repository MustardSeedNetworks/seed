// Package main is the entry point for LuminetIQ.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/krisarmstrong/seed/internal/api"
	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
)

var version = "dev"

// credentialsFileMode is the file permission for credential files (owner read/write only).
const credentialsFileMode = 0o600

// main starts the LuminetIQ network discovery and monitoring application.
func main() {
	// Handle subcommands before flag parsing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "credentials":
			handleCredentialsCommand()
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version")
	configPath := flag.String("config", "configs/luminetiq.yaml", "Path to configuration file")
	devMode := flag.Bool("dev", false, "Run in development mode")
	flag.Parse()

	if *showVersion {
		fmt.Printf("LuminetIQ %s\n", version)
		os.Exit(0)
	}

	icmpAvailable := checkICMPCapabilities()
	logPath := setupLogging()
	cfg := loadAndConfigureConfig(*configPath, *devMode)
	netMgr := setupNetworkInterface(cfg, *configPath)

	server := api.NewServer(cfg, *configPath, logPath, netMgr, icmpAvailable)
	runServerWithShutdown(server, cfg)
}

// checkICMPCapabilities checks for ICMP privileges and returns availability status.
func checkICMPCapabilities() bool {
	if err := discovery.CheckICMPPrivilegesWithMessage(); err != nil {
		slog.Warn("ICMP features disabled", "error", err)
		fmt.Fprintln(os.Stderr, "Warning: Running without ICMP privileges - ping features will be unavailable")
		fmt.Fprintln(os.Stderr, "For full functionality, run with: sudo ./luminetiq")
		fmt.Fprintln(os.Stderr, "Or grant capability: sudo setcap cap_net_raw=+ep ./luminetiq")
		return false
	}
	return true
}

// setupLogging configures logging with secure permissions and rotation.
func setupLogging() string {
	logPath := filepath.Join("logs", "luminetiq.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		slog.Error("Failed to create log directory", "error", err)
		os.Exit(1)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G304: logPath is constructed from constants
		if err != nil {
			slog.Error("Failed to create log file with secure permissions", "error", err)
			os.Exit(1)
		}
		f.Close()
	} else if err := os.Chmod(logPath, 0o600); err != nil {
		slog.Warn("Failed to set secure permissions on existing log file", "error", err)
	}

	// Initialize structured logging with file rotation and redaction
	logCfg := &logging.LoggingConfig{
		Level:      "info",
		Format:     "text",
		AddSource:  true,
		File:       logPath,
		MaxSize:    20,
		MaxBackups: 7,
		MaxAge:     30,
		Compress:   true,
	}
	if err := logging.InitLogger(logCfg); err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}

	slog.Info("LuminetIQ starting", "version", version, "log_path", logPath)

	return logPath
}

// loadAndConfigureConfig loads configuration and applies necessary modifications.
func loadAndConfigureConfig(configPath string, devMode bool) *config.Config {
	cfg, _, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ensureJWTSecret(cfg, configPath)

	if errors.Is(err, config.ErrInsecureCredentials) {
		slog.Info("Initial setup required - visit the web UI to set your admin password")
		printSetupBanner(cfg.Server.Port, cfg.Server.HTTPS)
	}

	migrateSNMPCredentials(cfg, configPath)
	applyEnvironmentOverrides(cfg)

	if devMode {
		slog.Info("Running in development mode")
		cfg.Server.HTTPS = false
		slog.Info("Protocol: HTTP (development mode)")
	}

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	return cfg
}

// ensureJWTSecret generates and persists a JWT secret if not present.
func ensureJWTSecret(cfg *config.Config, configPath string) {
	if cfg.Auth.JWTSecret != "" {
		return
	}
	cfg.UpdateJWTSecret(auth.GenerateJWTSecret())
	if err := cfg.Save(configPath); err != nil {
		slog.Warn("Failed to persist JWT secret", "error", err)
	} else {
		slog.Info("JWT secret generated and persisted to config file")
	}
}

// migrateSNMPCredentials encrypts plaintext SNMP credentials.
func migrateSNMPCredentials(cfg *config.Config, configPath string) {
	if cfg.Auth.JWTSecret == "" || len(cfg.SNMP.V3Credentials) == 0 {
		return
	}

	needsSave := false
	for i := range cfg.SNMP.V3Credentials {
		cred := &cfg.SNMP.V3Credentials[i]
		if (cred.AuthPassword != "" && !config.IsEncrypted(cred.AuthPassword)) ||
			(cred.PrivPassword != "" && !config.IsEncrypted(cred.PrivPassword)) {
			needsSave = true
			break
		}
	}

	if !needsSave {
		return
	}

	slog.Info("Migrating SNMP credentials to encrypted format")
	if err := cfg.EncryptSNMPCredentials(); err != nil {
		slog.Warn("Failed to encrypt SNMP credentials", "error", err)
	} else if err := cfg.Save(configPath); err != nil {
		slog.Warn("Failed to persist encrypted SNMP credentials", "error", err)
	} else {
		slog.Info("SNMP credentials encrypted and saved securely")
	}
}

// applyEnvironmentOverrides applies environment variable overrides to configuration.
// Note: LOG_ACCESS_TOKEN support removed per security fix #301 - JWT auth is sufficient.
func applyEnvironmentOverrides(_ *config.Config) {
	if token := os.Getenv("LOG_ACCESS_TOKEN"); token != "" {
		slog.Info("Environment variable override: LOG_ACCESS_TOKEN is set (deprecated, use JWT auth)")
	}
	if hdr := os.Getenv("LOG_ACCESS_HEADER"); hdr != "" {
		slog.Info("Environment variable override (deprecated)", "LOG_ACCESS_HEADER", hdr)
	}
}

// setupNetworkInterface initializes the network manager and finds an active interface.
func setupNetworkInterface(cfg *config.Config, configPath string) *network.Manager {
	if cfg.Interface.Default == "" {
		slog.Error("No default network interface specified in configuration")
		os.Exit(1)
	}

	netMgr, err := network.NewManager(cfg.Interface.Default)
	if err != nil {
		slog.Error("Failed to initialize network manager", "error", err)
		os.Exit(1)
	}

	preferred := append([]string{cfg.Interface.Default}, cfg.Interface.Fallbacks...)
	activeInterface := findActiveInterface(netMgr, preferred, cfg.Interface.StartupRetries, cfg.Interface.StartupRetryWait)

	if activeInterface == "" {
		logAvailableInterfaces(netMgr)
	} else {
		applyActiveInterface(cfg, netMgr, activeInterface, configPath)
	}

	return netMgr
}

// findActiveInterface attempts to find an active network interface with retries.
func findActiveInterface(netMgr *network.Manager, preferred []string, maxRetries int, retryWait time.Duration) string {
	activeInterface := netMgr.FindFirstAvailable(preferred)
	for retryCount := 0; activeInterface == "" && retryCount < maxRetries; retryCount++ {
		slog.Warn("No active network interface found, retrying", "retry_wait", retryWait)
		time.Sleep(retryWait)
		activeInterface = netMgr.FindFirstAvailable(preferred)
	}
	return activeInterface
}

// logAvailableInterfaces logs available interfaces grouped by type and status.
func logAvailableInterfaces(netMgr *network.Manager) {
	slog.Error("No active network interface found after multiple attempts")
	slog.Info("Please check your network configuration and ensure at least one interface is up")

	type ifaceGroup struct{ Type, Status string }
	grouped := make(map[ifaceGroup][]string)
	for _, iface := range netMgr.GetInterfaces() {
		status := "down"
		if iface.Up {
			status = "up"
		}
		key := ifaceGroup{Type: string(iface.Type), Status: status}
		grouped[key] = append(grouped[key], iface.Name)
	}
	for group, names := range grouped {
		slog.Info("Available interfaces", "type", group.Type, "status", group.Status, "interfaces", names)
	}
}

// applyActiveInterface sets the active interface as the default.
func applyActiveInterface(cfg *config.Config, netMgr *network.Manager, activeInterface, configPath string) {
	if activeInterface != cfg.Interface.Default {
		slog.Info("Using detected active interface instead of configured default",
			"active_interface", activeInterface, "configured_default", cfg.Interface.Default)
		cfg.Interface.Default = activeInterface
		if err := cfg.Save(configPath); err != nil {
			slog.Warn("Failed to save updated interface to config", "error", err)
		} else {
			slog.Info("Updated config with active interface", "interface", activeInterface)
		}
	}
	if err := netMgr.SetCurrentInterface(activeInterface); err != nil {
		slog.Warn("Failed to set active interface", "interface", activeInterface, "error", err)
	}
}

// runServerWithShutdown starts the server and handles graceful shutdown.
func runServerWithShutdown(server *api.Server, cfg *config.Config) {
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Starting server", "port", cfg.Server.Port, "https", cfg.Server.HTTPS)
		serverErrors <- server.Start()
	}()

	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-sigChan:
		slog.Info("Received signal, shutting down gracefully (press Ctrl+C again to force)", "signal", sig)

		go func() {
			<-sigChan
			slog.Info("Force quitting")
			os.Exit(1)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}

	slog.Info("LuminetIQ stopped")
}

// printSetupBanner displays a message directing users to the web UI for setup.
func printSetupBanner(port int, https bool) {
	protocol := "http"
	if https {
		protocol = "https"
	}
	banner := `
╔══════════════════════════════════════════════════════════════════╗
║                    LUMINETIQ INITIAL SETUP                       ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║  Welcome to LuminetIQ! Initial setup is required.                ║
║                                                                  ║
║  Please open your web browser and navigate to:                   ║
║                                                                  ║
║    %s://localhost:%-42d ║
║                                                                  ║
║  You will be prompted to set your admin password.                ║
║  A secure password will be suggested for you.                    ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝
`
	// Use fmt.Fprintf to stderr so it's visible even when stdout is redirected
	fmt.Fprintf(os.Stderr, banner, protocol, port)

	// Also log it
	slog.Info("Setup required - visit web UI to complete setup", "url", fmt.Sprintf("%s://localhost:%d", protocol, port))
}

// printUsage displays the CLI usage information.
func printUsage() {
	fmt.Printf(`LuminetIQ %s - Network Diagnostics and Monitoring

Usage:
  luminetiq [flags]              Start the server
  luminetiq credentials          Generate and display initial admin credentials
  luminetiq help                 Show this help message

Flags:
  -version    Show version and exit
  -config     Path to configuration file (default: configs/luminetiq.yaml)
  -dev        Run in development mode (HTTP instead of HTTPS)

First-Boot Credential Retrieval (fixes #489):
  Run 'luminetiq credentials' to generate secure initial credentials.
  This writes credentials to a secure file and displays them once.
  Use this instead of parsing logs for systemd deployments.

Examples:
  luminetiq                      Start with default config
  luminetiq -dev                 Start in development mode
  luminetiq -config /etc/luminetiq/config.yaml  Start with custom config
  luminetiq credentials          Generate initial admin credentials
`, version)
}

// handleCredentialsCommand generates and outputs initial credentials for first-boot setup.
// This provides a deterministic, non-log path for credential retrieval (fixes #489).
func handleCredentialsCommand() {
	// Parse credentials subcommand flags
	credFlags := flag.NewFlagSet("credentials", flag.ExitOnError)
	configPath := credFlags.String("config", "configs/luminetiq.yaml", "Path to configuration file")
	outputJSON := credFlags.Bool("json", false, "Output credentials as JSON")
	fileOutput := credFlags.String("file", "", "Write credentials to file (default: working directory)")
	if err := credFlags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load or create config
	cfg, result, err := config.EnsureConfig(*configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Check if credentials already exist and are secure
	if err == nil && !result.GeneratedCreds {
		fmt.Fprintln(os.Stderr, "Credentials are already configured. Use the web UI to change the password.")
		fmt.Fprintln(os.Stderr, "If you've lost access, delete the config file and restart.")
		os.Exit(0)
	}

	// Generate new credentials
	creds, err := auth.GenerateInitialCredentials(cfg.Auth.DefaultUsername)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to generate credentials: %v\n", err)
		os.Exit(1)
	}

	// Update config with new credentials
	cfg.UpdateCredentials(creds.Username, creds.PasswordHash, creds.JWTSecret)

	// Save the config
	if err := cfg.Save(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	// Prepare credential output
	credOutput := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Message  string `json:"message,omitempty"`
	}{
		Username: creds.Username,
		Password: creds.Password,
		Message:  "Save this password securely - it will not be shown again",
	}

	// Write to file if requested
	if *fileOutput != "" {
		credFilePath := *fileOutput
		if err := writeCredentialsFile(credFilePath, credOutput.Username, credOutput.Password); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write credentials file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Credentials written to: %s (mode 0600)\n", credFilePath)
	}

	// Output credentials
	if *outputJSON {
		jsonData, err := json.MarshalIndent(credOutput, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal credentials: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                LUMINETIQ INITIAL CREDENTIALS                     ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
		fmt.Printf("║  Username: %-53s ║\n", credOutput.Username)
		fmt.Printf("║  Password: %-53s ║\n", credOutput.Password)
		fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  IMPORTANT: Save this password securely!                         ║")
		fmt.Println("║  It will not be shown again.                                     ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	}
}

// writeCredentialsFile writes credentials to a secure file with restrictive permissions.
func writeCredentialsFile(path, username, password string) error {
	content := fmt.Sprintf("# LuminetIQ Initial Credentials\n# Generated: %s\n# DELETE THIS FILE after retrieving credentials\n\nUsername: %s\nPassword: %s\n",
		time.Now().Format(time.RFC3339), username, password)

	// Write with restrictive permissions (owner read/write only)
	return os.WriteFile(path, []byte(content), credentialsFileMode)
}
