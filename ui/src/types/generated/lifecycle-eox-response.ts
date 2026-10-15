/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface LifecycleEoXResponse {
  records: EoXRecord[];
}
export interface EoXRecord {
  productId: string;
  description?: string;
  endOfSale?: string;
  endOfSwMaintenance?: string;
  endOfSecuritySupport?: string;
  lastDateOfSupport?: string;
  bulletin?: string;
  migrationProductId?: string;
}
