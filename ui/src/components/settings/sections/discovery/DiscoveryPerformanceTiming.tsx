import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, layout, spacing } from '../../../../styles/theme';
import type { NetworkDiscoverySettings } from '../../../../types/settings';

interface DiscoveryPerformanceTimingProps {
  settings: NetworkDiscoverySettings;
  onSettingsChange: React.Dispatch<React.SetStateAction<NetworkDiscoverySettings>>;
}

/**
 * Discovery performance and timing sliders.
 */
export const DiscoveryPerformanceTiming: React.NamedExoticComponent<DiscoveryPerformanceTimingProps> =
  memo(function discoveryPerformanceTiming({
    settings,
    onSettingsChange,
  }: DiscoveryPerformanceTimingProps) {
    const { t } = useTranslation('settings');

    return (
      <div
        className={cn('border-t border-surface-border', spacing.pad.sm, spacing.margin.top.inline)}
      >
        <span className="caption text-text-muted font-medium">
          {t('discovery.performanceTiming')}
        </span>
        <p className="caption text-text-muted">{t('discovery.performanceTimingDesc')}</p>
        <div className={cn('stack-sm', spacing.margin.top.inline)}>
          {/* Probe Interval Slider */}
          <div>
            <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
              <label htmlFor="probe-interval-slider" className="caption text-text-muted">
                {t('discovery.probeInterval')}
              </label>
              <span className="caption text-text-primary font-medium">
                {settings.timing?.probeIntervalMs ?? 75}ms
              </span>
            </div>
            <input
              id="probe-interval-slider"
              type="range"
              min={25}
              max={500}
              step={25}
              value={settings.timing?.probeIntervalMs ?? 75}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                onSettingsChange((prev) => ({
                  ...prev,
                  timing: {
                    ...prev.timing,
                    probeIntervalMs: Number.parseInt(e.target.value, 10),
                    rescanIntervalMs: prev.timing?.rescanIntervalMs ?? 600000,
                    workers: prev.timing?.workers ?? 50,
                  },
                }))
              }
              className="w-full"
            />
            <div
              className={cn(
                layout.flex.between,
                'caption text-text-muted',
                spacing.margin.top.tight,
              )}
            >
              <span>{t('discovery.slower')}</span>
              <span>{t('discovery.faster')}</span>
            </div>
          </div>

          {/* Scan Timeout Slider */}
          <div>
            <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
              <label htmlFor="scan-timeout-slider" className="caption text-text-muted">
                {t('discovery.scanTimeout')}
              </label>
              <span className="caption text-text-primary font-medium">
                {settings.scanTimeoutMs ?? 2000}ms
              </span>
            </div>
            <input
              id="scan-timeout-slider"
              type="range"
              min={500}
              max={10000}
              step={500}
              value={settings.scanTimeoutMs ?? 2000}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                onSettingsChange((prev) => ({
                  ...prev,
                  scanTimeoutMs: Number.parseInt(e.target.value, 10),
                }))
              }
              className="w-full"
            />
            <div
              className={cn(
                layout.flex.between,
                'caption text-text-muted',
                spacing.margin.top.tight,
              )}
            >
              <span>500ms</span>
              <span>10s</span>
            </div>
          </div>

          {/* Workers Slider */}
          <div>
            <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
              <label htmlFor="workers-slider" className="caption text-text-muted">
                {t('discovery.workers')}
              </label>
              <span className="caption text-text-primary font-medium">
                {settings.timing?.workers ?? 20}
              </span>
            </div>
            <input
              id="workers-slider"
              type="range"
              min={5}
              max={100}
              step={5}
              value={settings.timing?.workers ?? 20}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                onSettingsChange((prev) => ({
                  ...prev,
                  timing: {
                    ...prev.timing,
                    probeIntervalMs: prev.timing?.probeIntervalMs ?? 75,
                    rescanIntervalMs: prev.timing?.rescanIntervalMs ?? 600000,
                    workers: Number.parseInt(e.target.value, 10),
                  },
                }))
              }
              className="w-full"
            />
            <div
              className={cn(
                layout.flex.between,
                'caption text-text-muted',
                spacing.margin.top.tight,
              )}
            >
              <span>{t('discovery.gentler')}</span>
              <span>{t('discovery.aggressive')}</span>
            </div>
          </div>

          {/* Rescan Interval Slider */}
          <div>
            <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
              <label htmlFor="rescan-interval-slider" className="caption text-text-muted">
                {t('discovery.rescanInterval')}
              </label>
              <span className="caption text-text-primary font-medium">
                {Math.round((settings.timing?.rescanIntervalMs ?? 600000) / 60000)}m
              </span>
            </div>
            <input
              id="rescan-interval-slider"
              type="range"
              min={60}
              max={3600}
              step={60}
              value={(settings.timing?.rescanIntervalMs ?? 600000) / 1000}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                onSettingsChange((prev) => ({
                  ...prev,
                  timing: {
                    ...prev.timing,
                    probeIntervalMs: prev.timing?.probeIntervalMs ?? 75,
                    rescanIntervalMs: Number.parseInt(e.target.value, 10) * 1000,
                    workers: prev.timing?.workers ?? 50,
                  },
                }))
              }
              className="w-full"
            />
            <div
              className={cn(
                layout.flex.between,
                'caption text-text-muted',
                spacing.margin.top.tight,
              )}
            >
              <span>1m</span>
              <span>60m</span>
            </div>
          </div>

          {/* Banner Timeout Slider (only shown when port scanning is enabled) */}
          {settings.options?.portScan?.enabled ? (
            <div>
              <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                <label htmlFor="banner-timeout-slider" className="caption text-text-muted">
                  {t('discovery.bannerTimeout')}
                </label>
                <span className="caption text-text-primary font-medium">
                  {settings.options?.portScan?.bannerTimeoutMs ?? 2000}ms
                </span>
              </div>
              <input
                id="banner-timeout-slider"
                type="range"
                min={500}
                max={10000}
                step={500}
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
                        bannerTimeoutMs: Number.parseInt(e.target.value, 10),
                      },
                    },
                  }))
                }
                className="w-full"
              />
              <div
                className={cn(
                  layout.flex.between,
                  'caption text-text-muted',
                  spacing.margin.top.tight,
                )}
              >
                <span>500ms</span>
                <span>10s</span>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    );
  });
