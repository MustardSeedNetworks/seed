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
 * - PublicIpCard: Public IPv4/IPv6 addresses
 * - PerformanceCard: Speedtest and iperf3 results
 * - HealthCheckCard: Health check monitoring
 * - SystemHealthCard: System resource monitoring
 * - WiFiSurveyCard: WiFi site survey management
 *
 * Exported Types:
 * - LinkData, SwitchData, VlanData, DHCPData, DNSData, GatewayData, WiFiData, etc.
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

export { CableCard, type CableData } from "./CableCard";
export { DnsCard, type DnsData } from "./DNSCard";
export { GatewayCard, type GatewayData } from "./GatewayCard";
export { LinkCard, type LinkData } from "./LinkCard";
export { LogViewerCard, type LogViewerCardProps } from "./LogViewerCard";
export {
  type DHCPData,
  type DHCPTiming,
  NetworkCard,
  type PublicIPInfo,
} from "./NetworkCard";
export {
  type DiscoveredDevice,
  type DiscoveryStatus,
  NetworkDiscoveryCard,
  type NetworkDiscoveryData,
} from "./NetworkDiscoveryCard";
export { PathDiscoveryCard } from "./PathDiscoveryCard";
export { PublicIpCard, type PublicIpData } from "./PublicIPCard";
export { SwitchCard, type SwitchData, type VlanData } from "./SwitchCard";
export { WiFiCard, type WiFiData } from "./WiFiCard";
