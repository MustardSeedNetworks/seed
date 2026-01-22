//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	api "github.com/krisarmstrong/seed/internal/api"
	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/harvest"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/netif"
	"github.com/krisarmstrong/seed/internal/paths"
	"github.com/krisarmstrong/seed/internal/pipeline"
	"github.com/krisarmstrong/seed/internal/services"
	"github.com/krisarmstrong/seed/internal/services/shell"
)

const (
	// windowsServiceName is the internal service name used by Windows SCM.
	windowsServiceName = "SeedNetworkDiagnostics"

	// windowsDisplayName is the human-readable service name shown in services.msc.
	windowsDisplayName = "Seed Network Diagnostic Service"

	// windowsDescription is the detailed description shown in service properties.
	windowsDescription = "Network diagnostics and monitoring service by Mustard Seed Networks"

	// serviceStopTimeout is the maximum time to wait for graceful shutdown.
	serviceStopTimeout = 30 * time.Second
)

// seedProgram implements service.Interface for Windows service management.
type seedProgram struct {
	state     *cliState
	server    *api.Server
	modules   *api.Modules
	stopChan  chan struct{}
	stoppedCh chan struct{}
}

// Start is called when the Windows Service Manager starts the service.
func (p *seedProgram) Start(_ service.Service) error {
	p.stopChan = make(chan struct{})
	p.stoppedCh = make(chan struct{})
	go p.run()
	return nil
}

// Stop is called when the Windows Service Manager requests service stop.
func (p *seedProgram) Stop(_ service.Service) error {
	close(p.stopChan)

	select {
	case <-p.stoppedCh:
		return nil
	case <-time.After(serviceStopTimeout):
		return errors.New("service stop timeout")
	}
}

// run executes the main service logic.
func (p *seedProgram) run() {
	defer close(p.stoppedCh)

	// Resolve config path using paths package
	configPath := paths.ResolveConfigPath(p.state.cfgFile, paths.ModeAuto)

	cfg := loadAndConfigureConfigForService(configPath)
	logPath := setupLoggingForService(cfg)

	// Check for deprecated SNMP settings after logging is initialized
	cfg.WarnDeprecatedSNMPSettings()

	netMgr := setupNetworkInterfaceForService(cfg, configPath)

	// Create trusted proxies configuration
	proxies := api.NewTrustedProxies(p.state.trustedProxies)
	if !proxies.IsEmpty() {
		logging.GetLogger().Info("Trusted proxies configured", "count", proxies.Count())
	}

	// Initialize database
	db := initializeDatabaseForService(cfg)

	// Initialize modules
	p.modules = initializeModulesForService(cfg, db)

	p.server = api.NewServer(cfg, configPath, logPath, netMgr, true, proxies, db, p.modules)

	// Start modules
	ctx := context.Background()
	if p.modules != nil {
		if err := p.modules.Start(ctx); err != nil {
			logging.GetLogger().Error("Failed to start modules", "error", err)
			return
		}
		logging.GetLogger().Info("All modules started successfully")
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logging.GetLogger().Info("Starting server", "port", cfg.Server.Port, "https", cfg.Server.HTTPS)
		serverErrors <- p.server.Start()
	}()

	// Wait for stop signal or server error
	select {
	case err := <-serverErrors:
		if err != nil {
			logging.GetLogger().Error("Server error", "error", err)
		}
	case <-p.stopChan:
		logging.GetLogger().Info("Service stop requested, shutting down...")

		// Stop modules first
		if p.modules != nil {
			logging.GetLogger().Info("Stopping modules...")
			if err := p.modules.Stop(); err != nil {
				logging.GetLogger().Error("Error stopping modules", "error", err)
			}
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), serviceStopTimeout)
		defer cancel()

		if err := p.server.Shutdown(shutdownCtx); err != nil {
			logging.GetLogger().Error("Error during shutdown", "error", err)
		}
	}

	logging.GetLogger().Info("The Seed service stopped")
}

// loadAndConfigureConfigForService loads config without console output.
func loadAndConfigureConfigForService(configPath string) *config.Config {
	cfg, _, err := config.EnsureConfig(configPath, auth.IsDefaultPasswordHash)
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		os.Exit(1)
	}

	if cfg.Auth.JWTSecret == "" {
		cfg.UpdateJWTSecret(auth.GenerateJWTSecret())
		_ = cfg.Save(configPath)
	}

	if errors.Is(err, config.ErrInsecureCredentials) {
		cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	}

	_ = cfg.Validate()
	return cfg
}

// setupLoggingForService configures logging for Windows service mode.
func setupLoggingForService(cfg *config.Config) string {
	logPath := cfg.Logging.File
	if logPath == "" {
		logPath = filepath.Join("logs", "seed.log")
	}

	_ = os.MkdirAll(filepath.Dir(logPath), 0o750)

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

	broadcaster := logging.InitBroadcaster(logBroadcasterBufferSize)
	_ = logging.InitLoggerWithBroadcaster(logCfg, broadcaster)

	return logPath
}

// setupNetworkInterfaceForService initializes network interface without console output.
func setupNetworkInterfaceForService(cfg *config.Config, configPath string) *netif.Manager {
	initialInterface := cfg.Interface.Default
	if initialInterface == "" {
		detected, _ := cfg.GetActiveInterface()
		if detected != "" {
			initialInterface = detected
		}
	}

	if initialInterface == "" {
		logging.GetLogger().Error("No network interface found")
		return nil
	}

	netMgr, err := netif.NewManager(initialInterface)
	if err != nil {
		logging.GetLogger().Error("Failed to initialize network manager", "error", err)
		return nil
	}

	preferred := append([]string{initialInterface}, cfg.Interface.Fallbacks...)
	activeInterface := netMgr.FindFirstAvailable(preferred)

	if activeInterface != "" && activeInterface != cfg.Interface.Default {
		cfg.Interface.Default = activeInterface
		_ = cfg.Save(configPath)
		_ = netMgr.SetCurrentInterface(activeInterface)
	}

	return netMgr
}

// initializeDatabaseForService initializes database for service mode.
func initializeDatabaseForService(cfg *config.Config) *database.DB {
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

// initializeModulesForService creates all application modules.
func initializeModulesForService(cfg *config.Config, db *database.DB) *api.Modules {
	modules := &api.Modules{}

	modules.Sap = services.New(cfg, db)
	modules.Shell = shell.New(cfg, db)
	modules.Canopy = canopy.New(cfg, db)
	modules.Roots = pipeline.New(cfg, db)
	modules.Harvest = harvest.New(cfg, db)

	return modules
}

func initServiceCmd(state *cliState) {
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Windows service management commands",
		Long:  `Manage the Seed Windows service (install, uninstall, start, stop, run).`,
	}

	// Run command - runs as Windows service (internal use)
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run as Windows service (internal use)",
		Long:  `This command is called by the Windows Service Manager. Do not run directly.`,
		Run: func(_ *cobra.Command, _ []string) {
			runWindowsService(state)
		},
	}

	// Install command
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install as Windows service",
		Long:  `Install Seed as a Windows service that starts automatically on boot.`,
		Run: func(_ *cobra.Command, _ []string) {
			installWindowsService()
		},
	}

	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Windows service",
		Long:  `Remove the Seed Windows service.`,
		Run: func(_ *cobra.Command, _ []string) {
			uninstallWindowsService()
		},
	}

	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Windows service",
		Long:  `Start the Seed Windows service.`,
		Run: func(_ *cobra.Command, _ []string) {
			controlWindowsService("start")
		},
	}

	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop Windows service",
		Long:  `Stop the Seed Windows service.`,
		Run: func(_ *cobra.Command, _ []string) {
			controlWindowsService("stop")
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show Windows service status",
		Long:  `Display the current status of the Seed Windows service.`,
		Run: func(_ *cobra.Command, _ []string) {
			showWindowsServiceStatus()
		},
	}

	serviceCmd.AddCommand(runCmd, installCmd, uninstallCmd, startCmd, stopCmd, statusCmd)
	state.rootCmd.AddCommand(serviceCmd)
}

func getServiceConfig() *service.Config {
	execPath, _ := os.Executable()
	return &service.Config{
		Name:        windowsServiceName,
		DisplayName: windowsDisplayName,
		Description: windowsDescription,
		Arguments:   []string{"service", "run"},
		Executable:  execPath,
	}
}

func runWindowsService(state *cliState) {
	prg := &seedProgram{state: state}
	svc, err := service.New(prg, getServiceConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service: %v\n", err)
		os.Exit(1)
	}

	if err := svc.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Service run failed: %v\n", err)
		os.Exit(1)
	}
}

func installWindowsService() {
	prg := &seedProgram{}
	svc, err := service.New(prg, getServiceConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service: %v\n", err)
		os.Exit(1)
	}

	if err := svc.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service installed successfully.")
	fmt.Println("")
	fmt.Println("To start the service:")
	fmt.Println("  seed service start")
	fmt.Println("  or: sc start SeedNetworkDiagnostics")
	fmt.Println("")
	fmt.Println("To configure automatic startup:")
	fmt.Println("  sc config SeedNetworkDiagnostics start= auto")
}

func uninstallWindowsService() {
	prg := &seedProgram{}
	svc, err := service.New(prg, getServiceConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service: %v\n", err)
		os.Exit(1)
	}

	// Stop service first if running
	_ = svc.Stop()

	if err := svc.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to uninstall service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service uninstalled successfully.")
}

func controlWindowsService(action string) {
	prg := &seedProgram{}
	svc, err := service.New(prg, getServiceConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service: %v\n", err)
		os.Exit(1)
	}

	switch action {
	case "start":
		if err := svc.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service started.")
	case "stop":
		if err := svc.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stop service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service stopped.")
	}
}

func showWindowsServiceStatus() {
	prg := &seedProgram{}
	svc, err := service.New(prg, getServiceConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service: %v\n", err)
		os.Exit(1)
	}

	status, err := svc.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get service status: %v\n", err)
		os.Exit(1)
	}

	statusStr := "Unknown"
	switch status {
	case service.StatusRunning:
		statusStr = "Running"
	case service.StatusStopped:
		statusStr = "Stopped"
	case service.StatusUnknown:
		statusStr = "Unknown (service may not be installed)"
	}

	fmt.Printf("Service: %s\n", windowsDisplayName)
	fmt.Printf("Status:  %s\n", statusStr)
}
