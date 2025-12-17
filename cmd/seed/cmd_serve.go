package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/api"
	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/paths"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Seed server",
	Long: `Start the Seed network diagnostics server.

The server provides a web-based UI for network diagnostics, monitoring,
and analysis. By default, it runs with HTTPS enabled on port 8443.

Use the --dev flag to run in development mode (HTTP on port 8080).`,
	Run: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) {
	// Resolve config path using paths package
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

	icmpAvailable := checkICMPCapabilities()
	cfg := loadAndConfigureConfig(configPath, devMode)
	logPath := setupLogging(cfg)
	netMgr := setupNetworkInterface(cfg, configPath)

	server := api.NewServer(cfg, configPath, logPath, netMgr, icmpAvailable)
	runServerWithShutdown(server, cfg)
}

// checkICMPCapabilities checks for ICMP privileges and returns availability status.
// Note: Called before logging is initialized, so uses fmt.Fprintf.
func checkICMPCapabilities() bool {
	if err := discovery.CheckICMPPrivilegesWithMessage(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: ICMP features disabled - %v\n", err)
		fmt.Fprintln(os.Stderr, "Warning: Running without ICMP privileges - ping features will be unavailable")
		fmt.Fprintln(os.Stderr, "For full functionality, run with: sudo ./seed")
		fmt.Fprintln(os.Stderr, "Or grant capability: sudo setcap cap_net_raw=+ep ./seed")
		return false
	}
	return true
}

// setupLogging configures structured logging with secure permissions and rotation.
func setupLogging(cfg *config.Config) string {
	logPath := filepath.Join("logs", "seed.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G304: logPath is constructed from constants
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal: Failed to create log file with secure permissions: %v\n", err)
			os.Exit(1)
		}
		f.Close()
	} else if err := os.Chmod(logPath, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set secure permissions on existing log file: %v\n", err)
	}

	// Use logging config from config file if available, otherwise defaults
	logCfg := &logging.LoggingConfig{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		AddSource:  cfg.Logging.AddSource,
		File:       logPath,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
	}

	if err := logging.InitLogger(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	slog.Info("The Seed starting", "version", version, "log_path", logPath)

	return logPath
}

// loadAndConfigureConfig loads configuration and applies necessary modifications.
// Note: Called before logging is initialized, so uses fmt.Fprintf for errors.
func loadAndConfigureConfig(configPath string, devMode bool) *config.Config {
	cfg, _, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		fmt.Fprintf(os.Stderr, "Fatal: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	ensureJWTSecret(cfg, configPath)

	if errors.Is(err, config.ErrInsecureCredentials) {
		fmt.Fprintln(os.Stderr, "Initial setup required - visit the web UI to set your admin password")
		printSetupBanner(cfg.Server.Port, cfg.Server.HTTPS)
		// Set placeholder hash to pass validation - wizard will set the real password
		cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	}

	migrateSNMPCredentials(cfg, configPath)
	// Security fix #301: Removed applyEnvironmentOverrides (LOG_ACCESS_TOKEN) - JWT auth is sufficient

	if devMode {
		fmt.Fprintln(os.Stderr, "Running in development mode")
		cfg.Server.HTTPS = false
		fmt.Fprintln(os.Stderr, "Protocol: HTTP (development mode)")
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	return cfg
}

// ensureJWTSecret generates and persists a JWT secret if not present.
// Note: Called before logging is initialized, so uses fmt.Fprintf.
func ensureJWTSecret(cfg *config.Config, configPath string) {
	if cfg.Auth.JWTSecret != "" {
		return
	}
	cfg.UpdateJWTSecret(auth.GenerateJWTSecret())
	if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to persist JWT secret: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "JWT secret generated and persisted to config file")
	}
}

// migrateSNMPCredentials encrypts plaintext SNMP credentials.
// Note: Called before logging is initialized, so uses fmt.Fprintf.
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

	fmt.Fprintln(os.Stderr, "Migrating SNMP credentials to encrypted format...")
	if err := cfg.EncryptSNMPCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to encrypt SNMP credentials: %v\n", err)
	} else if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to persist encrypted SNMP credentials: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "SNMP credentials encrypted and saved securely")
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
		slog.Info("Available interfaces", "type", group.Type, "status", group.Status, "names", names)
	}
}

// applyActiveInterface sets the active interface as the default.
func applyActiveInterface(cfg *config.Config, netMgr *network.Manager, activeInterface, configPath string) {
	if activeInterface != cfg.Interface.Default {
		slog.Info("Using detected active interface instead of configured default",
			"active", activeInterface, "configured", cfg.Interface.Default)
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
			slog.Info("Force quitting...")
			os.Exit(1)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}

	slog.Info("The Seed stopped")
}

// printSetupBanner displays a message directing users to the web UI for setup.
func printSetupBanner(port int, https bool) {
	protocol := "http"
	if https {
		protocol = "https"
	}
	banner := `
╔══════════════════════════════════════════════════════════════════╗
║                   THE SEED - INITIAL SETUP                       ║
║               Mustard Seed Networks                              ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║  Welcome to The Seed! Initial setup is required.                 ║
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
	// Note: Called before logging is initialized, so banner is stderr-only
}
