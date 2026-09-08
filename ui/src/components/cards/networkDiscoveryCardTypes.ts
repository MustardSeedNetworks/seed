/**
 * Shared types for NetworkDiscoveryCard and its DiscoveryModal companion.
 *
 * Mirrors what the backend ARP/LLDP/CDP/SNMP discovery pipeline returns,
 * plus the local-only port-scan and deep-scan view models the UI keeps.
 * Kept in their own module so the card body, the modal, and the
 * helper subviews can share without coupling to the card file itself.
 */

import type {
  CDPDeviceInfo,
  DeviceProfile,
  DiscoveredDevice,
  EDPDeviceInfo,
  HTTPInfo,
  LLDPDeviceInfo,
  NDPDeviceInfo,
  OpenPort,
  SNMPEntity,
  SNMPFullData,
  SNMPInterface,
  SNMPIPAddress,
  SNMPVLAN,
  SystemInfo,
} from '../../types/generated/engine-discovery-response';

// The discovery pipeline's own shapes come from the generated wire types. The
// hand-typed mirrors that used to live here were behind the wire by a whole
// SNMP collection pass — no MAC table, no LLDP neighbours, no interface
// counters — and the card could not name what it had not declared (seed#2393).
export type {
  CDPDeviceInfo,
  DeviceProfile,
  DiscoveredDevice,
  EDPDeviceInfo,
  HTTPInfo,
  LLDPDeviceInfo,
  NDPDeviceInfo,
  OpenPort,
  SNMPEntity,
  SNMPFullData,
  SNMPInterface,
  SNMPIPAddress,
  SNMPVLAN,
  SystemInfo,
};

/** Discovery methods the backend reports in `discoveryMethod`. */
export type DiscoveryMethod = 'arp' | 'ndp' | 'lldp' | 'cdp' | 'edp' | 'mdns' | 'ping' | 'snmp';

export interface DiscoveryStatus {
  scanning: boolean;
  deviceCount: number;
  lastScan: string;
  subnet: string;
  subnets?: string[]; // All subnets being scanned (I3)
  localIP: string;
  interface: string;
}

export interface NetworkDiscoveryData {
  devices: DiscoveredDevice[];
  status: DiscoveryStatus;
}

// Deep Scan (Port Scan) Types - matches backend discovery.ServiceInfo
export interface ServiceInfo {
  port: number;
  state: 'open' | 'closed' | 'filtered';
  service: string;
  banner?: string;
  version?: string;
  protocol?: string;
}

// PortScanResult for display - normalized from backend response
export interface PortScanResult {
  port: number;
  state: 'open' | 'closed' | 'filtered';
  service: string;
  banner?: string;
  version?: string;
  rtt: number; // nanoseconds (0 if not available from backend)
}

// Backend API response structure (internal to discovery pipeline)
export interface PortScanApiResponse {
  ip: string;
  hostname?: string;
  services: ServiceInfo[];
  scanTime: number;
  error?: string;
}

export interface DeepScanResult {
  target: string;
  results: PortScanResult[];
  osGuess?: string;
  scannedAt: Date;
}

export interface DiscoverySettingsForAutoScan {
  portScanEnabled?: boolean;
  vulnScanEnabled?: boolean;
  vulnAutoScan?: boolean;
}
