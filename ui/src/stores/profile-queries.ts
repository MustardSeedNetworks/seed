// biome-ignore-all lint/style/noInferrableTypes: useExplicitType requires types on default params
/**
 * Profile Queries - React Query Hooks for Profile API
 *
 * Provides cached, deduplicated API calls with automatic
 * background refetching and stale-while-revalidate patterns.
 *
 * Benefits:
 * - Automatic request deduplication
 * - Built-in caching with configurable stale time
 * - Background refetching
 * - Optimistic updates support
 *
 * Note: Using React Query v5 patterns - no onSuccess/onError in useQuery.
 * State sync to Zustand store happens via useEffect.
 *
 * Related: #890
 */

import type { UseMutationResult, UseQueryResult } from "@tanstack/react-query";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { api } from "../api";
import { LogComponents, logger } from "../lib/logger";
import type { DefaultSettings } from "../types/defaults";
import type {
  Profile,
  ProfileExportResponse,
  ProfileImportRequest,
  ProfileImportResponse,
  ProfileListResponse,
  ProfileRequest,
} from "../types/profile";
import { useProfileStore } from "./profile-store";

// ============================================================================
// Query Keys
// ============================================================================

export const profileKeys = {
  all: ["profiles"] as const,
  lists: () => [...profileKeys.all, "list"] as const,
  list: () => [...profileKeys.lists()] as const,
  active: () => [...profileKeys.all, "active"] as const,
  details: () => [...profileKeys.all, "detail"] as const,
  detail: (id: string) => [...profileKeys.details(), id] as const,
  defaults: () => [...profileKeys.all, "defaults"] as const,
};

// ============================================================================
// Query Hooks
// ============================================================================

/**
 * Fetch all profiles with caching.
 * Stale time: 30 seconds (profiles don't change often).
 */
export function useProfilesQuery(): UseQueryResult<Profile[], Error> {
  const setProfiles = useProfileStore((s) => s.setProfiles);
  const setError = useProfileStore((s) => s.setError);
  const setIsLoading = useProfileStore((s) => s.setIsLoading);

  const query = useQuery({
    queryKey: profileKeys.list(),
    queryFn: async () => {
      const data = await api.get<ProfileListResponse>("/api/v1/profiles");
      return data.profiles || [];
    },
    staleTime: 30 * 1000, // 30 seconds
  });

  // React Query v5: Sync state via useEffect instead of callbacks
  useEffect(() => {
    if (query.isSuccess && query.data) {
      setProfiles(query.data);
      setError(null);
    }
  }, [query.isSuccess, query.data, setProfiles, setError]);

  useEffect(() => {
    if (query.isError && query.error) {
      const message =
        query.error instanceof Error ? query.error.message : "Failed to fetch profiles";
      if (!message.toLowerCase().includes("session")) {
        setError(message);
      }
      logger.error(LogComponents.Profiles, "Failed to fetch profiles", query.error);
    }
  }, [query.isError, query.error, setError]);

  useEffect(() => {
    if (!query.isFetching) {
      setIsLoading(false);
    }
  }, [query.isFetching, setIsLoading]);

  return query;
}

/**
 * Fetch active profile with caching.
 * Stale time: 10 seconds (active profile may change more often).
 */
export function useActiveProfileQuery(): UseQueryResult<Profile, Error> {
  const setActiveProfile = useProfileStore((s) => s.setActiveProfile);
  const setIsSettingsLoaded = useProfileStore((s) => s.setIsSettingsLoaded);
  const setError = useProfileStore((s) => s.setError);

  const query = useQuery({
    queryKey: profileKeys.active(),
    queryFn: async () => {
      const profile = await api.get<Profile>("/api/v1/profiles/active");
      return profile;
    },
    staleTime: 10 * 1000, // 10 seconds
    retry: (failureCount: number, error: Error) => {
      // Don't retry on 404 - profile may not exist yet
      if (error instanceof Error && error.message.includes("404")) {
        return false;
      }
      return failureCount < 3;
    },
  });

  // React Query v5: Sync state via useEffect instead of callbacks
  useEffect(() => {
    if (query.isSuccess && query.data) {
      setActiveProfile(query.data);
      setIsSettingsLoaded(true);
      setError(null);
    }
  }, [query.isSuccess, query.data, setActiveProfile, setIsSettingsLoaded, setError]);

  useEffect(() => {
    if (query.isError && query.error) {
      // Active profile may not exist yet, which is okay
      if (query.error instanceof Error && query.error.message.includes("404")) {
        setIsSettingsLoaded(true);
        return;
      }
      const message =
        query.error instanceof Error ? query.error.message : "Failed to fetch active profile";
      if (!message.toLowerCase().includes("session")) {
        setError(message);
        logger.error(LogComponents.Profiles, "Failed to fetch active profile", query.error);
      }
      setIsSettingsLoaded(true);
    }
  }, [query.isError, query.error, setError, setIsSettingsLoaded]);

  return query;
}

/**
 * Fetch backend defaults with longer cache.
 * Stale time: 5 minutes (defaults rarely change).
 */
export function useBackendDefaultsQuery(): UseQueryResult<DefaultSettings, Error> {
  const setBackendDefaults = useProfileStore((s) => s.setBackendDefaults);

  const query = useQuery({
    queryKey: profileKeys.defaults(),
    queryFn: async () => {
      const defaults = await api.get<DefaultSettings>("/api/v1/settings/defaults");
      return defaults;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  // React Query v5: Sync state via useEffect instead of callbacks
  useEffect(() => {
    if (query.isSuccess && query.data) {
      setBackendDefaults(query.data);
    }
  }, [query.isSuccess, query.data, setBackendDefaults]);

  useEffect(() => {
    if (query.isError && query.error) {
      logger.warn(
        LogComponents.Profiles,
        "Could not load backend defaults, using hardcoded",
        query.error,
      );
    }
  }, [query.isError, query.error]);

  return query;
}

// ============================================================================
// Mutation Hooks
// ============================================================================

/**
 * Create a new profile.
 */
export function useCreateProfileMutation(): UseMutationResult<Profile, Error, ProfileRequest> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (profile: ProfileRequest) => {
      const created = await api.post<Profile>("/api/v1/profiles", profile);
      logger.info(LogComponents.Profiles, "Profile created", {
        id: created.id,
        name: created.name,
      });
      return created;
    },
    onSuccess: () => {
      // Invalidate profile list to refetch
      queryClient.invalidateQueries({ queryKey: profileKeys.list() });
    },
    onError: (err: Error) => {
      logger.error(LogComponents.Profiles, "Failed to create profile", err);
    },
  });
}

/**
 * Update an existing profile.
 */
export function useUpdateProfileMutation(): UseMutationResult<
  Profile,
  Error,
  { id: string; profile: ProfileRequest }
> {
  const queryClient = useQueryClient();
  const setActiveProfile = useProfileStore((s) => s.setActiveProfile);
  const activeProfile = useProfileStore((s) => s.activeProfile);

  return useMutation({
    mutationFn: async ({ id, profile }: { id: string; profile: ProfileRequest }) => {
      const updated = await api.put<Profile>(`/api/v1/profiles/${id}`, profile);
      logger.info(LogComponents.Profiles, "Profile updated", {
        id: updated.id,
        name: updated.name,
      });
      return updated;
    },
    onSuccess: (updated: Profile) => {
      // Update active profile if it was the one updated
      if (activeProfile?.id === updated.id) {
        setActiveProfile(updated);
      }
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: profileKeys.list() });
      queryClient.invalidateQueries({ queryKey: profileKeys.detail(updated.id) });
    },
    onError: (err: Error) => {
      logger.error(LogComponents.Profiles, "Failed to update profile", err);
    },
  });
}

/**
 * Delete a profile.
 */
export function useDeleteProfileMutation(): UseMutationResult<string, Error, string> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/v1/profiles/${id}`);
      logger.info(LogComponents.Profiles, "Profile deleted", { id });
      return id;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profileKeys.list() });
      queryClient.invalidateQueries({ queryKey: profileKeys.active() });
    },
    onError: (err: Error) => {
      logger.error(LogComponents.Profiles, "Failed to delete profile", err);
    },
  });
}

/**
 * Switch active profile - batched update to reduce API thrashing.
 */
export function useSwitchProfileMutation(): UseMutationResult<
  { activeProfile: Profile; profiles: Profile[] },
  Error,
  string
> {
  const queryClient = useQueryClient();
  const batchProfileSwitch = useProfileStore((s) => s.batchProfileSwitch);

  return useMutation({
    mutationFn: async (profileId: string) => {
      // Single API call to switch profile
      await api.post<void>("/api/v1/profiles/switch", { profileId });

      // Fetch updated data in parallel
      const [activeProfile, profileList] = await Promise.all([
        api.get<Profile>("/api/v1/profiles/active"),
        api.get<ProfileListResponse>("/api/v1/profiles"),
      ]);

      return { activeProfile, profiles: profileList.profiles || [] };
    },
    onSuccess: ({ activeProfile, profiles }: { activeProfile: Profile; profiles: Profile[] }) => {
      // Single batched state update instead of multiple
      batchProfileSwitch(activeProfile, profiles);
      logger.info(LogComponents.Profiles, "Profile switched", {
        id: activeProfile.id,
        name: activeProfile.name,
      });

      // Invalidate queries to ensure fresh data on next access
      queryClient.invalidateQueries({ queryKey: profileKeys.all });
    },
    onError: (err: Error) => {
      logger.error(LogComponents.Profiles, "Failed to switch profile", err);
    },
  });
}

/**
 * Duplicate a profile.
 */
export function useDuplicateProfileMutation(): UseMutationResult<Profile, Error, string> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const duplicated = await api.post<Profile>(`/api/v1/profiles/${id}/duplicate`);
      logger.info(LogComponents.Profiles, "Profile duplicated", {
        sourceId: id,
        newId: duplicated.id,
      });
      return duplicated;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profileKeys.list() });
    },
    onError: (err: Error) => {
      logger.error(LogComponents.Profiles, "Failed to duplicate profile", err);
    },
  });
}

/**
 * Import profiles from JSON.
 */
export function useImportProfilesMutation(): UseMutationResult<
  ProfileImportResponse,
  Error,
  ProfileImportRequest
> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: ProfileImportRequest) => {
      const result = await api.post<ProfileImportResponse>("/api/v1/profiles/import", data);
      logger.info(LogComponents.Profiles, "Profiles imported", {
        imported: result.imported,
        skipped: result.skipped,
      });
      return result;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: profileKeys.list() });
    },
    onError: (err: Error) => {
      logger.error(LogComponents.Profiles, "Failed to import profiles", err);
    },
  });
}

/**
 * Export all profiles to JSON.
 */
export function useExportProfilesQuery(
  enabled: boolean = false,
): UseQueryResult<ProfileExportResponse, Error> {
  return useQuery({
    queryKey: [...profileKeys.all, "export"],
    queryFn: async () => {
      const data = await api.get<ProfileExportResponse>("/api/v1/profiles/export");
      return data;
    },
    enabled,
    staleTime: 0, // Always fetch fresh data for export
  });
}

/**
 * Save settings to profile with debouncing handled by caller.
 */
export function useSaveSettingsMutation(): UseMutationResult<
  Profile,
  Error,
  { profileId: string; settings: Record<string, unknown> }
> {
  const queryClient = useQueryClient();
  const setSettingsStatus = useProfileStore((s) => s.setSettingsStatus);
  const updateActiveProfileSettings = useProfileStore((s) => s.updateActiveProfileSettings);

  return useMutation({
    mutationFn: async ({
      profileId,
      settings,
    }: {
      profileId: string;
      settings: Record<string, unknown>;
    }) => {
      setSettingsStatus("saving");
      const updated = await api.patch<Profile>(`/api/v1/profiles/${profileId}/settings`, settings);
      return updated;
    },
    onSuccess: (updated: Profile) => {
      updateActiveProfileSettings(updated.settings);
      setSettingsStatus("saved");

      // Reset status after delay
      setTimeout(() => setSettingsStatus("idle"), 2000);

      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: profileKeys.active() });
    },
    onError: (err: Error) => {
      setSettingsStatus("error");
      logger.error(LogComponents.Profiles, "Failed to save settings", err);

      // Reset status after delay
      setTimeout(() => setSettingsStatus("idle"), 3000);
    },
  });
}
