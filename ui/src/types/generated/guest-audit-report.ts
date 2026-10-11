/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface GuestAuditReport {
  startedAt: string;
  completedAt: string;
  isolationFailed: boolean;
  reachableTargets: number;
  totalTargets: number;
  results: TargetResult[];
}
export interface TargetResult {
  target: Target;
  reachable: boolean;
  pingResponded: boolean;
  openPorts: number[];
  portResults: PortResult[];
  durationSeconds: number;
}
export interface Target {
  ip: string;
  label?: string;
}
export interface PortResult {
  port: number;
  open: boolean;
  error?: string;
}
