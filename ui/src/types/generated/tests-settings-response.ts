/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface TestsSettingsResponse {
  dnsHostname: string;
  dnsServers: DNSServerResponse[];
  pingTargets: PingTargetResponse[];
  tcpPorts: TCPPortResponse[];
  udpPorts: UDPPortResponse[];
  httpEndpoints: HTTPEndpointResponse[];
  rtspEndpoints: RTSPEndpointResponse[];
  dicomEndpoints: DICOMEndpointResponse[];
  hl7Endpoints: HL7EndpointResponse[];
  fhirEndpoints: FHIREndpointResponse[];
  sqlEndpoints: SQLEndpointResponse[];
  fileShareEndpoints: FileShareEndpointResponse[];
  ldapEndpoints: LDAPEndpointResponse[];
  ltiEndpoints: LTIEndpointResponse[];
  opcuaEndpoints: OPCUAEndpointResponse[];
  modbusEndpoints: ModbusEndpointResponse[];
  speedtest: SpeedtestSettingsResponse;
  iperf: IperfSettingsResponse;
  runPerformance: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runDiscovery: boolean;
}
export interface DNSServerResponse {
  address: string;
  enabled: boolean;
}
export interface PingTargetResponse {
  name: string;
  host: string;
  enabled: boolean;
}
export interface TCPPortResponse {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}
export interface UDPPortResponse {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}
export interface HTTPEndpointResponse {
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
  bodyMatch?: string;
  bodyMatchIsRegex?: boolean;
  checkSecurityHeaders?: boolean;
  followRedirects?: boolean;
  maxRedirects?: number;
}
export interface RTSPEndpointResponse {
  name: string;
  url: string;
  enabled: boolean;
}
export interface DICOMEndpointResponse {
  name: string;
  host: string;
  port: number;
  calledAe: string;
  callingAe: string;
  enabled: boolean;
}
export interface HL7EndpointResponse {
  name: string;
  host: string;
  port: number;
  sendingApp: string;
  sendingFacility: string;
  receivingApp: string;
  receivingFacility: string;
  enabled: boolean;
  criticality: number;
}
export interface FHIREndpointResponse {
  name: string;
  baseUrl: string;
  authType: string;
  enabled: boolean;
  criticality: number;
}
export interface SQLEndpointResponse {
  name: string;
  driver: string;
  host: string;
  port: number;
  database: string;
  sslMode?: string;
  enabled: boolean;
  criticality: number;
}
export interface FileShareEndpointResponse {
  name: string;
  protocol: string;
  host: string;
  share: string;
  path?: string;
  testReadPerformance?: boolean;
  testWritePerformance?: boolean;
  testFileSizeMb?: number;
  enabled: boolean;
  criticality: number;
}
export interface LDAPEndpointResponse {
  name: string;
  host: string;
  port: number;
  useTls: boolean;
  startTls: boolean;
  baseDn: string;
  searchFilter?: string;
  enabled: boolean;
  criticality: number;
}
export interface LTIEndpointResponse {
  name: string;
  launchUrl: string;
  ltiVersion?: string;
  enabled: boolean;
  criticality: number;
}
export interface OPCUAEndpointResponse {
  name: string;
  endpointUrl: string;
  securityMode?: string;
  securityPolicy?: string;
  enabled: boolean;
  criticality: number;
}
export interface ModbusEndpointResponse {
  name: string;
  host: string;
  port: number;
  unitId: number;
  testRegister: number;
  registerType?: string;
  enabled: boolean;
  criticality: number;
}
export interface SpeedtestSettingsResponse {
  serverId: string;
  autoRunOnLink: boolean;
}
export interface IperfSettingsResponse {
  autoRunOnLink: boolean;
}
