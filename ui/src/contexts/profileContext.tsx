/**
 * ProfileContext - MSP Profile Management & Settings
 *
 * Provides a React Context for managing MSP profiles and ALL user settings.
 * Profiles are the SINGLE SOURCE OF TRUTH for all configuration.
 *
 * Architecture (#890):
 * - Zustand store for state management (atomic updates, memoized selectors)
 * - React Query for API calls (caching, deduplication, background refetch)
 * - Context provides backwards-compatible interface for consumers
 *
 * Layout:
 *  - profileContext.tsx    — ProfileContextValue interface + ProfileProvider
 *                            assembly + the public useProfileContext hook
 *  - useProfileApi.ts      — backwards-compat wrappers around React Query
 *                            queries / mutations (refresh / CRUD / import / export)
 *  - useProfileSettingsUpdates.ts — auto-save settings updaters that target
 *                            the active profile via the saveSettings mutation
 *  - useProfileInterfaces.ts — ethernet / wifi multi-interface helpers
 *                            (get / set / add / remove / setActive)
 */

import { createContext, type ReactNode, useCallback, useContext } from 'react';
import { useBackendDefaultsQuery } from '../stores/profileQueries';
import {
  type SettingsSaveStatus,
  useAppearanceSettings,
  useCableTestSettings,
  useCardSettings,
  useDisplayOptions,
  useDnsSettings,
  useIperfSettings,
  useLinkSettings,
  useNetworkDiscoverySettings,
  useProfileStore,
  useSnmpSettings,
  useSpeedtestSettings,
  useTestsSettings,
  useThresholds,
  useVulnerabilitySettings,
  useWifiSettings,
} from '../stores/profileStore';
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
  ProfileExportResponse,
  ProfileImportRequest,
  ProfileImportResponse,
  ProfileInterfaceSelection,
  ProfileRequest,
  ProfileSettings,
  ProfileThresholdsConfig,
  SnmpConfig,
  SpeedtestConfig,
  TestsConfig,
  VulnerabilityConfig,
  WifiSettingsConfig,
} from '../types/profile';
import { useProfileApi } from './useProfileApi';
import { useProfileInterfaces } from './useProfileInterfaces';
import { useProfileSettingsUpdates } from './useProfileSettingsUpdates';

// Re-export for consumers
export type { SettingsSaveStatus };

// ============================================================================
// Context Type Definition
// ============================================================================

export interface ProfileContextValue {
  // Profile State
  profiles: Profile[];
  activeProfile: Profile | null;
  isLoading: boolean;
  error: string | null;

  // Settings State (derived from active profile, merged with defaults)
  // Order matches Settings Drawer UI for consistency
  linkSettings: LinkConfig;
  cableTestSettings: CableTestConfig;
  displayOptions: DisplayOptionsConfig;
  wifiSettings: WifiSettingsConfig;
  dnsSettings: DnsSettingsConfig;
  testsSettings: TestsConfig;
  speedtestSettings: SpeedtestConfig;
  iperfSettings: IperfConfig;
  networkDiscoverySettings: NetworkDiscoveryConfig;
  snmpSettings: SnmpConfig;
  vulnerabilitySettings: VulnerabilityConfig;
  thresholds: ProfileThresholdsConfig;
  appearanceSettings: AppearanceConfig;
  cardSettings: CardSettingsConfig;
  settingsStatus: SettingsSaveStatus;
  isSettingsLoaded: boolean;

  // List/fetch operations
  refreshProfiles: () => Promise<void>;
  refreshActiveProfile: () => Promise<void>;

  // CRUD operations
  createProfile: (profile: ProfileRequest) => Promise<Profile | null>;
  updateProfile: (id: string, profile: ProfileRequest) => Promise<Profile | null>;
  deleteProfile: (id: string) => Promise<boolean>;

  // Profile switching
  switchProfile: (profileId: string) => Promise<boolean>;

  // Duplication
  duplicateProfile: (id: string, newName?: string) => Promise<Profile | null>;

  // Import/Export
  importProfiles: (request: ProfileImportRequest) => Promise<ProfileImportResponse | null>;
  exportProfiles: () => Promise<ProfileExportResponse | null>;
  downloadProfiles: () => Promise<boolean>;

  // Settings update methods - auto-save to active profile with debounce
  // Order matches Settings Drawer UI for consistency
  updateLinkSettings: (updates: Partial<LinkConfig>) => void;
  updateCableTestSettings: (updates: Partial<CableTestConfig>) => void;
  updateDisplayOptions: (updates: Partial<DisplayOptionsConfig>) => void;
  updateWifiSettings: (updates: Partial<WifiSettingsConfig>) => void;
  updateDnsSettings: (updates: Partial<DnsSettingsConfig>) => void;
  updateTestsSettings: (updates: Partial<TestsConfig>) => void;
  updateSpeedtestSettings: (updates: Partial<SpeedtestConfig>) => void;
  updateIperfSettings: (updates: Partial<IperfConfig>) => void;
  updateNetworkDiscoverySettings: (updates: Partial<NetworkDiscoveryConfig>) => void;
  updateSnmpSettings: (updates: Partial<SnmpConfig>) => void;
  updateVulnerabilitySettings: (updates: Partial<VulnerabilityConfig>) => void;
  updateThresholds: (updates: Partial<ProfileThresholdsConfig>) => void;
  updateAppearanceSettings: (updates: Partial<AppearanceConfig>) => void;
  updateCardSettings: (updates: Partial<CardSettingsConfig>) => void;
  /** Update any part of the profile settings */
  updateSettings: (updates: Partial<ProfileSettings>) => void;
  /** Force refresh settings from backend */
  refreshSettings: () => Promise<void>;

  // Interface selection helpers (multi-interface support)
  /** Get the active ethernet interface from the active profile */
  getEthernetInterface: () => ProfileInterfaceSelection | null;
  /** Get the active Wifi interface from the active profile */
  getWifiInterface: () => ProfileInterfaceSelection | null;
  /** Get all ethernet interfaces from the active profile */
  getAllEthernetInterfaces: () => ProfileInterfaceSelection[];
  /** Get all Wifi interfaces from the active profile */
  getAllWifiInterfaces: () => ProfileInterfaceSelection[];
  /** Add or update an ethernet interface and set it as active */
  setEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add or update a Wifi interface and set it as active */
  setWifiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add an ethernet interface without changing the active one */
  addEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add a Wifi interface without changing the active one */
  addWifiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Remove an ethernet interface */
  removeEthernetInterface: (name: string) => Promise<boolean>;
  /** Remove a Wifi interface */
  removeWifiInterface: (name: string) => Promise<boolean>;
  /** Set the active ethernet interface (must already be in the list) */
  setActiveEthernetInterface: (name: string) => Promise<boolean>;
  /** Set the active Wifi interface (must already be in the list) */
  setActiveWifiInterface: (name: string) => Promise<boolean>;
}

// Create context with undefined default to enforce provider requirement
const PROFILE_CONTEXT: React.Context<ProfileContextValue | undefined> = createContext<
  ProfileContextValue | undefined
>(undefined);

// ============================================================================
// Provider Component
// ============================================================================

interface ProfileProviderProps {
  children: ReactNode;
}

/**
 * Context provider that manages profile state, settings, and API synchronization.
 * Profiles are the SINGLE SOURCE OF TRUTH for all user settings.
 *
 * Uses Zustand for state and React Query for API calls (#890).
 */
export function ProfileProvider({ children }: ProfileProviderProps): React.JSX.Element {
  // Zustand store state
  const profiles = useProfileStore((s) => s.profiles);
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const isLoading = useProfileStore((s) => s.isLoading);
  const error = useProfileStore((s) => s.error);
  const settingsStatus = useProfileStore((s) => s.settingsStatus);
  const isSettingsLoaded = useProfileStore((s) => s.isSettingsLoaded);
  const setActiveProfile = useProfileStore((s) => s.setActiveProfile);

  // Memoized settings selectors
  const linkSettings = useLinkSettings();
  const cableTestSettings = useCableTestSettings();
  const displayOptions = useDisplayOptions();
  const wifiSettings = useWifiSettings();
  const dnsSettings = useDnsSettings();
  const testsSettings = useTestsSettings();
  const speedtestSettings = useSpeedtestSettings();
  const iperfSettings = useIperfSettings();
  const networkDiscoverySettings = useNetworkDiscoverySettings();
  const snmpSettings = useSnmpSettings();
  const vulnerabilitySettings = useVulnerabilitySettings();
  const thresholds = useThresholds();
  const appearanceSettings = useAppearanceSettings();
  const cardSettings = useCardSettings();

  // Trigger backend defaults query (state synced via effect inside the hook)
  useBackendDefaultsQuery();

  // Extracted hooks: API wrappers, settings updaters, interface helpers
  const apiOps = useProfileApi();
  const settingsUpdaters = useProfileSettingsUpdates(activeProfile);
  const interfaceOps = useProfileInterfaces(activeProfile, setActiveProfile);

  const refreshSettings = useCallback(async () => {
    await apiOps.refreshActiveProfile();
  }, [apiOps.refreshActiveProfile]);

  const contextValue: ProfileContextValue = {
    // Profile state (from Zustand)
    profiles,
    activeProfile,
    isLoading,
    error,

    // Settings state (from Zustand selectors)
    linkSettings,
    cableTestSettings,
    displayOptions,
    wifiSettings,
    dnsSettings,
    testsSettings,
    speedtestSettings,
    iperfSettings,
    networkDiscoverySettings,
    snmpSettings,
    vulnerabilitySettings,
    thresholds,
    appearanceSettings,
    cardSettings,
    settingsStatus,
    isSettingsLoaded,

    // Profile list/fetch + CRUD + import/export
    ...apiOps,

    // Settings auto-save
    ...settingsUpdaters,
    refreshSettings,

    // Interface helpers
    ...interfaceOps,
  };

  return <PROFILE_CONTEXT.Provider value={contextValue}>{children}</PROFILE_CONTEXT.Provider>;
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook to access the profile context.
 * Must be used within a ProfileProvider.
 */
export function useProfileContext(): ProfileContextValue {
  const context = useContext(PROFILE_CONTEXT);
  if (context === undefined) {
    throw new Error('useProfileContext must be used within a ProfileProvider');
  }
  return context;
}

/**
 * Hook to optionally access the profile context.
 * Returns null if used outside ProfileProvider (for non-critical usage).
 */
export function useProfileContextOptional(): ProfileContextValue | null {
  const context = useContext(PROFILE_CONTEXT);
  return context === undefined ? null : context;
}

// Note: ProfileContext is intentionally not exported to keep fast-refresh happy.
