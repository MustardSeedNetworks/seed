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
  /** The escalation stage last sent for this alert (P-B2); absent until one is. */
  escalationStage?: number;
  /** When that stage was sent. */
  escalatedAt?: string;
  /** An earlier alert that probably caused this one (#409). */
  rootCauseId?: number;
  /**
   * What happened when each configured receiver was offered this alert, one
   * entry per channel. Absent means delivery never applied — no receiver is
   * configured, which is the default — and must never render as a failure.
   */
  deliveries?: AlertDelivery[];
}

/** One channel's delivery outcome for one alert. */
export interface AlertDelivery {
  channel: 'webhook' | 'email' | 'syslog';
  status: 'pending' | 'delivered' | 'failed' | 'dropped';
  /** When the last attempt finished; absent while still queued. */
  attemptedAt?: string;
  /** The last attempt's error text, so the reason is readable without the log. */
  error?: string;
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
