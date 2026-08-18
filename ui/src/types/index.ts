/**
 * The Seed Type Definitions
 */

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

// TracerouteRequest and PathRequest now come from the generated schema
// (code-first contract): src/types/generated/traceroute-request.ts and path.ts.
// The result/view-model types below (TracerouteResult, L2PathResult,
// PathResponse) stay hand-maintained — PathResponse is a Phase-3-deferred DTO
// (it nests discovery.* domain types) and the others are its building blocks.

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

export interface PathResponse {
  l3Path?: TracerouteResult;
  l2Path?: L2PathResult;
}
