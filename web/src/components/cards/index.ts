/**
 * Card Components Index
 *
 * Purpose: Central export file for all diagnostic card components.
 * Provides convenient re-exports and type exports for cleaner imports in App.
 *
 * Exported Components:
 * - LinkCard: Network link status (speed, duplex, carrier)
 * - SwitchCard: Switch/VLAN information (LLDP/CDP/EDP)
 * - NetworkCard: IPv4/IPv6 DHCP configuration
 * - DNSCard: DNS resolution testing
 * - GatewayCard: Gateway/router reachability
 * - WiFiCard: WiFi connection status
 * - CableCard: Ethernet cable test results
 * - NetworkDiscoveryCard: Device inventory (1300+ lines)
 * - PublicIPCard: Public IPv4/IPv6 addresses
 * - PerformanceCard: Speedtest and iperf3 results
 * - HealthCheckCard: Health check monitoring
 * - SystemHealthCard: System resource monitoring
 * - WiFiSurveyCard: WiFi site survey management
 *
 * Exported Types:
 * - LinkData, SwitchData, VLANData, DHCPData, DNSData, GatewayData, WiFiData, etc.
 *
 * Usage:
 * ```typescript
 * import {
 *   LinkCard,
 *   NetworkCard,
 *   type LinkData,
 *   type DHCPData
 * } from './cards';
 * ```
 *
 * Dependencies: Individual card component files
 */

export { LinkCard, type LinkData } from "./LinkCard";
export { SwitchCard, type SwitchData, type VLANData } from "./SwitchCard";
export {
  NetworkCard,
  type DHCPData,
  type DHCPTiming,
  type PublicIPInfo,
} from "./NetworkCard";
export { DNSCard, type DNSData } from "./DNSCard";
export { GatewayCard, type GatewayData } from "./GatewayCard";
export { WiFiCard, type WiFiData } from "./WiFiCard";
export { CableCard, type CableData } from "./CableCard";
export {
  NetworkDiscoveryCard,
  type NetworkDiscoveryData,
  type DiscoveredDevice,
  type DiscoveryStatus,
} from "./NetworkDiscoveryCard";
export { PublicIPCard, type PublicIPData } from "./PublicIPCard";
export { LogViewerCard, type LogViewerCardProps } from "./LogViewerCard";
export { PathDiscoveryCard } from "./PathDiscoveryCard";
