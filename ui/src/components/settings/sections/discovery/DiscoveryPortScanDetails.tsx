import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, input as inputTokens, radius, spacing } from '../../../../styles/theme';
import type { NetworkDiscoverySettings, PortPreset } from '../../../../types/settings';
import { PORT_PRESETS } from './DiscoveryCustomOptions.constants';

interface DiscoveryPortScanDetailsProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Port-scan preset, port lists and banner timeout, shown when port scanning is enabled.
 */
export const DiscoveryPortScanDetails: React.NamedExoticComponent<DiscoveryPortScanDetailsProps> =
  memo(function discoveryPortScanDetails({
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
                    udpPorts:
                      newPreset === 'custom'
                        ? (prev.options?.portScan?.udpPorts ?? '53,161')
                        : presetConfig.udp,
                    bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
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
                    udpPorts: prev.options?.portScan?.udpPorts ?? '53,161',
                    bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
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
        <div>
          <label className="caption text-text-muted" htmlFor="port-scan-udp">
            {t('discovery.portScanUdpPorts')}
            {(settings.options?.portScan?.preset ?? 'common') !== 'custom' && (
              <span className="ml-inline text-text-muted italic">
                {t('discovery.portPresetReadOnly')}
              </span>
            )}
          </label>
          <input
            id="port-scan-udp"
            type="text"
            value={settings.options?.portScan?.udpPorts ?? '53,161'}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  portScan: {
                    ...prev.options?.portScan,
                    enabled: prev.options?.portScan?.enabled ?? false,
                    preset: prev.options?.portScan?.preset ?? 'common',
                    tcpPorts: prev.options?.portScan?.tcpPorts ?? '22,80,443',
                    udpPorts: e.target.value,
                    bannerTimeoutMs: prev.options?.portScan?.bannerTimeoutMs ?? 2000,
                  },
                },
              }))
            }
            placeholder="53,123,161"
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
        <div>
          <label className="caption text-text-muted" htmlFor="port-scan-banner">
            {t('discovery.portScanBannerTimeout')}
          </label>
          <input
            id="port-scan-banner"
            type="number"
            value={settings.options?.portScan?.bannerTimeoutMs ?? 2000}
            onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
              onSettingsChange((prev) => ({
                ...prev,
                options: {
                  ...prev.options,
                  portScan: {
                    ...prev.options?.portScan,
                    enabled: prev.options?.portScan?.enabled ?? false,
                    preset: prev.options?.portScan?.preset ?? 'common',
                    tcpPorts: prev.options?.portScan?.tcpPorts ?? '22,80,443',
                    udpPorts: prev.options?.portScan?.udpPorts ?? '53,161',
                    bannerTimeoutMs: Number.parseInt(e.target.value, 10) || 2000,
                  },
                },
              }))
            }
            min={100}
            max={10000}
            className={cn(
              'w-24',
              spacing.margin.top.tight,
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.sm,
              'body-small',
            )}
          />
        </div>
      </div>
    );
  });
