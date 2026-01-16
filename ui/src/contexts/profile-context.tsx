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
 * Features:
 * - Load and cache profiles
 * - Track active profile
 * - Handle profile switching
 * - Manage ALL user settings (cardSettings, displayOptions, thresholds, etc.)
 * - Auto-save settings to active profile with debouncing
 */

import { createContext, type ReactNode, useCallback, useContext, useRef } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";
import { getQueryClient } from "../lib/query-client";
import {
  profileKeys,
  useActiveProfileQuery,
  useBackendDefaultsQuery,
  useCreateProfileMutation,
  useDeleteProfileMutation,
  useDuplicateProfileMutation,
  useImportProfilesMutation,
  useProfilesQuery,
  useSaveSettingsMutation,
  useSwitchProfileMutation,
  useUpdateProfileMutation,
} from "../stores/profile-queries";
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
} from "../stores/profile-store";
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
} from "../types/profile";

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
  // ============================================================================
  // Zustand Store State
  // ============================================================================
  const profiles = useProfileStore((s) => s.profiles);
  const activeProfile = useProfileStore((s) => s.activeProfile);
  const isLoading = useProfileStore((s) => s.isLoading);
  const error = useProfileStore((s) => s.error);
  const settingsStatus = useProfileStore((s) => s.settingsStatus);
  const isSettingsLoaded = useProfileStore((s) => s.isSettingsLoaded);
  const setActiveProfile = useProfileStore((s) => s.setActiveProfile);

  // ============================================================================
  // Zustand Derived Selectors (memoized settings)
  // ============================================================================
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

  // ============================================================================
  // React Query Hooks
  // ============================================================================
  const profilesQuery = useProfilesQuery();
  const activeProfileQuery = useActiveProfileQuery();
  useBackendDefaultsQuery(); // Just triggers the query, state synced via effect

  const createProfileMutation = useCreateProfileMutation();
  const updateProfileMutation = useUpdateProfileMutation();
  const deleteProfileMutation = useDeleteProfileMutation();
  const switchProfileMutation = useSwitchProfileMutation();
  const duplicateProfileMutation = useDuplicateProfileMutation();
  const importProfilesMutation = useImportProfilesMutation();
  const saveSettingsMutation = useSaveSettingsMutation();

  // ============================================================================
  // Refs for interface helpers
  // ============================================================================
  const activeProfileRef = useRef(activeProfile);
  activeProfileRef.current = activeProfile;

  // ============================================================================
  // API Methods - Wrapped for backwards compatibility
  // ============================================================================

  const refreshProfiles = useCallback(async () => {
    // refetch returns a thenable (QueryObserverResult), awaiting for completion
    await Promise.resolve(profilesQuery.refetch());
  }, [profilesQuery]);

  const refreshActiveProfile = useCallback(async () => {
    // refetch returns a thenable (QueryObserverResult), awaiting for completion
    await Promise.resolve(activeProfileQuery.refetch());
  }, [activeProfileQuery]);

  const createProfile = useCallback(
    async (profile: ProfileRequest): Promise<Profile | null> => {
      try {
        // mutateAsync returns a thenable, wrap in Promise.resolve for linter
        const result = await Promise.resolve(createProfileMutation.mutateAsync(profile));
        return result;
      } catch {
        return null;
      }
    },
    [createProfileMutation],
  );

  const updateProfile = useCallback(
    async (id: string, profile: ProfileRequest): Promise<Profile | null> => {
      try {
        // mutateAsync returns a thenable, wrap in Promise.resolve for linter
        const result = await Promise.resolve(updateProfileMutation.mutateAsync({ id, profile }));
        return result;
      } catch {
        return null;
      }
    },
    [updateProfileMutation],
  );

  const deleteProfile = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        // mutateAsync returns a thenable, wrap in Promise.resolve for linter
        await Promise.resolve(deleteProfileMutation.mutateAsync(id));
        return true;
      } catch {
        return false;
      }
    },
    [deleteProfileMutation],
  );

  const switchProfile = useCallback(
    async (profileId: string): Promise<boolean> => {
      try {
        // mutateAsync returns a thenable, wrap in Promise.resolve for linter
        await Promise.resolve(switchProfileMutation.mutateAsync(profileId));
        return true;
      } catch {
        return false;
      }
    },
    [switchProfileMutation],
  );

  const duplicateProfile = useCallback(
    async (id: string, _newName?: string): Promise<Profile | null> => {
      try {
        // mutateAsync returns a thenable, wrap in Promise.resolve for linter
        const result = await Promise.resolve(duplicateProfileMutation.mutateAsync(id));
        return result;
      } catch {
        return null;
      }
    },
    [duplicateProfileMutation],
  );

  const importProfiles = useCallback(
    async (request: ProfileImportRequest): Promise<ProfileImportResponse | null> => {
      try {
        // mutateAsync returns a thenable, wrap in Promise.resolve for linter
        const result = await Promise.resolve(importProfilesMutation.mutateAsync(request));
        return result;
      } catch {
        return null;
      }
    },
    [importProfilesMutation],
  );

  const exportProfiles = useCallback(async (): Promise<ProfileExportResponse | null> => {
    try {
      const queryClient = getQueryClient();
      // fetchQuery returns a thenable, wrap in Promise.resolve for linter
      const result = await Promise.resolve(
        queryClient.fetchQuery({
          queryKey: [...profileKeys.all, "export"],
          queryFn: async () => {
            const data = await api.get<ProfileExportResponse>("/api/v1/profiles/export");
            return data;
          },
          staleTime: 0,
        }),
      );
      logger.info(LogComponents.Profiles, "Profiles exported", {
        count: result.profiles.length,
      });
      return result;
    } catch (err) {
      logger.error(LogComponents.Profiles, "Failed to export profiles", err);
      return null;
    }
  }, []);

  const downloadProfiles = useCallback(async (): Promise<boolean> => {
    try {
      const result = await exportProfiles();
      if (!result) {
        return false;
      }

      // Create and trigger download
      const blob = new Blob([JSON.stringify(result, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `seed-profiles-${new Date().toISOString().split("T")[0]}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);

      return true;
    } catch (err) {
      logger.error(LogComponents.Profiles, "Failed to download profiles", err);
      return false;
    }
  }, [exportProfiles]);

  // ============================================================================
  // Settings Update Methods
  // ============================================================================

  /**
   * Generic settings updater that uses the save mutation.
   */
  const updateSettingsField = useCallback(
    <T extends keyof ProfileSettings>(field: T, updates: Partial<ProfileSettings[T]>) => {
      if (!activeProfile) {
        logger.warn(LogComponents.Profiles, "Cannot save settings: no active profile");
        return;
      }

      const currentSettings = activeProfile.settings ?? {};
      const currentFieldValue = currentSettings[field] ?? {};
      const newSettings = {
        [field]: { ...currentFieldValue, ...updates },
      };

      saveSettingsMutation.mutate({
        profileId: activeProfile.id,
        settings: newSettings,
      });
    },
    [activeProfile, saveSettingsMutation],
  );

  const updateCardSettings = useCallback(
    (updates: Partial<CardSettingsConfig>) => {
      updateSettingsField("cardSettings", updates);
    },
    [updateSettingsField],
  );

  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptionsConfig>) => {
      updateSettingsField("displayOptions", updates);
    },
    [updateSettingsField],
  );

  const updateIperfSettings = useCallback(
    (updates: Partial<IperfConfig>) => {
      updateSettingsField("iperf", updates);
    },
    [updateSettingsField],
  );

  const updateThresholds = useCallback(
    (updates: Partial<ProfileThresholdsConfig>) => {
      updateSettingsField("thresholds", updates);
    },
    [updateSettingsField],
  );

  const updateSpeedtestSettings = useCallback(
    (updates: Partial<SpeedtestConfig>) => {
      updateSettingsField("speedtest", updates);
    },
    [updateSettingsField],
  );

  const updateTestsSettings = useCallback(
    (updates: Partial<TestsConfig>) => {
      updateSettingsField("tests", updates);
    },
    [updateSettingsField],
  );

  const updateNetworkDiscoverySettings = useCallback(
    (updates: Partial<NetworkDiscoveryConfig>) => {
      updateSettingsField("networkDiscovery", updates);
    },
    [updateSettingsField],
  );

  const updateSnmpSettings = useCallback(
    (updates: Partial<SnmpConfig>) => {
      updateSettingsField("snmp", updates);
    },
    [updateSettingsField],
  );

  const updateWifiSettings = useCallback(
    (updates: Partial<WifiSettingsConfig>) => {
      updateSettingsField("wifi", updates);
    },
    [updateSettingsField],
  );

  const updateLinkSettings = useCallback(
    (updates: Partial<LinkConfig>) => {
      updateSettingsField("link", updates);
    },
    [updateSettingsField],
  );

  const updateCableTestSettings = useCallback(
    (updates: Partial<CableTestConfig>) => {
      updateSettingsField("cableTest", updates);
    },
    [updateSettingsField],
  );

  const updateVulnerabilitySettings = useCallback(
    (updates: Partial<VulnerabilityConfig>) => {
      updateSettingsField("vulnerability", updates);
    },
    [updateSettingsField],
  );

  const updateDnsSettings = useCallback(
    (updates: Partial<DnsSettingsConfig>) => {
      updateSettingsField("dns", updates);
    },
    [updateSettingsField],
  );

  const updateAppearanceSettings = useCallback(
    (updates: Partial<AppearanceConfig>) => {
      updateSettingsField("appearance", updates);
    },
    [updateSettingsField],
  );

  const updateSettings = useCallback(
    (updates: Partial<ProfileSettings>) => {
      if (!activeProfile) {
        logger.warn(LogComponents.Profiles, "Cannot save settings: no active profile");
        return;
      }

      saveSettingsMutation.mutate({
        profileId: activeProfile.id,
        settings: updates,
      });
    },
    [activeProfile, saveSettingsMutation],
  );

  const refreshSettings = useCallback(async () => {
    await refreshActiveProfile();
  }, [refreshActiveProfile]);

  // ============================================================================
  // Interface Selection Helpers
  // ============================================================================

  const getEthernetInterface = useCallback((): ProfileInterfaceSelection | null => {
    const interfaces = activeProfileRef.current?.config?.interfaces;
    if (!(interfaces?.activeEthernet && interfaces.ethernet)) {
      return null;
    }
    return interfaces.ethernet.find((i) => i.name === interfaces.activeEthernet) ?? null;
  }, []);

  const getWifiInterface = useCallback((): ProfileInterfaceSelection | null => {
    const interfaces = activeProfileRef.current?.config?.interfaces;
    if (!(interfaces?.activeWifi && interfaces.wifi)) {
      return null;
    }
    return interfaces.wifi.find((i) => i.name === interfaces.activeWifi) ?? null;
  }, []);

  const getAllEthernetInterfaces = useCallback(
    (): ProfileInterfaceSelection[] => activeProfileRef.current?.config?.interfaces?.ethernet ?? [],
    [],
  );

  const getAllWifiInterfaces = useCallback(
    (): ProfileInterfaceSelection[] => activeProfileRef.current?.config?.interfaces?.wifi ?? [],
    [],
  );

  /**
   * Helper to update interface config on the backend.
   */
  const updateInterfaceConfig = useCallback(
    async (
      updater: (
        interfaces: NonNullable<NonNullable<Profile["config"]>["interfaces"]>,
      ) => NonNullable<NonNullable<Profile["config"]>["interfaces"]>,
    ): Promise<boolean> => {
      const currentProfile = activeProfileRef.current;
      if (!currentProfile) {
        logger.warn(LogComponents.Profiles, "Cannot update interfaces: no active profile");
        return false;
      }

      try {
        const currentInterfaces = currentProfile.config?.interfaces ?? {
          ethernet: [],
          wifi: [],
        };
        const updatedInterfaces = updater(currentInterfaces);

        const updatedConfig = {
          ...currentProfile.config,
          interfaces: updatedInterfaces,
        };

        await api.put(`/api/profiles/${currentProfile.id}`, {
          name: currentProfile.name,
          description: currentProfile.description,
          config: updatedConfig,
        });

        // Update local state
        setActiveProfile({
          ...currentProfile,
          config: updatedConfig,
        });

        // Invalidate queries
        const queryClient = getQueryClient();
        queryClient.invalidateQueries({ queryKey: profileKeys.active() });

        return true;
      } catch (err) {
        logger.error(LogComponents.Profiles, "Failed to update interface config", err);
        return false;
      }
    },
    [setActiveProfile],
  );

  const setEthernetInterface = useCallback(
    async (name: string, enabled = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = [...(interfaces.ethernet ?? [])];
        const existingIdx = ethernet.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          ethernet[existingIdx] = { ...ethernet[existingIdx], enabled };
        } else {
          ethernet.push({ name, enabled });
        }
        return { ...interfaces, ethernet, activeEthernet: name };
      });
      if (result) {
        logger.info(LogComponents.Profiles, "Ethernet interface set as active", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const setWifiInterface = useCallback(
    async (name: string, enabled = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = [...(interfaces.wifi ?? [])];
        const existingIdx = wifi.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          wifi[existingIdx] = { ...wifi[existingIdx], enabled };
        } else {
          wifi.push({ name, enabled });
        }
        return { ...interfaces, wifi, activeWifi: name };
      });
      if (result) {
        logger.info(LogComponents.Profiles, "Wifi interface set as active", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const addEthernetInterface = useCallback(
    async (name: string, enabled = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = [...(interfaces.ethernet ?? [])];
        const existingIdx = ethernet.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          ethernet[existingIdx] = { ...ethernet[existingIdx], enabled };
        } else {
          ethernet.push({ name, enabled });
        }
        return { ...interfaces, ethernet };
      });
      if (result) {
        logger.info(LogComponents.Profiles, "Ethernet interface added", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const addWifiInterface = useCallback(
    async (name: string, enabled = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = [...(interfaces.wifi ?? [])];
        const existingIdx = wifi.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          wifi[existingIdx] = { ...wifi[existingIdx], enabled };
        } else {
          wifi.push({ name, enabled });
        }
        return { ...interfaces, wifi };
      });
      if (result) {
        logger.info(LogComponents.Profiles, "Wifi interface added", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const removeEthernetInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = (interfaces.ethernet ?? []).filter((i) => i.name !== name);
        const activeEthernetVal =
          interfaces.activeEthernet === name ? "" : interfaces.activeEthernet;
        return { ...interfaces, ethernet, activeEthernet: activeEthernetVal };
      });
      if (result) {
        logger.info(LogComponents.Profiles, "Ethernet interface removed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const removeWifiInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = (interfaces.wifi ?? []).filter((i) => i.name !== name);
        const activeWifiVal = interfaces.activeWifi === name ? "" : interfaces.activeWifi;
        return { ...interfaces, wifi, activeWifi: activeWifiVal };
      });
      if (result) {
        logger.info(LogComponents.Profiles, "Wifi interface removed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const setActiveEthernetInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const currentProfile = activeProfileRef.current;
      const exists = (currentProfile?.config?.interfaces?.ethernet ?? []).some(
        (i) => i.name === name,
      );
      if (!exists) {
        logger.warn(
          LogComponents.Profiles,
          "Cannot set active ethernet interface: interface not in list",
          { interface: name },
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        activeEthernet: name,
      }));
      if (result) {
        logger.info(LogComponents.Profiles, "Active ethernet interface changed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  const setActiveWifiInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const currentProfile = activeProfileRef.current;
      const exists = (currentProfile?.config?.interfaces?.wifi ?? []).some((i) => i.name === name);
      if (!exists) {
        logger.warn(
          LogComponents.Profiles,
          "Cannot set active Wifi interface: interface not in list",
          { interface: name },
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        activeWifi: name,
      }));
      if (result) {
        logger.info(LogComponents.Profiles, "Active Wifi interface changed", {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  // ============================================================================
  // Context Value
  // ============================================================================

  const contextValue: ProfileContextValue = {
    // Profile state (from Zustand)
    profiles,
    activeProfile,
    isLoading,
    error,

    // Settings state (from Zustand selectors)
    // Order matches Settings Drawer UI for consistency
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

    // Profile list/fetch operations
    refreshProfiles,
    refreshActiveProfile,

    // Profile CRUD operations
    createProfile,
    updateProfile,
    deleteProfile,
    switchProfile,
    duplicateProfile,

    // Profile import/export
    importProfiles,
    exportProfiles,
    downloadProfiles,

    // Settings update methods - order matches Settings Drawer UI
    updateLinkSettings,
    updateCableTestSettings,
    updateDisplayOptions,
    updateWifiSettings,
    updateDnsSettings,
    updateTestsSettings,
    updateSpeedtestSettings,
    updateIperfSettings,
    updateNetworkDiscoverySettings,
    updateSnmpSettings,
    updateVulnerabilitySettings,
    updateThresholds,
    updateAppearanceSettings,
    updateCardSettings,
    updateSettings,
    refreshSettings,

    // Interface selection helpers
    getEthernetInterface,
    getWifiInterface,
    getAllEthernetInterfaces,
    getAllWifiInterfaces,
    setEthernetInterface,
    setWifiInterface,
    addEthernetInterface,
    addWifiInterface,
    removeEthernetInterface,
    removeWifiInterface,
    setActiveEthernetInterface,
    setActiveWifiInterface,
  };

  return <PROFILE_CONTEXT.Provider value={contextValue}>{children}</PROFILE_CONTEXT.Provider>;
}

// ============================================================================
// Hook
// ============================================================================

/**
 * Hook to access the profile context.
 * Must be used within a ProfileProvider.
 */
export function useProfileContext(): ProfileContextValue {
  const context = useContext(PROFILE_CONTEXT);
  if (context === undefined) {
    throw new Error("useProfileContext must be used within a ProfileProvider");
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
