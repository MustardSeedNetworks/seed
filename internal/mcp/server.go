package mcp

import (
	"context"
	"os"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/version"
)

// DiscoveryServiceProvider provides access to discovery-related services.
type DiscoveryServiceProvider interface {
	GetDiscoveryService() DiscoveryService
	GetDeviceDiscovery() DeviceDiscovery
}

// NetworkServiceProvider provides access to network-related services.
type NetworkServiceProvider interface {
	GetNetManager() NetworkManager
	GetLinkMonitor() LinkMonitor
	GetVLANManager() VLANManager
}

// TestingServiceProvider provides access to testing-related services.
type TestingServiceProvider interface {
	GetDNSTester() DNSTester
	GetGatewayTester() GatewayTester
	GetSpeedtestTester() SpeedtestTester
	GetIperfManager() IperfManager
}

// WiFiServiceProvider provides access to WiFi-related services.
type WiFiServiceProvider interface {
	GetWiFiScanner() WiFiScanner
	GetWiFiManager() WiFiManager
}

// SecurityServiceProvider provides access to security-related services.
type SecurityServiceProvider interface {
	GetRogueDetector() RogueDetector
	GetVulnScanner() VulnScanner
}

// SystemServiceProvider provides access to system-related services.
type SystemServiceProvider interface {
	GetPublicIPChecker() PublicIPChecker
	GetConfig() *config.Config
	IsICMPAvailable() bool
}

// ServiceProvider defines the interface for accessing Seed services.
// This is implemented by api.Server to provide access to all service managers.
// It composes smaller, focused interfaces for better maintainability.
type ServiceProvider interface {
	DiscoveryServiceProvider
	NetworkServiceProvider
	TestingServiceProvider
	WiFiServiceProvider
	SecurityServiceProvider
	SystemServiceProvider
}

// Server wraps the MCP server and provides tool registration.
type Server struct {
	mcpServer *server.MCPServer
	config    *config.MCPConfig
	services  ServiceProvider
}

// NewServer creates a new MCP server with all tools registered.
func NewServer(cfg *config.MCPConfig, services ServiceProvider) *Server {
	s := &Server{
		config:   cfg,
		services: services,
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"The Seed Network Diagnostics",
		version.GetVersion(),
		server.WithToolCapabilities(false), // We handle tool listing
		server.WithRecovery(),              // Recover from panics
	)

	// Register all tools
	s.registerTools()

	return s
}

// registerTools registers all available tools with the MCP server.
func (s *Server) registerTools() {
	// Check if tool is allowed based on config
	isAllowed := func(toolName string) bool {
		if len(s.config.AllowedTools) == 0 {
			return true // All tools allowed when list is empty
		}
		return slices.Contains(s.config.AllowedTools, toolName)
	}

	// Register discovery tools
	s.registerDiscoveryTools(isAllowed)

	// Register testing tools
	s.registerTestingTools(isAllowed)

	// Register WiFi tools
	s.registerWiFiTools(isAllowed)

	// Register security tools
	s.registerSecurityTools(isAllowed)

	// Register system tools
	s.registerSystemTools(isAllowed)
}

// ServeStdio runs the MCP server over stdin/stdout.
// This is used for direct integration with Claude Code.
func (s *Server) ServeStdio() error {
	logging.GetLogger().Info("Starting MCP server over stdio")
	return server.ServeStdio(s.mcpServer)
}

// ServeStdioWithContext runs the MCP server over stdin/stdout with context.
func (s *Server) ServeStdioWithContext(ctx context.Context) error {
	logging.GetLogger().InfoContext(ctx, "Starting MCP server over stdio with context")

	// Create a channel to signal completion
	done := make(chan error, 1)

	go func() {
		done <- server.ServeStdio(s.mcpServer)
	}()

	select {
	case <-ctx.Done():
		// Context canceled, close stdin to stop the server
		_ = os.Stdin.Close()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// GetMCPServer returns the underlying MCP server for advanced usage.
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// addTool is a helper to add a tool if it's allowed by config.
//

func (s *Server) addTool(
	name string,
	isAllowed func(string) bool,
	tool mcp.Tool,
	handler server.ToolHandlerFunc,
) {
	if !isAllowed(name) {
		logging.GetLogger().Debug("Tool not in allowed list, skipping", "tool", name)
		return
	}
	s.mcpServer.AddTool(tool, handler)
	logging.GetLogger().Debug("Registered MCP tool", "tool", name)
}
