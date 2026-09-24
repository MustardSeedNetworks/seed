/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface SendResult {
  target: string;
  port: number;
  runId: string;
  count: number;
  classes: SentClass[];
  marked: boolean;
}
export interface SentClass {
  dscp: number;
  name?: string;
  sent: number;
}
