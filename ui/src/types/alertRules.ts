/**
 * Alert rules — wire-shape mirror of /api/v1/alert-rules.
 * Keep field names aligned with internal/api/handlers_alert_rules.go
 * encodeAlertRule output so the JSON parses cleanly.
 *
 * Operators CRUD rules here; the listener pipeline reloads them on
 * each scan and falls back to the hardcoded DefaultListenerRules
 * when the table is empty (semantics decision in PR #1377).
 */

export interface AlertRule {
  id: number;
  name: string;
  enabled: boolean;
  matchKind: string;
  matchSeverity: string;
  matchPayloadContains: string;
  alertType: string;
  alertSeverity: string;
  alertTitle: string;
  alertMessage: string;
  windowSeconds: number;
  thresholdCount: number;
  createdAt: string;
  updatedAt: string;
}

/** Wire shape for POST/PUT — omits server-managed audit columns. */
export interface AlertRuleInput {
  name: string;
  enabled: boolean;
  matchKind?: string;
  matchSeverity?: string;
  matchPayloadContains?: string;
  alertType: string;
  alertSeverity: string;
  alertTitle: string;
  alertMessage?: string;
  windowSeconds?: number;
  thresholdCount?: number;
}

/** GET /api/v1/alert-rules list envelope. */
export interface AlertRulesListResponse {
  count: number;
  rules: AlertRule[];
}
