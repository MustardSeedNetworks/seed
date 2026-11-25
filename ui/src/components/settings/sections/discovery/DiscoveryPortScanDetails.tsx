import type React from 'react';
import { useTranslation } from 'react-i18next';
import { FormField } from '../../../../shared/FormField';
import { Select } from '../../../../shared/Select';
import { cn, input as inputTokens, radius, spacing } from '../../../../styles/theme';
import type { NetworkDiscoverySettings, PortPreset } from '../../../../types/settings';
import { PORT_PRESETS } from './DiscoveryCustomOptions.constants';

interface DiscoveryPortScanDetailsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

function toPreset(value: string): PortPreset {
  return value === 'secure' || value === 'insecure' || value === 'custom' ? value : 'common';
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
      <FormField
        label={t('discovery.portScanPreset')}
        hint={PORT_PRESETS[settings.options?.portScan?.preset ?? 'common'].description}
      >
        {(control) => (
          <Select
            {...control}
            value={settings.options?.portScan?.preset ?? 'common'}
            onValueChange={(value): void => {
              const newPreset = toPreset(value);
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
            options={[
              { value: 'common', label: t('discovery.portPresetCommon') },
              { value: 'secure', label: t('discovery.portPresetSecure') },
              { value: 'insecure', label: t('discovery.portPresetInsecure') },
              { value: 'custom', label: t('discovery.portPresetCustom') },
            ]}
          />
        )}
      </FormField>
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
              ? 'disabled:control-disabled cursor-not-allowed'
              : inputTokens.state.default,
            inputTokens.size.sm,
            'body-small',
          )}
        />
      </div>
    </div>
  );
}
