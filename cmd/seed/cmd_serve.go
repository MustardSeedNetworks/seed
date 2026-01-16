package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/harvest"
	api "github.com/krisarmstrong/seed/internal/httpapi"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/paths"
	"github.com/krisarmstrong/seed/internal/roots"
	"github.com/krisarmstrong/seed/internal/sap"
	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/version"
)

const (
	// logBroadcasterBufferSize is the number of log entries to buffer for streaming to the frontend.
	logBroadcasterBufferSize = 1000

	// signalChannelBufferSize is the buffer size for the OS signal channel to handle SIGINT/SIGTERM.
	signalChannelBufferSize = 2

	// shutdownTimeoutSeconds is the maximum time to wait for graceful server shutdown.
	shutdownTimeoutSeconds = 30
)

func initServeCmd(state *cliState) {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start The Seed server",
		Long: `Start The Seed network diagnostics server.

The server provides a web-based UI for network diagnostics, monitoring,
and analysis. By default, it runs with HTTPS enabled on port 8443.

Use the --dev flag to run in development mode (HTTP on port 8080).`,
		Run: func(cmd *cobra.Command, args []string) {
			runServe(cmd, args, state)
		},
	}
	state.rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string, state *cliState) {
	// Resolve config path using paths package
	configPath := paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto)

	icmpAvailable := checkICMPCapabilities()
	cfg := loadAndConfigureConfig(configPath, state.devMode)
	logPath := setupLogging(cfg)

	// Check for deprecated SNMP settings after logging is initialized
	cfg.WarnDeprecatedSNMPSettings()

	netMgr := setupNetworkInterface(cfg, configPath)

	// Create trusted proxies configuration
	proxies := api.NewTrustedProxies(state.trustedProxies)
	if !proxies.IsEmpty() {
		logging.GetLogger().Info("Trusted proxies configured", "count", proxies.Count())
	}

	// Initialize database
	db := initializeDatabase(cfg)

	// Initialize modules
	modules := initializeModules(cfg, db)

	server := api.NewServer(cfg, configPath, logPath, netMgr, icmpAvailable, proxies, db, modules)
	runServerWithShutdown(server, cfg, modules)
}

// initializeDatabase opens and configures the SQLite database.
func initializeDatabase(cfg *config.Config) *database.DB {
	dbPath := cfg.Database.Path
	if dbPath == "" {
		dbPath = "data/seed.db"
	}

	db, err := database.OpenWithAutoRebuild(dbPath)
	if err != nil {
		logging.GetLogger().Error("Failed to open database", "path", dbPath, "error", err)
		return nil
	}

	logging.GetLogger().Info("Database initialized", "path", dbPath)
	return db
}

// initializeModules creates all application modules.
func initializeModules(cfg *config.Config, db *database.DB) *api.Modules {
	modules := &api.Modules{}

	// Sap: Live telemetry (gateway, DNS, speedtest, iperf monitoring)
	modules.Sap = sap.New(cfg, db)
	logging.GetLogger().Info("Sap module initialized")

	// Shell: Security posture (DHCP monitoring, vulnerability scanning)
	modules.Shell = shell.New(cfg, db)
	logging.GetLogger().Info("Shell module initialized")

	// Canopy: Wi-Fi planning (surveys, site planning)
	modules.Canopy = canopy.New(cfg, db)
	logging.GetLogger().Info("Canopy module initialized")

	// Roots: Path analysis (traceroute, topology, IP enrichment)
	modules.Roots = roots.New(cfg, db)
	logging.GetLogger().Info("Roots module initialized")

	// Harvest: Reporting (report generation, templates, scheduling)
	modules.Harvest = harvest.New(cfg, db)
	logging.GetLogger().Info("Harvest module initialized")

	return modules
}

// checkICMPCapabilities checks for ICMP privileges and returns availability status.
// Note: Called before logging is initialized, so uses [fmt.Fprintf].
func checkICMPCapabilities() bool {
	if err := discovery.CheckICMPPrivilegesWithMessage(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: ICMP features disabled - %v\n", err)
		fmt.Fprintln(
			os.Stderr,
			"Warning: Running without ICMP privileges - ping features will be unavailable",
		)
		fmt.Fprintln(os.Stderr, "For full functionality, run with: sudo ./seed")
		fmt.Fprintln(
			os.Stderr,
			"Or grant capability: sudo setcap cap_net_raw,cap_net_admin=+ep ./seed",
		)
		return false
	}
	return true
}

// setupLogging configures structured logging with secure permissions and rotation.
func setupLogging(cfg *config.Config) string {
	// Use configured log path, or default to logs/seed.log
	logPath := cfg.Logging.File
	if logPath == "" {
		logPath = filepath.Join("logs", "seed.log")
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	if _, statErr := os.Stat(logPath); os.IsNotExist(statErr) {
		f, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0o600)
		if openErr != nil {
			fmt.Fprintf(
				os.Stderr,
				"Fatal: Failed to create log file with secure permissions: %v\n",
				openErr,
			)
			os.Exit(1)
		}
		_ = f.Close()
	} else if chmodErr := os.Chmod(logPath, 0o600); chmodErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set secure permissions on existing log file: %v\n", chmodErr)
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

	// Initialize logger with broadcaster to enable log streaming to frontend (#959)
	broadcaster := logging.InitBroadcaster(logBroadcasterBufferSize)
	if err := logging.InitLoggerWithBroadcaster(logCfg, broadcaster); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logging.GetLogger().Info("The Seed starting", "version", version.GetVersion(), "log_path", logPath)

	return logPath
}

// loadAndConfigureConfig loads configuration and applies necessary modifications.
// Note: Called before logging is initialized, so uses [fmt.Fprintf] for errors.
func loadAndConfigureConfig(configPath string, devMode bool) *config.Config {
	cfg, _, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		fmt.Fprintf(os.Stderr, "Fatal: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	ensureJWTSecret(cfg, configPath)

	if errors.Is(err, config.ErrInsecureCredentials) {
		fmt.Fprintln(
			os.Stderr,
			"Initial setup required - visit the web UI to set your admin password",
		)
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

	if validateErr := cfg.Validate(); validateErr != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Invalid configuration: %v\n", validateErr)
		os.Exit(1)
	}

	return cfg
}

// ensureJWTSecret generates and persists a JWT secret if not present.
// Note: Called before logging is initialized, so uses [fmt.Fprintf].
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
// Note: Called before logging is initialized, so uses [fmt.Fprintf].
// Security fix #884: Warn if JWT secret is missing (should never happen after ensureJWTSecret).
func migrateSNMPCredentials(cfg *config.Config, configPath string) {
	if len(cfg.SNMP.V3Credentials) == 0 {
		return
	}
	if cfg.Auth.JWTSecret == "" {
		fmt.Fprintln(os.Stderr, "Warning: Cannot encrypt SNMP credentials - JWT secret is missing")
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
	if encryptErr := cfg.EncryptSNMPCredentials(); encryptErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to encrypt SNMP credentials: %v\n", encryptErr)
	} else if saveErr := cfg.Save(configPath); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to persist encrypted SNMP credentials: %v\n", saveErr)
	} else {
		fmt.Fprintln(os.Stderr, "SNMP credentials encrypted and saved securely")
	}
}

// setupNetworkInterface initializes the network manager and finds an active interface.
// #756: Auto-detects available interfaces; uses config default if valid, otherwise selects best available.
func setupNetworkInterface(cfg *config.Config, configPath string) *network.Manager {
	// #756: Try configured default first, but fall back to auto-detection if invalid
	initialInterface := cfg.Interface.Default
	if initialInterface == "" {
		// Use config's GetActiveInterface which does auto-detection
		detected, usedFallback := cfg.GetActiveInterface()
		if detected != "" {
			if usedFallback {
				logging.GetLogger().Info("Auto-detected active interface", "interface", detected)
			}
			initialInterface = detected
		}
	}

	// Still require at least some interface to start with
	if initialInterface == "" {
		logging.GetLogger().
			Error("No network interface found - please ensure at least one interface is up with an IP address")
		os.Exit(1)
	}

	netMgr, err := network.NewManager(initialInterface)
	if err != nil {
		logging.GetLogger().Error("Failed to initialize network manager", "error", err)
		os.Exit(1)
	}

	preferred := append([]string{initialInterface}, cfg.Interface.Fallbacks...)
	activeInterface := findActiveInterface(
		netMgr,
		preferred,
		cfg.Interface.StartupRetries,
		cfg.Interface.StartupRetryWait,
	)

	if activeInterface == "" {
		logAvailableInterfaces(netMgr)
	} else {
		applyActiveInterface(cfg, netMgr, activeInterface, configPath)
	}

	return netMgr
}

// findActiveInterface attempts to find an active network interface with retries.
func findActiveInterface(
	netMgr *network.Manager,
	preferred []string,
	maxRetries int,
	retryWait time.Duration,
) string {
	activeInterface := netMgr.FindFirstAvailable(preferred)
	for retryCount := 0; activeInterface == "" && retryCount < maxRetries; retryCount++ {
		logging.GetLogger().
			Warn("No active network interface found, retrying", "retry_wait", retryWait)
		time.Sleep(retryWait)
		activeInterface = netMgr.FindFirstAvailable(preferred)
	}
	return activeInterface
}

// logAvailableInterfaces logs available interfaces grouped by type and status.
func logAvailableInterfaces(netMgr *network.Manager) {
	logging.GetLogger().Error("No active network interface found after multiple attempts")
	logging.GetLogger().
		Info("Please check your network configuration and ensure at least one interface is up")

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
		logging.GetLogger().
			Info("Available interfaces", "type", group.Type, "status", group.Status, "names", names)
	}
}

// applyActiveInterface sets the active interface and optionally saves to config.
// #756: Interface selection persists to profile, not global config.
func applyActiveInterface(
	cfg *config.Config,
	netMgr *network.Manager,
	activeInterface, configPath string,
) {
	if activeInterface != cfg.Interface.Default {
		logging.GetLogger().Info("Using detected active interface instead of configured default",
			"active", activeInterface, "configured", cfg.Interface.Default)
		cfg.Interface.Default = activeInterface
		if err := cfg.Save(configPath); err != nil {
			logging.GetLogger().Warn("Failed to save updated interface to config", "error", err)
		} else {
			logging.GetLogger().Info("Updated config with active interface", "interface", activeInterface)
		}
	}
	if err := netMgr.SetCurrentInterface(activeInterface); err != nil {
		logging.GetLogger().
			Warn("Failed to set active interface", "interface", activeInterface, "error", err)
	}
}

// runServerWithShutdown starts the server and handles graceful shutdown.
func runServerWithShutdown(server *api.Server, cfg *config.Config, modules *api.Modules) {
	// Start modules
	ctx := context.Background()
	if modules != nil {
		if err := modules.Start(ctx); err != nil {
			logging.GetLogger().Error("Failed to start modules", "error", err)
			os.Exit(1)
		}
		logging.GetLogger().Info("All modules started successfully")
	}

	serverErrors := make(chan error, 1)
	go func() {
		logging.GetLogger().
			Info("Starting server", "port", cfg.Server.Port, "https", cfg.Server.HTTPS)
		serverErrors <- server.Start()
	}()

	sigChan := make(chan os.Signal, signalChannelBufferSize)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil {
			logging.GetLogger().Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-sigChan:
		logging.GetLogger().
			Info("Received signal, shutting down gracefully (press Ctrl+C again to force)", "signal", sig)

		go func() {
			<-sigChan
			logging.GetLogger().Info("Force quitting...")
			os.Exit(1)
		}()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
		defer cancel()

		// Stop modules first
		if modules != nil {
			logging.GetLogger().Info("Stopping modules...")
			if err := modules.Stop(); err != nil {
				logging.GetLogger().Error("Error stopping modules", "error", err)
			}
		}

		if err := server.Shutdown(shutdownCtx); err != nil {
			logging.GetLogger().Error("Error during shutdown", "error", err)
		}
	}

	logging.GetLogger().Info("The Seed stopped")
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
