/**
 * Default-config constants and payload normaliser used by SettingsDrawer.
 *
 * Pulled out so the drawer file isn't carrying ~110 lines of pure data
 * shape before its component definition.
 */

import type { TestsSettingsResponse } from '../../types/generated/tests-settings-response';
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
  hasNvdApiKey: false,
  nvdApiKey: '',
  clearNvdApiKey: false,
  updateInterval: 86400,
  severityThreshold: 'medium',
  maxConcurrent: 5,
  autoScan: true,
};

// Give every loaded wire row a stable id for React keying/updating.
export const withIds = <Row extends object>(items: Row[] = []): Array<Row & { id: string }> =>
  items.map((item) => ({ ...item, id: generateId() }));

// Drop the client-only React key; the server decodes the body strictly.
const withoutId = <Row extends { id?: string }>({ id: _id, ...row }: Row): Omit<Row, 'id'> => row;

// Normalize the probe settings before PUT /telemetry/probes/settings. Every
// list is sent: the server replaces each one with what it receives, so a list
// left out is a list deleted.
export const normalizeTestsSettingsForSave = (settings: TestsSettings): TestsSettingsResponse => {
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
      ...withoutId(endpoint),
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
    rtspEndpoints: settings.rtspEndpoints.map(withoutId),
    dicomEndpoints: settings.dicomEndpoints.map(withoutId),
    hl7Endpoints: settings.hl7Endpoints.map(withoutId),
    fhirEndpoints: settings.fhirEndpoints.map(withoutId),
    sqlEndpoints: settings.sqlEndpoints.map(withoutId),
    fileShareEndpoints: settings.fileShareEndpoints.map(withoutId),
    ldapEndpoints: settings.ldapEndpoints.map(withoutId),
    ltiEndpoints: settings.ltiEndpoints.map(withoutId),
    opcuaEndpoints: settings.opcuaEndpoints.map(withoutId),
    modbusEndpoints: settings.modbusEndpoints.map(withoutId),
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
