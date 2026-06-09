/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface NetworkProblemsResponse {
  problems: NetworkProblem[];
  summary: ProblemSummary;
  total: number;
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
export interface ProblemSummary {
  totalActive: number;
  bySeverity: {
    [k: string]: number;
  };
  byCategory: {
    [k: string]: number;
  };
  recentCount: number;
  resolvedToday: number;
  lastScanTime: string;
}
