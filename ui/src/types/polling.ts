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

/** Wire shape for POST/PUT — omits server-managed audit columns. */
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
