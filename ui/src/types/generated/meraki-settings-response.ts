/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface MerakiSettingsResponse {
  choice: 'unanswered' | 'off' | 'on';
  endpoint: string;
  hasApiKey: boolean;
  lastAttempt?: string;
  lastSuccess?: string;
  lastError?: string;
}
