/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ProblemScanResponse {
  problems: NetworkProblem[];
  ipConflicts: IPConflict[];
  interfaceErrors: InterfaceErrorStats[];
  wifiProblems?: WiFiProblem[];
  scanTime: string;
  durationMs: number;
}
export interface NetworkProblem {
  id: string;
  category: string;
  type: string;
  severity: string;
  status: string;
  title: string;
  description: string;
  deviceId?: string;
  deviceMac?: string;
  interfaceName?: string;
  ipAddress?: string;
  affectedMacs?: string;
  ssid?: string;
  bssid?: string;
  channel?: number;
  currentValue?: number;
  thresholdValue?: number;
  unit?: string;
  firstSeen: string;
  lastSeen: string;
  resolvedAt?: string;
  occurrenceCount: number;
  metadata?: {};
}
export interface IPConflict {
  ipAddress: string;
  macs: string[];
  deviceIds: string[];
  firstSeen: string;
  lastSeen: string;
  isResolved: boolean;
}
export interface InterfaceErrorStats {
  deviceId: string;
  interfaceName: string;
  inputErrors: number;
  crcErrors: number;
  frameErrors: number;
  overruns: number;
  droppedInput: number;
  outputErrors: number;
  collisions: number;
  lateCollision: number;
  carrierErrors: number;
  droppedOutput: number;
  inputErrorsDelta?: number;
  outputErrorsDelta?: number;
  recordedAt: string;
}
export interface WiFiProblem {
  problemType: string;
  ssid?: string;
  bssid?: string;
  channel?: number;
  band?: string;
  signalDbm?: number;
  noiseDbm?: number;
  snr?: number;
  retryPercent?: number;
  coChannelAps?: number;
  adjacentChannelAps?: number;
  utilizationPercent?: number;
  isRogue?: boolean;
  isUnauthorized?: boolean;
  vendorMismatch?: boolean;
  expectedVendor?: string;
  actualVendor?: string;
  firstSeen: string;
  lastSeen: string;
}
