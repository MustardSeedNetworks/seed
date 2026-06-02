/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface PathResponse {
  l3Path?: TracerouteResult;
  l2Path?: L2PathResult;
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
export interface TracerouteHop {
  ttl: number;
  ip?: string;
  hostname?: string;
  rtt: number;
  state: string;
}
export interface L2PathResult {
  hops: L2Hop[];
}
export interface L2Hop {
  device: string;
  deviceIp: string;
  ingressPort: PortInfo;
  egressPort: PortInfo;
  source: string;
}
export interface PortInfo {
  name: string;
  index: number;
  speed: string;
  duplex: string;
  vlans: number[];
  isTrunk: boolean;
  connectedTo: string;
}
