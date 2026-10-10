import type React from 'react';
import { useTranslation } from 'react-i18next';
import { cn, input as inputTokens, radius, spacing } from '../../../../styles/theme';
import type { NetworkDiscoverySettings, PortPreset } from '../../../../types/settings';
import { PORT_PRESETS } from './DiscoveryCustomOptions.constants';

interface DiscoveryPortScanDetailsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Port-scan preset and TCP port list, shown when port scanning is enabled.
 */
export function DiscoveryPortScanDetails({
  settings,
  onSettingsChange,
}: DiscoveryPortScanDetailsProps) {
  const { t } = useTranslation('settings');

  return (
    <div
      className={cn(
        'ml-spacious stack-sm',
        spacing.pad.sm,
        'bg-surface-base',
        radius.default,
        'border border-surface-border',
      )}
    >
      <div>
        <label className="caption text-text-muted" htmlFor="port-scan-preset">
          {t('discovery.portScanPreset')}
        </label>
        <select
          id="port-scan-preset"
          value={settings.options?.portScan?.preset ?? 'common'}
          onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void => {
            const newPreset = e.target.value as PortPreset;
            const presetConfig = PORT_PRESETS[newPreset];
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                portScan: {
                  ...prev.options?.portScan,
                  enabled: prev.options?.portScan?.enabled ?? false,
                  preset: newPreset,
                  // Auto-populate for non-custom presets
                  tcpPorts:
                    newPreset === 'custom'
                      ? (prev.options?.portScan?.tcpPorts ?? '22,80,443')
                      : presetConfig.tcp,
                },
              },
            }));
          }}
          className={cn(
            'w-full',
            spacing.margin.top.tight,
            inputTokens.base,
            inputTokens.state.default,
            inputTokens.size.sm,
            'body-small',
          )}
        >
          <option value="common">{t('discovery.portPresetCommon')}</option>
          <option value="secure">{t('discovery.portPresetSecure')}</option>
          <option value="insecure">{t('discovery.portPresetInsecure')}</option>
          <option value="custom">{t('discovery.portPresetCustom')}</option>
        </select>
        {/* Description for selected preset */}
        <p className={cn('caption text-text-muted', spacing.margin.top.tight)}>
          {PORT_PRESETS[settings.options?.portScan?.preset ?? 'common'].description}
        </p>
      </div>
      <div>
        <label className="caption text-text-muted" htmlFor="port-scan-tcp">
          {t('discovery.portScanTcpPorts')}
          {(settings.options?.portScan?.preset ?? 'common') !== 'custom' && (
            <span className="ml-inline text-text-muted italic">
              {t('discovery.portPresetReadOnly')}
            </span>
          )}
        </label>
        <input
          id="port-scan-tcp"
          type="text"
          value={settings.options?.portScan?.tcpPorts ?? '22,80,443'}
          onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
            onSettingsChange((prev) => ({
              ...prev,
              options: {
                ...prev.options,
                portScan: {
                  ...prev.options?.portScan,
                  enabled: prev.options?.portScan?.enabled ?? false,
                  preset: prev.options?.portScan?.preset ?? 'common',
                  tcpPorts: e.target.value,
                },
              },
            }))
          }
          placeholder="22,80,443,8080-8100"
          readOnly={(settings.options?.portScan?.preset ?? 'common') !== 'custom'}
          disabled={(settings.options?.portScan?.preset ?? 'common') !== 'custom'}
          className={cn(
            'w-full',
            spacing.margin.top.tight,
            inputTokens.base,
            (settings.options?.portScan?.preset ?? 'common') !== 'custom'
              ? 'bg-surface-hover cursor-not-allowed opacity-60'
              : inputTokens.state.default,
            inputTokens.size.sm,
            'body-small',
          )}
        />
      </div>
    </div>
  );
}
