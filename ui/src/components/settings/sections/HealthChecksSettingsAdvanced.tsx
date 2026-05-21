/**
 * SLA / Alert / Anomaly Detection sub-sections of HealthChecksSettings.
 *
 * These trail the per-protocol endpoint sections and own a few
 * settings on the testsSettings shape (slaConfigs, alertConfig,
 * anomalyConfig). Extracted as a sibling component so the main
 * HealthChecksSettings file stays slim.
 */

import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, input, layout, radius, spacing } from '../../../styles/theme';
import type { TestsSettings } from '../../../types/settings';

interface HealthChecksSettingsAdvancedProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

export function HealthChecksSettingsAdvanced({
  testsSettings,
  setTestsSettings,
}: HealthChecksSettingsAdvancedProps): JSX.Element {
  const { t } = useTranslation('settings');

  return (
    <>
      {/* SLA Configuration */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.slaConfig')}</span>
        </div>
        <div className={spacing.stack.xs}>
          <label
            className={cn(
              layout.flex.between,
              spacing.pad.sm,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t('health.enableSla')}
              </span>
              <p className="caption text-text-muted">{t('health.slaDescription')}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.slaConfigs?.[0]?.enabled ?? false}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  slaConfigs: [
                    {
                      ...(prev.slaConfigs?.[0] ?? {
                        endpointName: '*',
                        targetUptime: 99.9,
                        targetLatencyP95: 500,
                        reportingPeriod: 'daily',
                      }),
                      enabled: e.target.checked,
                    },
                  ],
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>
          <div className={cn('flex items-center', spacing.gap.compact)}>
            <label htmlFor="sla-target-uptime" className="caption text-text-muted w-32">
              {t('health.targetUptime')}
            </label>
            <input
              id="sla-target-uptime"
              type="number"
              min={90}
              max={100}
              step={0.1}
              value={testsSettings.slaConfigs?.[0]?.targetUptime ?? 99.9}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  slaConfigs: [
                    {
                      ...(prev.slaConfigs?.[0] ?? {
                        endpointName: '*',
                        enabled: true,
                        targetLatencyP95: 500,
                        reportingPeriod: 'daily',
                      }),
                      targetUptime: Number.parseFloat(e.target.value) || 99.9,
                    },
                  ],
                }))
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <span className="caption text-text-muted">%</span>
          </div>
          <div className={cn('flex items-center', spacing.gap.compact)}>
            <label htmlFor="sla-target-latency" className="caption text-text-muted w-32">
              {t('health.targetLatency')}
            </label>
            <input
              id="sla-target-latency"
              type="number"
              min={10}
              max={10000}
              step={10}
              value={testsSettings.slaConfigs?.[0]?.targetLatencyP95 ?? 500}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  slaConfigs: [
                    {
                      ...(prev.slaConfigs?.[0] ?? {
                        endpointName: '*',
                        enabled: true,
                        targetUptime: 99.9,
                        reportingPeriod: 'daily',
                      }),
                      targetLatencyP95: Number.parseInt(e.target.value, 10) || 500,
                    },
                  ],
                }))
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-24')}
            />
            <span className="caption text-text-muted">ms (P95)</span>
          </div>
        </div>
      </div>
      {/* Alert Configuration */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.alertConfig')}</span>
        </div>
        <div className={spacing.stack.xs}>
          <label
            className={cn(
              layout.flex.between,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t('health.enableAlerts')}
              </span>
              <p className="caption text-text-muted">{t('health.alertsDescription')}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.alertConfig?.enabled ?? true}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  alertConfig: {
                    ...(prev.alertConfig ?? {
                      enabled: true,
                      consecutiveFailures: 3,
                      cooldownMinutes: 5,
                      digestMode: false,
                    }),
                    enabled: e.target.checked,
                  },
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>

          <div className={cn('flex items-center', spacing.gap.compact)}>
            <label htmlFor="alert-consecutive-failures" className="caption text-text-muted flex-1">
              {t('health.consecutiveFailures')}
            </label>
            <input
              id="alert-consecutive-failures"
              type="number"
              min={1}
              max={10}
              value={testsSettings.alertConfig?.consecutiveFailures ?? 3}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  alertConfig: {
                    ...(prev.alertConfig ?? {
                      enabled: true,
                      consecutiveFailures: 3,
                      cooldownMinutes: 5,
                      digestMode: false,
                    }),
                    consecutiveFailures: Number.parseInt(e.target.value, 10) || 3,
                  },
                }))
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
            />
          </div>

          <div className={cn('flex items-center', spacing.gap.compact)}>
            <label htmlFor="alert-cooldown-minutes" className="caption text-text-muted flex-1">
              {t('health.cooldownMinutes')}
            </label>
            <input
              id="alert-cooldown-minutes"
              type="number"
              min={1}
              max={60}
              value={testsSettings.alertConfig?.cooldownMinutes ?? 5}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  alertConfig: {
                    ...(prev.alertConfig ?? {
                      enabled: true,
                      consecutiveFailures: 3,
                      cooldownMinutes: 5,
                      digestMode: false,
                    }),
                    cooldownMinutes: Number.parseInt(e.target.value, 10) || 5,
                  },
                }))
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
            />
          </div>

          <label
            className={cn(
              layout.flex.between,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t('health.digestMode')}
              </span>
              <p className="caption text-text-muted">{t('health.digestDescription')}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.alertConfig?.digestMode ?? false}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  alertConfig: {
                    ...(prev.alertConfig ?? {
                      enabled: true,
                      consecutiveFailures: 3,
                      cooldownMinutes: 5,
                      digestMode: false,
                    }),
                    digestMode: e.target.checked,
                  },
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>
        </div>
      </div>
      {/* Anomaly Detection Configuration */}
      <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
          <span className="caption text-text-muted font-medium">{t('health.anomalyConfig')}</span>
        </div>
        <div className={spacing.stack.xs}>
          <label
            className={cn(
              layout.flex.between,
              spacing.pad.xs,
              'bg-surface-base border border-surface-border',
              radius.default,
            )}
          >
            <div>
              <span className="body-small text-text-primary font-medium">
                {t('health.enableAnomaly')}
              </span>
              <p className="caption text-text-muted">{t('health.anomalyDescription')}</p>
            </div>
            <input
              type="checkbox"
              checked={testsSettings.anomalyConfig?.enabled ?? true}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  anomalyConfig: {
                    ...(prev.anomalyConfig ?? {
                      enabled: true,
                      stdDevThreshold: 2,
                      maxSamples: 100,
                    }),
                    enabled: e.target.checked,
                  },
                }))
              }
              className={iconTokens.size.sm}
            />
          </label>

          <div className={cn('flex items-center', spacing.gap.compact)}>
            <label htmlFor="anomaly-std-dev-threshold" className="caption text-text-muted flex-1">
              {t('health.stdDevThreshold')}
            </label>
            <input
              id="anomaly-std-dev-threshold"
              type="number"
              min={1}
              max={5}
              step={0.5}
              value={testsSettings.anomalyConfig?.stdDevThreshold ?? 2}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  anomalyConfig: {
                    ...(prev.anomalyConfig ?? {
                      enabled: true,
                      stdDevThreshold: 2,
                      maxSamples: 100,
                    }),
                    stdDevThreshold: Number.parseFloat(e.target.value) || 2,
                  },
                }))
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
            />
          </div>

          <div className={cn('flex items-center', spacing.gap.compact)}>
            <label htmlFor="anomaly-max-samples" className="caption text-text-muted flex-1">
              {t('health.maxSamples')}
            </label>
            <input
              id="anomaly-max-samples"
              type="number"
              min={10}
              max={500}
              step={10}
              value={testsSettings.anomalyConfig?.maxSamples ?? 100}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTestsSettings((prev) => ({
                  ...prev,
                  anomalyConfig: {
                    ...(prev.anomalyConfig ?? {
                      enabled: true,
                      stdDevThreshold: 2,
                      maxSamples: 100,
                    }),
                    maxSamples: Number.parseInt(e.target.value, 10) || 100,
                  },
                }))
              }
              className={cn(input.base, input.state.default, input.size.md, 'w-20 text-center')}
            />
          </div>
        </div>
      </div>
    </>
  );
}
