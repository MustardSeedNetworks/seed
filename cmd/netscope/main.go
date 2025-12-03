// Package main is the entry point for NetScope.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krisarmstrong/netscope/internal/api"
	"github.com/krisarmstrong/netscope/internal/config"
	"github.com/krisarmstrong/netscope/internal/network"
)

var version = "dev"

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

	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("NetScope %s starting...", version)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *devMode {
		log.Println("Running in development mode")
		cfg.Server.HTTPS = false // Use HTTP in dev mode
	}

	// Initialize network manager
	netMgr := network.NewManager(cfg.Interface.Default)

	// Find first available interface
	preferred := append([]string{cfg.Interface.Default}, cfg.Interface.Fallbacks...)
	activeInterface := netMgr.FindFirstAvailable(preferred)
	if activeInterface == "" {
		log.Println("Warning: No active network interface found")
	} else {
		log.Printf("Using interface: %s", activeInterface)
		netMgr.SetCurrentInterface(activeInterface)
	}

	// Log available interfaces
	for _, iface := range netMgr.GetInterfaces() {
		status := "down"
		if iface.Up {
			status = "up"
		}
		log.Printf("  Interface: %s (%s) - %s", iface.Name, iface.Type, status)
	}

	// Create and start the server
	server := api.NewServer(cfg, netMgr)

	// Handle shutdown gracefully
	done := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
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
