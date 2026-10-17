/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface SettingsResponse {
  vlan: VLANSetting;
  ip: IPSetting;
  thresholds: ThresholdSettings;
  healthChecks: HealthCheckSettings;
  speedtest: SpeedtestSettings;
  iperf: IperfTestSettings;
  alerts: AlertSettings;
  displayOptions: DisplaySettings;
  identity: IdentitySettings;
}
export interface VLANSetting {
  enabled: boolean;
  id: number;
}
export interface IPSetting {
  mode: string;
}
export interface ThresholdSettings {
  dns: ThresholdPair;
  gateway: ThresholdPair;
  wifi: ThresholdPair;
  customPing: ThresholdPair;
  customTcp: ThresholdPair;
  customHttp: ThresholdPair;
  httpTimings: HTTPTimingSettings;
}
export interface ThresholdPair {
  good: number;
  warning: number;
}
export interface HTTPTimingSettings {
  dns: ThresholdPair;
  tcp: ThresholdPair;
  tls: ThresholdPair;
  ttfb: ThresholdPair;
}
export interface HealthCheckSettings {
  runPerformance: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runDiscovery: boolean;
}
export interface SpeedtestSettings {
  serverId: string;
  autoRunOnLink: boolean;
}
export interface IperfTestSettings {
  autoRunOnLink: boolean;
  server: string;
  port: number;
  protocol: string;
  direction: string;
  duration: number;
  serverPort: number;
  enableServer: boolean;
}
export interface AlertSettings {
  webhook: WebhookSettings;
  email: EmailSettings;
  escalations: EscalationLadder[];
  syslog: SyslogSettings;
  trap: TrapSettings;
}
export interface WebhookSettings {
  url: string;
  secretSet: boolean;
}
export interface EmailSettings {
  host: string;
  port: number;
  tls: string;
  username: string;
  passwordSet: boolean;
  from: string;
  to: string[];
}
export interface EscalationLadder {
  rule: string;
  stages: EscalationStage[];
  repeatSeconds: number;
}
export interface EscalationStage {
  afterSeconds: number;
  channels: ('webhook' | 'email' | 'syslog' | 'trap')[];
}
export interface SyslogSettings {
  host: string;
  port: number;
  transport: string;
}
export interface TrapSettings {
  host: string;
  port: number;
  communitySet: boolean;
}
export interface DisplaySettings {
  showPublicIP: boolean;
  unitSystem: string;
}
export interface IdentitySettings {
  name: string;
  location: string;
}
