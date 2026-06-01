/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface LogStatsResponse {
  total_count: number;
  by_level: {
    [k: string]: number;
  };
  by_layer: {
    [k: string]: number;
  };
  by_component: {
    [k: string]: number;
  };
  errors_last_hour: number;
  warnings_last_hour: number;
}
