/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface Result {
  id: string;
  interface: string;
  filter?: string;
  packets: number;
  bytes: number;
  durationMs: number;
  stopReason: 'duration' | 'size' | 'stopped';
}
