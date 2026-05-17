/**
 * useSettingsDrawerLoaders
 *
 * Bundles every per-section fetch callback that SettingsDrawer used to
 * declare inline (thresholds, IP, tests, iperf suggestions, wifi,
 * network-discovery, snmp, link, cable, logs) and the open-time
 * initial-load useEffect that fires them. The drawer passes in the
 * relevant state setters and init refs; the hook owns the network
 * calls and the on-open orchestration.
 */

import type React from 'react';
import { useCallback, useEffect } from 'react';
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

interface InitRefs {
  initialLoadRef: React.MutableRefObject<boolean>;
  thresholdsInitRef: React.MutableRefObject<boolean>;
  testsInitRef: React.MutableRefObject<boolean>;
  wifiInitRef: React.MutableRefObject<boolean>;
  linkInitRef: React.MutableRefObject<boolean>;
  cableTestInitRef: React.MutableRefObject<boolean>;
  networkDiscoveryInitRef: React.MutableRefObject<boolean>;
  snmpInitRef: React.MutableRefObject<boolean>;
  vulnInitRef: React.MutableRefObject<boolean>;
}

interface UseSettingsDrawerLoadersArgs {
  isOpen: boolean;
  initRefs: InitRefs;
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

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Aggregates ten per-section fetch callbacks and the open-time orchestration that previously lived inline in SettingsDrawer
export function useSettingsDrawerLoaders({
  isOpen,
  initRefs,
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
  const fetchThresholds = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        if (data.thresholds) {
          setThresholds((prev) => ({ ...prev, ...data.thresholds }));
        }
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch thresholds', err);
    }
  }, [setThresholds]);

  const fetchIpSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setIpSettings({
          mode: data.mode || 'dhcp',
          address: data.address || '',
          netmask: data.netmask || '24',
          gateway: data.gateway || '',
          dns: data.dns || [],
        });
        setDnsInput((data.dns || []).join(', '));
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch IP settings', err);
    }
  }, [setIpSettings, setDnsInput]);

  const fetchTestsSettings = useCallback(
    // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Reads many optional fields off the API payload; same shape as the original inline code
    async () => {
      try {
        const response = await fetch(`${API_BASE}/api/health-checks/settings`, {
          credentials: 'include',
        });
        if (response.ok) {
          const data = await (response.json() as Promise<Record<string, unknown>>);
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
              autoRunOnLink: data.iperf?.autoRunOnLink,
            },
          });
        }
      } catch (err) {
        logger.error(LogComponents.CONFIG, 'Failed to fetch tests settings', err);
      }
    },
    [setTestsSettings],
  );

  const fetchIperfSuggestions = useCallback(async () => {
    setIperfSuggestionsStatus('loading');
    setIperfSuggestionsError(null);
    try {
      const response = await fetch(`${API_BASE}/api/sap/iperf/suggestions`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setIperfSuggestions(Array.isArray(data) ? data : []);
        setIperfSuggestionsStatus('idle');
      } else {
        setIperfSuggestionsStatus('error');
        setIperfSuggestionsError('No iperf hosts found');
      }
    } catch (err) {
      setIperfSuggestionsStatus('error');
      setIperfSuggestionsError(err instanceof Error ? err.message : 'Failed to find iperf hosts');
    }
  }, [setIperfSuggestions, setIperfSuggestionsStatus, setIperfSuggestionsError]);

  const fetchWifiSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/canopy/wifi/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setWifiSettings({
          interface: data.interface || '',
          availableWifi: data.availableWifi || [],
          isWireless: data.isWireless,
        });
      }
    } catch (err) {
      logger.error(LogComponents.Wifi, 'Failed to fetch WiFi settings', err);
    }
  }, [setWifiSettings]);

  const fetchNetworkDiscoverySettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setNetworkDiscoverySettings({
          enabled: data.enabled ?? true,
          arpScanWorkers: data.arpScanWorkers ?? 50,
          pingTimeoutMs: data.pingTimeoutMs ?? 500,
          scanTimeoutMs: data.scanTimeoutMs ?? 30000,
          autoScan: data.autoScan ?? false,
          scanIntervalMs: data.scanIntervalMs ?? 0,
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
              bannerTimeoutMs: 2000,
            },
            tcpProbe: { timeoutMs: 2000, workers: 20 },
            traceroute: false,
            snmpQuery: false,
          },
          timing: data.timing ?? {
            probeIntervalMs: 75,
            rescanIntervalMs: 600000,
            workers: 50,
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
    } catch (err) {
      logger.error(LogComponents.Discovery, 'Failed to fetch network discovery settings', err);
    }
  }, [setNetworkDiscoverySettings]);

  const fetchSnmpSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/sap/snmp/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setSnmpSettings({
          communities: data.communities ?? ['public'],
          v3Credentials: data.v3Credentials ?? [],
          timeout: data.timeout ?? 5000,
          retries: data.retries ?? 2,
          port: data.port ?? 161,
        });
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch SNMP settings', err);
    }
  }, [setSnmpSettings]);

  const fetchLinkSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings/link`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        const mode = data.mode ?? (data.auto_negotiation ? 'auto' : `${data.speed}/${data.duplex}`);
        setLinkSettings({
          mode: mode,
          availableModes: data.available_modes ?? [],
        });
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch link settings', err);
    }
  }, [setLinkSettings]);

  const fetchCableTestSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings/cable`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setCableTestSettings({
          enabled: data.enabled ?? true,
        });
      }
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch cable test settings', err);
    }
  }, [setCableTestSettings]);

  const fetchLogPreview = useCallback(async () => {
    setLogLoading(true);
    setLogError(null);
    try {
      const response = await fetch(`${API_BASE}/api/harvest/logs?lines=200`, {
        credentials: 'include',
      });
      if (!response.ok) {
        throw new Error('Unable to load logs');
      }
      const data = await (response.json() as Promise<LogsResponse>);
      setLogPreview(data.lines || []);
    } catch (err) {
      setLogPreview([]);
      setLogError(err instanceof Error ? err.message : 'Failed to load log file');
    } finally {
      setLogLoading(false);
    }
  }, [setLogPreview, setLogLoading, setLogError]);

  // Open-time orchestration: reset init refs, fire every fetch, then
  // clear init refs after a short delay so the auto-save hooks ignore
  // the seeded values.
  useEffect(() => {
    if (!isOpen) {
      return;
    }
    initRefs.initialLoadRef.current = true;
    initRefs.thresholdsInitRef.current = true;
    initRefs.testsInitRef.current = true;
    initRefs.wifiInitRef.current = true;
    initRefs.linkInitRef.current = true;
    initRefs.cableTestInitRef.current = true;
    initRefs.networkDiscoveryInitRef.current = true;
    initRefs.snmpInitRef.current = true;
    initRefs.vulnInitRef.current = true;

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

    const timer = setTimeout(() => {
      initRefs.initialLoadRef.current = false;
      initRefs.thresholdsInitRef.current = false;
      initRefs.testsInitRef.current = false;
      initRefs.wifiInitRef.current = false;
      initRefs.linkInitRef.current = false;
      initRefs.cableTestInitRef.current = false;
      initRefs.networkDiscoveryInitRef.current = false;
      initRefs.snmpInitRef.current = false;
      initRefs.vulnInitRef.current = false;
    }, 500);

    return (): void => clearTimeout(timer);
  }, [
    isOpen,
    initRefs,
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
