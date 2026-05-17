/**
 * useProfileApi — backwards-compatible wrappers around the profile-related
 * React Query mutations / queries. Extracted from ProfileContext so the
 * provider stays focused on assembling context value rather than holding
 * dozens of inline useCallback hooks.
 */

import { useCallback } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import { getQueryClient } from '../lib/queryClient';
import {
  profileKeys,
  useActiveProfileQuery,
  useCreateProfileMutation,
  useDeleteProfileMutation,
  useDuplicateProfileMutation,
  useImportProfilesMutation,
  useProfilesQuery,
  useSwitchProfileMutation,
  useUpdateProfileMutation,
} from '../stores/profileQueries';
import type {
  Profile,
  ProfileExportResponse,
  ProfileImportRequest,
  ProfileImportResponse,
  ProfileRequest,
} from '../types/profile';

export interface ProfileApi {
  refreshProfiles: () => Promise<void>;
  refreshActiveProfile: () => Promise<void>;
  createProfile: (profile: ProfileRequest) => Promise<Profile | null>;
  updateProfile: (id: string, profile: ProfileRequest) => Promise<Profile | null>;
  deleteProfile: (id: string) => Promise<boolean>;
  switchProfile: (profileId: string) => Promise<boolean>;
  duplicateProfile: (id: string, newName?: string) => Promise<Profile | null>;
  importProfiles: (request: ProfileImportRequest) => Promise<ProfileImportResponse | null>;
  exportProfiles: () => Promise<ProfileExportResponse | null>;
  downloadProfiles: () => Promise<boolean>;
}

/**
 * Returns memoized wrappers around the profile-related queries / mutations.
 */
export function useProfileApi(): ProfileApi {
  const profilesQuery = useProfilesQuery();
  const activeProfileQuery = useActiveProfileQuery();
  const createProfileMutation = useCreateProfileMutation();
  const updateProfileMutation = useUpdateProfileMutation();
  const deleteProfileMutation = useDeleteProfileMutation();
  const switchProfileMutation = useSwitchProfileMutation();
  const duplicateProfileMutation = useDuplicateProfileMutation();
  const importProfilesMutation = useImportProfilesMutation();

  const refreshProfiles = useCallback(async () => {
    await Promise.resolve(profilesQuery.refetch());
  }, [profilesQuery]);

  const refreshActiveProfile = useCallback(async () => {
    await Promise.resolve(activeProfileQuery.refetch());
  }, [activeProfileQuery]);

  const createProfile = useCallback(
    async (profile: ProfileRequest): Promise<Profile | null> => {
      try {
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
      const result = await Promise.resolve(
        queryClient.fetchQuery({
          queryKey: [...profileKeys.all, 'export'],
          queryFn: async () => {
            const data = await api.get<ProfileExportResponse>('/api/v1/profiles/export');
            return data;
          },
          staleTime: 0,
        }),
      );
      logger.info(LogComponents.Profiles, 'Profiles exported', {
        count: result.profiles.length,
      });
      return result;
    } catch (err) {
      logger.error(LogComponents.Profiles, 'Failed to export profiles', err);
      return null;
    }
  }, []);

  const downloadProfiles = useCallback(async (): Promise<boolean> => {
    try {
      const result = await exportProfiles();
      if (!result) {
        return false;
      }
      const blob = new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `seed-profiles-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      return true;
    } catch (err) {
      logger.error(LogComponents.Profiles, 'Failed to download profiles', err);
      return false;
    }
  }, [exportProfiles]);

  return {
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
}
