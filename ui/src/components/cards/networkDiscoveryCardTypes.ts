/**
 * Shared types for NetworkDiscoveryCard and its DiscoveryModal companion.
 *
 * Mirrors what the backend ARP/LLDP/CDP/SNMP discovery pipeline returns,
 * plus the local-only port-scan and deep-scan view models the UI keeps.
 * Kept in their own module so the card body, the modal, and the
 * helper subviews can share without coupling to the card file itself.
 */

export interface LldpInfo {
  chassisId: string;
  portId: string;
  portDescription?: string;
  systemName?: string;
  systemDescription?: string;
  capabilities?: string[];
  managementAddress?: string;
}

export interface CdpInfo {
  deviceId: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  capabilities?: string[];
  managementAddress?: string;
  nativeVlan?: number;
  voiceVlan?: number;
}

export interface EdpInfo {
  deviceId: string;
  displayName?: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  vlan?: number;
}

export interface NdpInfo {
  linkLayerAddress: string;
  isRouter: boolean;
  reachableTime?: number;
  retransTimer?: number;
  flags?: number;
  lastAdvertisement?: string;
}

export type DiscoveryMethod = 'arp' | 'ndp' | 'lldp' | 'cdp' | 'edp' | 'mdns' | 'ping' | 'snmp';

// Auto-profiling types from backend
export interface OpenPort {
  port: number;
  protocol: string;
  service?: string;
  banner?: string;
  isOpen: boolean;
}

export interface HttpInfo {
  port: number;
  statusCode: number;
  title?: string;
  server?: string;
  isHttps: boolean;
}

export interface DeviceProfile {
  profiledAt: string;
  openPorts?: OpenPort[];
  httpInfo?: HttpInfo;
  deviceType?: string;
  deviceIcons?: string[];
}

// SNMP-related interfaces for extended device data
export interface SnmpSystemInfo {
  sysDescr?: string;
  sysObjectId?: string;
  sysName?: string;
  sysContact?: string;
  sysLocation?: string;
  sysUpTime?: number;
}

export interface SnmpInterface {
  index: number;
  name?: string;
  description?: string;
  alias?: string;
  type?: number;
  mtu?: number;
  speedMbps?: number;
  mac?: string;
  adminStatus?: string;
  operStatus?: string;
}

export interface SnmpIpAddress {
  address: string;
  prefix?: number;
  ifIndex: number;
  type?: string;
}

export interface SnmpVlan {
  id: number;
  name?: string;
  status?: string;
  egressPorts?: number[];
}

export interface SnmpEntity {
  index: number;
  description?: string;
  className?: string;
  name?: string;
  hardwareRev?: string;
  firmwareRev?: string;
  softwareRev?: string;
  serialNum?: string;
  modelName?: string;
}

export interface SnmpFullData {
  collectedAt?: string;
  system?: SnmpSystemInfo;
  interfaces?: SnmpInterface[];
  ipAddresses?: SnmpIpAddress[];
  vlans?: SnmpVlan[];
  inventory?: SnmpEntity[];
  errors?: string[];
}

export interface DiscoveredDevice {
  ip: string;
  ipv6?: string;
  ipv6Addresses?: string[];
  mac: string;
  hostname?: string; // DNS PTR resolved name
  netbiosName?: string; // Windows NetBIOS name (UDP 137)
  mdnsName?: string; // mDNS/Bonjour .local name
  displayName?: string; // Best available name for UI display
  vendor?: string;
  osGuess?: string;
  ttl?: number;
  discoveryMethod: DiscoveryMethod[];
  lastSeen: string;
  isLocal: boolean; // true if on local subnet, false for extended networks
  isRouter?: boolean;
  lldpInfo?: LldpInfo;
  cdpInfo?: CdpInfo;
  edpInfo?: EdpInfo;
  ndpInfo?: NdpInfo;
  profile?: DeviceProfile;
  snmpData?: SnmpFullData; // Extended SNMP data from Phase 3 scanning
  vulnerabilities?: {
    count: number;
    highestSeverity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  };
}

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
