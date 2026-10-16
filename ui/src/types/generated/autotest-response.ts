/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface Report {
  interface: string;
  verdict: 'pass' | 'warn' | 'fail';
  startedAt: string;
  durationMs: number;
  stages: StageReport[];
}
export interface StageReport {
  stage: 'link' | 'cable' | 'dhcp' | 'dns' | 'gateway' | 'vlan' | 'internet';
  outcome: 'pass' | 'warn' | 'fail' | 'skipped';
  reason:
    | 'ok'
    | 'noCarrier'
    | 'halfDuplex'
    | 'linkUp'
    | 'unsupported'
    | 'cableFault'
    | 'incompleteLease'
    | 'noAddress'
    | 'leaseExpired'
    | 'noLeaseFound'
    | 'slow'
    | 'noResolvers'
    | 'noAnswer'
    | 'lossy'
    | 'noGateway'
    | 'unreachable'
    | 'notAdvertised'
    | 'vlanMismatch'
    | 'partial'
    | 'noTargets'
    | 'cancelled'
    | 'error';
  message?: string;
  readings?: Reading[];
  durationMs: number;
}
export interface Reading {
  label: string;
  value: string;
}
