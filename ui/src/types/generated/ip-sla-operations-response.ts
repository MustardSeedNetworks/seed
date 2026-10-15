/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface IPSLAOperationsResponse {
  operations: IPSLAOperationResponse[];
  count: number;
}
export interface IPSLAOperationResponse {
  targetId: string;
  targetName: string;
  observedAt: string;
  index: number;
  tag?: string;
  rttType: number;
  target?: string;
  thresholdMs: number;
  frequencySeconds: number;
  status: number;
  latest?: IPSLAResultResponse;
}
export interface IPSLAResultResponse {
  completed: boolean;
  sense: number;
  rttMs: number;
  jitter?: IPSLAJitterResponse;
}
export interface IPSLAJitterResponse {
  avgJitterMs: number;
  received: number;
  lost: number;
  mos: number;
}
