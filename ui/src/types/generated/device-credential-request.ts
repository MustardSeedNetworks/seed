/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DeviceCredentialRequest {
  name: string;
  community?: string;
  snmpV3User?: string;
  snmpV3AuthSecret?: string;
  snmpV3PrivSecret?: string;
  snmpV3AuthProto?: string;
  snmpV3PrivProto?: string;
  sshUser?: string;
  sshPassword?: string;
}
