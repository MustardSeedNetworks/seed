/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface GatewayResponse {
  gateway: string;
  reachable: boolean;
  sent: number;
  received: number;
  lossPercent: number;
  minTime: number;
  maxTime: number;
  avgTime: number;
  lastTime: number;
  status: string;
  ipv6?: GatewayPingResult;
}
export interface GatewayPingResult {
  gateway: string;
  reachable: boolean;
  sent: number;
  received: number;
  lossPercent: number;
  minTime: number;
  maxTime: number;
  avgTime: number;
  lastTime: number;
  status: string;
}
