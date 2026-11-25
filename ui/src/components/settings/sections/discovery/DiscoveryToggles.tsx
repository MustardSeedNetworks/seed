import type React from 'react';
import { useTranslation } from 'react-i18next';
import { Toggle } from '../../../../shared/Toggle';
import type { NetworkDiscoverySettings } from '../../../../types/settings';

interface DiscoveryTogglesProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Enable/disable toggle for the discovery service.
 */
export function DiscoveryToggles({ settings, onSettingsChange }: DiscoveryTogglesProps) {
  const { t } = useTranslation('settings');

  return (
    <Toggle
      label={t('discovery.enableDiscovery')}
      description={t('discovery.scanForDevices')}
      checked={settings.enabled}
      onChange={(checked: boolean): void =>
        onSettingsChange((prev) => ({
          ...prev,
          enabled: checked,
        }))
      }
      className="rounded-lg border border-surface-border bg-surface-base p-3"
    />
  );
}
