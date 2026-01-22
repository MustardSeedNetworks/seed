/**
 * Profile Store - Zustand-based State Management
 *
 * Replaces the 48-hook ProfileContext with a streamlined Zustand store.
 * Benefits:
 * - Atomic state updates without re-render cascade
 * - Derived selectors with automatic memoization
 * - Simpler testing through store isolation
 *
 * Related: #890
 */

import type { StoreApi, UseBoundStore } from 'zustand';
import { create } from 'zustand';
import { devtools, persist, subscribeWithSelector } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import type { DefaultSettings } from '../types/defaults';
import type {
  AppearanceConfig,
  CableTestConfig,
  CardSettingsConfig,
  DisplayOptionsConfig,
  DnsSettingsConfig,
  IperfConfig,
  LinkConfig,
  NetworkDiscoveryConfig,
  Profile,
  ProfileSettings,
  ProfileThresholdsConfig,
  SnmpConfig,
  SpeedtestConfig,
  TestsConfig,
  VulnerabilityConfig,
  WifiSettingsConfig,
} from '../types/profile';

// ============================================================================
// State Types
// ============================================================================

export type SettingsSaveStatus = 'idle' | 'saving' | 'saved' | 'error';

interface ProfileState {
  // Core state
  profiles: Profile[];
  activeProfile: Profile | null;
  backendDefaults: DefaultSettings | null;

  // Loading/error state
  isLoading: boolean;
  isSettingsLoaded: boolean;
  error: string | null;
  settingsStatus: SettingsSaveStatus;
}

interface ProfileActions {
  // State setters
  setProfiles: (profiles: Profile[]) => void;
  setActiveProfile: (profile: Profile | null) => void;
  setBackendDefaults: (defaults: DefaultSettings | null) => void;
  setIsLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setSettingsStatus: (status: SettingsSaveStatus) => void;
  setIsSettingsLoaded: (loaded: boolean) => void;

  // Batch update for profile switch (reduces API thrashing)
  batchProfileSwitch: (profile: Profile, profiles: Profile[]) => void;

  // Update active profile settings
  updateActiveProfileSettings: (settings: Partial<ProfileSettings>) => void;

  // Reset store
  reset: () => void;
}

// ============================================================================
// Default Values
// ============================================================================

const DEFAULT_CARD_SETTINGS: CardSettingsConfig = {
  link: { enabled: true, autoRunOnLink: true },
  cable: { enabled: true, autoRunOnLink: true },
  switch: { enabled: true, autoRunOnLink: true },
  vlan: { enabled: true, autoRunOnLink: true },
  network: { enabled: true, autoRunOnLink: true },
  gateway: { enabled: true, autoRunOnLink: true },
  dns: { enabled: true, autoRunOnLink: true },
  publicIp: { enabled: true, autoRunOnLink: true },
  wifi: { enabled: true, autoRunOnLink: true },
  wifiSurvey: { enabled: true, autoRunOnLink: true },
  healthChecks: { enabled: true, autoRunOnLink: true },
  networkDiscovery: { enabled: true, autoRunOnLink: true },
  pathDiscovery: { enabled: true, autoRunOnLink: true },
  systemHealth: { enabled: true, autoRunOnLink: true },
  performance: {
    enabled: true,
    autoRunOnLink: true,
    speedtest: { enabled: true, autoRunOnLink: true },
    iperf: { enabled: false, autoRunOnLink: false },
  },
};

const DEFAULT_DISPLAY_OPTIONS: DisplayOptionsConfig = {
  showAdvancedMetrics: false,
  compactMode: false,
  autoRefresh: true,
  refreshInterval: 30,
};

const DEFAULT_IPERF_SETTINGS: IperfConfig = {
  server: '',
  port: 5201,
  duration: 10,
  parallel: 1,
  reverse: false,
};

const DEFAULT_THRESHOLDS: ProfileThresholdsConfig = {
  latency: { good: 50, warning: 100 },
  jitter: { good: 10, warning: 30 },
  packetLoss: { good: 0.1, warning: 1.0 },
  downloadSpeed: { good: 100, warning: 25 },
  uploadSpeed: { good: 50, warning: 10 },
};

const DEFAULT_SPEEDTEST_SETTINGS: SpeedtestConfig = {
  serverId: null,
  duration: 10,
  parallelConnections: 4,
};

const DEFAULT_TESTS_SETTINGS: TestsConfig = {
  autoRunOnStart: false,
  testsToRun: ['gateway', 'dns', 'speedtest'],
  runInterval: 0,
};

const DEFAULT_NETWORK_DISCOVERY_SETTINGS: NetworkDiscoveryConfig = {
  autoScanOnStart: true,
  scanInterval: 300,
  maxDevices: 256,
  includeOfflineDevices: true,
  scanMethods: ['arp', 'ping', 'mdns'],
  additionalSubnets: [],
};

const DEFAULT_SNMP_SETTINGS: SnmpConfig = {
  enabled: false,
  communities: ['public'],
  version: '2c',
  timeout: 5,
};

const DEFAULT_WIFI_SETTINGS: WifiSettingsConfig = {
  scanDuration: 5,
  includeHiddenNetworks: false,
  signalThreshold: -80,
};

const DEFAULT_LINK_SETTINGS: LinkConfig = {
  mtu: 1500,
  speed: 'auto',
  duplex: 'auto',
};

const DEFAULT_CABLE_TEST_SETTINGS: CableTestConfig = {
  testDuration: 10,
  targetHost: '8.8.8.8',
};

const DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityConfig = {
  enabled: false,
  severityThreshold: 'medium',
  maxConcurrent: 5,
  autoScan: true,
};

const DEFAULT_DNS_SETTINGS: DnsSettingsConfig = {
  testHostname: 'google.com',
  servers: [],
};

const DEFAULT_APPEARANCE_SETTINGS: AppearanceConfig = {
  theme: 'system',
  language: 'en',
};

// Initial state
const initialState: ProfileState = {
  profiles: [],
  activeProfile: null,
  backendDefaults: null,
  isLoading: false,
  isSettingsLoaded: false,
  error: null,
  settingsStatus: 'idle',
};

// ============================================================================
// Store Creation
// ============================================================================

export const useProfileStore: UseBoundStore<StoreApi<ProfileState & ProfileActions>> = create<
  ProfileState & ProfileActions
>()(
  devtools(
    persist(
      subscribeWithSelector(
        immer((set) => ({
          ...initialState,

          setProfiles: (profiles: Profile[]) =>
            set((state: ProfileState) => {
              state.profiles = profiles;
            }),

          setActiveProfile: (profile: Profile | null) =>
            set((state: ProfileState) => {
              state.activeProfile = profile;
            }),

          setBackendDefaults: (defaults: DefaultSettings | null) =>
            set((state: ProfileState) => {
              state.backendDefaults = defaults;
            }),

          setIsLoading: (loading: boolean) =>
            set((state: ProfileState) => {
              state.isLoading = loading;
            }),

          setError: (error: string | null) =>
            set((state: ProfileState) => {
              state.error = error;
            }),

          setSettingsStatus: (status: SettingsSaveStatus) =>
            set((state: ProfileState) => {
              state.settingsStatus = status;
            }),

          setIsSettingsLoaded: (loaded: boolean) =>
            set((state: ProfileState) => {
              state.isSettingsLoaded = loaded;
            }),

          // Batch update for profile switch - single state update instead of multiple
          batchProfileSwitch: (profile: Profile, profiles: Profile[]) =>
            set((state: ProfileState) => {
              state.activeProfile = profile;
              state.profiles = profiles;
              state.isSettingsLoaded = true;
              state.error = null;
            }),

          updateActiveProfileSettings: (settings: Partial<ProfileSettings>) =>
            set((state: ProfileState) => {
              if (state.activeProfile) {
                state.activeProfile.settings = {
                  ...state.activeProfile.settings,
                  ...settings,
                };
              }
            }),

          reset: () => set(initialState),
        })),
      ),
      {
        name: 'seed-profile-store',
        // Only persist the active profile ID, not the full data
        partialize: (state: ProfileState) => ({
          activeProfileId: state.activeProfile?.id,
        }),
      },
    ),
    { name: 'profile-store' },
  ),
);

// ============================================================================
// Derived Selectors (memoized automatically by Zustand)
// ============================================================================

// Helper to merge settings with defaults
function mergeWithDefaults<T>(
  profileValue: T | undefined,
  backendDefault: T | undefined,
  hardcodedDefault: T,
): T {
  return profileValue ?? backendDefault ?? hardcodedDefault;
}

// Settings selectors - these replace the 15+ useMemo hooks
export const useCardSettings = (): CardSettingsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.cardSettings,
    backendDefaults?.cardSettings,
    DEFAULT_CARD_SETTINGS,
  );
};

export const useDisplayOptions = (): DisplayOptionsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.displayOptions,
    backendDefaults?.displayOptions,
    DEFAULT_DISPLAY_OPTIONS,
  );
};

export const useIperfSettings = (): IperfConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.iperf,
    backendDefaults?.iperf,
    DEFAULT_IPERF_SETTINGS,
  );
};

export const useThresholds = (): ProfileThresholdsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.thresholds,
    backendDefaults?.thresholds,
    DEFAULT_THRESHOLDS,
  );
};

export const useSpeedtestSettings = (): SpeedtestConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.speedtest,
    backendDefaults?.speedtest,
    DEFAULT_SPEEDTEST_SETTINGS,
  );
};

export const useTestsSettings = (): TestsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.tests,
    backendDefaults?.tests,
    DEFAULT_TESTS_SETTINGS,
  );
};

export const useNetworkDiscoverySettings = (): NetworkDiscoveryConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.networkDiscovery,
    backendDefaults?.networkDiscovery,
    DEFAULT_NETWORK_DISCOVERY_SETTINGS,
  );
};

export const useSnmpSettings = (): SnmpConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.snmp,
    backendDefaults?.snmp,
    DEFAULT_SNMP_SETTINGS,
  );
};

export const useWifiSettings = (): WifiSettingsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.wifi,
    backendDefaults?.wifi,
    DEFAULT_WIFI_SETTINGS,
  );
};

export const useLinkSettings = (): LinkConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.link,
    backendDefaults?.link,
    DEFAULT_LINK_SETTINGS,
  );
};

export const useCableTestSettings = (): CableTestConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.cableTest,
    backendDefaults?.cableTest,
    DEFAULT_CABLE_TEST_SETTINGS,
  );
};

export const useVulnerabilitySettings = (): VulnerabilityConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.vulnerability,
    backendDefaults?.vulnerability,
    DEFAULT_VULNERABILITY_SETTINGS,
  );
};

export const useDnsSettings = (): DnsSettingsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.dns,
    backendDefaults?.dns,
    DEFAULT_DNS_SETTINGS,
  );
};

export const useAppearanceSettings = (): AppearanceConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.settings?.appearance,
    backendDefaults?.appearance,
    DEFAULT_APPEARANCE_SETTINGS,
  );
};
