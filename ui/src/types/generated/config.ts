/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface Config {
  version: number;
  server: ServerConfig;
  interface: InterfaceConfig;
  vlan: VLANConfig;
  ip: IPConfig;
  discovery: DiscoveryConfig;
  networkDiscovery: NetworkDiscoveryConfig;
  dns: DNSConfig;
  healthChecks: HealthChecksConfig;
  speedtest: SpeedtestConfig;
  iperf: IperfConfig;
  thresholds: ThresholdsConfig;
  auth: AuthConfig;
  security: SecurityConfig;
  dhcp: DHCPConfig;
  snmp: SNMPConfig;
  fabOptions: FABOptionsConfig;
  displayOptions: DisplayOptionsConfig;
  logging: LoggingConfig;
  database: DatabaseConfig;
  link?: LinkConfig;
  cableTest?: CableTestConfig;
}
export interface ServerConfig {
  port: number;
  https: boolean;
  cert_file: string;
  key_file: string;
  acme?: ACMEConfig;
}
export interface ACMEConfig {
  enabled: boolean;
  domain: string;
  email: string;
  cache_dir?: string;
  staging?: boolean;
}
export interface InterfaceConfig {
  default: string;
  fallbacks: string[];
  wifi?: string;
  ethernet?: string[];
  wifi_list?: string[];
  startup_retries: number;
  startup_retry_wait: number;
}
export interface VLANConfig {
  enabled: boolean;
  id: number;
}
export interface IPConfig {
  mode: string;
  static?: StaticIP;
}
export interface StaticIP {
  address: string;
  netmask: string;
  gateway: string;
  dns: string[];
}
export interface DiscoveryConfig {
  protocol: string;
  timeout: number;
}
export interface NetworkDiscoveryConfig {
  options: DiscoveryOptions;
  timing: DiscoveryTiming;
  additional_subnets: SubnetConfig[];
  enabled: boolean;
  arp_scan_workers: number;
  ping_timeout: number;
  scan_timeout: number;
  auto_scan: boolean;
  scan_interval: number;
  oui_file_path: string;
  oui_max_age: number;
  fingerprinting?: FingerprintingConfig;
  profiler?: DeviceProfilerConfig;
  ipv6_enabled: boolean;
}
export interface DiscoveryOptions {
  passiveProtocols: PassiveProtocolConfig;
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanConfig;
  tcpProbe: TCPProbeConfig;
  traceroute: boolean;
  snmpQuery: boolean;
}
export interface PassiveProtocolConfig {
  lldp: boolean;
  cdp: boolean;
  edp: boolean;
  ndp: boolean;
}
export interface PortScanConfig {
  enabled: boolean;
  preset: string;
  tcpPorts: string;
  udpPorts: string;
  bannerTimeout: number;
}
export interface TCPProbeConfig {
  timeout: number;
  workers: number;
}
export interface DiscoveryTiming {
  probe_interval: number;
  rescan_interval: number;
  workers: number;
}
export interface SubnetConfig {
  cidr: string;
  name: string;
  enabled: boolean;
}
export interface FingerprintingConfig {
  enabled: boolean;
  os_detection: boolean;
  service_probes: boolean;
}
export interface DeviceProfilerConfig {
  enabled: boolean;
  timeout: number;
  max_concurrent: number;
  quick_ports: number[];
}
export interface DNSConfig {
  test_hostname: string;
  timeout: number;
  servers?: DNSServer[];
}
export interface DNSServer {
  address: string;
  enabled: boolean;
}
export interface HealthChecksConfig {
  ping_targets: PingTarget[];
  tcp_ports: TCPPortTest[];
  udp_ports: UDPPortTest[];
  http_endpoints: HTTPEndpoint[];
  rtsp_endpoints: RTSPEndpoint[];
  dicom_endpoints: DICOMEndpoint[];
  hl7_endpoints: HL7Endpoint[];
  fhir_endpoints: FHIREndpoint[];
  sql_endpoints: SQLEndpoint[];
  fileshare_endpoints: FileShareEndpoint[];
  ldap_endpoints: LDAPEndpoint[];
  lti_endpoints: LTIEndpoint[];
  opcua_endpoints: OPCUAEndpoint[];
  modbus_endpoints: ModbusEndpoint[];
  run_performance: boolean;
  run_speedtest: boolean;
  run_iperf: boolean;
  run_discovery: boolean;
}
export interface PingTarget {
  name: string;
  host: string;
  enabled: boolean;
}
export interface TCPPortTest {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}
export interface UDPPortTest {
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}
export interface HTTPEndpoint {
  name: string;
  url: string;
  expected_status: number;
  enabled: boolean;
  body_match?: string;
  body_match_is_regex?: boolean;
  check_security_headers?: boolean;
  follow_redirects?: boolean;
  max_redirects?: number;
}
export interface RTSPEndpoint {
  name: string;
  url: string;
  enabled: boolean;
}
export interface DICOMEndpoint {
  name: string;
  host: string;
  port: number;
  called_ae: string;
  calling_ae: string;
  enabled: boolean;
}
export interface HL7Endpoint {
  name: string;
  host: string;
  port: number;
  sending_app: string;
  sending_facility: string;
  receiving_app: string;
  receiving_facility: string;
  enabled: boolean;
}
export interface FHIREndpoint {
  name: string;
  base_url: string;
  auth_type: string;
  username?: string;
  password?: string;
  bearer_token?: string;
  client_id?: string;
  client_secret?: string;
  token_url?: string;
  enabled: boolean;
}
export interface SQLEndpoint {
  name: string;
  driver: string;
  host: string;
  port: number;
  database: string;
  username?: string;
  password?: string;
  ssl_mode?: string;
  test_query?: string;
  enabled: boolean;
}
export interface FileShareEndpoint {
  name: string;
  protocol: string;
  host: string;
  share: string;
  path?: string;
  username?: string;
  password?: string;
  domain?: string;
  test_read_performance?: boolean;
  test_write_performance?: boolean;
  test_file_size_mb?: number;
  enabled: boolean;
}
export interface LDAPEndpoint {
  name: string;
  host: string;
  port: number;
  use_tls: boolean;
  start_tls: boolean;
  base_dn: string;
  bind_dn?: string;
  bind_password?: string;
  search_filter?: string;
  enabled: boolean;
}
export interface LTIEndpoint {
  name: string;
  launch_url: string;
  consumer_key?: string;
  consumer_secret?: string;
  lti_version?: string;
  client_id?: string;
  deployment_id?: string;
  platform_url?: string;
  enabled: boolean;
}
export interface OPCUAEndpoint {
  name: string;
  endpoint_url: string;
  security_mode?: string;
  security_policy?: string;
  username?: string;
  password?: string;
  cert_path?: string;
  key_path?: string;
  enabled: boolean;
}
export interface ModbusEndpoint {
  name: string;
  host: string;
  port: number;
  unit_id: number;
  test_register: number;
  register_type?: string;
  enabled: boolean;
}
export interface SpeedtestConfig {
  server_id: string;
  auto_run_on_link: boolean;
}
export interface IperfConfig {
  auto_run_on_link: boolean;
  server: string;
  port: number;
  protocol: string;
  direction: string;
  duration: number;
  server_port: number;
  enable_server: boolean;
}
export interface ThresholdsConfig {
  dhcp: DHCPThresholds;
  dns: Threshold;
  ping: Threshold;
  wifi: WiFiThresholds;
  link: LinkThresholds;
  custom_tests: CustomThresholds;
}
export interface DHCPThresholds {
  total: Threshold;
  per_phase: Threshold;
}
export interface Threshold {
  warning: number;
  critical: number;
}
export interface WiFiThresholds {
  signal: SignalThreshold;
}
export interface SignalThreshold {
  warning: number;
  critical: number;
}
export interface LinkThresholds {
  flap_count_24h: IntThreshold;
}
export interface IntThreshold {
  warning: number;
  critical: number;
}
export interface CustomThresholds {
  ping: Threshold;
  tcp: Threshold;
  udp: Threshold;
  http: Threshold;
  http_timings: HTTPTimingThresholds;
  cert_expiry: CertExpiryThreshold;
}
export interface HTTPTimingThresholds {
  dns: Threshold;
  tcp: Threshold;
  tls: Threshold;
  ttfb: Threshold;
}
export interface CertExpiryThreshold {
  warning: number;
  critical: number;
}
export interface AuthConfig {
  default_username: string;
  default_password_hash: string;
  session_timeout: number;
  jwt_secret?: string;
  sso?: SSOConfig;
}
export interface SSOConfig {
  providers: SSOProviderConfig[];
}
export interface SSOProviderConfig {
  enabled: boolean;
  name: string;
  client_id: string;
  client_secret: string;
  redirect_url: string;
  scopes?: string[];
  tenant_id?: string;
}
export interface SecurityConfig {
  allowed_origins: string[];
  vulnerability_scanning: VulnerabilityScanConfig;
  guest_network_audit: GuestNetworkAuditConfig;
}
export interface VulnerabilityScanConfig {
  enabled: boolean;
  cve_database: string;
  nvd_api_key: string;
  update_interval: number;
  severity_threshold: string;
  max_concurrent: number;
  auto_scan: boolean;
}
export interface GuestNetworkAuditConfig {
  enabled: boolean;
  targets?: GuestAuditTarget[];
  ports?: number[];
}
export interface GuestAuditTarget {
  ip: string;
  label?: string;
}
export interface DHCPConfig {
  rogue_detection: RogueDetectionConfig;
}
export interface RogueDetectionConfig {
  enabled: boolean;
  known_servers: string[];
  alert_on_detection: boolean;
}
export interface SNMPConfig {
  communities: string[];
  v3_credentials?: SNMPv3Credential[];
  timeout: number;
  retries: number;
  port: number;
  max_repetitions: number;
}
export interface SNMPv3Credential {
  name: string;
  username: string;
  auth_protocol: string;
  auth_password: string;
  priv_protocol: string;
  priv_password: string;
  context_name: string;
  security_level: string;
}
export interface FABOptionsConfig {
  run_link: boolean;
  run_switch: boolean;
  run_vlan: boolean;
  run_ip_config: boolean;
  run_gateway: boolean;
  run_dns: boolean;
  run_health_checks: boolean;
  run_network_discovery: boolean;
  run_speedtest: boolean;
  run_iperf: boolean;
  run_performance: boolean;
  auto_scan_on_link: boolean;
}
export interface DisplayOptionsConfig {
  show_public_ip: boolean;
  unit_system: string;
}
export interface LoggingConfig {
  level: string;
  format: string;
  add_source: boolean;
  file: string;
  max_size: number;
  max_backups: number;
  max_age: number;
  compress: boolean;
}
export interface DatabaseConfig {
  path: string;
  retention_days: number;
  enable_wal: boolean;
  max_connections: number;
}
export interface LinkConfig {
  mode?: string;
  available_modes?: string[];
}
export interface CableTestConfig {
  enabled: boolean;
}
