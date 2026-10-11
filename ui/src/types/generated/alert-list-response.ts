/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface AlertListResponse {
  count: number;
  alerts: Alert[];
}
export interface Alert {
  id: number;
  type: string;
  severity: string;
  title: string;
  message: string;
  source: string;
  acknowledged: boolean;
  resolved: boolean;
  createdAt: string;
  metadata: {
    [k: string]: unknown;
  };
  deviceId?: string;
  acknowledgedBy?: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
  deliveries?: AlertDelivery[];
  rule?: string;
  rootCauseId?: number;
  narrative?: AlertNarrative;
  escalationStage?: number;
  escalatedAt?: string;
}
export interface AlertDelivery {
  channel: 'webhook' | 'email' | 'syslog';
  status: 'pending' | 'delivered' | 'failed' | 'dropped';
  attemptedAt?: string;
  error?: string;
}
export interface AlertNarrative {
  summary: string;
  evidence: string[];
  nextCheck: string;
}
