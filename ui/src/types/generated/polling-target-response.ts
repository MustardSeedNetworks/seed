/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface PollingTargetResponse {
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
