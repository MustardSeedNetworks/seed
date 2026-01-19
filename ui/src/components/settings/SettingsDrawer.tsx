// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * SettingsDrawer Component (~1399 lines)
 *
 * Purpose: Master settings configuration panel managing all application settings.
 * Provides tabbed interface to Appearance, Discovery, Network, Performance, Health,
 * DNS, WiFi, SNMP, and Thresholds configuration sections with persistent storage.
 *
 * Key Features:
 * - Multi-section settings: Appearance, Discovery, Network, Performance, Health, DNS, WiFi, SNMP, Thresholds
 * - Persistent storage: Auto-saves settings to server with save status indication
 * - Real-time updates: Changes immediately reflected in UI
 * - Validation: Input validation before saving
 * - Import/Export: Settings backup and restore functionality
 * - Advanced options: Expandable sections for power users
 * - Save status: Shows success/error/unsaved indicators
 * - iperf3 suggestions: Autocomplete for iperf3 server discovery
 * - Keyboard shortcuts: Enter to save, ESC to close
 *
 * Usage:
 * ```typescript
 * <SettingsDrawer isOpen={showSettings} onClose={handleClose} />
 * ```
 *
 * Dependencies: useTheme, useAuth, useSettings context, all settings section components
 * State: All settings state, save status, validation errors, iperf3 suggestions
 */

import type React from 'react';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSettings } from '../../contexts/useSettings';
import { useTheme } from '../../hooks/useTheme';
import { LogComponents, logger } from '../../lib/logger';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import type {
  CableTestSettings as CableTestSettingsType,
  IperfSuggestion,
  IpSettings,
  LinkSettings as LinkSettingsType,
  LogsResponse,
  NetworkDiscoverySettings,
  SettingsThresholds,
  SnmpSettings as SnmpSettingsType,
  SubnetConfig,
  TestsSettings,
  VulnerabilityScanSettings,
  WiFiSettings as WiFiSettingsType,
} from '../../types/settings';
import { generateId } from '../../utils/id';
import { CollapsibleSection } from '../ui/CollapsibleSection';
import { Network } from '../ui/Icons';
import { AppearanceSettings } from './sections/AppearanceSettings';
import { AutoSaveIndicator } from './sections/AutoSaveIndicator';
import { CableTestSettings } from './sections/CableTestSettings';
import { ConfigBackupsSection } from './sections/ConfigBackupsSection';
import { DiscoverySettings } from './sections/DiscoverySettings';
import { DnsSettings } from './sections/DnsSettings';
import { HealthChecksSettings } from './sections/HealthChecksSettings';
import { LinkSettings } from './sections/LinkSettings';
import { MtuControl } from './sections/MtuControl';
import { PerformanceSettings } from './sections/PerformanceSettings';
import { ThresholdsSettings } from './sections/ThresholdsSettings';
import { UpdateSettings } from './sections/UpdateSettings';
import { VlanControl } from './sections/VlanControl';
import { VulnerabilitySettings } from './sections/VulnerabilitySettings';
import { WiFiSettings } from './sections/WiFiSettings';

// Inline defaults - avoids deprecated imports while maintaining fallback behavior
// The backend is the single source of truth; these are used only for initial state
const INLINE_DEFAULT_LINK_SETTINGS: LinkSettingsType = {
  mode: 'auto',
  availableModes: [],
};

const INLINE_DEFAULT_CABLE_TEST_SETTINGS: CableTestSettingsType = {
  enabled: true,
};

const INLINE_DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityScanSettings = {
  enabled: true,
  cveDatabase: 'nvd',
  nvdApiKey: '',
  updateInterval: 86400,
  severityThreshold: 'medium',
  maxConcurrent: 5,
  autoScan: true,
};

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

// Utility: ensure every item in an array has a stable id for React keying/updating.
const withIds = <T extends { id?: string }>(items: T[] = []): Array<T & { id: string }> =>
  items.map((item) => ({ ...item, id: item.id ?? generateId() }));

// Normalize tests/DNS settings payload before sending to the API.
interface NormalizedTestsSettings {
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

const normalizeTestsSettingsForSave = (settings: TestsSettings): NormalizedTestsSettings => {
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

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  version?: string;
  /** Whether currently viewing WiFi mode (shows WiFi settings instead of Link/Cable). */
  isWifi?: boolean;
}

export const SettingsDrawer: React.MemoExoticComponent<
  (props: SettingsDrawerProps) => React.ReactElement | null
> = memo(function settingsDrawer({
  isOpen,
  onClose,
  version = 'dev',
  isWifi = false,
}: SettingsDrawerProps): React.ReactElement | null {
  const { t } = useTranslation('settings');
  const { theme, setTheme, isDark } = useTheme();

  // Get settings from context - single source of truth
  const {
    displayOptions,
    iperfSettings,
    cardSettings,
    status: settingsStatus,
    updateDisplayOptions,
    updateIperfSettings,
    updateCardSettings,
  } = useSettings();

  // Create setter wrappers that use context update methods
  const setDisplayOptions = useCallback(
    (updater: React.SetStateAction<typeof displayOptions>) => {
      const newValue = typeof updater === 'function' ? updater(displayOptions) : updater;
      updateDisplayOptions(newValue);
    },
    [displayOptions, updateDisplayOptions],
  );

  const setIperfSettings = useCallback(
    (updater: React.SetStateAction<typeof iperfSettings>) => {
      const newValue = typeof updater === 'function' ? updater(iperfSettings) : updater;
      updateIperfSettings(newValue);
    },
    [iperfSettings, updateIperfSettings],
  );

  const scrollRef = useRef<HTMLDivElement | null>(null);
  // Track if health check settings were modified - dispatch event on drawer close
  const testsSettingsChangedRef = useRef(false);
  const prevIsOpenRef = useRef(isOpen);

  // Smooth scroll to top when opened
  useEffect(() => {
    if (isOpen && scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [isOpen]);

  // When drawer closes, dispatch healthChecksUpdated if test settings changed
  useEffect(() => {
    if (prevIsOpenRef.current && !isOpen && testsSettingsChangedRef.current) {
      // Drawer just closed and test settings were changed
      window.dispatchEvent(new CustomEvent('healthChecksUpdated'));
      testsSettingsChangedRef.current = false;
    }
    prevIsOpenRef.current = isOpen;
  }, [isOpen]);
  const [thresholds, setThresholds] = useState<SettingsThresholds>({
    dns: { good: 50, warning: 100 },
    gateway: { good: 20, warning: 50 },
    wifi: { good: -50, warning: -70 },
    customPing: { good: 50, warning: 100 },
    customTcp: { good: 100, warning: 500 },
    customHttp: { good: 500, warning: 2000 },
    httpTimings: {
      dns: { good: 100, warning: 500 },
      tcp: { good: 100, warning: 500 },
      tls: { good: 150, warning: 500 },
      ttfb: { good: 500, warning: 2000 },
    },
  });
  const [ipSettings, setIpSettings] = useState<IpSettings>({
    mode: 'dhcp',
    address: '',
    netmask: '24',
    gateway: '',
    dns: [],
  });
  const [testsSettings, setTestsSettings] = useState<TestsSettings>({
    dnsHostname: 'google.com',
    dnsServers: [],
    pingTargets: [],
    tcpPorts: [],
    udpPorts: [],
    httpEndpoints: [],
    runPerformance: true,
    runSpeedtest: true,
    runIperf: true,
    runDiscovery: true,
    speedtest: {
      serverId: '',
      autoRunOnLink: false,
    },
    iperf: {
      autoRunOnLink: false,
    },
  });

  // FAB Options, Display Options, and iperf Settings now come from SettingsContext above

  const [wifiSettings, setWifiSettings] = useState<WiFiSettingsType>({
    interface: '',
    availableWifi: [],
    isWireless: false,
  });
  // Link settings (speed/duplex) - #734
  const [linkSettings, setLinkSettings] = useState<LinkSettingsType>(INLINE_DEFAULT_LINK_SETTINGS);
  // Cable test settings - #734, #740
  const [cableTestSettings, setCableTestSettings] = useState<CableTestSettingsType>(
    INLINE_DEFAULT_CABLE_TEST_SETTINGS,
  );
  const [dnsInput, setDnsInput] = useState('');
  const [iperfSuggestions, setIperfSuggestions] = useState<IperfSuggestion[]>([]);
  const [iperfSuggestionsStatus, setIperfSuggestionsStatus] = useState<
    'idle' | 'loading' | 'error'
  >('idle');
  const [iperfSuggestionsError, setIperfSuggestionsError] = useState<string | null>(null);
  // Network Discovery settings
  const [networkDiscoverySettings, setNetworkDiscoverySettings] =
    useState<NetworkDiscoverySettings>({
      enabled: true,
      arpScanWorkers: 50,
      pingTimeoutMs: 500,
      scanTimeoutMs: 30000,
      autoScan: false,
      scanIntervalMs: 0,
      ipv6Enabled: true,
      options: {
        passiveProtocols: {
          lldp: true,
          cdp: true,
          edp: true,
          ndp: true,
        },
        arpScan: true,
        icmpScan: true,
        portScan: {
          enabled: false,
          preset: 'common',
          tcpPorts: '22,80,443,8080-8100',
          udpPorts: '53,123,161',
          bannerTimeoutMs: 2000,
        },
        tcpProbe: {
          timeoutMs: 2000,
          workers: 20,
        },
        traceroute: false,
        snmpQuery: false,
      },
      timing: {
        probeIntervalMs: 75,
        rescanIntervalMs: 600000,
        workers: 50,
      },
      profiler: {
        enabled: true,
        timeoutMs: 2000,
        maxConcurrent: 5,
        quickPorts: [22, 80, 443, 8080],
      },
      fingerprinting: {
        enabled: false,
        osDetection: false,
        serviceProbes: false,
      },
    });
  // SNMP settings
  const [snmpSettings, setSnmpSettings] = useState<SnmpSettingsType>({
    communities: ['public'],
    v3Credentials: [],
    timeout: 5000,
    retries: 2,
    port: 161,
  });
  // Additional subnets for scanning
  const [subnets, setSubnets] = useState<SubnetConfig[]>([]);
  const [newSubnetCidr, setNewSubnetCidr] = useState('');
  const [newSubnetName, setNewSubnetName] = useState('');
  const [subnetError, setSubnetError] = useState<string | null>(null);
  // Log preview (debug)
  const [logPreview, setLogPreview] = useState<string[]>([]);
  const [logLoading, setLogLoading] = useState(false);
  const [logError, setLogError] = useState<string | null>(null);
  const [subnetsStatus, setSubnetsStatus] = useState<SaveStatus>('idle');
  // Auto-save status for each section
  type SaveStatus = 'idle' | 'saving' | 'saved' | 'error';
  const [thresholdsStatus, setThresholdsStatus] = useState<SaveStatus>('idle');
  const [testsStatus, setTestsStatus] = useState<SaveStatus>('idle');
  const [wifiStatus, setWifiStatus] = useState<SaveStatus>('idle');
  const [linkStatus, setLinkStatus] = useState<SaveStatus>('idle');
  const [cableTestStatus, setCableTestStatus] = useState<SaveStatus>('idle');
  const [snmpStatus, setSnmpStatus] = useState<SaveStatus>('idle');
  const [vulnSettings, setVulnSettings] = useState<VulnerabilityScanSettings>(
    INLINE_DEFAULT_VULNERABILITY_SETTINGS,
  );
  const [vulnStatus, setVulnStatus] = useState<SaveStatus>('idle');
  // Status for display, iperf comes from context (settingsStatus)
  const displayStatus = settingsStatus.display;
  const iperfStatus = settingsStatus.iperf;

  const [networkDiscoveryStatus, setNetworkDiscoveryStatus] = useState<SaveStatus>('idle');

  // Refs to track initial load (skip auto-save on first load)
  const initialLoadRef = useRef(true);
  const thresholdsInitRef = useRef(true);
  const testsInitRef = useRef(true);
  const wifiInitRef = useRef(true);
  const linkInitRef = useRef(true);
  const cableTestInitRef = useRef(true);
  const networkDiscoveryInitRef = useRef(true);
  const snmpInitRef = useRef(true);
  const vulnInitRef = useRef(true);

  // Debounce timers (fab, display, iperf now handled by context)
  const thresholdsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const testsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wifiTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const linkTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const cableTestTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const snmpTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const vulnTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Legacy state (keep for IP settings which still needs manual apply)
  const [savingIp, setSavingIp] = useState(false);
  const [ipMessage, setIpMessage] = useState<string | null>(null);

  // Fetch current thresholds
  const fetchThresholds = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        if (data.thresholds) {
          setThresholds((prev) => ({
            ...prev,
            ...data.thresholds,
          }));
        }
      }
    } catch (err) {
      logger.error(LogComponents.Config, 'Failed to fetch thresholds', err);
    }
  }, []);

  // Fetch current IP settings
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
      logger.error(LogComponents.Config, 'Failed to fetch IP settings', err);
    }
  }, []);

  // Fetch current tests settings
  const fetchTestsSettings = useCallback(async () => {
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
            autoRunOnLink: data.speedtest?.autoRunOnLink ?? true, // Default to true
          },
          iperf: {
            autoRunOnLink: data.iperf?.autoRunOnLink,
          },
        });
      }
    } catch (err) {
      logger.error(LogComponents.Config, 'Failed to fetch tests settings', err);
    }
  }, []);

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
  }, []);

  // Fetch WiFi settings
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
  }, []);

  // FAB options, display options, and iperf settings now come from SettingsContext
  // (loaded automatically by the context provider)

  // Fetch Network Discovery settings from API
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
            passiveProtocols: {
              lldp: true,
              cdp: true,
              edp: true,
              ndp: true,
            },
            arpScan: true,
            icmpScan: true,
            portScan: {
              enabled: false,
              preset: 'common',
              tcpPorts: '22,80,443,8080-8100',
              udpPorts: '53,123,161',
              bannerTimeoutMs: 2000,
            },
            tcpProbe: {
              timeoutMs: 2000,
              workers: 20,
            },
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
  }, []);

  // Fetch SNMP settings from API
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
      logger.error(LogComponents.Config, 'Failed to fetch SNMP settings', err);
    }
  }, []);

  // Fetch link settings from API (fixes #734)
  const fetchLinkSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/settings/link`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        // Backend may send separate speed/duplex or combined mode
        const mode = data.mode ?? (data.auto_negotiation ? 'auto' : `${data.speed}/${data.duplex}`);
        setLinkSettings({
          mode: mode,
          availableModes: data.available_modes ?? [],
        });
      }
    } catch (err) {
      logger.error(LogComponents.Config, 'Failed to fetch link settings', err);
    }
  }, []);

  // Fetch cable test settings from API (fixes #740)
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
      logger.error(LogComponents.Config, 'Failed to fetch cable test settings', err);
    }
  }, []);

  // Fetch configured subnets from API
  const fetchSubnets = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/subnets`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setSubnets(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      logger.error(LogComponents.Discovery, 'Failed to fetch subnets', err);
    }
  }, []);

  // Fetch a small tail of the application log (debug)
  // Security fix #301: Removed VITE_LOG_ACCESS_TOKEN - JWT authentication is sufficient
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
  }, []);

  // Fetch subnets when drawer opens
  useEffect(() => {
    if (isOpen) {
      fetchSubnets().catch(() => undefined);
    }
  }, [isOpen, fetchSubnets]);

  // Add a new subnet
  const addSubnet = async (): Promise<void> => {
    if (!newSubnetCidr.trim()) {
      setSubnetError(t('network.cidrRequired'));
      return;
    }

    setSubnetError(null);
    setSubnetsStatus('saving');

    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/subnets`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          cidr: newSubnetCidr.trim(),
          name: newSubnetName.trim() || newSubnetCidr.trim(),
          enabled: true,
        }),
      });

      if (response.ok) {
        setNewSubnetCidr('');
        setNewSubnetName('');
        setSubnetsStatus('saved');
        setTimeout(() => setSubnetsStatus('idle'), 2000);
        await fetchSubnets();
      } else {
        // Handle both JSON and plain text error responses
        const contentType = response.headers.get('content-type');
        if (contentType?.includes('application/json')) {
          const errorData = await (response.json() as Promise<{ error?: string }>);
          setSubnetError(errorData.error || 'Failed to add subnet');
        } else {
          const errorText = await (response.text() as Promise<string>);
          setSubnetError(errorText || 'Failed to add subnet');
        }
        setSubnetsStatus('error');
      }
    } catch (err) {
      setSubnetError(err instanceof Error ? err.message : 'Network error adding subnet');
      setSubnetsStatus('error');
    }
  };

  // Toggle subnet enabled state
  const toggleSubnet = async (cidr: string, enabled: boolean): Promise<void> => {
    setSubnetsStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/subnets`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ cidr, enabled }),
      });

      if (response.ok) {
        setSubnetsStatus('saved');
        setTimeout(() => setSubnetsStatus('idle'), 2000);
        await fetchSubnets();
      } else {
        setSubnetsStatus('error');
      }
    } catch {
      setSubnetsStatus('error');
    }
  };

  // Delete a subnet
  const deleteSubnet = async (cidr: string): Promise<void> => {
    setSubnetsStatus('saving');
    try {
      // Backend expects CIDR as query parameter, not in body
      const response = await fetch(
        `${API_BASE}/api/v1/shell/devices/subnets?cidr=${encodeURIComponent(cidr)}`,
        {
          method: 'DELETE',
          credentials: 'include',
        },
      );

      if (response.ok) {
        setSubnetsStatus('saved');
        setTimeout(() => setSubnetsStatus('idle'), 2000);
        await fetchSubnets();
      } else {
        setSubnetsStatus('error');
      }
    } catch {
      setSubnetsStatus('error');
    }
  };

  // Save Network Discovery settings to API
  const saveNetworkDiscoverySettings = useCallback(async () => {
    setNetworkDiscoveryStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(networkDiscoverySettings),
      });
      if (response.ok) {
        setNetworkDiscoveryStatus('saved');
        setTimeout(() => setNetworkDiscoveryStatus('idle'), 2000);
      } else {
        setNetworkDiscoveryStatus('error');
      }
    } catch {
      setNetworkDiscoveryStatus('error');
    }
  }, [networkDiscoverySettings]);

  const saveSnmpSettings = useCallback(async () => {
    setSnmpStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/sap/snmp/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(snmpSettings),
      });
      if (response.ok) {
        setSnmpStatus('saved');
        setTimeout(() => setSnmpStatus('idle'), 2000);
      } else {
        setSnmpStatus('error');
      }
    } catch {
      setSnmpStatus('error');
    }
  }, [snmpSettings]);

  // Fetch vulnerability scanner settings from API
  const fetchVulnSettings = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/vulnerabilities/settings`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setVulnSettings({
          enabled: data.enabled ?? false,
          cveDatabase: data.cve_database ?? data.cveDatabase ?? 'nvd',
          nvdApiKey: data.nvd_api_key ?? data.nvdApiKey ?? '',
          updateInterval: data.update_interval ?? data.updateInterval ?? 86400,
          severityThreshold: data.severity_threshold ?? data.severityThreshold ?? 'medium',
          maxConcurrent: data.max_concurrent ?? data.maxConcurrent ?? 5,
          autoScan: data.auto_scan ?? data.autoScan ?? false,
        });
      }
    } catch (err) {
      logger.error(LogComponents.Config, 'Failed to fetch vulnerability settings', err);
    }
  }, []);

  const saveVulnSettings = useCallback(async () => {
    setVulnStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/vulnerabilities/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          enabled: vulnSettings.enabled,
          cveDatabase: vulnSettings.cveDatabase,
          nvdApiKey: vulnSettings.nvdApiKey,
          updateInterval: vulnSettings.updateInterval,
          severityThreshold: vulnSettings.severityThreshold,
          maxConcurrent: vulnSettings.maxConcurrent,
          autoScan: vulnSettings.autoScan,
        }),
      });
      if (response.ok) {
        setVulnStatus('saved');
        setTimeout(() => setVulnStatus('idle'), 2000);
      } else {
        setVulnStatus('error');
      }
    } catch {
      setVulnStatus('error');
    }
  }, [vulnSettings]);

  useEffect(() => {
    if (isOpen) {
      // Reset init refs on open
      initialLoadRef.current = true;
      thresholdsInitRef.current = true;
      testsInitRef.current = true;
      wifiInitRef.current = true;
      linkInitRef.current = true;
      cableTestInitRef.current = true;
      networkDiscoveryInitRef.current = true;
      snmpInitRef.current = true;
      vulnInitRef.current = true;

      fetchThresholds().catch(() => undefined);
      fetchIpSettings().catch(() => undefined);
      fetchTestsSettings().catch(() => undefined);
      fetchWifiSettings().catch(() => undefined);
      // FAB options, display options, and iperf settings come from SettingsContext
      fetchNetworkDiscoverySettings().catch(() => undefined);
      fetchSnmpSettings().catch(() => undefined);
      fetchVulnSettings().catch(() => undefined);
      fetchLinkSettings().catch(() => undefined);
      fetchCableTestSettings().catch(() => undefined);
      fetchSubnets().catch(() => undefined);

      // Mark initial load as done after a short delay
      setTimeout(() => {
        initialLoadRef.current = false;
        thresholdsInitRef.current = false;
        testsInitRef.current = false;
        wifiInitRef.current = false;
        linkInitRef.current = false;
        cableTestInitRef.current = false;
        networkDiscoveryInitRef.current = false;
        snmpInitRef.current = false;
        vulnInitRef.current = false;
      }, 500);
    }
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

  const saveThresholds = useCallback(async () => {
    setThresholdsStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ thresholds }),
      });
      if (response.ok) {
        setThresholdsStatus('saved');
        setTimeout(() => setThresholdsStatus('idle'), 2000);
      } else {
        setThresholdsStatus('error');
      }
    } catch {
      setThresholdsStatus('error');
    }
  }, [thresholds]);

  const saveIpSettings = async (): Promise<void> => {
    setSavingIp(true);
    setIpMessage(null);
    try {
      // Parse DNS from input
      const dns = dnsInput
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s.length > 0);

      const response = await fetch(`${API_BASE}/api/ipconfig/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          mode: ipSettings.mode,
          address: ipSettings.address,
          netmask: ipSettings.netmask,
          gateway: ipSettings.gateway,
          dns,
        }),
      });
      if (response.ok) {
        setIpMessage('IP settings applied');
        setTimeout(() => setIpMessage(null), 3000);
      } else {
        const error = await (response.text() as Promise<string>);
        setIpMessage(`Failed: ${error}`);
      }
    } catch {
      setIpMessage('Error applying IP settings');
    } finally {
      setSavingIp(false);
    }
  };

  const saveTestsSettings = useCallback(async () => {
    setTestsStatus('saving');
    try {
      const payload = normalizeTestsSettingsForSave(testsSettings);
      const response = await fetch(`${API_BASE}/api/health-checks/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(payload),
      });
      if (response.ok) {
        setTestsStatus('saved');
        setTimeout(() => setTestsStatus('idle'), 2000);
        // Mark that test settings changed - event dispatched on drawer close
        testsSettingsChangedRef.current = true;
      } else {
        setTestsStatus('error');
      }
    } catch {
      setTestsStatus('error');
    }
  }, [testsSettings]);

  const saveWifiSettings = useCallback(async () => {
    setWifiStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/canopy/wifi/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ interface: wifiSettings.interface }),
      });
      if (response.ok) {
        setWifiStatus('saved');
        setTimeout(() => setWifiStatus('idle'), 2000);
      } else {
        setWifiStatus('error');
      }
    } catch {
      setWifiStatus('error');
    }
  }, [wifiSettings.interface]);

  // Save link settings to backend (fixes #734)
  const saveLinkSettings = useCallback(async () => {
    setLinkStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/settings/link`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          mode: linkSettings.mode,
          availableModes: linkSettings.availableModes,
        }),
      });
      if (response.ok) {
        setLinkStatus('saved');
        setTimeout(() => setLinkStatus('idle'), 2000);
      } else {
        setLinkStatus('error');
      }
    } catch {
      setLinkStatus('error');
    }
  }, [linkSettings]);

  // Save cable test settings to backend (fixes #740)
  const saveCableTestSettings = useCallback(async () => {
    setCableTestStatus('saving');
    try {
      const response = await fetch(`${API_BASE}/api/settings/cable`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          enabled: cableTestSettings.enabled,
        }),
      });
      if (response.ok) {
        setCableTestStatus('saved');
        setTimeout(() => setCableTestStatus('idle'), 2000);
      } else {
        setCableTestStatus('error');
      }
    } catch {
      setCableTestStatus('error');
    }
  }, [cableTestSettings]);

  // Auto-save thresholds with debounce
  useEffect(() => {
    if (thresholdsInitRef.current) {
      return;
    }
    if (thresholdsTimerRef.current) {
      clearTimeout(thresholdsTimerRef.current);
    }
    thresholdsTimerRef.current = setTimeout(() => {
      saveThresholds().catch(() => undefined);
    }, 800);
    return (): void => {
      if (thresholdsTimerRef.current) {
        clearTimeout(thresholdsTimerRef.current);
      }
    };
  }, [saveThresholds]);

  // Auto-save tests settings with debounce
  useEffect(() => {
    if (testsInitRef.current) {
      return;
    }
    if (testsTimerRef.current) {
      clearTimeout(testsTimerRef.current);
    }
    testsTimerRef.current = setTimeout(() => {
      saveTestsSettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (testsTimerRef.current) {
        clearTimeout(testsTimerRef.current);
      }
    };
  }, [saveTestsSettings]);

  // Auto-save wifi settings with debounce
  useEffect(() => {
    if (wifiInitRef.current) {
      return;
    }
    if (wifiTimerRef.current) {
      clearTimeout(wifiTimerRef.current);
    }
    wifiTimerRef.current = setTimeout(() => {
      saveWifiSettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (wifiTimerRef.current) {
        clearTimeout(wifiTimerRef.current);
      }
    };
  }, [saveWifiSettings]);

  // Auto-save link settings with debounce (fixes #734)
  useEffect(() => {
    if (linkInitRef.current) {
      return;
    }
    if (linkTimerRef.current) {
      clearTimeout(linkTimerRef.current);
    }
    linkTimerRef.current = setTimeout(() => {
      saveLinkSettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (linkTimerRef.current) {
        clearTimeout(linkTimerRef.current);
      }
    };
  }, [saveLinkSettings]);

  // Auto-save cable test settings with debounce (fixes #740)
  useEffect(() => {
    if (cableTestInitRef.current) {
      return;
    }
    if (cableTestTimerRef.current) {
      clearTimeout(cableTestTimerRef.current);
    }
    cableTestTimerRef.current = setTimeout(() => {
      saveCableTestSettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (cableTestTimerRef.current) {
        clearTimeout(cableTestTimerRef.current);
      }
    };
  }, [saveCableTestSettings]);

  // Display options and iperf settings auto-save is handled by SettingsContext

  // Auto-save Network Discovery settings with debounce
  useEffect(() => {
    if (networkDiscoveryInitRef.current) {
      return;
    }
    if (networkDiscoveryTimerRef.current) {
      clearTimeout(networkDiscoveryTimerRef.current);
    }
    networkDiscoveryTimerRef.current = setTimeout(() => {
      saveNetworkDiscoverySettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (networkDiscoveryTimerRef.current) {
        clearTimeout(networkDiscoveryTimerRef.current);
      }
    };
  }, [saveNetworkDiscoverySettings]);

  // Auto-save SNMP settings with debounce
  useEffect(() => {
    if (snmpInitRef.current) {
      return;
    }
    if (snmpTimerRef.current) {
      clearTimeout(snmpTimerRef.current);
    }
    snmpTimerRef.current = setTimeout(() => {
      saveSnmpSettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (snmpTimerRef.current) {
        clearTimeout(snmpTimerRef.current);
      }
    };
  }, [saveSnmpSettings]);

  // Auto-save vulnerability settings with debounce
  useEffect(() => {
    if (vulnInitRef.current) {
      return;
    }
    if (vulnTimerRef.current) {
      clearTimeout(vulnTimerRef.current);
    }
    vulnTimerRef.current = setTimeout(() => {
      saveVulnSettings().catch(() => undefined);
    }, 800);
    return (): void => {
      if (vulnTimerRef.current) {
        clearTimeout(vulnTimerRef.current);
      }
    };
  }, [saveVulnSettings]);

  // Fixes #917: Master cleanup effect for all timer refs on unmount
  // Individual useEffects clean up on re-render, but this ensures cleanup on unmount
  useEffect(
    (): (() => void) => (): void => {
      if (thresholdsTimerRef.current) {
        clearTimeout(thresholdsTimerRef.current);
      }
      if (testsTimerRef.current) {
        clearTimeout(testsTimerRef.current);
      }
      if (wifiTimerRef.current) {
        clearTimeout(wifiTimerRef.current);
      }
      if (linkTimerRef.current) {
        clearTimeout(linkTimerRef.current);
      }
      if (cableTestTimerRef.current) {
        clearTimeout(cableTestTimerRef.current);
      }
      if (networkDiscoveryTimerRef.current) {
        clearTimeout(networkDiscoveryTimerRef.current);
      }
      if (snmpTimerRef.current) {
        clearTimeout(snmpTimerRef.current);
      }
      if (vulnTimerRef.current) {
        clearTimeout(vulnTimerRef.current);
      }
    },
    [],
  );

  // Validate IP address format
  const isValidIp = (ip: string): boolean => {
    if (!ip) {
      return true; // Empty is OK for optional fields
    }
    const parts = ip.split('.');
    if (parts.length !== 4) {
      return false;
    }
    return parts.every((p) => {
      const n = Number.parseInt(p, 10);
      return !Number.isNaN(n) && n >= 0 && n <= 255 && p === String(n);
    });
  };

  const drawerRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Handle ESC key to close drawer
  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handleKeyDown = (e: globalThis.KeyboardEvent): void => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    // Focus the close button when drawer opens
    // Fixes #918: Track timeout for cleanup to prevent stale closure
    const focusTimeout = setTimeout(() => closeButtonRef.current?.focus(), 100);

    return (): void => {
      document.removeEventListener('keydown', handleKeyDown);
      clearTimeout(focusTimeout);
    };
  }, [isOpen, onClose]);

  if (!isOpen) {
    return null;
  }

  return (
    <>
      {/* Backdrop */}
      <div
        class="fixed inset-0 bg-black/50 backdrop-blur-sm z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer - full width on mobile, 384px on larger screens */}
      <div
        ref={drawerRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-drawer-title"
        onClick={(e: React.MouseEvent): void => e.stopPropagation()}
        onKeyDown={(e: React.KeyboardEvent): void => e.stopPropagation()}
        class="fixed right-0 top-0 h-full w-full sm:w-96 lg:w-lg bg-surface-raised border-l border-surface-border z-50 overflow-y-auto shadow-xl"
      >
        {/* Header */}
        <div
          class={cn(
            layout.flex.between,
            'pad sm:pad-lg border-b border-surface-border sticky top-0 bg-surface-raised z-10',
          )}
        >
          <div class="stack-xs">
            <h2 id="settings-drawer-title" class="heading-3">
              {t('title')}
            </h2>
            <p class="body-small">{t('subtitle')}</p>
          </div>
          <button
            type="button"
            ref={closeButtonRef}
            onClick={onClose}
            class={cn(
              button.size.md,
              radius.md,
              'hover:bg-surface-hover active:bg-surface-hover text-text-muted touch-manipulation focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-raised',
            )}
            aria-label={t('network.closeSettings')}
          >
            <svg
              class={iconTokens.size.lg}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <div
          class={cn(spacing.drawerPad, 'section-gap body-small leading-relaxed')}
          ref={scrollRef}
        >
          {/* Settings sections ordered to match dashboard card order */}
          {/* Link Settings - always visible for ethernet interface config */}
          <LinkSettings
            linkSettings={linkSettings}
            setLinkSettings={setLinkSettings}
            linkStatus={linkStatus}
            cardSettings={cardSettings}
            updateCardSettings={updateCardSettings}
          />

          {/* Cable Test Settings - always visible for cable diagnostics */}
          <CableTestSettings
            cableTestSettings={cableTestSettings}
            setCableTestSettings={setCableTestSettings}
            cableTestStatus={cableTestStatus}
          />

          {/* Network Section - IP/DHCP config (third) */}
          <CollapsibleSection
            title={
              <div class={layout.inline.default}>
                <Network class={iconTokens.size.sm} />
                <span>{t('sections.network')}</span>
              </div>
            }
          >
            {/* Network Configuration */}
            <div class="stack">
              <p class="section-title">{t('network.title')}</p>
              {/* Mode Toggle */}
              <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
                <button
                  type="button"
                  onClick={(): void => setIpSettings((prev) => ({ ...prev, mode: 'dhcp' }))}
                  class={cn(
                    spacing.tab,
                    radius.md,
                    'body-small font-medium transition-colors',
                    ipSettings.mode === 'dhcp'
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover',
                  )}
                >
                  {t('network.dhcp')}
                </button>
                <button
                  type="button"
                  onClick={(): void => setIpSettings((prev) => ({ ...prev, mode: 'static' }))}
                  class={cn(
                    spacing.tab,
                    radius.md,
                    'body-small font-medium transition-colors',
                    ipSettings.mode === 'static'
                      ? 'bg-brand-primary text-text-inverse'
                      : 'bg-surface-base border border-surface-border text-text-primary hover:bg-surface-hover',
                  )}
                >
                  {t('network.static')}
                </button>
              </div>

              {/* Static IP Fields */}
              {ipSettings.mode === 'static' && (
                <div
                  class={cn('stack', spacing.padding.top.heading, 'border-t border-surface-border')}
                >
                  <div>
                    <label for="static-ip-address" class="caption font-medium">
                      {t('network.ipAddress')} *
                    </label>
                    <input
                      id="static-ip-address"
                      type="text"
                      value={ipSettings.address}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void =>
                        setIpSettings((prev) => ({
                          ...prev,
                          address: e.target.value,
                        }))
                      }
                      placeholder="192.168.1.100"
                      class={cn(
                        'w-full',
                        spacing.margin.top.tight,
                        spacing.chip.sm,
                        'bg-surface-base border',
                        radius.md,
                        'body-small text-text-primary',
                        ipSettings.address && !isValidIp(ipSettings.address)
                          ? 'border-status-error'
                          : 'border-surface-border',
                      )}
                    />
                  </div>
                  <div>
                    <label for="static-subnet-mask" class="caption font-medium">
                      {t('network.subnetMask')} *
                    </label>
                    <input
                      id="static-subnet-mask"
                      type="text"
                      value={ipSettings.netmask}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void =>
                        setIpSettings((prev) => ({
                          ...prev,
                          netmask: e.target.value,
                        }))
                      }
                      placeholder="24 or 255.255.255.0"
                      class={cn(
                        'w-full',
                        spacing.margin.top.tight,
                        spacing.chip.lg,
                        'bg-surface-base border border-surface-border',
                        radius.md,
                        'body-small text-text-primary',
                      )}
                    />
                  </div>
                  <div>
                    <label for="static-gateway" class="caption font-medium">
                      {t('network.gateway')}
                    </label>
                    <input
                      id="static-gateway"
                      type="text"
                      value={ipSettings.gateway}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void =>
                        setIpSettings((prev) => ({
                          ...prev,
                          gateway: e.target.value,
                        }))
                      }
                      placeholder="192.168.1.1"
                      class={cn(
                        'w-full',
                        spacing.margin.top.tight,
                        spacing.chip.sm,
                        'bg-surface-base border',
                        radius.md,
                        'body-small text-text-primary',
                        ipSettings.gateway && !isValidIp(ipSettings.gateway)
                          ? 'border-status-error'
                          : 'border-surface-border',
                      )}
                    />
                  </div>
                  <div>
                    <label for="static-dns-servers" class="caption font-medium">
                      {t('network.dnsServers')}
                    </label>
                    <input
                      id="static-dns-servers"
                      type="text"
                      value={dnsInput}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void => setDnsInput(e.target.value)}
                      placeholder="8.8.8.8, 8.8.4.4"
                      class={cn(
                        'w-full',
                        spacing.margin.top.tight,
                        spacing.chip.lg,
                        'bg-surface-base border border-surface-border',
                        radius.md,
                        'body-small text-text-primary',
                      )}
                    />
                  </div>
                </div>
              )}

              {/* Apply Button */}
              <button
                type="button"
                onClick={saveIpSettings}
                disabled={savingIp || (ipSettings.mode === 'static' && !ipSettings.address)}
                class={cn(
                  'w-full',
                  button.size.md,
                  'bg-brand-primary text-text-inverse',
                  radius.md,
                  'font-medium hover:bg-brand-accent disabled:opacity-50 transition-colors',
                )}
              >
                {savingIp ? t('network.applying') : t('network.applyIpSettings')}
              </button>

              {ipMessage ? (
                <p
                  class={cn(
                    'caption text-center',
                    ipMessage.includes('Failed') || ipMessage.includes('Error')
                      ? 'text-status-error'
                      : 'text-status-success',
                  )}
                >
                  {ipMessage}
                </p>
              ) : null}

              <p class="caption">{t('network.requiresRoot')}</p>
            </div>

            {/* Display Options */}
            <div
              class={cn(
                'border-t border-surface-border',
                spacing.padding.top.heading,
                spacing.margin.top.heading,
              )}
            >
              <p class={cn('caption font-medium', spacing.margin.bottom.inline)}>
                {t('network.displayOptions')} <AutoSaveIndicator status={displayStatus} />
              </p>
              <div class="stack-sm">
                {/* Show Public IP */}
                <label
                  class={cn(
                    'flex items-center justify-between',
                    spacing.pad.xs,
                    'bg-surface-base',
                    radius.md,
                    'border border-surface-border',
                  )}
                >
                  <div>
                    <span class="body-small text-text-primary font-medium">
                      {t('network.showPublicIp')}
                    </span>
                    <p class="caption text-text-muted">{t('network.displayInNetworkCard')}</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={displayOptions.showPublicIp}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      setDisplayOptions((prev) => ({
                        ...prev,
                        showPublicIp: e.target.checked,
                      }))
                    }
                    class={iconTokens.size.sm}
                  />
                </label>
              </div>
            </div>

            {/* VLAN Configuration */}
            <div
              class={cn(
                'border-t border-surface-border',
                spacing.padding.top.heading,
                spacing.margin.top.heading,
              )}
            >
              <p class={cn('section-title', spacing.margin.bottom.inline)}>
                {t('network.vlanTag')}
              </p>
              <VlanControl />
            </div>

            {/* MTU Configuration */}
            <div
              class={cn(
                'border-t border-surface-border',
                spacing.padding.top.heading,
                spacing.margin.top.heading,
              )}
            >
              <p class={cn('section-title', spacing.margin.bottom.inline)}>
                {t('network.mtuSetting')}
              </p>
              <MtuControl />
            </div>
          </CollapsibleSection>

          {/* WiFi Settings - only shown in WiFi mode (#754) */}
          {isWifi ? (
            <WiFiSettings
              wifiSettings={wifiSettings}
              setWifiSettings={setWifiSettings}
              wifiStatus={wifiStatus}
            />
          ) : null}

          {/* DNS Settings - matches DnsCard position */}
          <DnsSettings
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
            testsStatus={testsStatus}
            cardSettings={cardSettings}
            updateCardSettings={updateCardSettings}
          />

          <HealthChecksSettings
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
            testsStatus={testsStatus}
            cardSettings={cardSettings}
            updateCardSettings={updateCardSettings}
          />

          <PerformanceSettings
            testsSettings={testsSettings}
            setTestsSettings={setTestsSettings}
            iperfSettings={iperfSettings}
            setIperfSettings={setIperfSettings}
            iperfStatus={iperfStatus}
            iperfSuggestions={iperfSuggestions}
            iperfSuggestionsStatus={iperfSuggestionsStatus}
            iperfSuggestionsError={iperfSuggestionsError}
            fetchIperfSuggestions={fetchIperfSuggestions}
            cardSettings={cardSettings}
            updateCardSettings={updateCardSettings}
          />

          <DiscoverySettings
            networkDiscoverySettings={networkDiscoverySettings}
            setNetworkDiscoverySettings={setNetworkDiscoverySettings}
            networkDiscoveryStatus={networkDiscoveryStatus}
            subnets={subnets}
            subnetsStatus={subnetsStatus}
            newSubnetCidr={newSubnetCidr}
            setNewSubnetCidr={setNewSubnetCidr}
            newSubnetName={newSubnetName}
            setNewSubnetName={setNewSubnetName}
            subnetError={subnetError}
            setSubnetError={setSubnetError}
            addSubnet={addSubnet}
            toggleSubnet={toggleSubnet}
            deleteSubnet={deleteSubnet}
            snmpSettings={snmpSettings}
            setSnmpSettings={setSnmpSettings}
            snmpStatus={snmpStatus}
            cardSettings={cardSettings}
            updateCardSettings={updateCardSettings}
          />

          <VulnerabilitySettings
            settings={vulnSettings}
            setSettings={setVulnSettings}
            status={vulnStatus}
          />

          <ThresholdsSettings
            thresholds={thresholds}
            setThresholds={setThresholds}
            thresholdsStatus={thresholdsStatus}
          />

          {/* Appearance Section */}
          <AppearanceSettings
            theme={theme}
            setTheme={setTheme}
            isDark={isDark}
            unitSystem={displayOptions.unitSystem || 'sae'}
            setUnitSystem={(unit: 'sae' | 'metric'): void =>
              setDisplayOptions((prev) => ({ ...prev, unitSystem: unit }))
            }
          />

          {/* Config Backups Section (implements #494) */}
          <ConfigBackupsSection />

          {/* Updates Section (implements #862) */}
          <UpdateSettings currentVersion={version} />

          {/* Logs (debug) */}
          <section class={cn(spacing.padding.top.section, 'border-t border-surface-border')}>
            <div class="flex items-start justify-between">
              <div>
                <h3 class="body-small font-medium text-text-muted">{t('logs.title')}</h3>
                <p class="caption text-text-muted">{t('logs.description')}</p>
              </div>
              <button
                type="button"
                onClick={fetchLogPreview}
                class={cn(
                  'caption',
                  spacing.chip.sm,
                  'border border-surface-border',
                  radius.md,
                  'text-text-muted hover:text-text-primary hover:border-text-muted transition-colors',
                )}
              >
                {logLoading ? t('logs.loading') : t('logs.view')}
              </button>
            </div>
            {logError ? (
              <p class={cn('caption text-status-error', spacing.margin.top.inline)}>{logError}</p>
            ) : null}
            {!logError && logPreview.length > 0 ? (
              <pre
                class={cn(
                  spacing.margin.top.inline,
                  'max-h-48 overflow-y-auto text-2xs leading-5 bg-surface-base border border-surface-border',
                  radius.md,
                  spacing.chip.lg,
                  'text-text-primary whitespace-pre-wrap',
                )}
              >
                {logPreview.join('\n')}
              </pre>
            ) : null}
          </section>

          {/* Export Section */}
          <section class={cn(spacing.padding.top.section, 'border-t border-surface-border')}>
            <h3 class={cn('body-small font-medium text-text-muted', spacing.margin.bottom.heading)}>
              {t('export.title')}
            </h3>
            <a
              href={`${API_BASE}/api/harvest/export`}
              download="seed-export.json"
              class={cn(
                'w-full',
                button.size.md,
                'bg-surface-base border border-surface-border text-text-primary',
                radius.md,
                'font-medium hover:bg-surface-hover transition-colors flex items-center justify-center',
                spacing.gap.compact,
                'touch-manipulation',
              )}
            >
              <svg
                class={iconTokens.size.sm}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              {t('export.download')}
            </a>
            <p class={cn('caption text-text-muted', spacing.margin.top.inline)}>
              {t('export.description')}
            </p>
          </section>

          {/* About Section */}
          <section class={cn(spacing.padding.top.section, 'border-t border-surface-border')}>
            <h3 class={cn('body-small font-medium text-text-muted', spacing.margin.bottom.inline)}>
              {t('about.title')}
            </h3>
            <p class="caption text-text-muted">
              {t('about.appName')} {version}
              <br />
              {t('about.description')}
            </p>
          </section>
        </div>
      </div>
    </>
  );
});
