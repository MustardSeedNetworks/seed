import type React from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../../styles/theme';
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
    <label
      className={cn(
        layout.flex.between,
        spacing.pad.xs,
        'bg-surface-base',
        radius.default,
        'border border-surface-border',
      )}
    >
      <div>
        <span className="body-small text-text-primary font-medium">
          {t('discovery.enableDiscovery')}
        </span>
        <p className="caption text-text-muted">{t('discovery.scanForDevices')}</p>
      </div>
      <input
        type="checkbox"
        checked={settings.enabled}
        onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
          onSettingsChange((prev) => ({
            ...prev,
            enabled: e.target.checked,
          }))
        }
        className={iconTokens.size.sm}
      />
    </label>
  );
}
