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
import { useDebouncedAutoSave } from '../../hooks/useDebouncedAutoSave';
import { useSettingsDrawerLoaders } from '../../hooks/useSettingsDrawerLoaders';
import { useSettingsDrawerSavers } from '../../hooks/useSettingsDrawerSavers';
import { useSubnetSettings } from '../../hooks/useSubnetSettings';
import { useTheme } from '../../hooks/useTheme';
import { useVulnerabilitySettings } from '../../hooks/useVulnerabilitySettings';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import type {
  CableTestSettings as CableTestSettingsType,
  IperfSuggestion,
  IpSettings,
  LinkSettings as LinkSettingsType,
  NetworkDiscoverySettings,
  SettingsThresholds,
  SnmpSettings as SnmpSettingsType,
  TestsSettings,
  WiFiSettings as WiFiSettingsType,
} from '../../types/settings';
import { SettingsDrawerFooter } from './SettingsDrawerFooter';
import { SettingsDrawerNetworkSection } from './SettingsDrawerNetworkSection';
import { AppearanceSettings } from './sections/AppearanceSettings';
import { CableTestSettings } from './sections/CableTestSettings';
import { ConfigBackupsSection } from './sections/ConfigBackupsSection';
import { DiscoverySettings } from './sections/DiscoverySettings';
import { DnsSettings } from './sections/DnsSettings';
import { HealthChecksSettings } from './sections/HealthChecksSettings';
import { LinkSettings } from './sections/LinkSettings';
import { PerformanceSettings } from './sections/PerformanceSettings';
import { ThresholdsSettings } from './sections/ThresholdsSettings';
import { UpdateSettings } from './sections/UpdateSettings';
import { VulnerabilitySettings } from './sections/VulnerabilitySettings';
import { WiFiSettings } from './sections/WiFiSettings';
import {
  INLINE_DEFAULT_CABLE_TEST_SETTINGS,
  INLINE_DEFAULT_LINK_SETTINGS,
} from './settingsDrawerNormalizer';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

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
  } = useSubnetSettings(isOpen);
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
  const {
    vulnSettings,
    setVulnSettings,
    vulnStatus,
    vulnInitRef,
    vulnTimerRef,
    fetchVulnSettings,
    saveVulnSettings,
  } = useVulnerabilitySettings();
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

  // Debounce timers (fab, display, iperf now handled by context)
  const thresholdsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const testsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wifiTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const linkTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const cableTestTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const networkDiscoveryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const snmpTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Legacy state (keep for IP settings which still needs manual apply)
  const [savingIp, setSavingIp] = useState(false);
  const [ipMessage, setIpMessage] = useState<string | null>(null);

  // Per-section fetch callbacks + open-time orchestration live in their hook
  const { fetchIperfSuggestions, fetchLogPreview } = useSettingsDrawerLoaders({
    isOpen,
    initRefs: {
      initialLoadRef,
      thresholdsInitRef,
      testsInitRef,
      wifiInitRef,
      linkInitRef,
      cableTestInitRef,
      networkDiscoveryInitRef,
      snmpInitRef,
      vulnInitRef,
    },
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

  // Debounced auto-save effects for every settings group
  useDebouncedAutoSave(saveThresholds, thresholdsInitRef, thresholdsTimerRef);
  useDebouncedAutoSave(saveTestsSettings, testsInitRef, testsTimerRef);
  useDebouncedAutoSave(saveWifiSettings, wifiInitRef, wifiTimerRef);
  useDebouncedAutoSave(saveLinkSettings, linkInitRef, linkTimerRef);
  useDebouncedAutoSave(saveCableTestSettings, cableTestInitRef, cableTestTimerRef);
  useDebouncedAutoSave(
    saveNetworkDiscoverySettings,
    networkDiscoveryInitRef,
    networkDiscoveryTimerRef,
  );
  useDebouncedAutoSave(saveSnmpSettings, snmpInitRef, snmpTimerRef);
  useDebouncedAutoSave(saveVulnSettings, vulnInitRef, vulnTimerRef);

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

          <SettingsDrawerFooter
            version={version}
            fetchLogPreview={fetchLogPreview}
            logLoading={logLoading}
            logError={logError}
            logPreview={logPreview}
          />
        </div>
      </div>
    </>
  );
});
