/**
 * useProfileInterfaces — get/set/add/remove helpers for the multi-interface
 * config (ethernet / wifi lists + active selection) embedded in each
 * profile. Extracted from ProfileContext.
 */

import type React from 'react';
import { useCallback, useRef } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import { getQueryClient } from '../lib/queryClient';
import { profileKeys } from '../stores/profileQueries';
import type { Profile, ProfileInterfaceSelection } from '../types/profile';

export interface ProfileInterfaceHelpers {
  getEthernetInterface: () => ProfileInterfaceSelection | null;
  getWifiInterface: () => ProfileInterfaceSelection | null;
  getAllEthernetInterfaces: () => ProfileInterfaceSelection[];
  getAllWifiInterfaces: () => ProfileInterfaceSelection[];
  setEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  setWifiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  addEthernetInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  addWifiInterface: (name: string, enabled?: boolean) => Promise<boolean>;
  removeEthernetInterface: (name: string) => Promise<boolean>;
  removeWifiInterface: (name: string) => Promise<boolean>;
  setActiveEthernetInterface: (name: string) => Promise<boolean>;
  setActiveWifiInterface: (name: string) => Promise<boolean>;
}

/**
 * Returns the get/set/add/remove helpers for ethernet + wifi interface
 * lists on the active profile.
 */
export function useProfileInterfaces(
  activeProfile: Profile | null,
  setActiveProfile: (profile: Profile | null) => void,
): ProfileInterfaceHelpers {
  // Hold the current profile in a ref so callbacks stay stable while still
  // reading the latest value when they fire (avoids busting memoized consumers).
  const activeProfileRef: React.MutableRefObject<Profile | null> = useRef(activeProfile);
  activeProfileRef.current = activeProfile;

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
        interfaces: NonNullable<NonNullable<Profile['config']>['interfaces']>,
      ) => NonNullable<NonNullable<Profile['config']>['interfaces']>,
    ): Promise<boolean> => {
      const currentProfile = activeProfileRef.current;
      if (!currentProfile) {
        logger.warn(LogComponents.Profiles, 'Cannot update interfaces: no active profile');
        return false;
      }

      try {
        const currentInterfaces = currentProfile.config?.interfaces ?? { ethernet: [], wifi: [] };
        const updatedInterfaces = updater(currentInterfaces);
        const updatedConfig = { ...currentProfile.config, interfaces: updatedInterfaces };

        await api.put(`/api/profiles/${currentProfile.id}`, {
          name: currentProfile.name,
          description: currentProfile.description,
          config: updatedConfig,
        });

        setActiveProfile({ ...currentProfile, config: updatedConfig });

        const queryClient = getQueryClient();
        queryClient.invalidateQueries({ queryKey: profileKeys.active() });

        return true;
      } catch (err) {
        logger.error(LogComponents.Profiles, 'Failed to update interface config', err);
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
        logger.info(LogComponents.Profiles, 'Ethernet interface set as active', {
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
        logger.info(LogComponents.Profiles, 'Wifi interface set as active', {
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
        logger.info(LogComponents.Profiles, 'Ethernet interface added', {
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
        logger.info(LogComponents.Profiles, 'Wifi interface added', {
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
          interfaces.activeEthernet === name ? '' : interfaces.activeEthernet;
        return { ...interfaces, ethernet, activeEthernet: activeEthernetVal };
      });
      if (result) {
        logger.info(LogComponents.Profiles, 'Ethernet interface removed', {
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
        const activeWifiVal = interfaces.activeWifi === name ? '' : interfaces.activeWifi;
        return { ...interfaces, wifi, activeWifi: activeWifiVal };
      });
      if (result) {
        logger.info(LogComponents.Profiles, 'Wifi interface removed', {
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
          'Cannot set active ethernet interface: interface not in list',
          { interface: name },
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        activeEthernet: name,
      }));
      if (result) {
        logger.info(LogComponents.Profiles, 'Active ethernet interface changed', {
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
          'Cannot set active Wifi interface: interface not in list',
          { interface: name },
        );
        return false;
      }

      const result = await updateInterfaceConfig((interfaces) => ({
        ...interfaces,
        activeWifi: name,
      }));
      if (result) {
        logger.info(LogComponents.Profiles, 'Active Wifi interface changed', {
          profileId: activeProfileRef.current?.id,
          interface: name,
        });
      }
      return result;
    },
    [updateInterfaceConfig],
  );

  return {
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
}
