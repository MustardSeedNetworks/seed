/**
 * Result types consumed by HealthCheckCard and its protocol section subviews.
 *
 * Mirrors the /api/v1/sap/health-checks/run response. Split out so the
 * card itself and the protocol sections can share these definitions
 * without one file owning all the noise.
 */

export type StatusValue = 'success' | 'warning' | 'error';

export interface TestResult {
  name: string;
  host?: string;
  port?: number;
  url?: string;
  success: boolean;
  latency: number;
  error?: string;
  status?: number;
  testStatus?: StatusValue;
  // Extended ping fields
  packetLoss?: number;
  jitter?: number;
  minLatency?: number;
  maxLatency?: number;
  dnsLatency?: number;
  tcpConnect?: number;
  tlsLatency?: number;
  ttfbLatency?: number;
  // Per-phase status fields
  dnsStatus?: StatusValue;
  tcpStatus?: StatusValue;
  tlsStatus?: StatusValue;
  ttfbStatus?: StatusValue;
  // Certificate expiry fields
  certDaysLeft?: number;
  certStatus?: StatusValue;
  certExpiry?: string;
  certCommonName?: string;
  tlsVersion?: string;
  certIssuer?: string;
}

export interface SqlTestResult {
  name: string;
  driver: string;
  host: string;
  port: number;
  database: string;
  success: boolean;
  connectTimeMs: number;
  queryTimeMs?: number;
  totalTimeMs: number;
  serverVersion?: string;
  error?: string;
}

export interface FileShareTestResult {
  name: string;
  protocol: string;
  host: string;
  share: string;
  success: boolean;
  connectTimeMs: number;
  readSpeedMbps?: number;
  writeSpeedMbps?: number;
  readLatencyMs?: number;
  writeLatencyMs?: number;
  testFileSizeMb?: number;
  totalTimeMs: number;
  error?: string;
}

export interface LdapTestResult {
  name: string;
  host: string;
  port: number;
  useTls: boolean;
  success: boolean;
  connectTimeMs: number;
  bindTimeMs?: number;
  searchTimeMs?: number;
  totalTimeMs: number;
  entriesFound?: number;
  serverInfo?: string;
  error?: string;
}

// Video protocol results
export interface RtspTestResult {
  name: string;
  url: string;
  success: boolean;
  connectTimeMs: number;
  streamInfo?: string;
  codec?: string;
  resolution?: string;
  error?: string;
}

// Medical protocol results
export interface DicomTestResult {
  name: string;
  host: string;
  port: number;
  aeTitle: string;
  success: boolean;
  connectTimeMs: number;
  echoTimeMs?: number;
  totalTimeMs: number;
  serverAeTitle?: string;
  error?: string;
}

export interface Hl7TestResult {
  name: string;
  host: string;
  port: number;
  success: boolean;
  connectTimeMs: number;
  responseTimeMs?: number;
  totalTimeMs: number;
  ackCode?: string; // AA, AE, AR
  serverVersion?: string;
  error?: string;
}

export interface FhirTestResult {
  name: string;
  baseUrl: string;
  success: boolean;
  connectTimeMs: number;
  responseTimeMs?: number;
  totalTimeMs: number;
  fhirVersion?: string;
  resourceCount?: number;
  serverName?: string;
  error?: string;
}

// Education protocol results
export interface LtiTestResult {
  name: string;
  launchUrl: string;
  success: boolean;
  connectTimeMs: number;
  responseTimeMs?: number;
  totalTimeMs: number;
  ltiVersion?: string;
  error?: string;
}

// Industrial protocol results
export interface OpcuaTestResult {
  name: string;
  endpointUrl: string;
  success: boolean;
  connectTimeMs: number;
  browseTimeMs?: number;
  totalTimeMs: number;
  securityMode?: string;
  serverState?: string;
  productName?: string;
  error?: string;
}

export interface ModbusTestResult {
  name: string;
  host: string;
  port: number;
  unitId: number;
  success: boolean;
  connectTimeMs: number;
  readTimeMs?: number;
  totalTimeMs: number;
  registerValue?: number;
  error?: string;
}

export interface EnterpriseResults {
  sqlResults?: SqlTestResult[];
  fileShareResults?: FileShareTestResult[];
  ldapResults?: LdapTestResult[];
}

export interface VideoResults {
  rtspResults?: RtspTestResult[];
}

export interface MedicalResults {
  dicomResults?: DicomTestResult[];
  hl7Results?: Hl7TestResult[];
  fhirResults?: FhirTestResult[];
}

export interface EducationResults {
  ltiResults?: LtiTestResult[];
}

export interface IndustrialResults {
  opcuaResults?: OpcuaTestResult[];
  modbusResults?: ModbusTestResult[];
}

export interface HealthCheckData {
  pingResults: TestResult[];
  tcpResults: TestResult[];
  udpResults: TestResult[];
  httpResults: TestResult[];
  enterpriseResults?: EnterpriseResults;
  videoResults?: VideoResults;
  medicalResults?: MedicalResults;
  educationResults?: EducationResults;
  industrialResults?: IndustrialResults;
  hasTests: boolean;
}
