/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ProfileExportResponse {
  version: string;
  exported_at: string;
  profiles: ProfileResponse[];
}
export interface ProfileResponse {
  id: string;
  name: string;
  description: string;
  config: unknown;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}
