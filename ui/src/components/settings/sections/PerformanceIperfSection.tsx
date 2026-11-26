import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { RadioGroup } from '../../../shared/RadioGroup';
import { Toggle } from '../../../shared/Toggle';
import { Tooltip } from '../../../shared/Tooltip';
import { Info } from '../../ui/Icons';
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
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import type { IperfSuggestion } from '../../../types/generated/iperf-suggestion';
import type { IperfSettings } from '../../../types/settings';

const DIRECTIONS = ['download', 'upload', 'bidirectional'] as const;

interface PerformanceIperfSectionProps {
  iperfSettings: IperfSettings;
  setIperfSettings: React.Dispatch<React.SetStateAction<IperfSettings>>;
  iperfSuggestions: IperfSuggestion[];
  iperfSuggestionsStatus: 'idle' | 'loading' | 'error';
  iperfSuggestionsError: string | null;
  fetchIperfSuggestions: () => void;
}

export function PerformanceIperfSection({
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
          <Tooltip text={readOnlyReason}>
            {(description) => (
              <fieldset disabled={!canWrite} className="stack min-w-0">
                {readOnlyReason ? (
                  <legend>
                    <button
                      type="button"
                      {...description}
                      aria-label={readOnlyReason}
                      className="text-text-muted inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
                    >
                      <Info className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </legend>
                ) : null}
                <FormField id="iperf-server-address" label={t('performance.serverAddress')}>
                  {(control) => (
                    <Input
                      {...control}
                      type="text"
                      value={iperfSettings.server}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          server: e.target.value,
                        }))
                      }
                      placeholder="192.168.1.100"
                    />
                  )}
                </FormField>
              </fieldset>
            )}
          </Tooltip>
          <div className={cn(layout.flex.between, spacing.margin.top.inline)}>
            <button
              type="button"
              disabled={iperfSuggestionsStatus === 'loading'}
              onClick={fetchIperfSuggestions}
              className="caption text-brand-primary hover:underline disabled:text-text-disabled disabled:cursor-not-allowed"
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
                <circle className="stroke-current/25" cx="12" cy="12" r="10" strokeWidth="4" />
                <path
                  className="fill-current/75"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
            )}
          </div>
          <Tooltip text={readOnlyReason}>
            {(description) => (
              <fieldset disabled={!canWrite} className="stack min-w-0">
                {readOnlyReason ? (
                  <legend>
                    <button
                      type="button"
                      {...description}
                      aria-label={readOnlyReason}
                      className="text-text-muted inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
                    >
                      <Info className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </legend>
                ) : null}
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
            )}
          </Tooltip>
        </div>

        <Tooltip text={readOnlyReason}>
          {(description) => (
            <fieldset disabled={!canWrite} className="stack min-w-0">
              {readOnlyReason ? (
                <legend>
                  <button
                    type="button"
                    {...description}
                    aria-label={readOnlyReason}
                    className="text-text-muted inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
                  >
                    <Info className="h-4 w-4" aria-hidden="true" />
                  </button>
                </legend>
              ) : null}
              {/* Port */}
              <FormField id="iperf-port" label={t('performance.port')}>
                {(control) => (
                  <Input
                    {...control}
                    type="number"
                    value={iperfSettings.port}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      setIperfSettings((prev) => ({
                        ...prev,
                        port: Number.parseInt(e.target.value, 10) || 5201,
                      }))
                    }
                  />
                )}
              </FormField>

              {/* Protocol Toggle */}
              <RadioGroup
                label={t('performance.protocol')}
                name="iperf-protocol"
                orientation="horizontal"
                value={iperfSettings.protocol}
                onValueChange={(value): void =>
                  setIperfSettings((prev) => ({
                    ...prev,
                    protocol: value === 'udp' ? 'udp' : 'tcp',
                  }))
                }
                options={[
                  { value: 'tcp', label: 'TCP' },
                  { value: 'udp', label: 'UDP' },
                ]}
              />

              {/* Direction Toggle */}
              <RadioGroup
                label={t('performance.direction')}
                name="iperf-direction"
                orientation="horizontal"
                value={iperfSettings.direction}
                onValueChange={(value): void => {
                  const next = DIRECTIONS.find((direction) => direction === value);
                  if (next) {
                    setIperfSettings((prev) => ({ ...prev, direction: next }));
                  }
                }}
                options={DIRECTIONS.map((direction) => ({
                  value: direction,
                  label: getDirectionLabel(direction),
                }))}
              />

              {/* Duration */}
              <FormField id="iperf-duration" label={t('performance.duration')}>
                {(control) => (
                  <Input
                    {...control}
                    type="number"
                    value={iperfSettings.duration}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      setIperfSettings((prev) => ({
                        ...prev,
                        duration: Number.parseInt(e.target.value, 10) || 10,
                      }))
                    }
                    min={1}
                    max={60}
                  />
                )}
              </FormField>

              {/* Server Mode */}
              <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
                <Toggle
                  label={t('performance.enableServer')}
                  checked={iperfSettings.enableServer}
                  onChange={(checked: boolean): void =>
                    setIperfSettings((prev) => ({
                      ...prev,
                      enableServer: checked,
                    }))
                  }
                  className="rounded-lg border border-surface-border bg-surface-base p-3"
                />
                <FormField id="iperf-server-port" label={t('performance.serverPort')}>
                  {(control) => (
                    <Input
                      {...control}
                      type="number"
                      value={iperfSettings.serverPort}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                        setIperfSettings((prev) => ({
                          ...prev,
                          serverPort: Number.parseInt(e.target.value, 10) || 5201,
                        }))
                      }
                    />
                  )}
                </FormField>
                <p className={cn('caption text-text-muted', spacing.margin.top.tight)}>
                  {t('performance.serverAutoStart')}
                </p>
              </div>
            </fieldset>
          )}
        </Tooltip>
      </div>
    </div>
  );
}
