/**
 * Topology wire shapes mirroring /api/v1/topology/*. Keep field
 * names aligned with internal/api/handlers_topology.go encoders.
 */

export interface TopologyNode {
  id: string;
  clientId: string;
  identityHash: string;
  displayName: string;
  deviceType: string;
  chassisId: string;
  sysName: string;
  primaryMac: string;
  primaryIp: string;
  firstSeen: string;
  lastSeen: string;
  metadata: Record<string, unknown>;
}

export interface TopologyInterface {
  id: number;
  nodeId: string;
  ifIndex: number;
  ifName: string;
  ifDescr: string;
  ifAlias: string;
  ifType: number;
  ifAdminStatus: number;
  ifOperStatus: number;
  ifPhysAddr: string;
  speedBps: number;
  lastSeen: string;
}

export interface TopologyLink {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  sourceInterface: string;
  targetInterface: string;
  linkType: string;
  status: string;
  speedMbps: number;
  utilizationPct: number;
  firstSeen: string;
  lastSeen: string;
  evidence: Record<string, unknown>;
}

export interface TopologyNodesResponse {
  count: number;
  nodes: TopologyNode[];
}

export interface TopologyNodeDetailResponse {
  node: TopologyNode;
  interfaces: TopologyInterface[];
  links: TopologyLink[];
}
