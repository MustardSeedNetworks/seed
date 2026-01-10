/**
 * Profile Management Hook
 *
 * Manages MSP profile operations including CRUD, switching active profiles,
 * and import/export functionality (#754).
 *
 * Features:
 * - List all profiles
 * - Create, update, and delete profiles
 * - Get and set active profile
 * - Duplicate profiles
 * - Import/export profiles
 *
 * Usage:
 * ```typescript
 * const { profiles, activeProfile, createProfile, switchProfile } = useProfiles();
 *
 * // Create a new profile
 * await createProfile({ name: 'Client A', description: 'Client A config' });
 *
 * // Switch to a different profile
 * await switchProfile('profile-id');
 * ```
 */

import { useCallback, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";
import type {
  ActiveProfileResponse,
  Profile,
  ProfileExportResponse,
  ProfileImportRequest,
  ProfileImportResponse,
  ProfileListResponse,
  ProfileRequest,
} from "../types/profile";

/**
 * Custom hook for managing profile operations.
 *
 * Provides functions to manage profiles including CRUD operations,
 * switching, duplication, and import/export.
 *
 * @returns Profile state and control functions
 */
export function useProfiles() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [activeProfile, setActiveProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetches all profiles.
   */
  const fetchProfiles = useCallback(async (): Promise<Profile[]> => {
    try {
      setError(null);
      setIsLoading(true);
      const data = await api.get<ProfileListResponse>("/api/v1/profiles");
      setProfiles(data.profiles || []);
      return data.profiles || [];
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch profiles";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to fetch profiles", err, {
        endpoint: "/api/v1/profiles",
      });
      return [];
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Fetches a single profile by ID.
   */
  const fetchProfile = useCallback(async (id: string): Promise<Profile | null> => {
    try {
      setError(null);
      return await api.get<Profile>(`/api/profiles/${id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch profile";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to fetch profile", err, {
        endpoint: `/api/profiles/${id}`,
        profileId: id,
      });
      return null;
    }
  }, []);

  /**
   * Fetches the currently active profile.
   */
  const fetchActiveProfile = useCallback(async (): Promise<Profile | null> => {
    try {
      setError(null);
      const profile = await api.get<Profile>("/api/v1/profiles/active");
      setActiveProfile(profile);
      return profile;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch active profile";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to fetch active profile", err, {
        endpoint: "/api/v1/profiles/active",
      });
      return null;
    }
  }, []);

  /**
   * Creates a new profile.
   */
  const createProfile = useCallback(async (profile: ProfileRequest): Promise<Profile | null> => {
    try {
      setError(null);
      setIsLoading(true);
      const created = await api.post<Profile>("/api/v1/profiles", profile);
      setProfiles((prev) => [...prev, created]);
      logger.info(LogComponents.Profiles, "Profile created", {
        profileId: created.id,
        name: created.name,
      });
      return created;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create profile";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to create profile", err, {
        endpoint: "/api/v1/profiles",
        name: profile.name,
      });
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Updates an existing profile.
   */
  const updateProfile = useCallback(
    async (id: string, profile: ProfileRequest): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const updated = await api.put<Profile>(`/api/profiles/${id}`, profile);
        setProfiles((prev) => prev.map((p) => (p.id === id ? updated : p)));
        if (activeProfile?.id === id) {
          setActiveProfile(updated);
        }
        logger.info(LogComponents.Profiles, "Profile updated", {
          profileId: id,
          name: updated.name,
        });
        return updated;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to update profile";
        setError(message);
        logger.error(LogComponents.Profiles, "Failed to update profile", err, {
          endpoint: `/api/profiles/${id}`,
          profileId: id,
        });
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [activeProfile],
  );

  /**
   * Deletes a profile.
   */
  const deleteProfile = useCallback(async (id: string): Promise<boolean> => {
    try {
      setError(null);
      setIsLoading(true);
      await api.delete(`/api/profiles/${id}`);
      setProfiles((prev) => prev.filter((p) => p.id !== id));
      logger.info(LogComponents.Profiles, "Profile deleted", { profileId: id });
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete profile";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to delete profile", err, {
        endpoint: `/api/profiles/${id}`,
        profileId: id,
      });
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Switches to a different profile.
   */
  const switchProfile = useCallback(async (profileId: string): Promise<boolean> => {
    try {
      setError(null);
      setIsLoading(true);
      const result = await api.post<ActiveProfileResponse>("/api/v1/profiles/active", {
        profileId: profileId,
      });
      setActiveProfile(result.profile);
      logger.info(LogComponents.Profiles, "Profile switched", {
        profileId,
        name: result.profile.name,
      });
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to switch profile";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to switch profile", err, {
        endpoint: "/api/v1/profiles/active",
        profileId,
      });
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Duplicates an existing profile.
   */
  const duplicateProfile = useCallback(
    async (id: string, newName?: string): Promise<Profile | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const duplicated = await api.post<Profile>(`/api/profiles/${id}/duplicate`, {
          name: newName,
        });
        setProfiles((prev) => [...prev, duplicated]);
        logger.info(LogComponents.Profiles, "Profile duplicated", {
          sourceId: id,
          newId: duplicated.id,
          name: duplicated.name,
        });
        return duplicated;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to duplicate profile";
        setError(message);
        logger.error(LogComponents.Profiles, "Failed to duplicate profile", err, {
          endpoint: `/api/profiles/${id}/duplicate`,
          profileId: id,
        });
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  /**
   * Imports profiles from JSON.
   */
  const importProfiles = useCallback(
    async (request: ProfileImportRequest): Promise<ProfileImportResponse | null> => {
      try {
        setError(null);
        setIsLoading(true);
        const result = await api.post<ProfileImportResponse>("/api/v1/profiles/import", request);
        // Refresh the profile list after import
        await fetchProfiles();
        logger.info(LogComponents.Profiles, "Profiles imported", {
          created: result.created,
          updated: result.updated,
          skipped: result.skipped,
        });
        return result;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to import profiles";
        setError(message);
        logger.error(LogComponents.Profiles, "Failed to import profiles", err, {
          endpoint: "/api/v1/profiles/import",
        });
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [fetchProfiles],
  );

  /**
   * Exports all profiles to JSON.
   */
  const exportProfiles = useCallback(async (): Promise<ProfileExportResponse | null> => {
    try {
      setError(null);
      const result = await api.get<ProfileExportResponse>("/api/v1/profiles/export");
      logger.info(LogComponents.Profiles, "Profiles exported", {
        count: result.profiles.length,
      });
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to export profiles";
      setError(message);
      logger.error(LogComponents.Profiles, "Failed to export profiles", err, {
        endpoint: "/api/v1/profiles/export",
      });
      return null;
    }
  }, []);

  /**
   * Downloads profiles as a JSON file.
   */
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
      logger.error(LogComponents.Profiles, "Failed to download profiles", err);
      return false;
    }
  }, [exportProfiles]);

  return {
    // State
    profiles,
    activeProfile,
    isLoading,
    error,

    // List/fetch operations
    fetchProfiles,
    fetchProfile,
    fetchActiveProfile,

    // CRUD operations
    createProfile,
    updateProfile,
    deleteProfile,

    // Profile switching
    switchProfile,

    // Duplication
    duplicateProfile,

    // Import/Export
    importProfiles,
    exportProfiles,
    downloadProfiles,
  };
}
