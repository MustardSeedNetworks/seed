/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ObserveResult {
  interface: string;
  observedMs: number;
  messages: number;
  groups: ObservedGroup[];
  queriers: ObservedQuerier[];
  truncated: boolean;
}
export interface ObservedGroup {
  group: string;
  reporters: ObservedReporter[];
}
export interface ObservedReporter {
  address: string;
  version: number;
  reports: number;
  left: boolean;
}
export interface ObservedQuerier {
  address: string;
  version: number;
  queries: number;
  elected: boolean;
}
