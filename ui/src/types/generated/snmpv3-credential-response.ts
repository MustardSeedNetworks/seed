/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface SNMPv3CredentialResponse {
  name: string;
  username: string;
  authProtocol: string;
  authPassword: string;
  privProtocol: string;
  privPassword: string;
  contextName: string;
  securityLevel: string;
}
