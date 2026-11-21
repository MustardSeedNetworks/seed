//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/paths"
)

const (
	// windowsServiceName is the internal service name used by Windows SCM.
	windowsServiceName = paths.WindowsServiceName

	// windowsDisplayName is the human-readable service name shown in services.msc.
	windowsDisplayName = "Seed Network Diagnostic Service"

	// windowsDescription is the detailed description shown in service properties.
	windowsDescription = "Network diagnostics and monitoring service by Mustard Seed Networks"

	// serviceStopTimeout is the maximum time to wait for graceful shutdown.
	serviceStopTimeout = 30 * time.Second
)

// seedProgram implements service.Interface for Windows service management.
type seedProgram struct {
	state      *cliState
	configPath string
	certDir    string
	lock       *instance.Lock
	config     *config.Config
	server     *api.Server
	components *api.BackgroundComponents
	stopChan   chan struct{}
	stoppedCh  chan struct{}
}

// Start is called when the Windows Service Manager starts the service.
func (p *seedProgram) Start(_ service.Service) error {
	// ModeSystem, not ModeAuto: os.Getuid is -1 on Windows, so auto-detection
	// said "user" and put the config and JWT secret under the profile of the
	// account the service runs as.
	sp := paths.Resolve(paths.ModeSystem)
	if err := prepareServiceDirs(sp); err != nil {
		return err
	}
	// The SCM starts the service in C:\Windows\System32. Every storage path in
	// the config is made absolute below; this catches the ones that are code
	// constants, such as the report directory, so none can fall back there.
	if err := os.Chdir(sp.Root); err != nil {
		return fmt.Errorf("enter service directory: %w", err)
	}
	p.certDir = sp.CertDir
	p.configPath = paths.ResolveConfigPath(p.state.cfgFile, paths.ModeSystem)
	// The single-instance lock is what `seed status` and the write guards read.
	// Without it a console could not tell the service from nothing running.
	lock, err := instance.Acquire(lockDir(p.configPath))
	if err != nil {
		return fmt.Errorf("take the single-instance lock: %w", err)
	}
	p.lock = lock
	cfg, err := loadAndConfigureConfigForService(p.configPath)
	if err != nil {
		_ = lock.Release()
		return err
	}
	anchorServicePaths(cfg, sp)
	p.config = cfg
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
	defer func() { _ = p.lock.Release() }()

	configPath := p.configPath
	cfg := p.config
	logPath := setupLoggingForService(cfg)

	netMgr := setupNetworkInterfaceForService(cfg, configPath)

	// Create trusted proxies configuration
	proxies := api.NewTrustedProxies(p.state.trustedProxies)
	if !proxies.IsEmpty() {
		logging.GetLogger().Info("Trusted proxies configured", "count", proxies.Count())
	}

	// Initialize database
	persist := initializeDatabaseForService(cfg)

	// Initialize components
	p.components = initializeBackgroundComponentsForService(cfg, persist)

	server, err := api.NewServer(api.Deps{
		Config:         cfg,
		ConfigPath:     configPath,
		LogPath:        logPath,
		CertDir:        p.certDir,
		NetManager:     netMgr,
		ICMPAvailable:  true,
		TrustedProxies: proxies,
		Persistence:    persist,
		Background:     p.components,
	})
	if err != nil {
		logging.GetLogger().Error("Failed to build the server", "error", err)
		return
	}
	p.server = server
	server.SetBoundPortObserver(func(port int) {
		if setErr := p.lock.SetPort(port); setErr != nil {
			logging.GetLogger().Warn("Could not publish the bound port", "error", setErr)
		}
	})

	// Start components
	ctx := context.Background()
	if p.components != nil {
		if err = p.components.Start(ctx); err != nil {
			logging.GetLogger().Error("Failed to start components", "error", err)
			return
		}
		logging.GetLogger().Info("All components started successfully")
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logging.GetLogger().Info("Starting HTTPS server", "port", cfg.Server.Port)
		serverErrors <- p.server.Start()
	}()

	// Wait for stop signal or server error
	select {
	case err = <-serverErrors:
		if err != nil {
			logging.GetLogger().Error("Server error", "error", err)
		}
	case <-p.stopChan:
		logging.GetLogger().Info("Service stop requested, shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), serviceStopTimeout)
		defer cancel()

		// The service carried its own copy of the shutdown sequence, with the
		// same wrong order runServerWithShutdown had (#2748). One seam now.
		var stopComponents func() error
		if p.components != nil {
			stopComponents = p.components.Stop
		}
		stopServing(shutdownCtx, p.server, stopComponents)
	}

	logging.GetLogger().Info("Seed service stopped")
}

// setupLoggingForService configures logging for Windows service mode.
func setupLoggingForService(cfg *config.Config) string {
	logPath := cfg.Logging.File

	_ = ensureDataDir(filepath.Dir(logPath))

	// A service has no console: os.Stdout is an invalid handle, and InitLogger
	// writes to stdout ahead of the file through an io.MultiWriter, which stops
	// at the first error. Without the discard writer the log file was never
	// written.
	logCfg := &logging.LoggingConfig{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		AddSource:  cfg.Logging.AddSource,
		File:       logPath,
		Writer:     io.Discard,
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
		_ = savePrivateConfig(cfg, configPath)
		_ = netMgr.SetCurrentInterface(activeInterface)
	}

	return netMgr
}

// initializeDatabaseForService initializes database for service mode.
func initializeDatabaseForService(cfg *config.Config) *app.Persistence {
	dbPath := cfg.Database.Path
	db, err := database.OpenWithAutoRebuild(dbPath)
	if err != nil {
		logging.GetLogger().Error("Failed to open database", "path", dbPath, "error", err)
		return nil
	}

	logging.GetLogger().Info("Database initialized", "path", dbPath)
	return app.NewPersistence(db)
}

// initializeBackgroundComponentsForService creates all application components.
func initializeBackgroundComponentsForService(cfg *config.Config, persist *app.Persistence) *api.BackgroundComponents {
	components := &api.BackgroundComponents{}

	components.Reporting = app.NewReporting(cfg, persist)

	return components
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

// checkServiceExecutable refuses an executable that is not inside installDir.
// The service runs as LocalSystem from whatever path it is registered with, so
// registering a binary an unprivileged user can replace is a local privilege
// escalation. exe must already have its symlinks resolved.
func checkServiceExecutable(exe, installDir string) error {
	if !filepath.IsAbs(exe) || !filepath.IsAbs(installDir) {
		return fmt.Errorf("service executable %q and install directory %q must be absolute paths", exe, installDir)
	}
	rel, err := filepath.Rel(installDir, exe)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to register %s as a service: it is not under %s; "+
			"run install.ps1 from the release archive, which copies seed.exe there", exe, installDir)
	}
	return nil
}

// installableExecutable returns the running binary with symlinks resolved, or
// an error if it is not where the service may be registered from.
func installableExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate seed.exe: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", exe, err)
	}
	if checkErr := checkServiceExecutable(resolved, paths.Resolve(paths.ModeSystem).BinaryDir); checkErr != nil {
		return "", checkErr
	}
	return resolved, nil
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

	if runErr := svc.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "Service run failed: %v\n", runErr)
		os.Exit(1)
	}
}

func installWindowsService() {
	execPath, err := installableExecutable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	svcCfg := getServiceConfig()
	svcCfg.Executable = execPath
	svc, err := service.New(&seedProgram{}, svcCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create service: %v\n", err)
		os.Exit(1)
	}

	if installErr := svc.Install(); installErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", installErr)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "Service installed successfully.")
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "To start the service:")
	fmt.Fprintln(os.Stdout, "  seed service start")
	fmt.Fprintln(os.Stdout, "  or: sc start SeedNetworkDiagnostics")
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "To configure automatic startup:")
	fmt.Fprintln(os.Stdout, "  sc config SeedNetworkDiagnostics start= auto")
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

	if uninstallErr := svc.Uninstall(); uninstallErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to uninstall service: %v\n", uninstallErr)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "Service uninstalled successfully.")
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
		if startErr := svc.Start(); startErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to start service: %v\n", startErr)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "Service started.")
	case "stop":
		if stopErr := svc.Stop(); stopErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to stop service: %v\n", stopErr)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "Service stopped.")
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

	fmt.Fprintf(os.Stdout, "Service: %s\n", windowsDisplayName)
	fmt.Fprintf(os.Stdout, "Status:  %s\n", statusStr)
}
