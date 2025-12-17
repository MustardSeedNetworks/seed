// Package mcp provides a Model Context Protocol (MCP) server for exposing
// network diagnostic tools to AI assistants like Claude.
//
// MCP is a protocol that allows AI models to interact with external tools
// in a standardized way. This package implements an MCP server that exposes
// the network diagnostic capabilities of Seed as tools that can be called
// by AI assistants.
//
// # Available Tools
//
// The MCP server exposes the following tool categories:
//
// Discovery tools:
//   - network_scan: Scan the local network for devices
//   - get_devices: List all discovered devices
//   - device_fingerprint: Fingerprint a specific device
//   - get_neighbors: Get LLDP/CDP/EDP neighbors
//   - traceroute: Trace route to a target
//   - tcp_probe: Probe a TCP port
//   - port_scan: Scan ports on a host
//
// Testing tools:
//   - dns_test: Test DNS resolution
//   - gateway_ping: Ping the default gateway
//   - speedtest: Run an internet speed test
//   - iperf_test: Run an iPerf3 throughput test
//
// WiFi tools:
//   - wifi_scan: Scan for WiFi networks
//   - wifi_info: Get current WiFi connection info
//
// Security tools:
//   - vulnerability_scan: Scan a device for CVEs
//   - rogue_dhcp_check: Check for rogue DHCP servers
//   - snmp_query: Query SNMP OID on a device
//
// System tools:
//   - get_interfaces: List network interfaces
//   - get_link_status: Get interface link status
//   - get_ip_config: Get IP configuration
//   - get_public_ip: Get public IP address
//   - get_vlan_info: Get VLAN information
//   - system_health: Get system health metrics
//
// # Usage
//
// The MCP server can be used in two modes:
//
// 1. Integrated mode: The MCP server runs alongside the HTTP server and
// accepts connections via SSE (Server-Sent Events) at the /mcp endpoint.
//
// 2. Stdio mode: The MCP server communicates over stdin/stdout for direct
// integration with Claude Code via the "seed mcp" command.
//
// # Security
//
// The MCP server supports JWT authentication and rate limiting to prevent
// abuse. These can be configured via the MCPConfig struct in the config package.
package mcp
