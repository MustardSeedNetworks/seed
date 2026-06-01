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
export interface IPConflict {
  ip_address: string;
  macs: string[];
  device_ids: string[];
  first_seen: string;
  last_seen: string;
  is_resolved: boolean;
}
export interface InterfaceErrorStats {
  device_id: string;
  interface_name: string;
  input_errors: number;
  crc_errors: number;
  frame_errors: number;
  overruns: number;
  dropped_input: number;
  output_errors: number;
  collisions: number;
  late_collision: number;
  carrier_errors: number;
  dropped_output: number;
  input_errors_delta?: number;
  output_errors_delta?: number;
  recorded_at: string;
}
export interface WiFiProblem {
  problem_type: string;
  ssid?: string;
  bssid?: string;
  channel?: number;
  band?: string;
  signal_dbm?: number;
  noise_dbm?: number;
  snr?: number;
  retry_percent?: number;
  co_channel_aps?: number;
  adjacent_channel_aps?: number;
  utilization_percent?: number;
  is_rogue?: boolean;
  is_unauthorized?: boolean;
  vendor_mismatch?: boolean;
  expected_vendor?: string;
  actual_vendor?: string;
  first_seen: string;
  last_seen: string;
}
