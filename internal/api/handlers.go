// Package api provides HTTP handlers and API endpoints for the LuminetIQ network monitoring application.
//
// This file serves as the main orchestrator for HTTP request handling. The actual handler implementations
// have been split into specialized files for better organization and maintainability (fixes #544):
//
//   - handlers_types.go     - Shared utility functions (sendJSONResponse, etc.)
//   - handlers_status.go    - System status, health, logs, and export endpoints
//   - handlers_network.go   - Network interface, link, IP config, VLAN, WiFi, cable handlers
//   - handlers_settings.go  - Application settings (GET/PUT)
//   - handlers_tools.go     - Advanced diagnostic tools (TCP probe, traceroute, port scan, fingerprinting)
//   - handlers_security.go  - Security features (rogue DHCP detection, SNMP settings)
//   - handlers_tests.go     - Network testing (DNS, custom tests, speedtest, iperf)
//   - handlers_discovery.go - Device discovery, public IP, LLDP/CDP, WiFi survey
//
// Route registration and server setup are handled in server.go.
//
// All handlers require authentication unless explicitly exempted (setup endpoints).
// Rate limiting is applied to expensive operations via middleware.
//
// Security considerations:
//   - Input validation via internal/validation package
//   - Authentication via JWT tokens (internal/auth)
//   - CORS restrictions based on configured allowed origins
//   - Rate limiting on resource-intensive endpoints
//   - Secure credential handling with encryption for sensitive data (SNMP, WiFi)
package api
