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
  device_id?: string;
  device_mac?: string;
  interface_name?: string;
  ip_address?: string;
  affected_macs?: string;
  ssid?: string;
  bssid?: string;
  channel?: number;
  current_value?: number;
  threshold_value?: number;
  unit?: string;
  first_seen: string;
  last_seen: string;
  resolved_at?: string;
  occurrence_count: number;
  metadata?: {};
}
export interface ProblemSummary {
  total_active: number;
  by_severity: {
    [k: string]: number;
  };
  by_category: {
    [k: string]: number;
  };
  recent_count: number;
  resolved_today: number;
  last_scan_time: string;
}
