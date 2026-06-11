# Route Inventory (current state)

**Generated:** 2026-05-31 from `internal/api/server_routes.go` + `internal/api/handlers_update.go`
**Prefixes updated:** 2026-06-02 — botanical route prefixes renamed to descriptive ones
(`sap→telemetry`, `roots→path`, `shell→security`, `canopy→wifi`, `harvest→reporting`); the
route set and policy trio are otherwise unchanged. Live source of truth is the
`/__capabilities` manifest (golden: `internal/api/testdata/golden/capabilities.txt`).
**Status:** Phase-0 artifact — source data for the Phase-1 capability registry and the golden-harness gating matrix.
**Regenerate:** re-run the extraction over the two route files (paren-balanced parse of `s.mux.Handle*` calls).

## Why this exists

This is the authoritative snapshot of the **per-route policy trio** that the
capability registry (ADR-0002) will move from hand-wrapping into declarative
`Route{}` values:

- **Min role** — `operator` where the route is wrapped in `writeGated` today.
- **Feature** — the `requireFeature("…")` license gate, if any.
- **Rate-limit** — wrapped in `endpointRateLimiter().RateLimitMiddleware` today.

**Not** in this table (enforced globally in `server_lifecycle.go`, unchanged by the
re-arch): **authentication** (`authManager.Middleware`) and **CSRF**
(`csrfManager.CSRFMiddleware`). **Method** is handler-internal under `http.ServeMux`
today; the registry makes it explicit per route.

## Summary

| Metric | Count |
|---|---|
| API routes | **183** (+1 SPA static `/`) |
| Operator-gated (`writeGated`) | 54 |
| Feature-gated (`requireFeature`) | 9 routes across 7 features |
| Rate-limited | 13 |

**Licensed features in use:** `airmapper_baseline_diff`, `anomaly_detection`,
`compliance_advanced`, `export_csv_json`, `live_telemetry`, `path_analysis`, `sso`.

> The actual count (183) is materially higher than the early "~120" estimate — recent
> `alerts`/`alert-rules` routes were missed by manual reads. This generated inventory is
> the reliable number and the Phase-1 registry's input of record.

## Routes

| Path | Handler | Min role | Feature | Rate-limit |
|------|---------|----------|---------|-----------|
| `None` | `handleUpdateCheck` | — | — | — |
| `None` | `handleUpdateStatus` | — | — | — |
| `None` | `handleUpdateInfo` | — | — | — |
| `None` | `handleUpdateDownload` | operator | — | — |
| `None` | `handleUpdateApply` | operator | — | — |
| `None` | `handleUpdateRollback` | operator | — | — |
| `None` | `handleGetUpdateConfig` | — | — | — |
| `None` | `handleUpdateConfig` | operator | — | — |
| `/__version` | `handleBuildVersion` | — | — | — |
| `/api/v1/alert-rules` | `handleAlertRules` | operator | — | — |
| `/api/v1/alert-rules/` | `handleAlertRuleByID` | operator | — | — |
| `/api/v1/alerts` | `handleAlerts` | — | — | — |
| `/api/v1/alerts/` | `handleAlertAction` | operator | — | — |
| `/api/v1/auth/csrf` | `handleCSRFToken` | — | — | — |
| `/api/v1/auth/login` | `handleLogin` | — | — | — |
| `/api/v1/auth/login/totp` | `handleLoginTOTP` | — | — | — |
| `/api/v1/auth/logout` | `handleLogout` | — | — | — |
| `/api/v1/auth/mfa/status` | `handleMFAStatus` | — | — | — |
| `/api/v1/auth/refresh` | `handleRefreshToken` | — | — | — |
| `/api/v1/auth/totp/disable` | `handleTOTPDisable` | — | — | — |
| `/api/v1/auth/totp/setup` | `handleTOTPSetup` | — | — | — |
| `/api/v1/auth/totp/verify` | `handleTOTPVerify` | — | — | — |
| `/api/v1/auth/webauthn/login/begin` | `handleWebAuthnLoginBegin` | — | — | — |
| `/api/v1/auth/webauthn/login/finish` | `handleWebAuthnLoginFinish` | — | — | — |
| `/api/v1/auth/webauthn/register/begin` | `handleWebAuthnRegisterBegin` | — | — | — |
| `/api/v1/auth/webauthn/register/finish` | `handleWebAuthnRegisterFinish` | — | — | — |
| `/api/v1/wifi/survey` | `getSurvey` | — | — | — |
| `/api/v1/wifi/survey/active-floor` | `setActiveFloor` | operator | — | — |
| `/api/v1/wifi/survey/complete` | `completeSurvey` | operator | — | — |
| `/api/v1/wifi/survey/create` | `createSurvey` | operator | — | — |
| `/api/v1/wifi/survey/dead-zones` | `getSurveyDeadZones` | — | — | — |
| `/api/v1/wifi/survey/delete` | `deleteSurvey` | operator | — | — |
| `/api/v1/wifi/survey/floor` | `handleSurveyFloor` | operator | — | — |
| `/api/v1/wifi/survey/floor/floorplan` | `updateFloorFloorPlan` | operator | — | — |
| `/api/v1/wifi/survey/floor/sample` | `addFloorSample` | operator | — | — |
| `/api/v1/wifi/survey/floorplan` | `updateSurveyFloorPlan` | operator | — | — |
| `/api/v1/wifi/survey/floors` | `handleSurveyFloors` | operator | — | — |
| `/api/v1/wifi/survey/heatmap` | `getSurveyHeatmap` | — | — | yes |
| `/api/v1/wifi/survey/import/airmapper` | `importAirMapper` | operator | airmapper_baseline_diff | yes |
| `/api/v1/wifi/survey/imported-data` | `updateSurveyImportedData` | operator | — | — |
| `/api/v1/wifi/survey/list` | `listSurveys` | — | — | — |
| `/api/v1/wifi/survey/pause` | `pauseSurvey` | operator | — | — |
| `/api/v1/wifi/survey/report` | `generateSurveyReport` | — | — | yes |
| `/api/v1/wifi/survey/sample` | `addSurveySample` | operator | — | — |
| `/api/v1/wifi/survey/settings` | `updateSurveySettings` | operator | — | — |
| `/api/v1/wifi/survey/start` | `startSurvey` | operator | — | — |
| `/api/v1/wifi/wifi` | `handleWiFi` | — | — | — |
| `/api/v1/wifi/wifi/channel-graph` | `handleWiFiChannelGraph` | — | — | — |
| `/api/v1/wifi/wifi/connect` | `handleWiFiConnect` | operator | — | — |
| `/api/v1/wifi/wifi/disconnect` | `handleWiFiDisconnect` | operator | — | — |
| `/api/v1/wifi/wifi/forget` | `handleWiFiForgetNetwork` | operator | — | — |
| `/api/v1/wifi/wifi/saved` | `handleWiFiSavedNetworks` | — | — | — |
| `/api/v1/wifi/wifi/scan` | `handleWiFiScan` | — | — | — |
| `/api/v1/wifi/wifi/settings` | `handleWiFiSettings` | operator | — | — |
| `/api/v1/wifi/wifi/status` | `handleWiFiStatus` | — | — | — |
| `/api/v1/config/backup` | `handleConfigBackupCreate` | operator | — | — |
| `/api/v1/config/backup/delete` | `handleConfigBackupDelete` | operator | — | — |
| `/api/v1/config/backups` | `handleConfigBackups` | — | — | — |
| `/api/v1/config/restore` | `handleConfigRestore` | operator | — | — |
| `/api/v1/config/version` | `handleConfigVersion` | — | — | — |
| `/api/v1/discovery/engine` | `handleEngineDiscovery` | — | — | — |
| `/api/v1/discovery/engine/capabilities` | `handleEngineCapabilities` | — | — | — |
| `/api/v1/discovery/engine/device/` | `handleEngineDevice` | — | — | — |
| `/api/v1/discovery/engine/events` | `handleEngineEvents` | — | — | — |
| `/api/v1/discovery/engine/full` | `handleEngineFullScan` | — | — | yes |
| `/api/v1/discovery/engine/quick` | `handleEngineQuickScan` | — | — | yes |
| `/api/v1/discovery/engine/scan` | `handleEngineScan` | — | — | yes |
| `/api/v1/discovery/engine/stats` | `handleEngineStats` | — | — | — |
| `/api/v1/engines` | `handleEngines` | — | — | — |
| `/api/v1/events` | `handleSSE` | — | live_telemetry | — |
| `/api/v1/reporting/export` | `handleExport` | — | export_csv_json | — |
| `/api/v1/reporting/logs` | `handleLogs` | — | — | — |
| `/api/v1/reporting/logs/client` | `handleClientLogs` | — | — | — |
| `/api/v1/reporting/logs/query` | `handleLogsQuery` | — | — | — |
| `/api/v1/reporting/logs/recent` | `handleLogsRecent` | — | — | — |
| `/api/v1/reporting/logs/stats` | `handleLogsStats` | — | — | — |
| `/api/v1/health` | `handleHealth` | — | — | — |
| `/api/v1/interface` | `handleInterface` | — | — | — |
| `/api/v1/interfaces` | `handleInterfaces` | — | — | — |
| `/api/v1/license` | `handleLicenseStatus` | — | — | — |
| `/api/v1/network/mtu` | `handleSetMTU` | operator | — | — |
| `/api/v1/polling-targets` | `handlePollingTargets` | operator | — | — |
| `/api/v1/polling-targets/` | `handlePollingTargetByID` | operator | — | — |
| `/api/v1/profiles` | `handleProfiles` | operator | — | — |
| `/api/v1/profiles/` | `handleProfiles` | operator | — | — |
| `/api/v1/profiles/active` | `handleActiveProfile` | operator | — | — |
| `/api/v1/profiles/export` | `handleExportProfiles` | — | — | — |
| `/api/v1/profiles/import` | `handleImportProfiles` | operator | — | — |
| `/api/v1/recovery/complete` | `handleRecoveryComplete` | — | — | — |
| `/api/v1/recovery/instructions` | `handleRecoveryInstructions` | — | — | — |
| `/api/v1/recovery/status` | `handleRecoveryStatus` | — | — | — |
| `/api/v1/path/path` | `handlePath` | — | path_analysis | — |
| `/api/v1/path/traceroute` | `handleTraceroute` | — | path_analysis | yes |
| `/api/v1/telemetry/cable` | `handleCable` | — | — | — |
| `/api/v1/telemetry/dhcp/rogue` | `handleRogueDHCP` | — | — | — |
| `/api/v1/telemetry/dhcp/rogue/config` | `handleRogueDHCPConfig` | operator | — | — |
| `/api/v1/telemetry/dhcp/rogue/servers` | `handleRogueDHCPServers` | — | — | — |
| `/api/v1/telemetry/dns` | `handleDNS` | — | — | — |
| `/api/v1/telemetry/dns/security` | `handleDNSSecurity` | — | — | — |
| `/api/v1/telemetry/dns/security/settings` | `handleDNSSecuritySettings` | operator | — | — |
| `/api/v1/telemetry/gateway` | `handleGateway` | — | — | — |
| `/api/v1/telemetry/probes/anomalies` | `handleHealthCheckAnomalies` | — | anomaly_detection | — |
| `/api/v1/telemetry/probes/run` | `handleHealthChecks` | — | — | yes |
| `/api/v1/telemetry/probes/settings` | `handleHealthChecksSettings` | operator | — | — |
| `/api/v1/telemetry/ipconfig` | `handleIPConfig` | — | — | — |
| `/api/v1/telemetry/ipconfig/settings` | `handleIPSettings` | operator | — | — |
| `/api/v1/telemetry/iperf/client` | `handleIperfClient` | — | — | yes |
| `/api/v1/telemetry/iperf/client/status` | `handleIperfClientStatus` | — | — | — |
| `/api/v1/telemetry/iperf/info` | `handleIperfInfo` | — | — | — |
| `/api/v1/telemetry/iperf/server` | `handleIperfServer` | — | — | — |
| `/api/v1/telemetry/iperf/server/status` | `handleIperfServerStatus` | — | — | — |
| `/api/v1/telemetry/iperf/suggestions` | `handleIperfSuggestions` | — | — | — |
| `/api/v1/telemetry/link` | `handleLink` | — | — | — |
| `/api/v1/telemetry/publicip` | `handlePublicIP` | — | — | — |
| `/api/v1/telemetry/snmp/settings` | `handleSNMPSettings` | operator | — | — |
| `/api/v1/telemetry/speedtest` | `handleSpeedtest` | — | — | yes |
| `/api/v1/telemetry/speedtest/status` | `handleSpeedtestStatus` | — | — | — |
| `/api/v1/telemetry/system/health` | `handleSystemHealth` | — | — | — |
| `/api/v1/telemetry/vlan` | `handleVLAN` | — | — | — |
| `/api/v1/telemetry/vlan/interface` | `handleVLANInterface` | — | — | — |
| `/api/v1/telemetry/vlan/traffic` | `handleVLANTraffic` | — | — | — |
| `/api/v1/settings` | `handleSettings` | operator | — | — |
| `/api/v1/settings/cable` | `handleCableTestSettings` | operator | — | — |
| `/api/v1/settings/defaults` | `handleSettingsDefaults` | operator | — | — |
| `/api/v1/settings/link` | `handleLinkSettings` | operator | — | — |
| `/api/v1/setup/complete` | `handleSetupComplete` | — | — | — |
| `/api/v1/setup/status` | `handleSetupStatus` | — | — | — |
| `/api/v1/security/bluetooth/devices` | `handleBluetoothDevices` | — | — | — |
| `/api/v1/security/bluetooth/scan` | `handleBluetoothScan` | — | — | — |
| `/api/v1/security/bluetooth/stats` | `handleBluetoothStats` | — | — | — |
| `/api/v1/security/bluetooth/status` | `handleBluetoothStatus` | — | — | — |
| `/api/v1/security/devices` | `handleDevices` | — | — | — |
| `/api/v1/security/devices/scan` | `handleDevicesScan` | — | — | yes |
| `/api/v1/security/devices/settings` | `handleDevicesSettings` | operator | — | — |
| `/api/v1/security/devices/status` | `handleDevicesStatus` | — | — | — |
| `/api/v1/security/devices/subnets` | `handleDevicesSubnets` | — | — | — |
| `/api/v1/security/discovery` | `handleDiscovery` | — | — | — |
| `/api/v1/security/discovery/fingerprint` | `handleAdvancedFingerprint` | — | — | — |
| `/api/v1/security/discovery/options` | `handleDiscoveryOptions` | — | — | — |
| `/api/v1/security/discovery/portscan` | `handlePortScan` | — | — | — |
| `/api/v1/security/discovery/probe` | `handleTCPProbe` | — | — | — |
| `/api/v1/security/discovery/service/status` | `handleDiscoveryServiceStatus` | — | — | — |
| `/api/v1/security/guest-audit/run` | `handleGuestAuditRun` | — | compliance_advanced | yes |
| `/api/v1/security/guest-audit/settings` | `handleGuestAuditSettings` | operator | — | — |
| `/api/v1/security/pipeline/cancel` | `handlePipelineCancel` | — | — | — |
| `/api/v1/security/pipeline/config` | `handlePipelineConfigRoute` | operator | — | — |
| `/api/v1/security/pipeline/port-intensity` | `handlePipelinePortIntensityInfo` | — | — | — |
| `/api/v1/security/pipeline/start` | `handlePipelineStart` | — | — | — |
| `/api/v1/security/pipeline/status` | `handlePipelineStatus` | — | — | — |
| `/api/v1/security/pipeline/timing-profiles` | `handlePipelineTimingProfiles` | — | — | — |
| `/api/v1/security/problems` | `handleNetworkProblems` | — | — | — |
| `/api/v1/security/problems/scan` | `handleProblemScan` | — | — | — |
| `/api/v1/security/problems/thresholds` | `handleProblemThresholds` | operator | — | — |
| `/api/v1/security/vulnerabilities/device` | `handleDeviceVulnerabilities` | — | — | — |
| `/api/v1/security/vulnerabilities/results` | `handleVulnerabilityResults` | — | — | — |
| `/api/v1/security/vulnerabilities/scan` | `handleVulnerabilityScan` | — | compliance_advanced | yes |
| `/api/v1/security/vulnerabilities/settings` | `handleVulnerabilitySettings` | operator | — | — |
| `/api/v1/security/vulnerabilities/status` | `handleVulnerabilityStatus` | — | — | — |
| `/api/v1/security/vulnerabilities/validate-api-key` | `handleNVDAPIKeyValidate` | — | — | — |
| `/api/v1/security/wifi/discovery/aps` | `handleWiFiDiscoveryAPs` | — | — | — |
| `/api/v1/security/wifi/discovery/networks` | `handleWiFiDiscoveryNetworks` | — | — | — |
| `/api/v1/security/wifi/discovery/scan` | `handleWiFiDiscoveryScan` | — | — | — |
| `/api/v1/security/wifi/discovery/stats` | `handleWiFiDiscoveryStats` | — | — | — |
| `/api/v1/sso/callback` | `handleSSOCallback` | — | — | — |
| `/api/v1/sso/login` | `handleSSOLogin` | — | — | — |
| `/api/v1/sso/providers` | `handleSSOProviders` | — | — | — |
| `/api/v1/sso/settings` | `handleSSOSettings` | operator | — | — |
| `/api/v1/sso/update` | `handleSSOUpdate` | operator | sso | — |
| `/api/v1/status` | `handleStatus` | — | — | — |
| `/api/v1/tokens` | `handleAPITokens` | operator | — | — |
| `/api/v1/tokens/` | `handleAPITokenByID` | operator | — | — |
| `/api/v1/topology/arp` | `handleTopologyARP` | — | — | — |
| `/api/v1/topology/links` | `handleTopologyLinks` | — | — | — |
| `/api/v1/topology/nodes` | `handleTopologyNodes` | — | — | — |
| `/api/v1/topology/nodes/` | `handleTopologyNodeByID` | — | — | — |
| `/api/v1/users` | `handleUsers` | — | — | — |
| `/api/v1/users/` | `handleUserByName` | — | — | — |
| `/api/v1/users/me` | `handleCurrentUser` | — | — | — |
