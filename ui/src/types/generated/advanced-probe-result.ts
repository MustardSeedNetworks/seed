/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface AdvancedProbeResult {
  ip: string;
  probedAt: string;
  osFingerprint?: OSFingerprint;
  serviceVersions?: ServiceVersion[];
  tlsInfo?: TLSInfo[];
}
export interface OSFingerprint {
  osFamily?: string;
  osVersion?: string;
  confidence: number;
  methods?: string[];
  ttlObserved?: number;
}
export interface ServiceVersion {
  port: number;
  service: string;
  product?: string;
  version?: string;
  extraInfo?: string;
  confidence: number;
}
export interface TLSInfo {
  port: number;
  version: string;
  cipherSuite: string;
  commonName: string;
  issuer: string;
  validFrom: string;
  validTo: string;
  daysUntilExpiry: number;
  selfSigned: boolean;
  subjectAltNames: string[];
  certErrors: string[];
}
