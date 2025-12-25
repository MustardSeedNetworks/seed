/**
 * ProfileContext - MSP Profile Management
 *
 * Provides a React Context for managing MSP profiles across the application.
 * Profiles enable client-specific configurations including health checks,
 * thresholds, and discovery settings (#754).
 *
 * Features:
 * - Load and cache profiles
 * - Track active profile
 * - Handle profile switching
 * - Listen for WebSocket profile change events
 */

import {
  createContext,
  useState,
  useCallback,
  useEffect,
  useContext,
  useRef,
  ReactNode,
} from "react";
import { api } from "../lib/api";
import { logger, LogComponents } from "../lib/logger";
import type {
  Profile,
  ProfileListResponse,
  ProfileRequest,
  ProfileImportRequest,
  ProfileImportResponse,
  ProfileExportResponse,
  ActiveProfileResponse,
  ProfileInterfaceSelection,
} from "../types/profile";

// ============================================================================
// Context Type Definition
// ============================================================================

export interface ProfileContextValue {
  // State
  profiles: Profile[];
  activeProfile: Profile | null;
  isLoading: boolean;
  error: string | null;

  // List/fetch operations
  refreshProfiles: () => Promise<void>;
  refreshActiveProfile: () => Promise<void>;

  // CRUD operations
  createProfile: (profile: ProfileRequest) => Promise<Profile | null>;
  updateProfile: (
    id: string,
    profile: ProfileRequest
  ) => Promise<Profile | null>;
  deleteProfile: (id: string) => Promise<boolean>;

  // Profile switching
  switchProfile: (profileId: string) => Promise<boolean>;

  // Duplication
  duplicateProfile: (id: string, newName?: string) => Promise<Profile | null>;

  // Import/Export
  importProfiles: (
    request: ProfileImportRequest
  ) => Promise<ProfileImportResponse | null>;
  exportProfiles: () => Promise<ProfileExportResponse | null>;
  downloadProfiles: () => Promise<boolean>;

  // Interface selection helpers (multi-interface support)
  /** Get the active ethernet interface from the active profile */
  getEthernetInterface: () => ProfileInterfaceSelection | null;
  /** Get the active WiFi interface from the active profile */
  getWifiInterface: () => ProfileInterfaceSelection | null;
  /** Get all ethernet interfaces from the active profile */
  getAllEthernetInterfaces: () => ProfileInterfaceSelection[];
  /** Get all WiFi interfaces from the active profile */
  getAllWiFiInterfaces: () => ProfileInterfaceSelection[];
  /** Add or update an ethernet interface and set it as active */
  setEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add or update a WiFi interface and set it as active */
  setWifiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add an ethernet interface without changing the active one */
  addEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Add a WiFi interface without changing the active one */
  addWiFiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  /** Remove an ethernet interface */
  removeEthernetInterface: (name: string) => Promise<boolean>;
  /** Remove a WiFi interface */
  removeWiFiInterface: (name: string) => Promise<boolean>;
  /** Set the active ethernet interface (must already be in the list) */
  setActiveEthernetInterface: (name: string) => Promise<boolean>;
  /** Set the active WiFi interface (must already be in the list) */
  setActiveWiFiInterface: (name: string) => Promise<boolean>;
}

// Create context with undefined default to enforce provider requirement
const ProfileContext = createContext<ProfileContextValue | undefined>(
  undefined
);

// ============================================================================
// Provider Component
// ============================================================================

interface ProfileProviderProps {
  children: ReactNode;
}

/**
 * Context provider that manages profile state and API synchronization.
 */
export function ProfileProvider({ children }: ProfileProviderProps) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [activeProfile, setActiveProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isMountedRef = useRef(true);

  // ============================================================================
  // Load Profiles
  // ============================================================================

  const refreshProfiles = useCallback(async () => {
    try {
      setError(null);
      setIsLoading(true);
      const data = await api.get<ProfileListResponse>("/api/profiles");
      if (isMountedRef.current) {
        setProfiles(data.profiles || []);
      }
    } catch (err) {
      if (isMountedRef.current) {
        // Don't show session expired errors in profile panel - handled globally
        const message =
          err instanceof Error ? err.message : "Failed to fetch profiles";
        if (!message.toLowerCase().includes("session")) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to fetch profiles", err);
      }
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, []);

  const refreshActiveProfile = useCallback(async () => {
    try {
      setError(null);
      const profile = await api.get<Profile>("/api/profiles/active");
      if (isMountedRef.current) {
        setActiveProfile(profile);
      }
    } catch (err) {
      if (isMountedRef.current) {
        // Active profile may not exist yet, which is okay
        // Don't show session expired errors - handled globally
        const message =
          err instanceof Error ? err.message : "Failed to fetch active profile";
        if (
          !(err instanceof Error && err.message.includes("404")) &&
          !message.toLowerCase().includes("session")
        ) {
          setError(message);
          logger.error(
            LogComponents.PROFILES,
            "Failed to fetch active profile",
            err
          );
        }
      }
    }
  }, []);

  // ============================================================================
  // CRUD Operations
  // ============================================================================

  const createProfile = useCallback(
    async (profile: ProfileRequest): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const created = await api.post<Profile>("/api/profiles", profile);
        if (isMountedRef.current) {
          setProfiles((prev) => [...prev, created]);
        }
        logger.info(LogComponents.PROFILES, "Profile created", {
          profileId: created.id,
          name: created.name,
        });
        return created;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to create profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to create profile", err);
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    []
  );

  const updateProfile = useCallback(
    async (id: string, profile: ProfileRequest): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const updated = await api.put<Profile>(`/api/profiles/${id}`, profile);
        if (isMountedRef.current) {
          setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
          if (activeProfile?.id === id) {
            setActiveProfile(updated);
          }
        }
        logger.info(LogComponents.PROFILES, "Profile updated", {
          profileId: id,
          name: updated.name,
        });
        return updated;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to update profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to update profile", err);
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [activeProfile]
  );

  const deleteProfile = useCallback(async (id: string): Promise<boolean> => {
    try {
      setError(null);
      setIsLoading(true);
      await api.delete(`/api/profiles/${id}`);
      if (isMountedRef.current) {
        setProfiles((prev) => prev.filter((p) => p.id !== id));
      }
      logger.info(LogComponents.PROFILES, "Profile deleted", { profileId: id });
      return true;
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to delete profile";
      if (isMountedRef.current) {
        setError(message);
      }
      logger.error(LogComponents.PROFILES, "Failed to delete profile", err);
      return false;
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, []);

  // ============================================================================
  // Profile Switching
  // ============================================================================

  const switchProfile = useCallback(
    async (profileId: string): Promise<boolean> => {
      try {
        setError(null);
        setIsLoading(true);
        const result = await api.post<ActiveProfileResponse>(
          "/api/profiles/active",
          {
            profile_id: profileId,
          }
        );
        if (isMountedRef.current) {
          setActiveProfile(result.profile);
        }
        logger.info(LogComponents.PROFILES, "Profile switched", {
          profileId,
          name: result.profile.name,
        });
        return true;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to switch profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to switch profile", err);
        return false;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    []
  );

  // ============================================================================
  // Duplication
  // ============================================================================

  const duplicateProfile = useCallback(
    async (id: string, newName?: string): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const duplicated = await api.post<Profile>(
          `/api/profiles/${id}/duplicate`,
          {
            name: newName,
          }
        );
        if (isMountedRef.current) {
          setProfiles((prev) => [...prev, duplicated]);
        }
        logger.info(LogComponents.PROFILES, "Profile duplicated", {
          sourceId: id,
          newId: duplicated.id,
          name: duplicated.name,
        });
        return duplicated;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to duplicate profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(
          LogComponents.PROFILES,
          "Failed to duplicate profile",
          err
        );
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    []
  );

  // ============================================================================
  // Import/Export
  // ============================================================================

  const importProfiles = useCallback(
    async (
      request: ProfileImportRequest
    ): Promise<ProfileImportResponse | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const result = await api.post<ProfileImportResponse>(
          "/api/profiles/import",
          request
        );
        // Refresh the profile list after import
        await refreshProfiles();
        logger.info(LogComponents.PROFILES, "Profiles imported", {
          created: result.created,
          updated: result.updated,
          skipped: result.skipped,
        });
        return result;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to import profiles";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to import profiles", err);
        return null;
      } finally {
        if (isMountedRef.current) {
          setIsLoading(false);
        }
      }
    },
    [refreshProfiles]
  );

  const exportProfiles =
    useCallback(async (): Promise<ProfileExportResponse | null> => {
      try {
        setError(null);
        const result = await api.get<ProfileExportResponse>(
          "/api/profiles/export"
        );
        logger.info(LogComponents.PROFILES, "Profiles exported", {
          count: result.profiles.length,
        });
        return result;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to export profiles";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to export profiles", err);
        return null;
      }
    }, []);

  const downloadProfiles = useCallback(async (): Promise<boolean> => {
    try {
      const result = await exportProfiles();
      if (!result) return false;

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
      logger.error(LogComponents.PROFILES, "Failed to download profiles", err);
      return false;
    }
  }, [exportProfiles]);

  // ============================================================================
  // Interface Selection Helpers
  // ============================================================================

  /**
   * Get the currently active ethernet interface from the active profile.
   */
  const getEthernetInterface =
    useCallback((): ProfileInterfaceSelection | null => {
      const interfaces = activeProfile?.config?.interfaces;
      if (!interfaces?.active_ethernet || !interfaces.ethernet) return null;
      return (
        interfaces.ethernet.find((i) => i.name === interfaces.active_ethernet) ??
        null
      );
    }, [activeProfile]);

  /**
   * Get the currently active WiFi interface from the active profile.
   */
  const getWifiInterface = useCallback((): ProfileInterfaceSelection | null => {
    const interfaces = activeProfile?.config?.interfaces;
    if (!interfaces?.active_wifi || !interfaces.wifi) return null;
    return (
      interfaces.wifi.find((i) => i.name === interfaces.active_wifi) ?? null
    );
  }, [activeProfile]);

  /**
   * Get all ethernet interfaces from the active profile.
   */
  const getAllEthernetInterfaces = useCallback((): ProfileInterfaceSelection[] => {
    return activeProfile?.config?.interfaces?.ethernet ?? [];
  }, [activeProfile]);

  /**
   * Get all WiFi interfaces from the active profile.
   */
  const getAllWiFiInterfaces = useCallback((): ProfileInterfaceSelection[] => {
    return activeProfile?.config?.interfaces?.wifi ?? [];
  }, [activeProfile]);

  /**
   * Helper to update interface config on the backend.
   */
  const updateInterfaceConfig = useCallback(
    async (
      updater: (
        interfaces: NonNullable<
          NonNullable<Profile["config"]>["interfaces"]
        >
      ) => NonNullable<NonNullable<Profile["config"]>["interfaces"]>
    ): Promise<boolean> => {
      if (!activeProfile) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot update interfaces: no active profile"
        );
        return false;
      }

      try {
        const currentInterfaces = activeProfile.config?.interfaces ?? {
          ethernet: [],
          wifi: [],
        };
        const updatedInterfaces = updater(currentInterfaces);

        const updatedConfig = {
          ...activeProfile.config,
          interfaces: updatedInterfaces,
        };

        await api.put(`/api/profiles/${activeProfile.id}`, {
          name: activeProfile.name,
          description: activeProfile.description,
          config: updatedConfig,
        });

        await refreshActiveProfile();
        return true;
      } catch (err) {
        logger.error(
          LogComponents.PROFILES,
          "Failed to update interface config",
          err
        );
        return false;
      }
    },
    [activeProfile, refreshActiveProfile]
  );

  /**
   * Add or update an ethernet interface and set it as active.
   */
  const setEthernetInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = [...(interfaces.ethernet ?? [])];
        const existingIdx = ethernet.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          ethernet[existingIdx] = { ...ethernet[existingIdx], enabled };
        } else {
          ethernet.push({ name, enabled });
        }
        return { ...interfaces, ethernet, active_ethernet: name };
      });
      if (result) {
        logger.info(LogComponents.PROFILES, "Ethernet interface set as active", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Add or update a WiFi interface and set it as active.
   */
  const setWifiInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = [...(interfaces.wifi ?? [])];
        const existingIdx = wifi.findIndex((i) => i.name === name);
        if (existingIdx >= 0) {
          wifi[existingIdx] = { ...wifi[existingIdx], enabled };
        } else {
          wifi.push({ name, enabled });
        }
        return { ...interfaces, wifi, active_wifi: name };
      });
      if (result) {
        logger.info(LogComponents.PROFILES, "WiFi interface set as active", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Add an ethernet interface without changing the active one.
   */
  const addEthernetInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
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
        logger.info(LogComponents.PROFILES, "Ethernet interface added", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Add a WiFi interface without changing the active one.
   */
  const addWiFiInterface = useCallback(
    async (name: string, enabled: boolean = true): Promise<boolean> => {
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
        logger.info(LogComponents.PROFILES, "WiFi interface added", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Remove an ethernet interface.
   */
  const removeEthernetInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const ethernet = (interfaces.ethernet ?? []).filter(
          (i) => i.name !== name
        );
        const active_ethernet =
          interfaces.active_ethernet === name ? "" : interfaces.active_ethernet;
        return { ...interfaces, ethernet, active_ethernet };
      });
      if (result) {
        logger.info(LogComponents.PROFILES, "Ethernet interface removed", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Remove a WiFi interface.
   */
  const removeWiFiInterface = useCallback(
    async (name: string): Promise<boolean> => {
      const result = await updateInterfaceConfig((interfaces) => {
        const wifi = (interfaces.wifi ?? []).filter((i) => i.name !== name);
        const active_wifi =
          interfaces.active_wifi === name ? "" : interfaces.active_wifi;
        return { ...interfaces, wifi, active_wifi };
      });
      if (result) {
        logger.info(LogComponents.PROFILES, "WiFi interface removed", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Set the active ethernet interface (must already be in the list).
   */
  const setActiveEthernetInterface = useCallback(
    async (name: string): Promise<boolean> => {
      // Check if the interface exists in the list
      const exists = (activeProfile?.config?.interfaces?.ethernet ?? []).some(
        (i) => i.name === name
      );
      if (!exists) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot set active ethernet interface: interface not in list",
          { interface: name }
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        active_ethernet: name,
      }));
      if (result) {
        logger.info(LogComponents.PROFILES, "Active ethernet interface changed", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  /**
   * Set the active WiFi interface (must already be in the list).
   */
  const setActiveWiFiInterface = useCallback(
    async (name: string): Promise<boolean> => {
      // Check if the interface exists in the list
      const exists = (activeProfile?.config?.interfaces?.wifi ?? []).some(
        (i) => i.name === name
      );
      if (!exists) {
        logger.warn(
          LogComponents.PROFILES,
          "Cannot set active WiFi interface: interface not in list",
          { interface: name }
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        active_wifi: name,
      }));
      if (result) {
        logger.info(LogComponents.PROFILES, "Active WiFi interface changed", {
          profileId: activeProfile?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig, activeProfile]
  );

  // ============================================================================
  // Initial Load
  // ============================================================================

  useEffect(() => {
    isMountedRef.current = true;

    // Load profiles and active profile on mount
    const loadInitialData = async () => {
      await Promise.all([refreshProfiles(), refreshActiveProfile()]);
    };

    loadInitialData();

    return () => {
      isMountedRef.current = false;
    };
  }, [refreshProfiles, refreshActiveProfile]);

  // ============================================================================
  // Context Value
  // ============================================================================

  const contextValue: ProfileContextValue = {
    profiles,
    activeProfile,
    isLoading,
    error,
    refreshProfiles,
    refreshActiveProfile,
    createProfile,
    updateProfile,
    deleteProfile,
    switchProfile,
    duplicateProfile,
    importProfiles,
    exportProfiles,
    downloadProfiles,
    getEthernetInterface,
    getWifiInterface,
    getAllEthernetInterfaces,
    getAllWiFiInterfaces,
    setEthernetInterface,
    setWifiInterface,
    addEthernetInterface,
    addWiFiInterface,
    removeEthernetInterface,
    removeWiFiInterface,
    setActiveEthernetInterface,
    setActiveWiFiInterface,
  };

  return (
    <ProfileContext.Provider value={contextValue}>
      {children}
    </ProfileContext.Provider>
  );
}

// ============================================================================
// Hook
// ============================================================================

/**
 * Hook to access the profile context.
 * Must be used within a ProfileProvider.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useProfileContext(): ProfileContextValue {
  const context = useContext(ProfileContext);
  if (context === undefined) {
    throw new Error("useProfileContext must be used within a ProfileProvider");
  }
  return context;
}

// Note: ProfileContext is intentionally not exported to keep fast-refresh happy.
