/**
 * Polling targets — wire-shape mirror of /api/v1/polling-targets.
 * Keep field names aligned with internal/api/handlers_polling_targets.go
 * encodePollingTarget output so the JSON parses cleanly.
 */

export interface PollingTarget {
  id: string;
  clientId: string;
  name: string;
  ipAddress: string;
  snmpVersion: string;
  credentialsId: string;
  pollIntervalSeconds: number;
  enabled: boolean;
  collectorChain: string[];
  lastStatus: string;
  lastError: string;
  lastPolledAt?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * Wire shape for POST/PUT — omits server-managed audit columns.
 *
 * Deliberately carries no `clientId`: the owning tenant comes from the
 * session on the server side, and the handler rejects unknown fields, so
 * adding one here would turn every create into a 400.
 */
export interface PollingTargetInput {
  name: string;
  ipAddress: string;
  snmpVersion?: string;
  credentialsId?: string;
  pollIntervalSeconds?: number;
  enabled: boolean;
  collectorChain?: string[];
}

/** GET /api/v1/polling-targets list envelope. */
export interface PollingTargetsListResponse {
  count: number;
  targets: PollingTarget[];
}
