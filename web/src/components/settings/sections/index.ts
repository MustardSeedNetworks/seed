/**
 * Settings Sections Index
 *
 * Purpose: Central export file for all settings section components.
 * Provides convenient re-exports for cleaner imports in SettingsDrawer.
 *
 * Exported Components:
 * - AutoSaveIndicator: Status indicator for unsaved changes
 * - AppearanceSettings: Theme selection (light/dark/system)
 * - DiscoverySettings: Network discovery configuration
 * - DNSSettings: DNS server and test configuration
 * - HealthChecksSettings: Ping/TCP/UDP/HTTP health check configuration
 * - PerformanceSettings: Speedtest and iperf3 configuration
 * - SNMPSettings: SNMP v2c and v3 credentials
 * - ThresholdsSettings: Performance threshold configuration
 * - WiFiSettings: WiFi interface and scan configuration
 *
 * Usage:
 * ```typescript
 * import {
 *   AppearanceSettings,
 *   DiscoverySettings,
 *   DNSSettings
 * } from './sections';
 * ```
 *
 * Dependencies: Individual component files in this directory
 */

export { AutoSaveIndicator } from "./AutoSaveIndicator";
export { AppearanceSettings } from "./AppearanceSettings";
export { CableTestSettings } from "./CableTestSettings";
export { ConfigBackupsSection } from "./ConfigBackupsSection";
export { DiscoverySettings } from "./DiscoverySettings";
export { DNSSettings } from "./DNSSettings";
export { HealthChecksSettings } from "./HealthChecksSettings";
export { LinkSettings } from "./LinkSettings";
export { PerformanceSettings } from "./PerformanceSettings";
export { SNMPSettings } from "./SNMPSettings";
export { ThresholdsSettings } from "./ThresholdsSettings";
export { WiFiSettings } from "./WiFiSettings";
