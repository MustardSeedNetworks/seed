/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WMITargetListResponse {
  count: number;
  targets: WMITargetResponse[];
}
export interface WMITargetResponse {
  id: string;
  name: string;
  host: string;
  port: number;
  credentialsId: string;
  certSha256?: string;
  pendingCertSha256?: string;
  lastAttemptAt?: string;
  lastStatus?: 'ok' | 'auth_failed' | 'cert_mismatch' | 'cert_unconfirmed' | 'unreachable' | 'query_failed' | 'error';
  lastError?: string;
  createdAt: string;
  updatedAt: string;
}
