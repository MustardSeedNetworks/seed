/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface FileShareEndpointResponse {
  name: string;
  protocol: string;
  host: string;
  share: string;
  path?: string;
  testReadPerformance?: boolean;
  testWritePerformance?: boolean;
  testFileSizeMb?: number;
  enabled: boolean;
  criticality: number;
}
