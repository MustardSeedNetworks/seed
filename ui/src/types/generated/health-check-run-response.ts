/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface HealthCheckRunResponse {
  pingResults: TestResult[];
  tcpResults: TestResult[];
  udpResults: TestResult[];
  httpResults: TestResult[];
  hasTests: boolean;
  enterpriseResults?: EnterpriseResults;
  videoResults?: VideoResults;
  medicalResults?: MedicalResults;
  educationResults?: EducationResults;
  industrialResults?: IndustrialResults;
}
export interface TestResult {
  name: string;
  success: boolean;
  latency: number;
  error?: string;
  status?: number;
  testStatus?: 'success' | 'warning' | 'error';
  packetLoss?: number;
  jitter?: number;
  dnsLatency?: number;
  tcpConnect?: number;
  tlsLatency?: number;
  ttfbLatency?: number;
  dnsStatus?: 'success' | 'warning' | 'error';
  tcpStatus?: 'success' | 'warning' | 'error';
  tlsStatus?: 'success' | 'warning' | 'error';
  ttfbStatus?: 'success' | 'warning' | 'error';
  certDaysLeft?: number;
  certStatus?: 'success' | 'warning' | 'error';
  certExpiry?: string;
  certIssuer?: string;
  tlsVersion?: string;
  certWithheld?: boolean;
}
export interface EnterpriseResults {
  sqlResults?: SQLTestResult[];
  fileShareResults?: FileShareTestResult[];
  ldapResults?: LDAPTestResult[];
}
export interface SQLTestResult {
  name: string;
  success: boolean;
  driver: string;
  host: string;
  port: number;
  serverVersion?: string;
  error?: string;
  totalTimeMs: number;
  connectTimeMs?: number;
  queryTimeMs?: number;
}
export interface FileShareTestResult {
  name: string;
  success: boolean;
  protocol: string;
  host: string;
  share: string;
  error?: string;
  connectTimeMs: number;
  readSpeedMbps?: number;
  writeSpeedMbps?: number;
}
export interface LDAPTestResult {
  name: string;
  success: boolean;
  useTls: boolean;
  host: string;
  port: number;
  serverInfo?: string;
  error?: string;
  totalTimeMs: number;
  connectTimeMs?: number;
  bindTimeMs?: number;
}
export interface VideoResults {
  rtspResults?: RTSPTestResult[];
}
export interface RTSPTestResult {
  name: string;
  success: boolean;
  url: string;
  codec?: string;
  resolution?: string;
  error?: string;
  connectTimeMs: number;
}
export interface MedicalResults {
  dicomResults?: DICOMTestResult[];
  hl7Results?: HL7TestResult[];
  fhirResults?: FHIRTestResult[];
}
export interface DICOMTestResult {
  name: string;
  success: boolean;
  host: string;
  port: number;
  aeTitle: string;
  serverAeTitle?: string;
  error?: string;
  totalTimeMs: number;
  echoTimeMs?: number;
}
export interface HL7TestResult {
  name: string;
  success: boolean;
  host: string;
  port: number;
  ackCode?: string;
  serverVersion?: string;
  error?: string;
  totalTimeMs: number;
  responseTimeMs?: number;
}
export interface FHIRTestResult {
  name: string;
  success: boolean;
  baseUrl: string;
  fhirVersion?: string;
  serverName?: string;
  error?: string;
  totalTimeMs: number;
  resourceCount?: number;
}
export interface EducationResults {
  ltiResults?: LTITestResult[];
}
export interface LTITestResult {
  name: string;
  success: boolean;
  launchUrl: string;
  ltiVersion?: string;
  error?: string;
  totalTimeMs: number;
}
export interface IndustrialResults {
  opcuaResults?: OPCUATestResult[];
  modbusResults?: ModbusTestResult[];
}
export interface OPCUATestResult {
  name: string;
  success: boolean;
  endpointUrl: string;
  securityMode?: string;
  productName?: string;
  serverState?: string;
  error?: string;
  totalTimeMs: number;
}
export interface ModbusTestResult {
  name: string;
  success: boolean;
  host: string;
  port: number;
  unitId: number;
  error?: string;
  totalTimeMs: number;
  registerValue?: number;
}
