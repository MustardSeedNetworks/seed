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
 * ProfileConfig contains the client-specific settings stored in a profile.
 */
export interface ProfileConfig {
  health_checks?: HealthChecksConfig;
  thresholds?: ThresholdsConfig;
  discovery?: DiscoveryConfig;
  notes?: string;
  /** Interface configurations for multi-interface support */
  interfaces?: ProfileInterfaceConfigs;
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
