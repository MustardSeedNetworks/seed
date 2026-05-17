/**
 * useProfileSettingsUpdates — auto-save settings updaters that target the
 * active profile via the saveSettings mutation. Extracted from
 * ProfileContext so the provider stays focused on assembling its value.
 */

import { useCallback } from 'react';
import { LogComponents, logger } from '../lib/logger';
import { useSaveSettingsMutation } from '../stores/profileQueries';
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
  ProfileSettings,
  ProfileThresholdsConfig,
  SnmpConfig,
  SpeedtestConfig,
  TestsConfig,
  VulnerabilityConfig,
  WifiSettingsConfig,
} from '../types/profile';

export interface ProfileSettingsUpdaters {
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
  updateSettings: (updates: Partial<ProfileSettings>) => void;
}

/**
 * Returns memoized settings updaters bound to the active profile.
 */
export function useProfileSettingsUpdates(activeProfile: Profile | null): ProfileSettingsUpdaters {
  const saveSettingsMutation = useSaveSettingsMutation();

  const updateSettingsField = useCallback(
    <T extends keyof ProfileSettings>(field: T, updates: Partial<ProfileSettings[T]>) => {
      if (!activeProfile) {
        logger.warn(LogComponents.Profiles, 'Cannot save settings: no active profile');
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
    (updates: Partial<CardSettingsConfig>) => updateSettingsField('cardSettings', updates),
    [updateSettingsField],
  );
  const updateDisplayOptions = useCallback(
    (updates: Partial<DisplayOptionsConfig>) => updateSettingsField('displayOptions', updates),
    [updateSettingsField],
  );
  const updateIperfSettings = useCallback(
    (updates: Partial<IperfConfig>) => updateSettingsField('iperf', updates),
    [updateSettingsField],
  );
  const updateThresholds = useCallback(
    (updates: Partial<ProfileThresholdsConfig>) => updateSettingsField('thresholds', updates),
    [updateSettingsField],
  );
  const updateSpeedtestSettings = useCallback(
    (updates: Partial<SpeedtestConfig>) => updateSettingsField('speedtest', updates),
    [updateSettingsField],
  );
  const updateTestsSettings = useCallback(
    (updates: Partial<TestsConfig>) => updateSettingsField('tests', updates),
    [updateSettingsField],
  );
  const updateNetworkDiscoverySettings = useCallback(
    (updates: Partial<NetworkDiscoveryConfig>) => updateSettingsField('networkDiscovery', updates),
    [updateSettingsField],
  );
  const updateSnmpSettings = useCallback(
    (updates: Partial<SnmpConfig>) => updateSettingsField('snmp', updates),
    [updateSettingsField],
  );
  const updateWifiSettings = useCallback(
    (updates: Partial<WifiSettingsConfig>) => updateSettingsField('wifi', updates),
    [updateSettingsField],
  );
  const updateLinkSettings = useCallback(
    (updates: Partial<LinkConfig>) => updateSettingsField('link', updates),
    [updateSettingsField],
  );
  const updateCableTestSettings = useCallback(
    (updates: Partial<CableTestConfig>) => updateSettingsField('cableTest', updates),
    [updateSettingsField],
  );
  const updateVulnerabilitySettings = useCallback(
    (updates: Partial<VulnerabilityConfig>) => updateSettingsField('vulnerability', updates),
    [updateSettingsField],
  );
  const updateDnsSettings = useCallback(
    (updates: Partial<DnsSettingsConfig>) => updateSettingsField('dns', updates),
    [updateSettingsField],
  );
  const updateAppearanceSettings = useCallback(
    (updates: Partial<AppearanceConfig>) => updateSettingsField('appearance', updates),
    [updateSettingsField],
  );

  const updateSettings = useCallback(
    (updates: Partial<ProfileSettings>) => {
      if (!activeProfile) {
        logger.warn(LogComponents.Profiles, 'Cannot save settings: no active profile');
        return;
      }

      saveSettingsMutation.mutate({
        profileId: activeProfile.id,
        settings: updates,
      });
    },
    [activeProfile, saveSettingsMutation],
  );

  return {
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
  };
}
