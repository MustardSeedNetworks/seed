/**
 * useProfileApi — backwards-compatible wrappers around the profile-related
 * React Query mutations / queries. Extracted from ProfileContext so the
 * provider stays focused on assembling its context value.
 */

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
import type { ProfileImportResponse } from '../types/generated/profile-import-response';
import type {
  Profile,
  ProfileExportResponse,
  ProfileImportRequest,
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
 * Returns wrappers around the profile-related queries / mutations. Each
 * takes only the stable `refetch` / `mutateAsync`, never the whole result
 * object, which is new on every render and would change every wrapper.
 */
export function useProfileApi(): ProfileApi {
  const { refetch: refetchProfiles } = useProfilesQuery();
  const { refetch: refetchActiveProfile } = useActiveProfileQuery();
  const { mutateAsync: createProfileAsync } = useCreateProfileMutation();
  const { mutateAsync: updateProfileAsync } = useUpdateProfileMutation();
  const { mutateAsync: deleteProfileAsync } = useDeleteProfileMutation();
  const { mutateAsync: switchProfileAsync } = useSwitchProfileMutation();
  const { mutateAsync: duplicateProfileAsync } = useDuplicateProfileMutation();
  const { mutateAsync: importProfilesAsync } = useImportProfilesMutation();

  const refreshProfiles = async () => {
    await Promise.resolve(refetchProfiles());
  };

  const refreshActiveProfile = async () => {
    await Promise.resolve(refetchActiveProfile());
  };

  const createProfile = async (profile: ProfileRequest): Promise<Profile | null> => {
    try {
      const result = await Promise.resolve(createProfileAsync(profile));
      return result;
    } catch {
      return null;
    }
  };

  const updateProfile = async (id: string, profile: ProfileRequest): Promise<Profile | null> => {
    try {
      const result = await Promise.resolve(updateProfileAsync({ id, profile }));
      return result;
    } catch {
      return null;
    }
  };

  const deleteProfile = async (id: string): Promise<boolean> => {
    try {
      await Promise.resolve(deleteProfileAsync(id));
      return true;
    } catch {
      return false;
    }
  };

  const switchProfile = async (profileId: string): Promise<boolean> => {
    try {
      await Promise.resolve(switchProfileAsync(profileId));
      return true;
    } catch {
      return false;
    }
  };

  const duplicateProfile = async (id: string, _newName?: string): Promise<Profile | null> => {
    try {
      const result = await Promise.resolve(duplicateProfileAsync(id));
      return result;
    } catch {
      return null;
    }
  };

  const importProfiles = async (
    request: ProfileImportRequest,
  ): Promise<ProfileImportResponse | null> => {
    try {
      const result = await Promise.resolve(importProfilesAsync(request));
      return result;
    } catch {
      return null;
    }
  };

  const exportProfiles = async (): Promise<ProfileExportResponse | null> => {
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
      logger.info(LogComponents.PROFILES, 'Profiles exported', {
        count: result.profiles.length,
      });
      return result;
    } catch (err) {
      logger.error(LogComponents.PROFILES, 'Failed to export profiles', err);
      return null;
    }
  };

  const downloadProfiles = async (): Promise<boolean> => {
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
      logger.error(LogComponents.PROFILES, 'Failed to download profiles', err);
      return false;
    }
  };

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
