/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface IPConfigResponse {
  interface: string;
  mac: string;
  mode: string;
  ipv4?: IPv4Info;
  ipv6: IPv6Info[];
  dns: string[];
  timing?: DHCPTimingInfo;
}
export interface IPv4Info {
  address: string;
  subnet: string;
  gateway?: string;
  dhcpServer?: string;
  leaseTime?: number;
}
export interface IPv6Info {
  address: string;
  prefix: number;
  scope: string;
  source: string;
}
export interface DHCPTimingInfo {
  discover: number;
  offer: number;
  request: number;
  ack: number;
  total: number;
}
