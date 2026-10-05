/**
 * useSettingsDrawerLoaders
 *
 * Bundles every per-section fetch callback that SettingsDrawer used to
 * declare inline (thresholds, IP, tests, iperf suggestions, wifi,
 * network-discovery, snmp, link, cable, logs) and the open-time
 * initial-load useEffect that fires them. The drawer passes in the
 * relevant state setters; the hook owns the network
 * calls and the on-open orchestration.
 */

import type React from 'react';
import { useEffect } from 'react';
import { withIds } from '../components/settings/settingsDrawerNormalizer';
import { LogComponents, logger } from '../lib/logger';
import type {
  CableTestSettings as CableTestSettingsType,
  IperfSuggestion,
  IpSettings,
  LinkSettings as LinkSettingsType,
  LogsResponse,
  NetworkDiscoverySettings,
  SettingsThresholds,
  SnmpSettings as SnmpSettingsType,
  TestsSettings,
  WiFiSettings as WiFiSettingsType,
} from '../types/settings';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

interface UseSettingsDrawerLoadersArgs {
  isOpen: boolean;
  setThresholds: React.Dispatch<React.SetStateAction<SettingsThresholds>>;
  setIpSettings: (s: IpSettings) => void;
  setDnsInput: (value: string) => void;
  setTestsSettings: (s: TestsSettings) => void;
  setIperfSuggestions: (list: IperfSuggestion[]) => void;
  setIperfSuggestionsStatus: (status: 'idle' | 'loading' | 'error') => void;
  setIperfSuggestionsError: (msg: string | null) => void;
  setWifiSettings: (s: WiFiSettingsType) => void;
  setNetworkDiscoverySettings: (s: NetworkDiscoverySettings) => void;
  setSnmpSettings: (s: SnmpSettingsType) => void;
  setLinkSettings: (s: LinkSettingsType) => void;
  setCableTestSettings: (s: CableTestSettingsType) => void;
  setLogPreview: (lines: string[]) => void;
  setLogLoading: (loading: boolean) => void;
  setLogError: (msg: string | null) => void;
  fetchSubnets: () => Promise<void>;
  fetchVulnSettings: () => Promise<void>;
}

interface UseSettingsDrawerLoadersResult {
  fetchIperfSuggestions: () => Promise<void>;
  fetchLogPreview: () => Promise<void>;
}

export function useSettingsDrawerLoaders({
  isOpen,
  setThresholds,
  setIpSettings,
  setDnsInput,
  setTestsSettings,
  setIperfSuggestions,
  setIperfSuggestionsStatus,
  setIperfSuggestionsError,
  setWifiSettings,
  setNetworkDiscoverySettings,
  setSnmpSettings,
  setLinkSettings,
  setCableTestSettings,
  setLogPreview,
  setLogLoading,
  setLogError,
  fetchSubnets,
  fetchVulnSettings,
}: UseSettingsDrawerLoadersArgs): UseSettingsDrawerLoadersResult {
  const fetchThresholds = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/settings`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<{
            thresholds?: Partial<SettingsThresholds>;
          }>);
          if (data.thresholds) {
            setThresholds((prev) => ({ ...prev, ...data.thresholds }));
          }
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch thresholds', err);
      });
  };

  const fetchIpSettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/telemetry/ipconfig/settings`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Partial<IpSettings>>);
          setIpSettings({
            mode: data.mode || 'dhcp',
            address: data.address || '',
            netmask: data.netmask || '24',
            gateway: data.gateway || '',
            dns: data.dns || [],
          });
          setDnsInput((data.dns || []).join(', '));
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch IP settings', err);
      });
  };

  const fetchTestsSettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/telemetry/probes/settings`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Partial<TestsSettings>>);
          setTestsSettings({
            dnsHostname: data.dnsHostname || 'google.com',
            dnsServers: withIds(data.dnsServers || []).map((server) => ({
              ...server,
              enabled: server.enabled !== false,
            })),
            pingTargets: withIds(data.pingTargets || []).map((target) => ({
              ...target,
              enabled: target.enabled !== false,
            })),
            tcpPorts: withIds(data.tcpPorts || []).map((port) => ({
              ...port,
              port: port.port || 80,
              enabled: port.enabled !== false,
            })),
            udpPorts: withIds(data.udpPorts || []).map((port) => ({
              ...port,
              port: port.port || 53,
              enabled: port.enabled !== false,
            })),
            httpEndpoints: withIds(data.httpEndpoints || []).map((endpoint) => ({
              ...endpoint,
              expectedStatus: endpoint.expectedStatus || 200,
              enabled: endpoint.enabled !== false,
            })),
            runPerformance: data.runPerformance ?? true,
            runSpeedtest: data.runSpeedtest ?? true,
            runIperf: data.runIperf ?? true,
            runDiscovery: data.runDiscovery ?? true,
            speedtest: {
              serverId: data.speedtest?.serverId || '',
              autoRunOnLink: data.speedtest?.autoRunOnLink ?? true,
            },
            iperf: {
              autoRunOnLink: data.iperf?.autoRunOnLink ?? true,
            },
          });
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch tests settings', err);
      });
  };

  const fetchIperfSuggestions = async (): Promise<void> => {
    setIperfSuggestionsStatus('loading');
    setIperfSuggestionsError(null);
    await fetch(`${API_BASE}/api/v1/telemetry/iperf/suggestions`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<IperfSuggestion[]>);
          setIperfSuggestions(Array.isArray(data) ? data : []);
          setIperfSuggestionsStatus('idle');
        } else {
          setIperfSuggestionsStatus('error');
          setIperfSuggestionsError('No iperf hosts found');
        }
      })
      .catch((err: unknown) => {
        setIperfSuggestionsStatus('error');
        setIperfSuggestionsError(err instanceof Error ? err.message : 'Failed to find iperf hosts');
      });
  };

  const fetchWifiSettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/wifi/wifi/settings`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Partial<WiFiSettingsType>>);
          setWifiSettings({
            interface: data.interface || '',
            availableWifi: data.availableWifi || [],
            isWireless: data.isWireless ?? false,
          });
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.WIFI, 'Failed to fetch WiFi settings', err);
      });
  };

  const fetchNetworkDiscoverySettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/security/devices/settings`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Partial<NetworkDiscoverySettings>>);
          setNetworkDiscoverySettings({
            enabled: data.enabled ?? true,
            scanTimeoutMs: data.scanTimeoutMs ?? 30000,
            autoScan: data.autoScan ?? false,
            ipv6Enabled: data.ipv6Enabled ?? true,
            options: data.options ?? {
              passiveProtocols: { lldp: true, cdp: true, edp: true, ndp: true },
              arpScan: true,
              icmpScan: true,
              portScan: {
                enabled: false,
                preset: 'common',
                tcpPorts: '22,80,443,8080-8100',
                udpPorts: '53,123,161',
              },
              tcpProbe: { timeoutMs: 2000, workers: 20 },
              traceroute: false,
              snmpQuery: false,
            },
            timing: data.timing ?? {
              rescanIntervalMs: 60000,
            },
            profiler: data.profiler ?? {
              enabled: true,
              timeoutMs: 2000,
              maxConcurrent: 5,
              quickPorts: [22, 80, 443, 8080],
            },
            fingerprinting: data.fingerprinting ?? {
              enabled: false,
              osDetection: false,
              serviceProbes: false,
            },
          });
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.DISCOVERY, 'Failed to fetch network discovery settings', err);
      });
  };

  const fetchSnmpSettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/telemetry/snmp/settings`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Partial<SnmpSettingsType>>);
          setSnmpSettings({
            timeout: data.timeout ?? 5000,
            retries: data.retries ?? 2,
            port: data.port ?? 161,
          });
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch SNMP settings', err);
      });
  };

  const fetchLinkSettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/settings/link`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<{
            mode?: string;
            auto_negotiation?: boolean;
            speed?: string;
            duplex?: string;
            available_modes?: string[];
          }>);
          const mode =
            data.mode ?? (data.auto_negotiation ? 'auto' : `${data.speed}/${data.duplex}`);
          setLinkSettings({
            mode: mode,
            availableModes: data.available_modes ?? [],
          });
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch link settings', err);
      });
  };

  const fetchCableTestSettings = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/settings/cable`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Partial<CableTestSettingsType>>);
          setCableTestSettings({
            enabled: data.enabled ?? true,
          });
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.CONFIG, 'Failed to fetch cable test settings', err);
      });
  };

  const fetchLogPreview = async (): Promise<void> => {
    setLogLoading(true);
    setLogError(null);
    await fetch(`${API_BASE}/api/v1/reporting/logs?lines=200`, { credentials: 'include' })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error('Unable to load logs');
        }
        const data = await (response.json() as Promise<LogsResponse>);
        setLogPreview(data.lines || []);
      })
      .catch((err: unknown) => {
        setLogPreview([]);
        setLogError(err instanceof Error ? err.message : 'Failed to load log file');
      });
    setLogLoading(false);
  };

  // Load every section each time the drawer opens. The drawer saves only on
  // an edit, so seeding these values never writes them back.
  useEffect(() => {
    if (!isOpen) {
      return;
    }
    fetchThresholds().catch(() => undefined);
    fetchIpSettings().catch(() => undefined);
    fetchTestsSettings().catch(() => undefined);
    fetchWifiSettings().catch(() => undefined);
    fetchNetworkDiscoverySettings().catch(() => undefined);
    fetchSnmpSettings().catch(() => undefined);
    fetchVulnSettings().catch(() => undefined);
    fetchLinkSettings().catch(() => undefined);
    fetchCableTestSettings().catch(() => undefined);
    fetchSubnets().catch(() => undefined);
  }, [
    isOpen,
    fetchThresholds,
    fetchIpSettings,
    fetchTestsSettings,
    fetchWifiSettings,
    fetchNetworkDiscoverySettings,
    fetchSnmpSettings,
    fetchVulnSettings,
    fetchLinkSettings,
    fetchCableTestSettings,
    fetchSubnets,
  ]);

  return { fetchIperfSuggestions, fetchLogPreview };
}
