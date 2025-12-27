/**
 * Profile Type Definitions
 *
 * Types for the MSP Profile system (#754) that enables client-specific configurations.
 */

/**
 * Profile represents a client-specific configuration profile.
 */
export interface Profile {
  id: string;
  name: string;
  description: string;
  config: ProfileConfig;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * ProfileSettings contains all user-configurable settings that are stored per-profile.
 * Profiles are the single source of truth for all settings.
 *
 * ORDER: Settings are ordered to match the Settings Drawer UI for consistency:
 * 1. Link Settings - Ethernet link configuration
 * 2. Cable Test Settings - Cable diagnostics
 * 3. Network/Display Options - IP/DHCP and display preferences
 * 4. WiFi Settings - WiFi interface configuration
 * 5. DNS Settings - DNS test configuration
 * 6. Health Checks/Tests - Ping, TCP, HTTP tests
 * 7. Performance - Speedtest and iPerf
 * 8. Discovery - Network discovery and SNMP
 * 9. Vulnerability - CVE scanning
 * 10. Thresholds - All threshold values
 * 11. Card Settings - Card visibility/auto-run (last as it's meta-configuration)
 */
export interface ProfileSettings {
  // ============================================================================
  // 1. Link Settings (matches LinkSettings section in SettingsDrawer)
  // ============================================================================
  /** Link settings (speed, duplex, auto-neg) */
  link?: LinkConfig;

  // ============================================================================
  // 2. Cable Test Settings (matches CableTestSettings section)
  // ============================================================================
  /** Cable test settings */
  cableTest?: CableTestConfig;

  // ============================================================================
  // 3. Network/Display Options (matches Network CollapsibleSection)
  // ============================================================================
  /** Display options (unit system, public IP visibility) */
  displayOptions?: DisplayOptionsConfig;

  // ============================================================================
  // 4. WiFi Settings (matches WiFiSettings section - WiFi mode only)
  // ============================================================================
  /** WiFi settings (interface selection, survey options) */
  wifi?: WiFiSettingsConfig;

  // ============================================================================
  // 5. DNS Settings (matches DNSSettings section)
  // ============================================================================
  /** DNS settings (test hostname, custom servers) */
  dns?: DNSSettingsConfig;

  // ============================================================================
  // 6. Health Checks/Tests (matches HealthChecksSettings section)
  // ============================================================================
  /** Health check test configurations (ping targets, HTTP endpoints, etc.) */
  tests?: TestsConfig;

  // ============================================================================
  // 7. Performance Settings (matches PerformanceSettings section)
  // ============================================================================
  /** Speedtest settings */
  speedtest?: SpeedtestConfig;
  /** iPerf client/server settings */
  iperf?: IperfConfig;

  // ============================================================================
  // 8. Discovery Settings (matches DiscoverySettings section - includes SNMP)
  // ============================================================================
  /** Network discovery configuration */
  networkDiscovery?: NetworkDiscoveryConfig;
  /** SNMP configuration */
  snmp?: SNMPConfig;

  // ============================================================================
  // 9. Vulnerability Settings (matches VulnerabilitySettings section)
  // ============================================================================
  /** Vulnerability scanning settings */
  vulnerability?: VulnerabilityConfig;

  // ============================================================================
  // 10. Thresholds (matches ThresholdsSettings section)
  // ============================================================================
  /** Threshold values for various tests */
  thresholds?: ProfileThresholdsConfig;

  // ============================================================================
  // 11. Appearance Settings (matches AppearanceSettings section)
  // ============================================================================
  /** Appearance settings (theme, language) */
  appearance?: AppearanceConfig;

  // ============================================================================
  // 12. Card Settings (meta-configuration for card visibility/behavior)
  // ============================================================================
  /** Card visibility and auto-run settings */
  cardSettings?: CardSettingsConfig;
}

/** Card option configuration for visibility and auto-run behavior. */
export interface CardOptionConfig {
  enabled: boolean;
  autoRunOnLink: boolean;
}

/** Performance card has nested options for speedtest and iperf. */
export interface PerformanceCardConfig extends CardOptionConfig {
  speedtest: CardOptionConfig;
  iperf: CardOptionConfig;
}

/** Card settings - visibility and auto-run configuration for each card. */
export interface CardSettingsConfig {
  // Core network cards
  link?: CardOptionConfig;
  cable?: CardOptionConfig;
  switch?: CardOptionConfig;
  vlan?: CardOptionConfig;
  network?: CardOptionConfig;
  gateway?: CardOptionConfig;
  dns?: CardOptionConfig;
  publicIP?: CardOptionConfig;

  // WiFi cards
  wifi?: CardOptionConfig;
  wifiSurvey?: CardOptionConfig;

  // Diagnostic/analysis cards
  healthChecks?: CardOptionConfig;
  networkDiscovery?: CardOptionConfig;
  pathDiscovery?: CardOptionConfig;
  systemHealth?: CardOptionConfig;

  // Performance testing
  performance?: PerformanceCardConfig;
}

/** Display options configuration. */
export interface DisplayOptionsConfig {
  showPublicIP?: boolean;
  unitSystem?: "sae" | "metric";
}

/** Appearance configuration (theme and language). */
export interface AppearanceConfig {
  theme?: "light" | "dark" | "system";
  language?: string;
}

/**
 * iPerf configuration.
 * Note: iPerf is enabled by default but auto-run only triggers if server is
 * configured with a valid IP/hostname that responds to connection attempts.
 */
export interface IperfConfig {
  /** iPerf server address (IP or hostname) */
  server?: string;
  /** iPerf server port (default: 5201) */
  port?: number;
  /** Protocol for testing */
  protocol?: "tcp" | "udp";
  /** Test direction */
  direction?: "upload" | "download" | "bidirectional";
  /** Test duration in seconds */
  duration?: number;
  /** Local server port when running as server */
  serverPort?: number;
  /** Whether to run local iPerf server */
  enableServer?: boolean;
}

/** Threshold pair for good/warning values. */
export interface ThresholdPairConfig {
  good: number;
  warning: number;
}

/** HTTP timing thresholds. */
export interface HTTPTimingThresholdsConfig {
  dns?: ThresholdPairConfig;
  tcp?: ThresholdPairConfig;
  tls?: ThresholdPairConfig;
  ttfb?: ThresholdPairConfig;
}

/** Complete thresholds configuration stored in profile. */
export interface ProfileThresholdsConfig {
  dns?: ThresholdPairConfig;
  gateway?: ThresholdPairConfig;
  wifi?: ThresholdPairConfig;
  customPing?: ThresholdPairConfig;
  customTcp?: ThresholdPairConfig;
  customHttp?: ThresholdPairConfig;
  httpTimings?: HTTPTimingThresholdsConfig;
}

/** Passive protocol configuration. */
export interface PassiveProtocolsConfig {
  lldp?: boolean;
  cdp?: boolean;
  edp?: boolean;
  ndp?: boolean;
}

/** Port scan configuration. */
export interface PortScanSettingsConfig {
  enabled?: boolean;
  preset?: string;
  tcpPorts?: string;
  udpPorts?: string;
  bannerTimeoutMs?: number;
}

/** Network discovery options. */
export interface DiscoveryOptionsConfig {
  passiveProtocols?: PassiveProtocolsConfig;
  arpScan?: boolean;
  icmpScan?: boolean;
  portScan?: PortScanSettingsConfig;
  tcpProbe?: { timeoutMs?: number; workers?: number };
  traceroute?: boolean;
  snmpQuery?: boolean;
}

/**
 * Network discovery configuration.
 * Note: OUI database is baked into binary at build time - no runtime path needed.
 */
export interface NetworkDiscoveryConfig {
  enabled?: boolean;
  arpScanWorkers?: number;
  pingTimeoutMs?: number;
  scanTimeoutMs?: number;
  autoScan?: boolean;
  scanIntervalMs?: number;
  ipv6Enabled?: boolean;
  options?: DiscoveryOptionsConfig;
  timing?: { probeIntervalMs?: number; rescanIntervalMs?: number; workers?: number };
  profiler?: { enabled?: boolean; timeoutMs?: number; maxConcurrent?: number; quickPorts?: number[] };
  fingerprinting?: { enabled?: boolean; osDetection?: boolean; serviceProbes?: boolean };
}

/** SNMP v3 credential configuration. */
export interface SNMPv3CredentialConfig {
  id?: string;
  name: string;
  username: string;
  authProtocol?: string; // "MD5", "SHA", "SHA256", "SHA512", or "" for noAuth
  authPassword?: string;
  privProtocol?: string; // "DES", "AES", "AES192", "AES256", or "" for noPriv
  privPassword?: string;
  contextName?: string;
  securityLevel?: string; // "noAuthNoPriv", "authNoPriv", "authPriv"
}

/** SNMP configuration. */
export interface SNMPConfig {
  communities?: string[];
  v3Credentials?: SNMPv3CredentialConfig[];
  timeoutMs?: number;
  retries?: number;
  port?: number;
}

/**
 * Link settings configuration.
 * Uses combined mode format (e.g., "10/half", "100/full", "1000/full")
 * matching ethtool output for clarity.
 */
export interface LinkConfig {
  /** Combined speed/duplex mode (e.g., "100/full", "1000/full") or "auto" for auto-negotiation */
  mode?: string;
  /** Available modes from the interface (e.g., ["10/half", "10/full", "100/half", "100/full", "1000/full"]) */
  availableModes?: string[];
}

/**
 * Cable test configuration.
 * Note: Cable test auto-runs automatically when link is down AND PHY supports TDR.
 * No user toggle needed - it's either possible or not based on hardware capability.
 */
export interface CableTestConfig {
  /** Whether cable testing is enabled (requires PHY TDR support) */
  enabled?: boolean;
}

/** Vulnerability scanning configuration. */
export interface VulnerabilityConfig {
  enabled?: boolean;
  cveDatabase?: string;
  nvdApiKey?: string;
  updateInterval?: number;
  severityThreshold?: string;
  maxConcurrent?: number;
  autoScan?: boolean;
}

/** Speedtest configuration. */
export interface SpeedtestConfig {
  serverId?: string;
  autoRunOnLink?: boolean;
}

/** WiFi settings configuration. */
export interface WiFiSettingsConfig {
  interface?: string;
  surveyEnabled?: boolean;
  surveyIntervalMs?: number;
  signalThreshold?: number;
}

/** DNS settings configuration. */
export interface DNSSettingsConfig {
  testHostname?: string;
  servers?: DNSServerConfig[];
}

/** DNS server configuration. */
export interface DNSServerConfig {
  id?: string;
  address: string;
  enabled: boolean;
}

/** Health check tests configuration. */
export interface TestsConfig {
  dnsHostname?: string;
  pingTargets?: PingTargetConfig[];
  tcpPorts?: TCPPortConfig[];
  udpPorts?: UDPPortConfig[];
  httpEndpoints?: HTTPEndpointConfig[];
  runPerformance?: boolean;
  runSpeedtest?: boolean;
  runIperf?: boolean;
  runDiscovery?: boolean;
}

/** Ping target configuration. */
export interface PingTargetConfig {
  id?: string;
  name: string;
  host: string;
  enabled: boolean;
  count?: number;
}

/** TCP port test configuration. */
export interface TCPPortConfig {
  id?: string;
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

/** UDP port test configuration. */
export interface UDPPortConfig {
  id?: string;
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

/** HTTP endpoint test configuration. */
export interface HTTPEndpointConfig {
  id?: string;
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
}

/**
 * ProfileConfig contains the client-specific settings stored in a profile.
 * All settings are now stored within the profile - profiles own everything.
 */
export interface ProfileConfig {
  health_checks?: HealthChecksConfig;
  thresholds?: ThresholdsConfig;
  discovery?: DiscoveryConfig;
  notes?: string;
  /** Interface configurations for multi-interface support */
  interfaces?: ProfileInterfaceConfigs;
  /** All user settings - profiles are the single source of truth */
  settings?: ProfileSettings;
}

// ============================================================================
// Multi-Interface Support Types
// ============================================================================

/**
 * ProfileInterfaceConfigs stores the selected interfaces for a profile.
 * Each profile can have multiple ethernet and WiFi interfaces, each with independent settings.
 */
export interface ProfileInterfaceConfigs {
  /** All configured ethernet interfaces for this profile */
  ethernet?: ProfileInterfaceSelection[];
  /** All configured WiFi interfaces for this profile */
  wifi?: ProfileInterfaceSelection[];
  /** Name of the currently active ethernet interface */
  active_ethernet?: string;
  /** Name of the currently active WiFi interface */
  active_wifi?: string;
}

/**
 * ProfileInterfaceSelection stores configuration for a selected interface.
 * Each interface can have its own thresholds and health check configuration.
 */
export interface ProfileInterfaceSelection {
  /** Interface name (e.g., "eth0", "wlan0") */
  name: string;
  /** Whether this interface is enabled for testing */
  enabled: boolean;
  /** Optional per-interface thresholds */
  thresholds?: ProfileInterfaceThresholds;
  /** Optional per-interface health check overrides */
  health_checks?: ProfileInterfaceHealthChecks;
}

/**
 * Helper to get the active ethernet interface from a ProfileInterfaceConfigs.
 */
export function getActiveEthernetInterface(
  configs?: ProfileInterfaceConfigs
): ProfileInterfaceSelection | undefined {
  if (!configs?.active_ethernet || !configs.ethernet) return undefined;
  return configs.ethernet.find((i) => i.name === configs.active_ethernet);
}

/**
 * Helper to get the active WiFi interface from a ProfileInterfaceConfigs.
 */
export function getActiveWiFiInterface(
  configs?: ProfileInterfaceConfigs
): ProfileInterfaceSelection | undefined {
  if (!configs?.active_wifi || !configs.wifi) return undefined;
  return configs.wifi.find((i) => i.name === configs.active_wifi);
}

/**
 * ProfileInterfaceThresholds for per-interface latency/signal thresholds.
 */
export interface ProfileInterfaceThresholds {
  latency_warning_ms?: number;
  latency_critical_ms?: number;
  signal_warning_dbm?: number;
  signal_critical_dbm?: number;
}

/**
 * ProfileInterfaceHealthChecks for per-interface health check overrides.
 */
export interface ProfileInterfaceHealthChecks {
  ping_targets?: string[];
  tcp_checks?: TcpCheck[];
  http_checks?: HttpCheck[];
}

/**
 * HealthChecksConfig defines custom health check targets for a client.
 */
export interface HealthChecksConfig {
  ping_targets?: string[];
  tcp_checks?: TcpCheck[];
  http_checks?: HttpCheck[];
}

/**
 * TcpCheck defines a TCP port check configuration.
 */
export interface TcpCheck {
  host: string;
  port: number;
  name?: string;
}

/**
 * HttpCheck defines an HTTP health check configuration.
 */
export interface HttpCheck {
  url: string;
  name?: string;
  expected_status?: number;
}

/**
 * ThresholdsConfig defines custom thresholds for alerting.
 */
export interface ThresholdsConfig {
  latency_warning_ms?: number;
  latency_critical_ms?: number;
  packet_loss_warning_pct?: number;
  packet_loss_critical_pct?: number;
}

/**
 * DiscoveryConfig defines custom discovery settings.
 */
export interface DiscoveryConfig {
  additional_subnets?: string[];
  scan_interval_seconds?: number;
}

/**
 * ProfileListResponse is returned by GET /api/profiles.
 */
export interface ProfileListResponse {
  profiles: Profile[];
  total: number;
}

/**
 * ProfileRequest is used for creating and updating profiles.
 */
export interface ProfileRequest {
  name: string;
  description?: string;
  config?: ProfileConfig;
  is_default?: boolean;
}

/**
 * ProfileImportRequest is used for importing profiles from JSON.
 */
export interface ProfileImportRequest {
  version: string;
  profiles: ProfileRequest[];
  overwrite?: boolean;
}

/**
 * ProfileImportResponse is returned after importing profiles.
 */
export interface ProfileImportResponse {
  created: number;
  updated: number;
  skipped: number;
  errors?: string[];
}

/**
 * ProfileExportResponse is returned when exporting profiles.
 */
export interface ProfileExportResponse {
  version: string;
  exported_at: string;
  profiles: Profile[];
}

/**
 * ActiveProfileRequest is used to set the active profile.
 */
export interface ActiveProfileRequest {
  profile_id: string;
}

/**
 * ActiveProfileResponse is returned after setting the active profile.
 */
export interface ActiveProfileResponse {
  message: string;
  profile: Profile;
}

/**
 * ProfileChangedEvent is broadcast via WebSocket when the active profile changes.
 */
export interface ProfileChangedEvent {
  type: "profileChanged";
  payload: {
    profile_id: string;
    profile_name: string;
  };
}
