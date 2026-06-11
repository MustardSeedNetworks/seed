/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ClientLogRequest {
  entries: ClientLogEntry[];
}
export interface ClientLogEntry {
  timestamp: string;
  level: string;
  component: string;
  message: string;
  requestId?: string;
  sessionId?: string;
  metadata?: {};
  stack?: string;
}
