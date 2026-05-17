/**
 * Default-config constants and payload normaliser used by SettingsDrawer.
 *
 * Pulled out so the drawer file isn't carrying ~110 lines of pure data
 * shape before its component definition.
 */

import type {
  CableTestSettings as CableTestSettingsType,
  LinkSettings as LinkSettingsType,
  TestsSettings,
  VulnerabilityScanSettings,
} from '../../types/settings';
import { generateId } from '../../utils/id';

export const INLINE_DEFAULT_LINK_SETTINGS: LinkSettingsType = {
  mode: 'auto',
  availableModes: [],
};

export const INLINE_DEFAULT_CABLE_TEST_SETTINGS: CableTestSettingsType = {
  enabled: true,
};

export const INLINE_DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityScanSettings = {
  enabled: true,
  cveDatabase: 'nvd',
  nvdApiKey: '',
  updateInterval: 86400,
  severityThreshold: 'medium',
  maxConcurrent: 5,
  autoScan: true,
};

// Utility: ensure every item in an array has a stable id for React keying/updating.
export const withIds = <T extends { id?: string }>(items: T[] = []): Array<T & { id: string }> =>
  items.map((item) => ({ ...item, id: item.id ?? generateId() }));

// Normalize tests/DNS settings payload before sending to the API.
export interface NormalizedTestsSettings {
  dnsHostname: string;
  dnsServers: Array<{ address: string; enabled: boolean }>;
  pingTargets: Array<{ name: string; host: string; enabled: boolean }>;
  tcpPorts: Array<{ name: string; host: string; port: number; enabled: boolean }>;
  udpPorts: Array<{ name: string; host: string; port: number; enabled: boolean }>;
  httpEndpoints: Array<{ name: string; url: string; expectedStatus: number; enabled: boolean }>;
  runPerformance: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runDiscovery: boolean;
  speedtest: { serverId: string; autoRunOnLink: boolean };
  iperf: { autoRunOnLink: boolean | undefined };
}

export const normalizeTestsSettingsForSave = (settings: TestsSettings): NormalizedTestsSettings => {
  const dnsHostname = settings.dnsHostname?.trim() || 'google.com';

  const dnsServers = (settings.dnsServers || [])
    .map((server) => ({
      address: server.address.trim(),
      enabled: server.enabled !== false,
    }))
    .filter((server) => server.address.length > 0);

  const pingTargets = (settings.pingTargets || [])
    .map((target) => ({
      name: target.name?.trim() || target.host.trim(),
      host: target.host.trim(),
      enabled: target.enabled !== false,
    }))
    .filter((target) => target.host.length > 0);

  const tcpPorts = (settings.tcpPorts || [])
    .map((port) => ({
      name: port.name?.trim() || port.host.trim(),
      host: port.host.trim(),
      port: typeof port.port === 'number' ? port.port : Number.parseInt(String(port.port), 10) || 0,
      enabled: port.enabled !== false,
    }))
    .filter((port) => port.host.length > 0 && port.port > 0);

  const udpPorts = (settings.udpPorts || [])
    .map((port) => ({
      name: port.name?.trim() || port.host.trim(),
      host: port.host.trim(),
      port: typeof port.port === 'number' ? port.port : Number.parseInt(String(port.port), 10) || 0,
      enabled: port.enabled !== false,
    }))
    .filter((port) => port.host.length > 0 && port.port > 0);

  const httpEndpoints = (settings.httpEndpoints || [])
    .map((endpoint) => ({
      name: endpoint.name?.trim() || endpoint.url.trim(),
      url: endpoint.url.trim(),
      expectedStatus:
        typeof endpoint.expectedStatus === 'number' && endpoint.expectedStatus > 0
          ? endpoint.expectedStatus
          : 200,
      enabled: endpoint.enabled !== false,
    }))
    .filter((endpoint) => endpoint.url.length > 0);

  return {
    dnsHostname,
    dnsServers,
    pingTargets,
    tcpPorts,
    udpPorts,
    httpEndpoints,
    runPerformance: settings.runPerformance !== false,
    runSpeedtest: settings.runSpeedtest !== false,
    runIperf: settings.runIperf !== false,
    runDiscovery: settings.runDiscovery !== false,
    speedtest: {
      serverId: settings.speedtest?.serverId?.trim() || '',
      autoRunOnLink: !!settings.speedtest?.autoRunOnLink,
    },
    iperf: {
      autoRunOnLink: !!settings.iperf?.autoRunOnLink,
    },
  };
};
