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
import type { SaveStatus, SettingsThresholds } from '../../../types/settings';
import { THRESHOLD_HELP } from '../../help/HelpContent';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Info, SlidersHorizontal } from '../../ui/icons';
import { Tooltip } from '../../ui/tooltip';
import { AutoSaveIndicator } from './AutoSaveIndicator';

interface ThresholdsSettingsProps {
  thresholds: SettingsThresholds;
  setThresholds: React.Dispatch<React.SetStateAction<SettingsThresholds>>;
  thresholdsStatus: SaveStatus;
}

/**
 * Settings section for configuring alert thresholds across metrics.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const ThresholdsSettings: React.NamedExoticComponent<ThresholdsSettingsProps> = memo(
  function thresholdsSettings({
    thresholds,
    setThresholds,
    thresholdsStatus,
  }: ThresholdsSettingsProps) {
    const { t } = useTranslation('settings');

    // Type-safe threshold category getter
    function getThresholdCategory(
      prev: SettingsThresholds,
      category: keyof Omit<SettingsThresholds, 'httpTimings'>,
    ): { good: number; warning: number } {
      switch (category) {
        case 'dns':
          return prev.dns;
        case 'gateway':
          return prev.gateway;
        case 'wifi':
          return prev.wifi;
        case 'customPing':
          return prev.customPing;
        case 'customTcp':
          return prev.customTcp;
        case 'customHttp':
          return prev.customHttp;
        default:
          return prev.dns;
      }
    }

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

    const updateThreshold = (
      category: keyof Omit<SettingsThresholds, 'httpTimings'>,
      level: 'good' | 'warning',
      value: number,
    ): void => {
      setThresholds((prev) => {
        const current = getThresholdCategory(prev, category);
        const updated =
          level === 'good' ? { ...current, good: value } : { ...current, warning: value };
        return { ...prev, [category]: updated };
      });
    };

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
      <CollapsibleSection
        title={
          <div class={layout.inline.default}>
            <SlidersHorizontal class={iconTokens.size.sm} />
            <span>{t('sections.thresholds')}</span>
            <AutoSaveIndicator status={thresholdsStatus} />
          </div>
        }
      >
        <div class="stack-sm">
          {/* DNS Thresholds */}
          <div
            class={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
          >
            <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span class="body-small font-medium text-text-primary">
                {t('thresholds.dnsLookup')}
              </span>
              <Tooltip content={THRESHOLD_HELP.dnsLookup} position="top">
                <Info
                  class={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </Tooltip>
            </div>
            <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
              <div>
                <label class="caption text-text-muted" for="dns-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="dns-good"
                  type="number"
                  value={thresholds.dns.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('dns', 'good', Number(e.target.value))
                  }
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    spacing.margin.top.tight,
                    'body-small',
                  )}
                />
              </div>
              <div>
                <label class="caption text-text-muted" for="dns-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="dns-warning"
                  type="number"
                  value={thresholds.dns.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('dns', 'warning', Number(e.target.value))
                  }
                  class={cn(
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

          {/* Gateway Thresholds */}
          <div
            class={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
          >
            <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span class="body-small font-medium text-text-primary">
                {t('thresholds.gatewayPing')}
              </span>
              <Tooltip content={THRESHOLD_HELP.gatewayPing} position="top">
                <Info
                  class={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </Tooltip>
            </div>
            <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
              <div>
                <label class="caption text-text-muted" for="gateway-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="gateway-good"
                  type="number"
                  value={thresholds.gateway.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('gateway', 'good', Number(e.target.value))
                  }
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    spacing.margin.top.tight,
                    'body-small',
                  )}
                />
              </div>
              <div>
                <label class="caption text-text-muted" for="gateway-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="gateway-warning"
                  type="number"
                  value={thresholds.gateway.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('gateway', 'warning', Number(e.target.value))
                  }
                  class={cn(
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

          {/* Wi-Fi Signal Thresholds */}
          <div
            class={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
          >
            <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span class="body-small font-medium text-text-primary">
                {t('thresholds.wifiSignal')}
              </span>
              <Tooltip content={THRESHOLD_HELP.wifiSignal} position="top">
                <Info
                  class={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </Tooltip>
            </div>
            <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
              <div>
                <label class="caption text-text-muted" for="wifi-good">
                  {t('thresholds.goodGreater')}
                </label>
                <input
                  id="wifi-good"
                  type="number"
                  value={thresholds.wifi.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('wifi', 'good', Number(e.target.value))
                  }
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    spacing.margin.top.tight,
                    'body-small',
                  )}
                />
              </div>
              <div>
                <label class="caption text-text-muted" for="wifi-warning">
                  {t('thresholds.warningGreater')}
                </label>
                <input
                  id="wifi-warning"
                  type="number"
                  value={thresholds.wifi.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('wifi', 'warning', Number(e.target.value))
                  }
                  class={cn(
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

          {/* Health Check Ping Thresholds */}
          <div
            class={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
          >
            <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span class="body-small font-medium text-text-primary">
                {t('thresholds.healthPing')}
              </span>
              <Tooltip content={THRESHOLD_HELP.healthCheckPing} position="top">
                <Info
                  class={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </Tooltip>
            </div>
            <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
              <div>
                <label class="caption text-text-muted" for="ping-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="ping-good"
                  type="number"
                  value={thresholds.customPing.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customPing', 'good', Number(e.target.value))
                  }
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    spacing.margin.top.tight,
                    'body-small',
                  )}
                />
              </div>
              <div>
                <label class="caption text-text-muted" for="ping-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="ping-warning"
                  type="number"
                  value={thresholds.customPing.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customPing', 'warning', Number(e.target.value))
                  }
                  class={cn(
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

          {/* Health Check TCP Thresholds */}
          <div
            class={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
          >
            <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span class="body-small font-medium text-text-primary">
                {t('thresholds.healthTcp')}
              </span>
              <Tooltip content={THRESHOLD_HELP.healthCheckTcp} position="top">
                <Info
                  class={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </Tooltip>
            </div>
            <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
              <div>
                <label class="caption text-text-muted" for="tcp-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="tcp-good"
                  type="number"
                  value={thresholds.customTcp.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customTcp', 'good', Number(e.target.value))
                  }
                  class={cn(
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    spacing.margin.top.tight,
                    'body-small',
                  )}
                />
              </div>
              <div>
                <label class="caption text-text-muted" for="tcp-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="tcp-warning"
                  type="number"
                  value={thresholds.customTcp.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customTcp', 'warning', Number(e.target.value))
                  }
                  class={cn(
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

          {/* HTTP Thresholds (Total + Timing Phases) */}
          <div
            class={cn(spacing.pad.sm, 'bg-surface-base', radius.md, 'border border-surface-border')}
          >
            <span
              class={cn(
                'body-small font-medium text-text-primary block',
                spacing.margin.bottom.inline,
              )}
            >
              {t('thresholds.httpThresholds')}
            </span>

            {/* Total */}
            <div class={spacing.margin.bottom.heading}>
              <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
                <span class="caption font-medium text-text-primary">
                  {t('thresholds.totalResponseTime')}
                </span>
                <Tooltip content={THRESHOLD_HELP.httpTotal} position="top">
                  <Info
                    class={cn(
                      iconTokens.size.xs,
                      'text-text-muted hover:text-text-secondary cursor-help',
                    )}
                  />
                </Tooltip>
              </div>
              <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
                <div>
                  <label class="caption text-text-muted" for="http-total-good">
                    {t('thresholds.goodLess')}
                  </label>
                  <input
                    id="http-total-good"
                    type="number"
                    value={thresholds.customHttp.good}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateThreshold('customHttp', 'good', Number(e.target.value))
                    }
                    class={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.sm,
                      spacing.margin.top.tight,
                      'body-small',
                    )}
                  />
                </div>
                <div>
                  <label class="caption text-text-muted" for="http-total-warning">
                    {t('thresholds.warningLess')}
                  </label>
                  <input
                    id="http-total-warning"
                    type="number"
                    value={thresholds.customHttp.warning}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateThreshold('customHttp', 'warning', Number(e.target.value))
                    }
                    class={cn(
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
              class={cn(
                'caption text-text-muted',
                spacing.margin.bottom.heading,
                'border-t border-surface-border',
                spacing.pad.sm,
              )}
            >
              {t('thresholds.perPhaseThresholds')}
            </p>

            {/* DNS */}
            <div class={spacing.margin.bottom.heading}>
              <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
                <span class="caption font-medium text-text-primary">
                  {t('thresholds.dnsLookupPhase')}
                </span>
                <Tooltip content={THRESHOLD_HELP.httpDns} position="top">
                  <Info
                    class={cn(
                      iconTokens.size.xs,
                      'text-text-muted hover:text-text-secondary cursor-help',
                    )}
                  />
                </Tooltip>
              </div>
              <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
                <div>
                  <label class="caption text-text-muted" for="http-dns-good">
                    {t('thresholds.goodLess')}
                  </label>
                  <input
                    id="http-dns-good"
                    type="number"
                    value={thresholds.httpTimings.dns.good}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('dns', 'good', Number(e.target.value))
                    }
                    class={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.sm,
                      spacing.margin.top.tight,
                      'body-small',
                    )}
                  />
                </div>
                <div>
                  <label class="caption text-text-muted" for="http-dns-warning">
                    {t('thresholds.warningLess')}
                  </label>
                  <input
                    id="http-dns-warning"
                    type="number"
                    value={thresholds.httpTimings.dns.warning}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('dns', 'warning', Number(e.target.value))
                    }
                    class={cn(
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
            <div class={spacing.margin.bottom.heading}>
              <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
                <span class="caption font-medium text-text-primary">
                  {t('thresholds.tcpConnect')}
                </span>
                <Tooltip content={THRESHOLD_HELP.httpTcp} position="top">
                  <Info
                    class={cn(
                      iconTokens.size.xs,
                      'text-text-muted hover:text-text-secondary cursor-help',
                    )}
                  />
                </Tooltip>
              </div>
              <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
                <div>
                  <label class="caption text-text-muted" for="http-tcp-good">
                    {t('thresholds.goodLess')}
                  </label>
                  <input
                    id="http-tcp-good"
                    type="number"
                    value={thresholds.httpTimings.tcp.good}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('tcp', 'good', Number(e.target.value))
                    }
                    class={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.sm,
                      spacing.margin.top.tight,
                      'body-small',
                    )}
                  />
                </div>
                <div>
                  <label class="caption text-text-muted" for="http-tcp-warning">
                    {t('thresholds.warningLess')}
                  </label>
                  <input
                    id="http-tcp-warning"
                    type="number"
                    value={thresholds.httpTimings.tcp.warning}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('tcp', 'warning', Number(e.target.value))
                    }
                    class={cn(
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
            <div class={spacing.margin.bottom.heading}>
              <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
                <span class="caption font-medium text-text-primary">
                  {t('thresholds.tlsHandshake')}
                </span>
                <Tooltip content={THRESHOLD_HELP.httpTls} position="top">
                  <Info
                    class={cn(
                      iconTokens.size.xs,
                      'text-text-muted hover:text-text-secondary cursor-help',
                    )}
                  />
                </Tooltip>
              </div>
              <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
                <div>
                  <label class="caption text-text-muted" for="http-tls-good">
                    {t('thresholds.goodLess')}
                  </label>
                  <input
                    id="http-tls-good"
                    type="number"
                    value={thresholds.httpTimings.tls.good}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('tls', 'good', Number(e.target.value))
                    }
                    class={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.sm,
                      spacing.margin.top.tight,
                      'body-small',
                    )}
                  />
                </div>
                <div>
                  <label class="caption text-text-muted" for="http-tls-warning">
                    {t('thresholds.warningLess')}
                  </label>
                  <input
                    id="http-tls-warning"
                    type="number"
                    value={thresholds.httpTimings.tls.warning}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('tls', 'warning', Number(e.target.value))
                    }
                    class={cn(
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
              <div class={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
                <span class="caption font-medium text-text-primary">{t('thresholds.ttfb')}</span>
                <Tooltip content={THRESHOLD_HELP.httpTtfb} position="top">
                  <Info
                    class={cn(
                      iconTokens.size.xs,
                      'text-text-muted hover:text-text-secondary cursor-help',
                    )}
                  />
                </Tooltip>
              </div>
              <div class={cn('grid grid-cols-2', spacing.gap.compact)}>
                <div>
                  <label class="caption text-text-muted" for="http-ttfb-good">
                    {t('thresholds.goodLess')}
                  </label>
                  <input
                    id="http-ttfb-good"
                    type="number"
                    value={thresholds.httpTimings.ttfb.good}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('ttfb', 'good', Number(e.target.value))
                    }
                    class={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.sm,
                      spacing.margin.top.tight,
                      'body-small',
                    )}
                  />
                </div>
                <div>
                  <label class="caption text-text-muted" for="http-ttfb-warning">
                    {t('thresholds.warningLess')}
                  </label>
                  <input
                    id="http-ttfb-warning"
                    type="number"
                    value={thresholds.httpTimings.ttfb.warning}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      updateHttpTimingThreshold('ttfb', 'warning', Number(e.target.value))
                    }
                    class={cn(
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
        </div>
      </CollapsibleSection>
    );
  },
);
