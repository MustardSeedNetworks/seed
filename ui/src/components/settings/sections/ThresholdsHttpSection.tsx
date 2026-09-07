/**
 * The HTTP half of the thresholds settings section — total request budget plus
 * the four timing phases (DNS, TCP, TLS, TTFB).
 *
 * Split out of `ThresholdsSettings` so that file could take the viewer
 * read-only gate (#2467): it sat on the shrink-only size baseline, and the gate
 * could not be added to a file that may not grow. No behaviour change.
 */

import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
} from '../../../styles/theme';
import type { SettingsThresholds } from '../../../types/settings';
import { THRESHOLD_HELP } from '../../help/HelpContent';
import { Info } from '../../ui/icons';
import { Tooltip } from '../../ui/tooltip';

interface ThresholdsHttpSectionProps {
  thresholds: SettingsThresholds;
  setThresholds: React.Dispatch<React.SetStateAction<SettingsThresholds>>;
  /** The parent's category updater; the total-request row writes `customHttp`. */
  updateThreshold: (
    category: keyof Omit<SettingsThresholds, 'httpTimings'>,
    level: 'good' | 'warning',
    value: number,
  ) => void;
}

export const ThresholdsHttpSection: React.NamedExoticComponent<ThresholdsHttpSectionProps> = memo(
  function thresholdsHttpSection({
    thresholds,
    setThresholds,
    updateThreshold,
  }: ThresholdsHttpSectionProps) {
    const { t } = useTranslation('settings');

    // Type-safe HTTP timing phase getter
    function getHttpTimingPhase(
      httpTimings: SettingsThresholds['httpTimings'],
      phase: keyof SettingsThresholds['httpTimings'],
    ): { good: number; warning: number } {
      switch (phase) {
        case 'dns':
          return httpTimings.dns;
        case 'tcp':
          return httpTimings.tcp;
        case 'tls':
          return httpTimings.tls;
        case 'ttfb':
          return httpTimings.ttfb;
        default:
          return httpTimings.dns;
      }
    }

    const updateHttpTimingThreshold = (
      phase: keyof SettingsThresholds['httpTimings'],
      level: 'good' | 'warning',
      value: number,
    ): void => {
      setThresholds((prev) => {
        const current = getHttpTimingPhase(prev.httpTimings, phase);
        const updated =
          level === 'good' ? { ...current, good: value } : { ...current, warning: value };
        return {
          ...prev,
          httpTimings: { ...prev.httpTimings, [phase]: updated },
        };
      });
    };

    return (
      <div
        className={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
      >
        <span
          className={cn(
            'body-small font-medium text-text-primary block',
            spacing.margin.bottom.inline,
          )}
        >
          {t('thresholds.httpThresholds')}
        </span>

        {/* Total */}
        <div className={spacing.margin.bottom.heading}>
          <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
            <span className="caption font-medium text-text-primary">
              {t('thresholds.totalResponseTime')}
            </span>
            <Tooltip text={THRESHOLD_HELP.httpTotal} side="top">
              <Info
                className={cn(
                  iconTokens.size.xs,
                  'text-text-muted hover:text-text-secondary cursor-help',
                )}
              />
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <div>
              <label className="caption text-text-muted" htmlFor="http-total-good">
                {t('thresholds.goodLess')}
              </label>
              <input
                id="http-total-good"
                type="number"
                value={thresholds.customHttp.good}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateThreshold('customHttp', 'good', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="http-total-warning">
                {t('thresholds.warningLess')}
              </label>
              <input
                id="http-total-warning"
                type="number"
                value={thresholds.customHttp.warning}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateThreshold('customHttp', 'warning', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
          </div>
        </div>

        <p
          className={cn(
            'caption text-text-muted',
            spacing.margin.bottom.heading,
            'border-t border-surface-border',
            spacing.pad.sm,
          )}
        >
          {t('thresholds.perPhaseThresholds')}
        </p>

        {/* DNS */}
        <div className={spacing.margin.bottom.heading}>
          <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
            <span className="caption font-medium text-text-primary">
              {t('thresholds.dnsLookupPhase')}
            </span>
            <Tooltip text={THRESHOLD_HELP.httpDns} side="top">
              <Info
                className={cn(
                  iconTokens.size.xs,
                  'text-text-muted hover:text-text-secondary cursor-help',
                )}
              />
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <div>
              <label className="caption text-text-muted" htmlFor="http-dns-good">
                {t('thresholds.goodLess')}
              </label>
              <input
                id="http-dns-good"
                type="number"
                value={thresholds.httpTimings.dns.good}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('dns', 'good', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="http-dns-warning">
                {t('thresholds.warningLess')}
              </label>
              <input
                id="http-dns-warning"
                type="number"
                value={thresholds.httpTimings.dns.warning}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('dns', 'warning', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
          </div>
        </div>

        {/* TCP */}
        <div className={spacing.margin.bottom.heading}>
          <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
            <span className="caption font-medium text-text-primary">
              {t('thresholds.tcpConnect')}
            </span>
            <Tooltip text={THRESHOLD_HELP.httpTcp} side="top">
              <Info
                className={cn(
                  iconTokens.size.xs,
                  'text-text-muted hover:text-text-secondary cursor-help',
                )}
              />
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <div>
              <label className="caption text-text-muted" htmlFor="http-tcp-good">
                {t('thresholds.goodLess')}
              </label>
              <input
                id="http-tcp-good"
                type="number"
                value={thresholds.httpTimings.tcp.good}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('tcp', 'good', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="http-tcp-warning">
                {t('thresholds.warningLess')}
              </label>
              <input
                id="http-tcp-warning"
                type="number"
                value={thresholds.httpTimings.tcp.warning}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('tcp', 'warning', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
          </div>
        </div>

        {/* TLS */}
        <div className={spacing.margin.bottom.heading}>
          <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
            <span className="caption font-medium text-text-primary">
              {t('thresholds.tlsHandshake')}
            </span>
            <Tooltip text={THRESHOLD_HELP.httpTls} side="top">
              <Info
                className={cn(
                  iconTokens.size.xs,
                  'text-text-muted hover:text-text-secondary cursor-help',
                )}
              />
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <div>
              <label className="caption text-text-muted" htmlFor="http-tls-good">
                {t('thresholds.goodLess')}
              </label>
              <input
                id="http-tls-good"
                type="number"
                value={thresholds.httpTimings.tls.good}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('tls', 'good', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="http-tls-warning">
                {t('thresholds.warningLess')}
              </label>
              <input
                id="http-tls-warning"
                type="number"
                value={thresholds.httpTimings.tls.warning}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('tls', 'warning', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
          </div>
        </div>

        {/* TTFB */}
        <div>
          <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
            <span className="caption font-medium text-text-primary">{t('thresholds.ttfb')}</span>
            <Tooltip text={THRESHOLD_HELP.httpTtfb} side="top">
              <Info
                className={cn(
                  iconTokens.size.xs,
                  'text-text-muted hover:text-text-secondary cursor-help',
                )}
              />
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <div>
              <label className="caption text-text-muted" htmlFor="http-ttfb-good">
                {t('thresholds.goodLess')}
              </label>
              <input
                id="http-ttfb-good"
                type="number"
                value={thresholds.httpTimings.ttfb.good}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('ttfb', 'good', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
            <div>
              <label className="caption text-text-muted" htmlFor="http-ttfb-warning">
                {t('thresholds.warningLess')}
              </label>
              <input
                id="http-ttfb-warning"
                type="number"
                value={thresholds.httpTimings.ttfb.warning}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  updateHttpTimingThreshold('ttfb', 'warning', Number(e.target.value))
                }
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  spacing.margin.top.tight,
                  'body-small',
                )}
              />
            </div>
          </div>
        </div>
      </div>
    );
  },
);
