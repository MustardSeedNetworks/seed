/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ListenResult {
  port: number;
  family: 'ipv4' | 'ipv6';
  listenedMs: number;
  probes: number;
  ignored: number;
  runs: RunResult[];
  runsTruncated: boolean;
  dscpObserved: boolean;
}
export interface RunResult {
  runId: string;
  sender: string;
  classes: ClassResult[];
  preserved: boolean;
}
export interface ClassResult {
  sentDscp: number;
  sentName?: string;
  expected: number;
  received: number;
  observed: ObservedDSCP[];
  verdict: 'preserved' | 'remarked' | 'mixed' | 'lost' | 'unobserved';
}
export interface ObservedDSCP {
  dscp: number;
  name?: string;
  count: number;
}
