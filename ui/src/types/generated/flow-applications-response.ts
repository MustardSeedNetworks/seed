/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface FlowApplicationsResponse {
  window: HistoryWindowResponse;
  by: string;
  applications: Application[];
}
export interface HistoryWindowResponse {
  from: string;
  to: string;
  days: number;
  resolution: string;
  source: string;
  clamped: boolean;
  requestedDays: number;
}
export interface Application {
  name: string;
  bytes: number;
  packets: number;
}
