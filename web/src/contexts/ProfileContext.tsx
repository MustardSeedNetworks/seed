/* eslint-disable react-refresh/only-export-components */
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
}

// Create context with undefined default to enforce provider requirement
const ProfileContext = createContext<ProfileContextValue | undefined>(undefined);

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
        const message = err instanceof Error ? err.message : "Failed to fetch profiles";
        setError(message);
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
        if (!(err instanceof Error && err.message.includes("404"))) {
          const message = err instanceof Error ? err.message : "Failed to fetch active profile";
          setError(message);
          logger.error(LogComponents.PROFILES, "Failed to fetch active profile", err);
        }
      }
    }
  }, []);

  // ============================================================================
  // CRUD Operations
  // ============================================================================

  const createProfile = useCallback(async (profile: ProfileRequest): Promise<Profile | null> => {
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
      const message = err instanceof Error ? err.message : "Failed to create profile";
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
  }, []);

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
        const message = err instanceof Error ? err.message : "Failed to update profile";
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
      const message = err instanceof Error ? err.message : "Failed to delete profile";
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

  const switchProfile = useCallback(async (profileId: string): Promise<boolean> => {
    try {
      setError(null);
      setIsLoading(true);
      const result = await api.post<ActiveProfileResponse>("/api/profiles/active", {
        profile_id: profileId,
      });
      if (isMountedRef.current) {
        setActiveProfile(result.profile);
      }
      logger.info(LogComponents.PROFILES, "Profile switched", {
        profileId,
        name: result.profile.name,
      });
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to switch profile";
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
  }, []);

  // ============================================================================
  // Duplication
  // ============================================================================

  const duplicateProfile = useCallback(
    async (id: string, newName?: string): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const duplicated = await api.post<Profile>(`/api/profiles/${id}/duplicate`, {
          name: newName,
        });
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
        const message = err instanceof Error ? err.message : "Failed to duplicate profile";
        if (isMountedRef.current) {
          setError(message);
        }
        logger.error(LogComponents.PROFILES, "Failed to duplicate profile", err);
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
    async (request: ProfileImportRequest): Promise<ProfileImportResponse | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const result = await api.post<ProfileImportResponse>("/api/profiles/import", request);
        // Refresh the profile list after import
        await refreshProfiles();
        logger.info(LogComponents.PROFILES, "Profiles imported", {
          created: result.created,
          updated: result.updated,
          skipped: result.skipped,
        });
        return result;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to import profiles";
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

  const exportProfiles = useCallback(async (): Promise<ProfileExportResponse | null> => {
    try {
      setError(null);
      const result = await api.get<ProfileExportResponse>("/api/profiles/export");
      logger.info(LogComponents.PROFILES, "Profiles exported", {
        count: result.profiles.length,
      });
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to export profiles";
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
  };

  return <ProfileContext.Provider value={contextValue}>{children}</ProfileContext.Provider>;
}

// ============================================================================
// Hook
// ============================================================================

/**
 * Hook to access the profile context.
 * Must be used within a ProfileProvider.
 */
export function useProfileContext(): ProfileContextValue {
  const context = useContext(ProfileContext);
  if (context === undefined) {
    throw new Error("useProfileContext must be used within a ProfileProvider");
  }
  return context;
}

// Note: ProfileContext is intentionally not exported to keep fast-refresh happy.
