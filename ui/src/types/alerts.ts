/**
 * UI-side filter state for the alerts list view. Mirrors the query
 * parameters parseAlertListOptions reads; the wire shapes are generated
 * (types/generated/alert-list-response).
 */
export interface AlertsFilter {
  severity: string;
  unacknowledgedOnly: boolean;
  unresolvedOnly: boolean;
}
