/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface SNMPWalkResponse {
  targetId: string;
  oid: string;
  rows: SNMPWalkRow[];
  truncated: boolean;
}
export interface SNMPWalkRow {
  oid: string;
  name?: string;
  instance?: string;
  value: string;
}
