/**
 * The iperf3 (LAN speed) half of the performance settings section.
 *
 * Split out of `PerformanceSettings` so that file could take the viewer
 * read-only gate (#2467): it sat on the shrink-only size baseline, and the
 * gate could not be added to a file that may not grow.
 *
 * The gate is per-group rather than one fieldset over the body, because the
 * "find iperf hosts" button is a *read* — a viewer may still scan for public
 * servers — and it sits between the server-address input and the suggestion
 * chips, both of which write.
 */

import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from '../../../styles/theme';
import type { IperfSettings, IperfSuggestion } from '../../../types/settings';

interface PerformanceIperfSectionProps {
  iperfSettings: IperfSettings;
  setIperfSettings: React.Dispatch<React.SetStateAction<IperfSettings>>;
  iperfSuggestions: IperfSuggestion[];
  iperfSuggestionsStatus: 'idle' | 'loading' | 'error';
  iperfSuggestionsError: string | null;
  fetchIperfSuggestions: () => void;
}

export const PerformanceIperfSection: React.NamedExoticComponent<PerformanceIperfSectionProps> =
  memo(function performanceIperfSection({
    iperfSettings,
    setIperfSettings,
    iperfSuggestions,
    iperfSuggestionsStatus,
    iperfSuggestionsError,
    fetchIperfSuggestions,
  }: PerformanceIperfSectionProps) {
    const { t } = useTranslation('settings');
    const { canWrite } = useRole();
    const readOnlyReason = canWrite ? undefined : t('common.readOnly');

    const getDirectionLabel = (direction: string): string => {
      switch (direction) {
        case 'download':
          return t('performance.download');
        case 'upload':
          return t('performance.upload');
        case 'bidirectional':
          return t('performance.both');
        default:
          return direction;
      }
    };

    return (
      <div>
        <h4
          className={cn(
            'body-small font-semibold text-text-primary',
            spacing.margin.bottom.inline,
            'uppercase tracking-wide',
          )}
        >
          {t('performance.lanSpeed')}
        </h4>
        <div className="stack">
          <p className="caption text-text-muted">{t('performance.lanSpeedDesc')}</p>

          {/* Server Address */}
          <div>
            <fieldset disabled={!canWrite} title={readOnlyReason} className="stack min-w-0">
              <label htmlFor="iperf-server-address" className="caption text-text-muted font-medium">
                {t('performance.serverAddress')}
              </label>
              <input
                id="iperf-server-address"
                type="text"
                value={iperfSettings.server}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    server: e.target.value,
                  }))
                }
                placeholder="192.168.1.100"
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  'w-full',
                  spacing.margin.top.tight,
                  'body-small disabled:opacity-60',
                )}
              />
            </fieldset>
            <div className={cn(layout.flex.between, spacing.margin.top.inline)}>
              <button
                type="button"
                disabled={iperfSuggestionsStatus === 'loading'}
                onClick={fetchIperfSuggestions}
                className="caption text-brand-primary hover:underline disabled:opacity-60 disabled:cursor-not-allowed"
              >
                {iperfSuggestionsStatus === 'loading'
                  ? t('performance.scanning')
                  : t('performance.findIperfHosts')}
              </button>
              {iperfSuggestionsStatus === 'loading' && (
                <svg
                  className={cn(iconTokens.size.sm, 'animate-spin text-text-muted')}
                  viewBox="0 0 24 24"
                  fill="none"
                  aria-hidden="true"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
              )}
            </div>
            <fieldset disabled={!canWrite} title={readOnlyReason} className="stack min-w-0">
              {iperfSuggestionsStatus === 'error' && (
                <p className={cn('caption text-status-warning', spacing.margin.top.tight)}>
                  {iperfSuggestionsError || t('performance.noIperfHosts')}
                </p>
              )}
              {iperfSuggestions.length > 0 && (
                <div
                  className={cn('flex flex-wrap', spacing.gap.compact, spacing.margin.top.inline)}
                >
                  {iperfSuggestions.map((sugg) => (
                    <button
                      type="button"
                      key={`${sugg.host}-${sugg.hostname || ''}`}
                      className={cn(
                        spacing.chip.sm,
                        radius.full,
                        'border border-surface-border bg-surface-base caption text-text-primary hover:bg-surface-hover',
                      )}
                      onClick={(): void =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          server: sugg.host,
                        }))
                      }
                    >
                      <span className="font-medium">{sugg.hostname || sugg.host}</span>
                      <span className={cn('text-text-muted', spacing.margin.left.tight)}>
                        {sugg.hostname ? `(${sugg.host})` : ''}
                        {sugg.latencyMs !== undefined ? ` · ${Math.round(sugg.latencyMs)}ms` : ''}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </fieldset>
          </div>

          <fieldset disabled={!canWrite} title={readOnlyReason} className="stack min-w-0">
            {/* Port */}
            <div>
              <label className="caption text-text-muted font-medium" htmlFor="iperf-port">
                {t('performance.port')}
              </label>
              <input
                id="iperf-port"
                type="number"
                value={iperfSettings.port}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    port: Number.parseInt(e.target.value, 10) || 5201,
                  }))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  'w-full',
                  spacing.margin.top.tight,
                  'body-small disabled:opacity-60',
                )}
              />
            </div>

            {/* Protocol Toggle */}
            <div>
              <span
                className={cn(
                  'caption text-text-muted font-medium block',
                  spacing.margin.bottom.inline,
                )}
              >
                {t('performance.protocol')}
              </span>
              <div
                className={cn('flex flex-wrap', spacing.gap.compact)}
                role="radiogroup"
                aria-label="Protocol selection"
              >
                {(['tcp', 'udp'] as const).map((proto) => {
                  const checked = iperfSettings.protocol === proto;
                  return (
                    <label
                      key={proto}
                      className={cn(
                        'cursor-pointer',
                        spacing.chip.md,
                        radius.full,
                        'border body-small font-medium transition-colors',
                        checked
                          ? 'bg-brand-primary text-on-brand border-brand-primary'
                          : 'bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover',
                      )}
                    >
                      <input
                        type="radio"
                        name="iperf-protocol"
                        value={proto}
                        checked={checked}
                        onChange={(): void =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            protocol: proto,
                          }))
                        }
                        className="sr-only"
                        aria-label={`${proto.toUpperCase()} protocol`}
                      />
                      {proto.toUpperCase()}
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Direction Toggle */}
            <div>
              <span
                className={cn(
                  'caption text-text-muted font-medium block',
                  spacing.margin.bottom.inline,
                )}
              >
                {t('performance.direction')}
              </span>
              <div
                className={cn('flex flex-wrap', spacing.gap.compact)}
                role="radiogroup"
                aria-label="Direction selection"
              >
                {(['download', 'upload', 'bidirectional'] as const).map((direction) => {
                  const checked = iperfSettings.direction === direction;
                  return (
                    <label
                      key={direction}
                      className={cn(
                        'cursor-pointer',
                        spacing.chip.md,
                        radius.full,
                        'border body-small font-medium transition-colors',
                        checked
                          ? 'bg-brand-primary text-on-brand border-brand-primary'
                          : 'bg-surface-base border-surface-border text-text-primary hover:bg-surface-hover',
                      )}
                    >
                      <input
                        type="radio"
                        name="iperf-direction"
                        value={direction}
                        checked={checked}
                        onChange={(): void =>
                          setIperfSettings((prev) => ({
                            ...prev,
                            direction: direction,
                          }))
                        }
                        className="sr-only"
                        aria-label={`${getDirectionLabel(direction)} direction`}
                      />
                      {getDirectionLabel(direction)}
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Duration */}
            <div>
              <label className="caption text-text-muted font-medium" htmlFor="iperf-duration">
                {t('performance.duration')}
              </label>
              <input
                id="iperf-duration"
                type="number"
                value={iperfSettings.duration}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    duration: Number.parseInt(e.target.value, 10) || 10,
                  }))
                }
                min={1}
                max={60}
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  'w-full',
                  spacing.margin.top.tight,
                  'body-small disabled:opacity-60',
                )}
              />
            </div>

            {/* Server Mode */}
            <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
              <label
                className={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  'bg-surface-base',
                  radius.default,
                  'border border-surface-border',
                  spacing.margin.bottom.inline,
                )}
              >
                <span className="body-small text-text-primary">
                  {t('performance.enableServer')}
                </span>
                <input
                  type="checkbox"
                  checked={iperfSettings.enableServer}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      enableServer: e.target.checked,
                    }))
                  }
                  className={iconTokens.size.sm}
                />
              </label>
              <div>
                <label className="caption text-text-muted font-medium" htmlFor="iperf-server-port">
                  {t('performance.serverPort')}
                </label>
                <input
                  id="iperf-server-port"
                  type="number"
                  value={iperfSettings.serverPort}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      serverPort: Number.parseInt(e.target.value, 10) || 5201,
                    }))
                  }
                  className={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.md,
                    'w-full',
                    spacing.margin.top.tight,
                    'body-small disabled:opacity-60',
                  )}
                />
              </div>
              <p className={cn('caption text-text-muted', spacing.margin.top.tight)}>
                {t('performance.serverAutoStart')}
              </p>
            </div>
          </fieldset>
        </div>
      </div>
    );
  });
