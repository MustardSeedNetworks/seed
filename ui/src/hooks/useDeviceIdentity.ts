/**
 * useDeviceIdentity
 *
 * The operator-assigned name and location of this Seed (#195), read from and
 * written to the main settings endpoint. The header, the tab title and the
 * Settings section share one cached query, so a save shows up in the header
 * the moment the server accepts it rather than on the next page load.
 */

import {
  type UseMutationResult,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { api } from '../api';

export interface DeviceIdentity {
  /** Shown in the header and the tab title. Empty means unset. */
  name: string;
  /** Where the device is installed. Empty means unset. */
  location: string;
}

/** The identity slice of GET /api/v1/settings. */
interface IdentitySettingsResponse {
  identity?: Partial<DeviceIdentity>;
}

export const EMPTY_DEVICE_IDENTITY: DeviceIdentity = { name: '', location: '' };

export const deviceIdentityKey = ['settings', 'identity'] as const;

/** The stored identity, or the empty one until it has loaded. */
export function useDeviceIdentity(): DeviceIdentity {
  const { data } = useQuery({
    queryKey: deviceIdentityKey,
    queryFn: async (): Promise<DeviceIdentity> => {
      const settings = await api.get<IdentitySettingsResponse>('/api/v1/settings');
      return {
        name: settings.identity?.name ?? '',
        location: settings.identity?.location ?? '',
      };
    },
    // This browser's own save invalidates it; the minute bounds how long a
    // rename made in another tab goes unseen here.
    staleTime: 60 * 1000,
  });
  return data ?? EMPTY_DEVICE_IDENTITY;
}

/**
 * Saves the identity. The server trims surrounding whitespace, so the cache
 * is refetched rather than filled with what was sent.
 */
export function useSaveDeviceIdentity(): UseMutationResult<void, Error, DeviceIdentity> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (identity: DeviceIdentity): Promise<void> => {
      await api.put('/api/v1/settings', { identity });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: deviceIdentityKey }),
  });
}
