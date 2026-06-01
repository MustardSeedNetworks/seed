/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface OPCUAEndpointResponse {
  name: string;
  endpointUrl: string;
  securityMode?: string;
  securityPolicy?: string;
  enabled: boolean;
  criticality: number;
}
