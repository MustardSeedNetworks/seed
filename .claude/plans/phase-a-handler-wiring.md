# Phase A: Wire API Handlers to Module Services

## Current State Analysis

### Server Struct Dependencies
The Server struct currently creates service instances directly:
```go
type Server struct {
    dnsTester           *dns.Tester
    dnsSecurityScanner  *dns.SecurityScanner
    gatewayTester       *gateway.Tester
    vlanManager         *vlan.Manager
    vlanTrafficMonitor  *vlan.TrafficMonitor
    wifiManager         *wifi.Manager
    wifiScanner         *wifi.Scanner
    cableTester         *cable.Tester
    speedtestTester     *speedtest.Tester
    surveyManager       *survey.Manager
    publicipChecker     *publicip.Checker
    // ...
}
```

### Module Services Available
Modules expose services through wrapper types:
- `s.modules.Sap.DNS()` → `*DNSService`
- `s.modules.Sap.Gateway()` → `*GatewayService`
- etc.

### Gap Analysis
Module services don't expose all methods handlers need. For example:
- Handler uses: `s.dnsTester.Test(ctx)`
- Module has: `sap.DNSService.Test(ctx, server, hostname)`

## Implementation Strategy

### Option 1: Full Module Integration (Recommended)
1. Expand module services to expose all needed methods
2. Remove direct package instances from Server struct
3. Update handlers to use module services

### Option 2: Proxy Pattern
1. Module services expose underlying instances
2. Handlers access via `s.modules.Sap.DNS().Tester()`
3. Less refactoring but less encapsulation

## Phase A Implementation Steps

### Step 1: Expand Sap Module Services
- [ ] DNSService: Add security scanner methods
- [ ] GatewayService: Add continuous monitoring methods
- [ ] VLANService: Add traffic monitor methods
- [ ] PerformanceService: Add speedtest/iperf methods

### Step 2: Expand Canopy Module Services
- [ ] WiFiService: Add scanner methods
- [ ] SurveyService: Add manager methods

### Step 3: Expand Roots Module Services
- [ ] PublicIPService: Add checker methods

### Step 4: Expand Shell Module Services
- [ ] DiscoveryService: Add all discovery methods
- [ ] VulnerabilityService: Add scanner methods

### Step 5: Update Server Struct
- [ ] Remove direct package instances
- [ ] Use modules for all service access
- [ ] Update NewServer to not create instances

### Step 6: Update Handlers
- [ ] handlers_dns.go → use s.modules.Sap.DNS()
- [ ] handlers_health_checks.go → use s.modules.Sap.*
- [ ] handlers_wifi.go → use s.modules.Canopy.WiFi()
- [ ] handlers_survey.go → use s.modules.Canopy.Survey()
- [ ] handlers_cable.go → use s.modules.Sap.Cable()
- [ ] handlers_vlan.go → use s.modules.Sap.VLAN()
- [ ] handlers_security.go → use s.modules.Shell.*
- [ ] handlers_discovery.go → use s.modules.Shell.Discovery()
- [ ] mcp_provider.go → use modules

### Step 7: Update Tests
- [ ] Update test helpers
- [ ] Mock modules for unit tests

## Estimated Effort
- Step 1-4: 2-3 hours (expand services)
- Step 5-6: 2-3 hours (update handlers)
- Step 7: 1-2 hours (tests)

## Files to Modify

### Module Service Files
- internal/sap/services.go
- internal/canopy/services.go
- internal/roots/services.go
- internal/shell/services.go

### API Handler Files
- internal/api/server.go
- internal/api/handlers_dns.go
- internal/api/handlers_health_checks.go
- internal/api/handlers_wifi.go
- internal/api/handlers_survey.go
- internal/api/handlers_cable.go
- internal/api/handlers_vlan.go
- internal/api/handlers_security.go
- internal/api/handlers_discovery.go
- internal/api/mcp_provider.go
- internal/api/testing.go
