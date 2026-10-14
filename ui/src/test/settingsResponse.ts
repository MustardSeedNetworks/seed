/**
 * A complete GET /api/v1/settings answer for tests that render the settings
 * drawer: the server always sends every section, so a fixture does too.
 */

import type { SettingsResponse, ThresholdPair } from '../types/generated/settings-response';

const pair = (good: number, warning: number): ThresholdPair => ({ good, warning });

export const SETTINGS_RESPONSE: SettingsResponse = {
  vlan: { enabled: false, id: 0 },
  ip: { mode: 'dhcp' },
  thresholds: {
    dns: pair(50, 200),
    gateway: pair(10, 50),
    wifi: pair(-70, -80),
    customPing: pair(10, 100),
    customTcp: pair(100, 500),
    customHttp: pair(500, 2000),
    httpTimings: {
      dns: pair(50, 500),
      tcp: pair(100, 500),
      tls: pair(150, 500),
      ttfb: pair(500, 2000),
    },
  },
  healthChecks: { runPerformance: true, runSpeedtest: true, runIperf: true, runDiscovery: true },
  speedtest: { serverId: '', autoRunOnLink: true },
  iperf: {
    autoRunOnLink: false,
    server: '',
    port: 5201,
    protocol: 'tcp',
    direction: 'download',
    duration: 10,
    serverPort: 5201,
    enableServer: true,
  },
  alerts: {
    webhook: { url: '', secretSet: false },
    email: { host: '', port: 0, tls: '', username: '', passwordSet: false, from: '', to: [] },
    escalations: [],
    syslog: { host: '', port: 0, transport: '' },
  },
  displayOptions: { showPublicIP: true, unitSystem: 'sae' },
  identity: { name: '', location: '' },
};
