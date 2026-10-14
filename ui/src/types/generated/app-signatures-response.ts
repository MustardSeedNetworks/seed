/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface AppSignaturesResponse {
  source: 'builtin' | 'custom';
  signatures: Signature[];
}
export interface Signature {
  name: string;
  description: string;
  protocol: number;
  ports?: string[];
}
