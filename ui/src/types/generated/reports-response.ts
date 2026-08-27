/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ReportsResponse {
  reports: ReportInfo[];
}
export interface ReportInfo {
  id: string;
  name: string;
  type: string;
  format: string;
  status: string;
  fileSize?: number;
  createdAt: string;
  completedAt?: string;
  expiresAt?: string;
  error?: string;
}
