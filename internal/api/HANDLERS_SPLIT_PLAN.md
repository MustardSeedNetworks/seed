# Handlers File Split Plan

## Current State

- `handlers.go`: 4674 lines (too large for maintainability)
- Contains 47+ HTTP handlers plus helper functions and types

## Proposed Split (fixes #544)

### File Structure

1. **handlers_types.go** - Shared types and helper functions
   - `sendJSONResponse()`
   - `readLastLines()`
   - Common request/response types used across multiple handlers

2. **handlers_auth.go** - Authentication & Setup
   - `handleLogin()`
   - `handleLogout()`
   - `handleSetupStatus()`
   - `handleSetupComplete()`
   - Types: `LoginRequest`, `LoginResponse`

3. **handlers_status.go** - System Status & Export
   - `handleStatus()`
   - `handleSystemHealth()`
   - `handleExport()`
   - `handleLogs()`
   - Types: `StatusResponse`, `ExportData`

4. **handlers_network.go** - Network Interfaces & Configuration
   - `handleInterface()`, `handleInterfaces()`
   - `handleLink()`, `handleSetMTU()`
   - `handleIPConfig()`, `handleIPSettings()`, `handleIPSettingsGet()`, `handleIPSettingsPut()`
   - `handleVLAN()`, `handleVLANInterface()`, `handleVLANTraffic()`
   - `handleWiFi()`, `handleWiFiSettings()`
   - `handleCable()`
   - Types: `SetInterfaceRequest`, `LinkResponse`, `IPConfigResponse`, etc.

5. **handlers_discovery.go** - Device Discovery
   - `handleDiscovery()`
   - `handleDevices()`, `handleDevicesScan()`, `handleDevicesStatus()`, `handleDevicesSettings()`,
     `handleDevicesSubnets()`
   - `handleDiscoveryProfile()`, `handleDiscoveryServiceStatus()`
   - Types: `DiscoveryResponse`, `DiscoveryNeighborInfo`

6. **handlers_tests.go** - Network Testing
   - `handleDNS()`, `handleGateway()`, `handlePublicIP()`
   - `handleSpeedtest()`, `handleSpeedtestStatus()`
   - `handleIperfClient()`, `handleIperfClientStatus()`, `handleIperfInfo()`, `handleIperfServer()`,
     `handleIperfServerStatus()`, `handleIperfSuggestions()`
   - `handleCustomTests()`, `handleTestsSettings()`
   - Types: DNS/speedtest/iperf related types

7. **handlers_security.go** - Security Features
   - `handleRogueDHCP()`, `handleRogueDHCPConfig()`, `handleRogueDHCPServers()`
   - `handleSNMPSettings()`
   - Types: Rogue DHCP, SNMP related types

8. **handlers_tools.go** - Advanced Network Tools
   - `handlePortScan()`
   - `handleTCPProbe()`
   - `handleTraceroute()`
   - `handleAdvancedFingerprint()`
   - Types: `PortScanRequest`, `TCPProbeRequest`, `TracerouteRequest`

9. **handlers_settings.go** - Application Settings
   - `handleSettings()`, `getSettings()`, `updateSettings()`

## Migration Strategy

1. Create `handlers_types.go` with shared utilities first
2. Extract one category at a time (start with auth as it's smallest)
3. Build and test after each extraction
4. Update imports in moved files
5. Remove extracted code from original `handlers.go`
6. Final verification that all handlers still work

## Testing Strategy

- Build verification after each file split
- Manual API testing for moved endpoints
- Ensure no duplicate function definitions
- Verify all types are accessible where needed

## Estimated Time

- 6 hours total for complete split and verification
- Can be done incrementally over multiple commits
