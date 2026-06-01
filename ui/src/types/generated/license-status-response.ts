/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
/**
 * LicenseStatusResponse — generated from the Go struct in internal/api; refresh with `make schema` after struct changes.
 */
export interface LicenseStatusResponse {
  tier: string;
  tierValue: number;
  isTrialMode: boolean;
  trialDaysLeft?: number;
  canMintTokens: boolean;
  activated: boolean;
  expiresAt?: string;
}
