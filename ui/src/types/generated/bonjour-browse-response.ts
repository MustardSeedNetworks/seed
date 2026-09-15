/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface BrowseResult {
  interface?: string;
  localPrefixes?: string[];
  serviceTypes: string[];
  services: ServiceInstance[];
  reflector: ReflectorStatus;
  responsesObserved: number;
  truncated: boolean;
  durationMs: number;
}
export interface ServiceInstance {
  instance: string;
  type: string;
  host?: string;
  port: number;
  addresses?: string[];
  txt?: {
    [k: string]: string;
  };
  origin: string;
  sourceAddresses?: string[];
}
export interface ReflectorStatus {
  state: string;
  remoteSubnets?: string[];
  forwardedBy?: string[];
  routedFrom?: string[];
}
