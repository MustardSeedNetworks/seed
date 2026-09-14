/**
 * Alerts wire shapes mirroring /api/v1/alerts. Severity + type are
 * strings rather than enums so the UI tolerates new server values
 * (e.g. when alert-rule editor adds custom severities).
 */

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
  metadata: Record<string, unknown>;
  deviceId?: string;
  acknowledgedBy?: string;
  acknowledgedAt?: string;
  resolvedAt?: string;
  /** The pipeline rule that raised this alert, e.g. "bgp.flap". */
  rule?: string;
  /** An earlier alert that probably caused this one (#409). */
  rootCauseId?: number;
}

export interface AlertsListResponse {
  count: number;
  alerts: Alert[];
}

export interface AlertActionResponse {
  id: number;
  acknowledged?: boolean;
  acknowledgedBy?: string;
  resolved?: boolean;
}

/** UI-side filter state for the list view. Mirrors the query
 * parameters parseAlertListOptions reads. */
export interface AlertsFilter {
  severity: string;
  unacknowledgedOnly: boolean;
  unresolvedOnly: boolean;
}
