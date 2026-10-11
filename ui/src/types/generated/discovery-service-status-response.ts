/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DiscoveryServiceStatusResponse {
  running: boolean;
  scanning: boolean;
  deviceCount: number;
  lastScan: string;
  subnet: string;
  subnets: string[];
  localIP: string;
  interface: string;
  activeMethods: string[];
  snmpProbe: SNMPProbeReport;
  rescanIntervalMs: number;
  profilingStatus?: ProfilingStatus;
  degradationStatus?: DegradationStatus;
}
export interface SNMPProbeReport {
  silent: number;
  tried: number;
  answered: number;
  skipped?: string;
}
export interface ProfilingStatus {
  totalProfiled: number;
  inProgress: number;
  queueLength: number;
  profilingIps: string[];
  enabled: boolean;
  maxConcurrent: number;
  portsToScan: number;
  scanIntensity: string;
}
export interface DegradationStatus {
  overallHealth: 'healthy' | 'degraded' | 'critical';
  healthyMethods: string[];
  failedMethods: string[];
  warnings: string[];
  recommendations: string[];
}
