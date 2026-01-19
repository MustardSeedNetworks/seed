/**
 * The Seed Type Definitions
 */

export type Status = 'success' | 'warning' | 'error' | 'unknown';

export interface CardData {
  id: string;
  title: string;
  status: Status;
  lastUpdated: string;
}

export interface LinkCard extends CardData {
  type: 'link';
  speed: string;
  duplex: 'full' | 'half';
  advertisedSpeeds: string[];
  linkUp: boolean;
}

export interface CableCard extends CardData {
  type: 'cable';
  supported: boolean;
  length?: number;
  status: Status;
  faults?: string[];
}

export interface VlanCard extends CardData {
  type: 'vlan';
  nativeVlan?: number;
  taggedVlans: number[];
  voiceVlan?: number;
}

export interface SwitchCard extends CardData {
  type: 'switch';
  protocol: 'lldp' | 'cdp' | 'edp' | 'fdp' | 'unknown';
  switchName?: string;
  portId?: string;
  portDescription?: string;
  managementIp?: string;
  systemDescription?: string;
}

export interface WifiCard extends CardData {
  type: 'wifi';
  ssid?: string;
  bssid?: string;
  signal: number; // dBm
  channel: number;
  frequency: number; // MHz
  security: string;
}

export interface DhcpCard extends CardData {
  type: 'dhcp';
  mode: 'dhcp' | 'static';
  ip?: string;
  subnet?: string;
  gateway?: string;
  dns: string[];
  server?: string;
  leaseTime?: number;
  timing?: {
    discover: number;
    offer: number;
    request: number;
    ack: number;
    total: number;
  };
}

export interface DnsCard extends CardData {
  type: 'dns';
  server: string;
  testHostname: string;
  forward?: {
    result: string;
    time: number;
    status: Status;
  };
  reverse?: {
    result: string;
    time: number;
    status: Status;
  };
}

export interface GatewayCard extends CardData {
  type: 'gateway';
  ip: string;
  pings: {
    time: number;
    success: boolean;
  }[];
  averageLatency: number;
  packetLoss: number;
}

export type DiagnosticCard =
  | LinkCard
  | CableCard
  | VlanCard
  | SwitchCard
  | WifiCard
  | DhcpCard
  | DnsCard
  | GatewayCard;

export interface Thresholds {
  dhcp: {
    total: { warning: number; critical: number };
    perPhase: { warning: number; critical: number };
  };
  dns: { warning: number; critical: number };
  ping: { warning: number; critical: number };
  wifi: { warning: number; critical: number };
}

export interface Settings {
  interface: string;
  availableInterfaces: string[];
  vlan: {
    enabled: boolean;
    id: number;
  };
  ip: {
    mode: 'dhcp' | 'static';
    static?: {
      address: string;
      netmask: string;
      gateway: string;
      dns: string[];
    };
  };
  thresholds: Thresholds;
  darkMode: boolean;
}

// ============================================================================
// Traceroute Types
// ============================================================================

export interface TracerouteRequest {
  target: string;
  protocol: 'icmp' | 'udp' | 'tcp';
  port?: number;
  maxHops?: number;
  timeout?: number;
}

export interface TracerouteHop {
  ttl: number;
  ip?: string;
  hostname?: string;
  rtt: number; // nanoseconds
  state: 'reply' | 'timeout' | 'error';
}

export interface TracerouteResult {
  target: string;
  targetIp: string;
  protocol: string;
  port?: number;
  hops: TracerouteHop[];
  completed: boolean;
  error?: string;
}

// ============================================================================
// L2 Path Discovery Types
// ============================================================================

export interface PortInfo {
  name: string; // "Gi0/1"
  index: number;
  speed: string; // "1Gbps"
  duplex: string;
  vlans: number[];
  isTrunk: boolean;
  connectedTo: string; // Device name/MAC
}

export interface L2Hop {
  device: string; // Switch name
  deviceIp: string;
  ingressPort: PortInfo | null;
  egressPort: PortInfo | null;
  source: 'lldp' | 'cdp' | 'snmp';
}

export interface L2PathResult {
  hops: L2Hop[];
}

export interface PathRequest {
  source: string; // IP or "self"
  destination: string; // IP or hostname
  method: 'l3' | 'l2' | 'both';
  protocol: 'icmp' | 'udp' | 'tcp';
  port?: number;
}

export interface PathResponse {
  l3Path?: TracerouteResult;
  l2Path?: L2PathResult;
}
