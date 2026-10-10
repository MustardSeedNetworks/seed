/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface AlertRuleListResponse {
  count: number;
  rules: AlertRuleResponse[];
}
export interface AlertRuleResponse {
  id: number;
  name: string;
  enabled: boolean;
  matchKind: string;
  matchSeverity: string;
  matchPayloadContains: string;
  alertType: string;
  alertSeverity: string;
  alertTitle: string;
  alertMessage: string;
  windowSeconds: number;
  thresholdCount: number;
  createdAt: string;
  updatedAt: string;
}
