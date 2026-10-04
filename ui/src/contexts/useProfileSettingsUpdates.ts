/**
 * useProfileSettingsUpdates — auto-save settings updaters that target the
 * active profile via the saveSettings mutation. Extracted from
 * ProfileContext so the provider stays focused on assembling its value.
 */

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
  WiFiSettingsConfig,
} from '../types/profile';

export interface ProfileSettingsUpdaters {
  updateLinkSettings: (updates: Partial<LinkConfig>) => void;
  updateCableTestSettings: (updates: Partial<CableTestConfig>) => void;
  updateDisplayOptions: (updates: Partial<DisplayOptionsConfig>) => void;
  updateWifiSettings: (updates: Partial<WiFiSettingsConfig>) => void;
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
 * Returns settings updaters bound to the active profile.
 */
export function useProfileSettingsUpdates(activeProfile: Profile | null): ProfileSettingsUpdaters {
  const { mutate: saveSettings } = useSaveSettingsMutation();

  const updateSettingsField = <T extends keyof ProfileSettings>(
    field: T,
    updates: Partial<ProfileSettings[T]>,
  ) => {
    if (!activeProfile) {
      logger.warn(LogComponents.PROFILES, 'Cannot save settings: no active profile');
      return;
    }

    const currentSettings = activeProfile.config?.settings ?? {};
    const currentFieldValue = currentSettings[field] ?? {};
    const newSettings = {
      [field]: { ...currentFieldValue, ...updates },
    };

    saveSettings({
      profileId: activeProfile.id,
      settings: newSettings,
    });
  };

  const updateCardSettings = (updates: Partial<CardSettingsConfig>) =>
    updateSettingsField('cardSettings', updates);
  const updateDisplayOptions = (updates: Partial<DisplayOptionsConfig>) =>
    updateSettingsField('displayOptions', updates);
  const updateIperfSettings = (updates: Partial<IperfConfig>) =>
    updateSettingsField('iperf', updates);
  const updateThresholds = (updates: Partial<ProfileThresholdsConfig>) =>
    updateSettingsField('thresholds', updates);
  const updateSpeedtestSettings = (updates: Partial<SpeedtestConfig>) =>
    updateSettingsField('speedtest', updates);
  const updateTestsSettings = (updates: Partial<TestsConfig>) =>
    updateSettingsField('tests', updates);
  const updateNetworkDiscoverySettings = (updates: Partial<NetworkDiscoveryConfig>) =>
    updateSettingsField('networkDiscovery', updates);
  const updateSnmpSettings = (updates: Partial<SnmpConfig>) => updateSettingsField('snmp', updates);
  const updateWifiSettings = (updates: Partial<WiFiSettingsConfig>) =>
    updateSettingsField('wifi', updates);
  const updateLinkSettings = (updates: Partial<LinkConfig>) => updateSettingsField('link', updates);
  const updateCableTestSettings = (updates: Partial<CableTestConfig>) =>
    updateSettingsField('cableTest', updates);
  const updateVulnerabilitySettings = (updates: Partial<VulnerabilityConfig>) =>
    updateSettingsField('vulnerability', updates);
  const updateDnsSettings = (updates: Partial<DnsSettingsConfig>) =>
    updateSettingsField('dns', updates);
  const updateAppearanceSettings = (updates: Partial<AppearanceConfig>) =>
    updateSettingsField('appearance', updates);

  const updateSettings = (updates: Partial<ProfileSettings>) => {
    if (!activeProfile) {
      logger.warn(LogComponents.PROFILES, 'Cannot save settings: no active profile');
      return;
    }

    saveSettings({
      profileId: activeProfile.id,
      settings: updates,
    });
  };

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
