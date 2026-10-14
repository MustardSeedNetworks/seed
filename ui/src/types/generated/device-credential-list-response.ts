/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DeviceCredentialListResponse {
  count: number;
  credentials: Credentials[];
}
export interface Credentials {
  id: string;
  clientId: string;
  name: string;
  kind: string;
  securityLevel?: string;
  snmpV3User?: string;
  snmpV3AuthProto?: string;
  snmpV3PrivProto?: string;
  createdAt: string;
  updatedAt: string;
}
