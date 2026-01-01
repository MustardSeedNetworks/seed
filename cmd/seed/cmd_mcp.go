package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/api"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/mcp"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/paths"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server over stdio",
	Long: `Start the Model Context Protocol (MCP) server for AI assistant integration.

The MCP server communicates over stdin/stdout and exposes network diagnostic
tools that can be used by AI assistants like Claude Code.

Example usage in .claude/mcp.json:
{
  "mcpServers": {
    "seed": {
      "command": "./seed",
      "args": ["mcp"],
      "env": {}
    }
  }
}`,
	Run: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(_ *cobra.Command, _ []string) {
	// Resolve config path
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Check ICMP capabilities (for network tools)
	icmpAvailable := false
	if icmpErr := discovery.CheckICMPPrivilegesWithMessage(); icmpErr == nil {
		icmpAvailable = true
	}

	// Initialize minimal network manager for MCP tools
	var netMgr *network.Manager
	activeIface, _ := cfg.GetActiveInterface()
	if activeIface != "" {
		var netErr error
		netMgr, netErr = network.NewManager(activeIface)
		if netErr != nil {
			slog.Warn("Failed to initialize network manager", "error", netErr)
		}
	}

	// Create a minimal API server instance for service access
	// This doesn't start the HTTP server, just provides service access
	// Note: MCP runs over stdio, not HTTP, so trusted proxies/db/modules not needed
	server := api.NewServer(cfg, configPath, "", netMgr, icmpAvailable, nil, nil, nil)

	// Create MCP server
	mcpServer := mcp.NewServer(&cfg.MCP, server)

	// Handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
	}()

	// Run MCP server over stdio
	slog.Info("Starting MCP server over stdio")
	serveErr := mcpServer.ServeStdioWithContext(ctx)
	cancel() // Ensure context is canceled
	if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", serveErr)
		os.Exit(1)
	}
}
