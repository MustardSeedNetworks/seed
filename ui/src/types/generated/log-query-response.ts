/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface LogQueryResponse {
  logs: LogEntry[];
  total_count: number;
  offset: number;
  limit: number;
}
export interface LogEntry {
  timestamp: string;
  level: string;
  layer: string;
  request_id?: string;
  session_id?: string;
  message: string;
  component?: string;
  duration_ms?: number;
  metadata?: {};
  stack?: string;
}
