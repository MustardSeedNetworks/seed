/**
 * useProfileInterfaces — get/set/add/remove helpers for the multi-interface
 * config (ethernet / wifi lists + active selection) embedded in each
 * profile. Extracted from ProfileContext.
 */

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import { getQueryClient } from '../lib/queryClient';
import { profileKeys } from '../stores/profileQueries';
import { useProfileStore } from '../stores/profileStore';
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
 *
 * The getters read the profile this render subscribed to, so a consumer
 * that calls them while rendering sees the current list. The mutators read
 * the store when they run, so two in a row each build on the last.
 */
export function useProfileInterfaces(): ProfileInterfaceHelpers {
  const activeProfile = useProfileStore((s) => s.activeProfile);

  const getEthernetInterface = (): ProfileInterfaceSelection | null => {
    const interfaces = activeProfile?.config?.interfaces;
    if (!(interfaces?.activeEthernet && interfaces.ethernet)) {
      return null;
    }
    return interfaces.ethernet.find((i) => i.name === interfaces.activeEthernet) ?? null;
  };

  const getWifiInterface = (): ProfileInterfaceSelection | null => {
    const interfaces = activeProfile?.config?.interfaces;
    if (!(interfaces?.activeWifi && interfaces.wifi)) {
      return null;
    }
    return interfaces.wifi.find((i) => i.name === interfaces.activeWifi) ?? null;
  };

  const getAllEthernetInterfaces = (): ProfileInterfaceSelection[] =>
    activeProfile?.config?.interfaces?.ethernet ?? [];

  const getAllWifiInterfaces = (): ProfileInterfaceSelection[] =>
    activeProfile?.config?.interfaces?.wifi ?? [];

  /**
   * Helper to update interface config on the backend.
   */
  const updateInterfaceConfig = async (
    updater: (
      interfaces: NonNullable<NonNullable<Profile['config']>['interfaces']>,
    ) => NonNullable<NonNullable<Profile['config']>['interfaces']>,
  ): Promise<boolean> => {
    const currentProfile = useProfileStore.getState().activeProfile;
    if (!currentProfile) {
      logger.warn(LogComponents.PROFILES, 'Cannot update interfaces: no active profile');
      return false;
    }

    // Built outside the try: the React Compiler cannot compile `?.` or `??`
    // inside one, and an uncompiled hook hands out new helpers every render.
    const currentInterfaces = currentProfile.config?.interfaces ?? { ethernet: [], wifi: [] };
    const updatedConfig = { ...currentProfile.config, interfaces: updater(currentInterfaces) };

    try {
      await api.put(`/api/v1/profiles/${currentProfile.id}`, {
        name: currentProfile.name,
        description: currentProfile.description,
        config: updatedConfig,
      });

      useProfileStore.getState().setActiveProfile({ ...currentProfile, config: updatedConfig });

      const queryClient = getQueryClient();
      await queryClient.invalidateQueries({ queryKey: profileKeys.active() });

      return true;
    } catch (err) {
      logger.error(LogComponents.PROFILES, 'Failed to update interface config', err);
      return false;
    }
  };

  const setEthernetInterface = async (name: string, enabled = true): Promise<boolean> => {
    const result = await updateInterfaceConfig((interfaces) => {
      const ethernet = [...(interfaces.ethernet ?? [])];
      const existingIdx = ethernet.findIndex((i) => i.name === name);
      const existing = ethernet.at(existingIdx);
      if (existing) {
        ethernet[existingIdx] = { ...existing, enabled };
      } else {
        ethernet.push({ name, enabled });
      }
      return { ...interfaces, ethernet, activeEthernet: name };
    });
    if (result) {
      logger.info(LogComponents.PROFILES, 'Ethernet interface set as active', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const setWifiInterface = async (name: string, enabled = true): Promise<boolean> => {
    const result = await updateInterfaceConfig((interfaces) => {
      const wifi = [...(interfaces.wifi ?? [])];
      const existingIdx = wifi.findIndex((i) => i.name === name);
      const existing = wifi.at(existingIdx);
      if (existing) {
        wifi[existingIdx] = { ...existing, enabled };
      } else {
        wifi.push({ name, enabled });
      }
      return { ...interfaces, wifi, activeWifi: name };
    });
    if (result) {
      logger.info(LogComponents.PROFILES, 'Wifi interface set as active', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const addEthernetInterface = async (name: string, enabled = true): Promise<boolean> => {
    const result = await updateInterfaceConfig((interfaces) => {
      const ethernet = [...(interfaces.ethernet ?? [])];
      const existingIdx = ethernet.findIndex((i) => i.name === name);
      const existing = ethernet.at(existingIdx);
      if (existing) {
        ethernet[existingIdx] = { ...existing, enabled };
      } else {
        ethernet.push({ name, enabled });
      }
      return { ...interfaces, ethernet };
    });
    if (result) {
      logger.info(LogComponents.PROFILES, 'Ethernet interface added', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const addWifiInterface = async (name: string, enabled = true): Promise<boolean> => {
    const result = await updateInterfaceConfig((interfaces) => {
      const wifi = [...(interfaces.wifi ?? [])];
      const existingIdx = wifi.findIndex((i) => i.name === name);
      const existing = wifi.at(existingIdx);
      if (existing) {
        wifi[existingIdx] = { ...existing, enabled };
      } else {
        wifi.push({ name, enabled });
      }
      return { ...interfaces, wifi };
    });
    if (result) {
      logger.info(LogComponents.PROFILES, 'Wifi interface added', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const removeEthernetInterface = async (name: string): Promise<boolean> => {
    const result = await updateInterfaceConfig((interfaces) => {
      const ethernet = (interfaces.ethernet ?? []).filter((i) => i.name !== name);
      const activeEthernetVal = interfaces.activeEthernet === name ? '' : interfaces.activeEthernet;
      return { ...interfaces, ethernet, activeEthernet: activeEthernetVal };
    });
    if (result) {
      logger.info(LogComponents.PROFILES, 'Ethernet interface removed', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const removeWifiInterface = async (name: string): Promise<boolean> => {
    const result = await updateInterfaceConfig((interfaces) => {
      const wifi = (interfaces.wifi ?? []).filter((i) => i.name !== name);
      const activeWifiVal = interfaces.activeWifi === name ? '' : interfaces.activeWifi;
      return { ...interfaces, wifi, activeWifi: activeWifiVal };
    });
    if (result) {
      logger.info(LogComponents.PROFILES, 'Wifi interface removed', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const setActiveEthernetInterface = async (name: string): Promise<boolean> => {
    const currentProfile = useProfileStore.getState().activeProfile;
    const exists = (currentProfile?.config?.interfaces?.ethernet ?? []).some(
      (i) => i.name === name,
    );
    if (!exists) {
      logger.warn(
        LogComponents.PROFILES,
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
      logger.info(LogComponents.PROFILES, 'Active ethernet interface changed', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

  const setActiveWifiInterface = async (name: string): Promise<boolean> => {
    const currentProfile = useProfileStore.getState().activeProfile;
    const exists = (currentProfile?.config?.interfaces?.wifi ?? []).some((i) => i.name === name);
    if (!exists) {
      logger.warn(
        LogComponents.PROFILES,
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
      logger.info(LogComponents.PROFILES, 'Active Wifi interface changed', {
        profileId: useProfileStore.getState().activeProfile?.id,
        interface: name,
      });
    }
    return result;
  };

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
