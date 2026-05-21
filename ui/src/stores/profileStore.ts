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
  WiFiSettingsConfig,
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

// All *Config types have fully optional fields. These last-resort defaults
// are only used when both the active profile AND backendDefaults are
// missing the section. The backend ships canonical defaults via
// /api/v1/sap/defaults, so the empty literals here are intentional —
// they exist so selectors return a non-undefined value rather than
// claiming any specific defaults that would drift from the backend.
const DEFAULT_DISPLAY_OPTIONS: DisplayOptionsConfig = {};
const DEFAULT_IPERF_SETTINGS: IperfConfig = {};
const DEFAULT_THRESHOLDS: ProfileThresholdsConfig = {};
const DEFAULT_SPEEDTEST_SETTINGS: SpeedtestConfig = {};
const DEFAULT_TESTS_SETTINGS: TestsConfig = {};
const DEFAULT_NETWORK_DISCOVERY_SETTINGS: NetworkDiscoveryConfig = {};
const DEFAULT_SNMP_SETTINGS: SnmpConfig = {};
const DEFAULT_WIFI_SETTINGS: WiFiSettingsConfig = {};
const DEFAULT_LINK_SETTINGS: LinkConfig = {};
const DEFAULT_CABLE_TEST_SETTINGS: CableTestConfig = {};
const DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityConfig = {};
const DEFAULT_DNS_SETTINGS: DnsSettingsConfig = {};
const DEFAULT_APPEARANCE_SETTINGS: AppearanceConfig = {};

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
                state.activeProfile.config = {
                  ...state.activeProfile.config,
                  settings: {
                    ...state.activeProfile.config?.settings,
                    ...settings,
                  },
                };
              }
            }),

          reset: () => set(initialState),
        })),
      ),
      {
        name: 'seed-profileStore',
        // Only persist the active profile ID, not the full data
        partialize: (state: ProfileState) => ({
          activeProfileId: state.activeProfile?.id,
        }),
      },
    ),
    { name: 'profileStore' },
  ),
);

// ============================================================================
// Derived Selectors (memoized automatically by Zustand)
// ============================================================================

/**
 * Merge order: profileValue (user-saved) → backendDefault (server-shipped
 * canonical) → hardcodedDefault (last-resort, always type-safe).
 *
 * backendDefault is typed as `unknown` because the backend Defaults
 * namespace (e.g. DisplayOptionsDefaults) is structurally similar but
 * not identical to the Config namespace (DisplayOptionsConfig). The
 * cast happens at the merge boundary to keep selector consumers
 * working without a giant Defaults↔Config type-map rewrite.
 */
function mergeWithDefaults<T>(
  profileValue: T | undefined,
  backendDefault: unknown,
  hardcodedDefault: T,
): T {
  return (profileValue ?? (backendDefault as T | undefined) ?? hardcodedDefault) as T;
}

// Settings selectors - these replace the 15+ useMemo hooks
export const useCardSettings = (): CardSettingsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.cardSettings,
    backendDefaults?.cardSettings,
    DEFAULT_CARD_SETTINGS,
  );
};

export const useDisplayOptions = (): DisplayOptionsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.displayOptions,
    backendDefaults?.displayOptions,
    DEFAULT_DISPLAY_OPTIONS,
  );
};

export const useIperfSettings = (): IperfConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.iperf,
    backendDefaults?.iperf,
    DEFAULT_IPERF_SETTINGS,
  );
};

export const useThresholds = (): ProfileThresholdsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.thresholds,
    backendDefaults?.thresholds,
    DEFAULT_THRESHOLDS,
  );
};

export const useSpeedtestSettings = (): SpeedtestConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const _backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.speedtest,
    undefined, // backend DefaultSettings does not currently expose speedtest
    DEFAULT_SPEEDTEST_SETTINGS,
  );
};

export const useTestsSettings = (): TestsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.tests,
    backendDefaults?.tests,
    DEFAULT_TESTS_SETTINGS,
  );
};

export const useNetworkDiscoverySettings = (): NetworkDiscoveryConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.networkDiscovery,
    backendDefaults?.networkDiscovery,
    DEFAULT_NETWORK_DISCOVERY_SETTINGS,
  );
};

export const useSnmpSettings = (): SnmpConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.snmp,
    backendDefaults?.snmp,
    DEFAULT_SNMP_SETTINGS,
  );
};

export const useWifiSettings = (): WiFiSettingsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const _backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.wifi,
    undefined, // backend DefaultSettings does not currently expose wifi
    DEFAULT_WIFI_SETTINGS,
  );
};

export const useLinkSettings = (): LinkConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.link,
    backendDefaults?.link,
    DEFAULT_LINK_SETTINGS,
  );
};

export const useCableTestSettings = (): CableTestConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.cableTest,
    backendDefaults?.cableTest,
    DEFAULT_CABLE_TEST_SETTINGS,
  );
};

export const useVulnerabilitySettings = (): VulnerabilityConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.vulnerability,
    backendDefaults?.vulnerability,
    DEFAULT_VULNERABILITY_SETTINGS,
  );
};

export const useDnsSettings = (): DnsSettingsConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const _backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.dns,
    undefined, // backend DefaultSettings does not currently expose dns
    DEFAULT_DNS_SETTINGS,
  );
};

export const useAppearanceSettings = (): AppearanceConfig => {
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const _backendDefaults = useProfileStore((s) => s.backendDefaults);
  return mergeWithDefaults(
    activeProfile?.config?.settings?.appearance,
    undefined, // backend DefaultSettings does not currently expose appearance
    DEFAULT_APPEARANCE_SETTINGS,
  );
};
