// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * SettingsDrawer Component
 *
 * Purpose: Master settings configuration panel managing all application settings.
 * Renders the sections as SettingsTabs inside a RightDrawer (General, Network, Tests,
 * Discovery, Alerts, Access, About) with persistent storage.
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

import { Bell, Gauge, Info, KeyRound, Network, Radar, SlidersHorizontal } from 'lucide-react';
import type React from 'react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import { useRole } from '../../contexts/RoleContext';
import { useSettings } from '../../contexts/useSettings';
import { useDebouncedAutoSave } from '../../hooks/useDebouncedAutoSave';
import { useSettingsDrawerLoaders } from '../../hooks/useSettingsDrawerLoaders';
import { useSettingsDrawerSavers } from '../../hooks/useSettingsDrawerSavers';
import { useSubnetSettings } from '../../hooks/useSubnetSettings';
import { useTheme } from '../../hooks/useTheme';
import { useVulnerabilitySettings } from '../../hooks/useVulnerabilitySettings';
import { RightDrawer } from '../../shared/RightDrawer';
import { SettingsTabs } from '../../shared/SettingsTabs';
import type { IperfSuggestion } from '../../types/generated/iperf-suggestion';
import type { ThresholdSettings } from '../../types/generated/settings-response';
import type {
  CableTestSettings as CableTestSettingsType,
  IpSettings,
  LinkSettings as LinkSettingsType,
  NetworkDiscoverySettings,
  SnmpSettings as SnmpSettingsType,
  TestsSettings,
  WiFiSettings as WiFiSettingsType,
} from '../../types/settings';
import { EMPTY_PROTOCOL_ENDPOINTS } from '../../types/settings';
import { RequireAdmin } from '../ui/RequireRole';
import { SettingsDrawerFooter } from './SettingsDrawerFooter';
import { SettingsDrawerNetworkSection } from './SettingsDrawerNetworkSection';
import { AlertDeliverySettings } from './sections/AlertDeliverySettings';
import { AlertEmailSettings } from './sections/AlertEmailSettings';
import { AlertEscalationSettings } from './sections/AlertEscalationSettings';
import { ApiTokensSettings } from './sections/ApiTokensSettings';
import { AppearanceSettings } from './sections/AppearanceSettings';
import { CableTestSettings } from './sections/CableTestSettings';
import { ConfigBackupsSection } from './sections/ConfigBackupsSection';
import { DeviceIdentitySettings } from './sections/DeviceIdentitySettings';
import { DiscoverySettings } from './sections/DiscoverySettings';
import { DnsSettings } from './sections/DnsSettings';
import { EoxFeedSettings } from './sections/EoxFeedSettings';
import { GuestNetworkAuditSettings } from './sections/GuestNetworkAuditSettings';
import { HealthChecksSettings } from './sections/HealthChecksSettings';
import { InterfacesSettings } from './sections/InterfacesSettings';
import { LinkSettings } from './sections/LinkSettings';
import { PerformanceSettings } from './sections/PerformanceSettings';
import { SsoSettings } from './sections/SsoSettings';
import { ThresholdsSettings } from './sections/ThresholdsSettings';
import { UsersSettings } from './sections/UsersSettings';
import { VulnerabilitySettings } from './sections/VulnerabilitySettings';
import { WiFiSettings } from './sections/WiFiSettings';
import {
  INLINE_DEFAULT_CABLE_TEST_SETTINGS,
  INLINE_DEFAULT_LINK_SETTINGS,
} from './settingsDrawerNormalizer';

type SettingsTabId = 'general' | 'network' | 'tests' | 'discovery' | 'alerts' | 'access' | 'about';

const TAB_ICON = 'h-4 w-4';

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  version?: string;
  /** Whether currently viewing WiFi mode (shows WiFi settings instead of Link/Cable). */
  isWifi?: boolean;
}

export function SettingsDrawer({
  isOpen,
  onClose,
  version = 'dev',
  isWifi = false,
}: SettingsDrawerProps): React.ReactElement | null {
  const { t } = useTranslation(['settings', 'errors']);
  const { canWrite } = useRole();
  const { theme, setTheme, isDark } = useTheme();
  const [activeTab, setActiveTab] = useState<SettingsTabId>('general');
  const tabs = [
    { id: 'general', label: t('tabs.general'), icon: <SlidersHorizontal className={TAB_ICON} /> },
    { id: 'network', label: t('tabs.network'), icon: <Network className={TAB_ICON} /> },
    { id: 'tests', label: t('tabs.tests'), icon: <Gauge className={TAB_ICON} /> },
    { id: 'discovery', label: t('tabs.discovery'), icon: <Radar className={TAB_ICON} /> },
    { id: 'alerts', label: t('tabs.alerts'), icon: <Bell className={TAB_ICON} /> },
    { id: 'access', label: t('tabs.access'), icon: <KeyRound className={TAB_ICON} /> },
    { id: 'about', label: t('tabs.about'), icon: <Info className={TAB_ICON} /> },
  ] as const;

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
  const setDisplayOptions = (updater: React.SetStateAction<typeof displayOptions>) => {
    const newValue = typeof updater === 'function' ? updater(displayOptions) : updater;
    updateDisplayOptions(newValue);
  };

  const setIperfSettings = (updater: React.SetStateAction<typeof iperfSettings>) => {
    const newValue = typeof updater === 'function' ? updater(iperfSettings) : updater;
    updateIperfSettings(newValue);
  };

  // Track if health check settings were modified - dispatch event on drawer close
  const testsSettingsChangedRef = useRef(false);
  const prevIsOpenRef = useRef(isOpen);

  // When drawer closes, dispatch healthChecksUpdated if test settings changed
  useEffect(() => {
    if (prevIsOpenRef.current && !isOpen && testsSettingsChangedRef.current) {
      // Drawer just closed and test settings were changed
      window.dispatchEvent(new CustomEvent('healthChecksUpdated'));
      testsSettingsChangedRef.current = false;
    }
    prevIsOpenRef.current = isOpen;
  }, [isOpen]);
  const [thresholds, setThresholds] = useState<ThresholdSettings>({
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
    ...EMPTY_PROTOCOL_ENDPOINTS,
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
      scanTimeoutMs: 30000,
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
        },
        traceroute: false,
        snmpQuery: false,
      },
      timing: {
        rescanIntervalMs: 60000,
      },
      fingerprinting: {
        enabled: false,
        osDetection: false,
        serviceProbes: false,
      },
    });
  // SNMP settings
  const [snmpSettings, setSnmpSettings] = useState<SnmpSettingsType>({
    timeout: 5000,
    retries: 2,
    port: 161,
  });
  // Subnet management lives in its own hook
  const {
    subnets,
    newSubnetCidr,
    setNewSubnetCidr,
    newSubnetName,
    setNewSubnetName,
    subnetError,
    setSubnetError,
    subnetsStatus,
    fetchSubnets,
    addSubnet,
    toggleSubnet,
    deleteSubnet,
  } = useSubnetSettings();
  // Log preview (debug)
  const [logPreview, setLogPreview] = useState<string[]>([]);
  const [logLoading, setLogLoading] = useState(false);
  const [logError, setLogError] = useState<string | null>(null);
  // Auto-save status for each section
  type SaveStatus = 'idle' | 'saving' | 'saved' | 'error';
  const [thresholdsStatus, setThresholdsStatus] = useState<SaveStatus>('idle');
  const [testsStatus, setTestsStatus] = useState<SaveStatus>('idle');
  const [wifiStatus, setWifiStatus] = useState<SaveStatus>('idle');
  const [linkStatus, setLinkStatus] = useState<SaveStatus>('idle');
  const [cableTestStatus, setCableTestStatus] = useState<SaveStatus>('idle');
  const [snmpStatus, setSnmpStatus] = useState<SaveStatus>('idle');
  const { vulnSettings, setVulnSettings, vulnStatus, fetchVulnSettings, saveVulnSettings } =
    useVulnerabilitySettings();
  // Status for display, iperf comes from context (settingsStatus)
  const displayStatus = settingsStatus.display;
  const iperfStatus = settingsStatus.iperf;

  const [networkDiscoveryStatus, setNetworkDiscoveryStatus] = useState<SaveStatus>('idle');

  // Legacy state (keep for IP settings which still needs manual apply)
  const [savingIp, setSavingIp] = useState(false);
  const [ipMessage, setIpMessage] = useState<string | null>(null);

  // Per-section fetch callbacks + open-time orchestration live in their hook
  const { fetchIperfSuggestions, fetchLogPreview } = useSettingsDrawerLoaders({
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
  });

  // Per-section save callbacks live in their own hook
  const {
    saveThresholds,
    saveTestsSettings,
    saveWifiSettings,
    saveLinkSettings,
    saveCableTestSettings,
    saveNetworkDiscoverySettings,
    saveSnmpSettings,
  } = useSettingsDrawerSavers({
    thresholds,
    setThresholdsStatus,
    testsSettings,
    setTestsStatus,
    testsSettingsChangedRef,
    wifiSettings,
    setWifiStatus,
    linkSettings,
    setLinkStatus,
    cableTestSettings,
    setCableTestStatus,
    networkDiscoverySettings,
    setNetworkDiscoveryStatus,
    snmpSettings,
    setSnmpStatus,
  });

  const saveIpSettings = async (): Promise<void> => {
    setSavingIp(true);
    setIpMessage(null);
    try {
      // Parse DNS from input
      const dns = dnsInput
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s.length > 0);

      await api.put('/api/v1/telemetry/ipconfig/settings', {
        mode: ipSettings.mode,
        address: ipSettings.address,
        netmask: ipSettings.netmask,
        gateway: ipSettings.gateway,
        dns,
      });
      setIpMessage('IP settings applied');
      setTimeout(() => setIpMessage(null), 3000);
    } catch (err) {
      // The client carries the server's own message through.
      setIpMessage(`Failed: ${err instanceof Error ? err.message : t('errors:network.ipFailed')}`);
    }
    setSavingIp(false);
  };

  // The setters the controls edit through: each edit schedules its group's
  // debounced save, and the loaders keep the raw setters. Every save behind
  // them is minRole: op, so a viewer arms none of them (#2467).
  const editThresholds = useDebouncedAutoSave(setThresholds, saveThresholds, canWrite);
  const editTestsSettings = useDebouncedAutoSave(setTestsSettings, saveTestsSettings, canWrite);
  const editWifiSettings = useDebouncedAutoSave(setWifiSettings, saveWifiSettings, canWrite);
  const editLinkSettings = useDebouncedAutoSave(setLinkSettings, saveLinkSettings, canWrite);
  const editCableTestSettings = useDebouncedAutoSave(
    setCableTestSettings,
    saveCableTestSettings,
    canWrite,
  );
  const editNetworkDiscoverySettings = useDebouncedAutoSave(
    setNetworkDiscoverySettings,
    saveNetworkDiscoverySettings,
    canWrite,
  );
  const editSnmpSettings = useDebouncedAutoSave(setSnmpSettings, saveSnmpSettings, canWrite);
  const editVulnSettings = useDebouncedAutoSave(setVulnSettings, saveVulnSettings, canWrite);

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

  const renderTab = (): React.ReactElement => {
    switch (activeTab) {
      case 'general':
        return (
          <>
            <DeviceIdentitySettings />

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

            <ConfigBackupsSection />
          </>
        );
      case 'network':
        return (
          <>
            <LinkSettings
              linkSettings={linkSettings}
              setLinkSettings={editLinkSettings}
              linkStatus={linkStatus}
              cardSettings={cardSettings}
              updateCardSettings={updateCardSettings}
            />

            <CableTestSettings
              cableTestSettings={cableTestSettings}
              setCableTestSettings={editCableTestSettings}
              cableTestStatus={cableTestStatus}
            />

            <SettingsDrawerNetworkSection
              ipSettings={ipSettings}
              setIpSettings={setIpSettings}
              dnsInput={dnsInput}
              setDnsInput={setDnsInput}
              saveIpSettings={saveIpSettings}
              savingIp={savingIp}
              ipMessage={ipMessage}
              displayOptions={displayOptions}
              setDisplayOptions={setDisplayOptions}
              displayStatus={displayStatus}
              isValidIp={isValidIp}
            />

            {isWifi ? (
              <WiFiSettings
                wifiSettings={wifiSettings}
                setWifiSettings={editWifiSettings}
                wifiStatus={wifiStatus}
              />
            ) : null}

            <InterfacesSettings />
          </>
        );
      case 'tests':
        return (
          <>
            <DnsSettings
              testsSettings={testsSettings}
              setTestsSettings={editTestsSettings}
              testsStatus={testsStatus}
              cardSettings={cardSettings}
              updateCardSettings={updateCardSettings}
            />

            <HealthChecksSettings
              testsSettings={testsSettings}
              setTestsSettings={editTestsSettings}
              testsStatus={testsStatus}
              cardSettings={cardSettings}
              updateCardSettings={updateCardSettings}
            />

            <PerformanceSettings
              testsSettings={testsSettings}
              setTestsSettings={editTestsSettings}
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

            <ThresholdsSettings
              thresholds={thresholds}
              setThresholds={editThresholds}
              thresholdsStatus={thresholdsStatus}
            />
          </>
        );
      case 'discovery':
        return (
          <>
            <DiscoverySettings
              networkDiscoverySettings={networkDiscoverySettings}
              setNetworkDiscoverySettings={editNetworkDiscoverySettings}
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
              setSnmpSettings={editSnmpSettings}
              snmpStatus={snmpStatus}
              cardSettings={cardSettings}
              updateCardSettings={updateCardSettings}
            />

            <VulnerabilitySettings
              settings={vulnSettings}
              setSettings={editVulnSettings}
              status={vulnStatus}
            />

            {/* Guest-network isolation audit target list (#1004). */}
            <GuestNetworkAuditSettings />
          </>
        );
      case 'alerts':
        return (
          <>
            <AlertDeliverySettings />

            <AlertEmailSettings />

            <AlertEscalationSettings />
          </>
        );
      case 'access':
        return (
          <>
            <ApiTokensSettings />

            {/* Users CRUD (multi_user, seed#1191) — admin-only section, so an
                operator/viewer never sees user management at all (#1254). The
                in-section buttons stay isAdmin-gated as defense-in-depth. */}
            <RequireAdmin>
              <UsersSettings />
            </RequireAdmin>

            {/* SSO admin panel (seed#1198). GET /sso/settings is readable by
                every role; PUT /sso/update is minRole: op and the section
                disables its own controls for a viewer. */}
            <SsoSettings />

            {/* Opt-in EoX feed (P-D9) — admin-only, as GET/PUT /lifecycle/feed are. */}
            <RequireAdmin>
              <EoxFeedSettings />
            </RequireAdmin>
          </>
        );
      case 'about':
        return (
          <SettingsDrawerFooter
            version={version}
            fetchLogPreview={fetchLogPreview}
            logLoading={logLoading}
            logError={logError}
            logPreview={logPreview}
          />
        );
    }
  };

  if (!isOpen) {
    return null;
  }

  return (
    <RightDrawer
      open={isOpen}
      onClose={onClose}
      title={t('title')}
      closeLabel={t('network.closeSettings')}
      data-testid="settings-drawer"
    >
      <div className="text-sm leading-relaxed">
        <SettingsTabs
          tabs={tabs}
          value={activeTab}
          onChange={(id) => setActiveTab(id as SettingsTabId)}
          label={t('tabs.label')}
        >
          <div className="space-y-4">{renderTab()}</div>
        </SettingsTabs>
      </div>
    </RightDrawer>
  );
}
