// Package main is the entry point for LuminetIQ.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/krisarmstrong/luminetiq/internal/api"
	"github.com/krisarmstrong/luminetiq/internal/auth"
	"github.com/krisarmstrong/luminetiq/internal/config"
	"github.com/krisarmstrong/luminetiq/internal/discovery"
	"github.com/krisarmstrong/luminetiq/internal/network"
)

var version = "dev"

// main starts the LuminetIQ network discovery and monitoring application.
// It initializes configuration from a YAML file, sets up logging, validates
// network interface availability, and starts the API server with graceful shutdown handling.
//
// Command-line flags:
//
//	-version: Display the application version and exit
//	-config: Path to the YAML configuration file (default: "configs/luminetiq.yaml")
//	-dev: Run in development mode using HTTP instead of HTTPS
//
// The application requires elevated privileges (root or CAP_NET_RAW) for ICMP ping operations
// on Linux systems. It validates that a default network interface is configured and attempts
// to find an active interface with retry logic (up to 3 retries with 5-second intervals).
// If no active interface is found, it logs available interfaces grouped by type and status.
//
// Graceful shutdown is handled via SIGINT and SIGTERM signals, allowing the server up to
// 30 seconds to clean up before terminating.
//
// Fatal conditions:
//   - Missing ICMP privileges
//   - Failed configuration loading
//   - No default network interface specified in configuration
func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version")
	configPath := flag.String("config", "configs/luminetiq.yaml", "Path to configuration file")
	devMode := flag.Bool("dev", false, "Run in development mode")
	flag.Parse()

	if *showVersion {
		fmt.Printf("LuminetIQ %s\n", version)
		os.Exit(0)
	}

	// Check for ICMP privileges (raw socket access for ping features)
	// Continues gracefully if unavailable - ICMP features will be disabled
	icmpAvailable := true
	if err := discovery.CheckICMPPrivilegesWithMessage(); err != nil {
		icmpAvailable = false
		log.Printf("Warning: ICMP features disabled - %v", err)
		fmt.Fprintln(os.Stderr, "Warning: Running without ICMP privileges - ping features will be unavailable")
		fmt.Fprintln(os.Stderr, "For full functionality, run with: sudo ./luminetiq")
		fmt.Fprintln(os.Stderr, "Or grant capability: sudo setcap cap_net_raw=+ep ./luminetiq")
	}

	// Set up logging with secure permissions (fixes #537)
	logPath := filepath.Join("logs", "luminetiq.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// Create log file with restrictive permissions (0600) before lumberjack uses it
	// This ensures log files containing sensitive information are only readable by owner
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			log.Fatalf("Failed to create log file with secure permissions: %v", err)
		}
		f.Close()
	} else {
		// If log file exists, ensure it has correct permissions
		if err := os.Chmod(logPath, 0o600); err != nil {
			log.Printf("Warning: Failed to set secure permissions on existing log file: %v", err)
		}
	}

	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    20, // megabytes
		MaxBackups: 7,  // keep last 7 files
		MaxAge:     30, // days
		Compress:   true,
	}
	log.SetOutput(rotator)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	log.Printf("LuminetIQ %s starting, logging to %s", version, logPath)

	// Load configuration with first-boot security check
	cfg, setupResult, err := config.EnsureConfig(*configPath, auth.IsDefaultPasswordHash)
	if err != nil {
		if errors.Is(err, config.ErrInsecureCredentials) {
			// Don't auto-generate credentials - let the web wizard handle it
			// Just ensure we have a JWT secret for the server to start
			if cfg.Auth.JWTSecret == "" {
				jwtSecret := auth.GenerateJWTSecret()
				cfg.UpdateJWTSecret(jwtSecret)
				if saveErr := cfg.Save(*configPath); saveErr != nil {
					log.Printf("Warning: Failed to persist JWT secret: %v", saveErr)
				}
			}
			// Log that setup is needed
			log.Println("Initial setup required - visit the web UI to set your admin password")
			printSetupBanner(cfg.Server.Port, cfg.Server.HTTPS)
		} else {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	} else if setupResult != nil && setupResult.JWTSecretStored {
		// JWT secret was empty and needs to be generated and persisted
		jwtSecret := auth.GenerateJWTSecret()
		cfg.UpdateJWTSecret(jwtSecret)
		if saveErr := cfg.Save(*configPath); saveErr != nil {
			log.Printf("Warning: Failed to persist JWT secret: %v", saveErr)
		} else {
			log.Println("JWT secret generated and persisted to config file")
		}
	}

	// Encrypt SNMP credentials if they're in plaintext (fixes #518 - migration)
	if cfg.Auth.JWTSecret != "" && len(cfg.SNMP.V3Credentials) > 0 {
		needsSave := false
		for i := range cfg.SNMP.V3Credentials {
			cred := &cfg.SNMP.V3Credentials[i]
			// Check if passwords need encryption
			if (cred.AuthPassword != "" && !config.IsEncrypted(cred.AuthPassword)) ||
				(cred.PrivPassword != "" && !config.IsEncrypted(cred.PrivPassword)) {
				needsSave = true
				break
			}
		}

		if needsSave {
			log.Println("Migrating SNMP credentials to encrypted format...")
			if err := cfg.EncryptSNMPCredentials(); err != nil {
				log.Printf("Warning: Failed to encrypt SNMP credentials: %v", err)
			} else if saveErr := cfg.Save(*configPath); saveErr != nil {
				log.Printf("Warning: Failed to persist encrypted SNMP credentials: %v", saveErr)
			} else {
				log.Println("SNMP credentials encrypted and saved securely")
			}
		}
	}

	// Optional log access token override via environment (fixes #538)
	if token := os.Getenv("LOG_ACCESS_TOKEN"); token != "" {
		log.Println("Environment variable override: LOG_ACCESS_TOKEN is set")
		cfg.Server.LogAccessToken = token
	}
	if hdr := os.Getenv("LOG_ACCESS_HEADER"); hdr != "" {
		log.Printf("Environment variable override: LOG_ACCESS_HEADER=%s", hdr)
		cfg.Server.LogAccessHeader = hdr
	}

	if *devMode {
		log.Println("Running in development mode")
		cfg.Server.HTTPS = false // Use HTTP in dev mode
		log.Println("Protocol: HTTP (development mode)")
	}

	// Validate configuration after all modifications (fixes #542)
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	// Initialize network manager
	if cfg.Interface.Default == "" {
		log.Fatalf("No default network interface specified in configuration")
	}
	netMgr, err := network.NewManager(cfg.Interface.Default)
	if err != nil {
		log.Fatalf("Failed to initialize network manager: %v", err)
	}
	preferred := append([]string{cfg.Interface.Default}, cfg.Interface.Fallbacks...)
	activeInterface := netMgr.FindFirstAvailable(preferred)
	retryCount := 0
	// Use configurable retry logic (fixes #528)
	maxRetries := cfg.Interface.StartupRetries
	retryWait := cfg.Interface.StartupRetryWait
	for activeInterface == "" && retryCount < maxRetries {
		log.Printf("Warning: No active network interface found. Retrying in %v...", retryWait)
		time.Sleep(retryWait)
		activeInterface = netMgr.FindFirstAvailable(preferred)
		retryCount++
	}
	if activeInterface == "" {
		log.Println("Error: No active network interface found after multiple attempts.")
		log.Println("Please check your network configuration and ensure at least one interface is up.")
		// Log available interfaces grouped by type and status
		type ifaceGroup struct {
			Type   string
			Status string
		}
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
			log.Printf("Interfaces (%s, %s): %v", group.Type, group.Status, names)
		}
	}

	// If we found a usable interface, make it the active/default everywhere
	if activeInterface != "" {
		if activeInterface != cfg.Interface.Default {
			log.Printf("Using detected active interface %s instead of configured default %s", activeInterface, cfg.Interface.Default)
			cfg.Interface.Default = activeInterface

			// Persist the updated interface to config (fixes #521)
			if err := cfg.Save(*configPath); err != nil {
				log.Printf("Warning: Failed to save updated interface to config: %v", err)
			} else {
				log.Printf("Updated config with active interface: %s", activeInterface)
			}
		}
		if err := netMgr.SetCurrentInterface(activeInterface); err != nil {
			log.Printf("Warning: failed to set active interface %s: %v", activeInterface, err)
		}
	}

	// Create and start the server
	server := api.NewServer(cfg, *configPath, logPath, netMgr, icmpAvailable)

	// Handle shutdown gracefully (fixes #541)
	// Buffer size 2 to catch second signal for force quit
	done := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 2)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// First signal: graceful shutdown
		<-sigChan
		log.Println("Shutting down gracefully... (press Ctrl+C again to force)")

		// Start graceful shutdown in goroutine
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
				log.Printf("Error during shutdown: %v", shutdownErr)
			}
			close(done)
		}()

		// Second signal: force quit
		<-sigChan
		log.Println("Force quitting...")
		os.Exit(1)
	}()

	// Start the server
	log.Printf("Starting server on port %d (HTTPS: %v)", cfg.Server.Port, cfg.Server.HTTPS)
	if err := server.Start(); err != nil {
		log.Printf("Server error: %v", err)
	}

	<-done
	log.Println("LuminetIQ stopped")
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
	log.Printf("Setup required - visit %s://localhost:%d to complete setup", protocol, port)
}
