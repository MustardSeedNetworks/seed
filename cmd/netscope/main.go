// Package main is the entry point for NetScope.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/krisarmstrong/netscope/internal/api"
	"github.com/krisarmstrong/netscope/internal/config"
	"github.com/krisarmstrong/netscope/internal/discovery"
	"github.com/krisarmstrong/netscope/internal/network"
)

var version = "dev"

// main starts the NetScope network discovery and monitoring application.
// It initializes configuration from a YAML file, sets up logging, validates
// network interface availability, and starts the API server with graceful shutdown handling.
//
// Command-line flags:
//
//	-version: Display the application version and exit
//	-config: Path to the YAML configuration file (default: "configs/netscope.yaml")
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
	configPath := flag.String("config", "configs/netscope.yaml", "Path to configuration file")
	devMode := flag.Bool("dev", false, "Run in development mode")
	flag.Parse()

	if *showVersion {
		fmt.Printf("NetScope %s\n", version)
		os.Exit(0)
	}

	// Check for required privileges (raw socket access for ICMP ping)
	// This is a Linux-only tool that requires root or CAP_NET_RAW
	discovery.MustHaveICMPPrivileges()

	// Set up logging
	logPath := filepath.Join("logs", "netscope.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
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
	log.Printf("NetScope %s starting, logging to %s", version, logPath)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Optional log access token override via environment
	if token := os.Getenv("LOG_ACCESS_TOKEN"); token != "" {
		cfg.Server.LogAccessToken = token
	}
	if hdr := os.Getenv("LOG_ACCESS_HEADER"); hdr != "" {
		cfg.Server.LogAccessHeader = hdr
	}

	if *devMode {
		log.Println("Running in development mode")
		cfg.Server.HTTPS = false // Use HTTP in dev mode
		log.Println("Protocol: HTTP (development mode)")
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
	for activeInterface == "" && retryCount < 3 {
		log.Println("Warning: No active network interface found. Retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
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

	// Create and start the server
	server := api.NewServer(cfg, *configPath, logPath, netMgr)

	// Handle shutdown gracefully
	done := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
			log.Printf("Error during shutdown: %v", shutdownErr)
		}
		close(done)
	}()

	// Start the server
	log.Printf("Starting server on port %d (HTTPS: %v)", cfg.Server.Port, cfg.Server.HTTPS)
	if err := server.Start(); err != nil {
		log.Printf("Server error: %v", err)
	}

	<-done
	log.Println("NetScope stopped")
}
