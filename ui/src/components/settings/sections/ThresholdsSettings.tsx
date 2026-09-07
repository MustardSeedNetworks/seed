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
import type { SaveStatus, SettingsThresholds } from '../../../types/settings';
import { THRESHOLD_HELP } from '../../help/HelpContent';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Info, SlidersHorizontal } from '../../ui/icons';
import { Tooltip } from '../../ui/tooltip';
import { AutoSaveIndicator } from './AutoSaveIndicator';
import { ThresholdsHttpSection } from './ThresholdsHttpSection';

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
    const { canWrite } = useRole();
    const readOnlyReason = canWrite ? undefined : t('common.readOnly');

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

    return (
      <CollapsibleSection
        data-testid="thresholds-settings-section"
        readOnlyReason={readOnlyReason}
        title={
          <div className={layout.inline.default}>
            <SlidersHorizontal className={iconTokens.size.sm} />
            <span>{t('sections.thresholds')}</span>
            <AutoSaveIndicator status={thresholdsStatus} />
          </div>
        }
      >
        <div className="stack-sm">
          {/* DNS Thresholds */}
          <div
            className={cn(
              spacing.pad.sm,
              'bg-surface-base',
              radius.md,
              'border border-surface-border',
            )}
          >
            <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span className="body-small font-medium text-text-primary">
                {t('thresholds.dnsLookup')}
              </span>
              <Tooltip text={THRESHOLD_HELP.dnsLookup} side="top">
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
                <label className="caption text-text-muted" htmlFor="dns-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="dns-good"
                  type="number"
                  value={thresholds.dns.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('dns', 'good', Number(e.target.value))
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
                <label className="caption text-text-muted" htmlFor="dns-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="dns-warning"
                  type="number"
                  value={thresholds.dns.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('dns', 'warning', Number(e.target.value))
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

          {/* Gateway Thresholds */}
          <div
            className={cn(
              spacing.pad.sm,
              'bg-surface-base',
              radius.md,
              'border border-surface-border',
            )}
          >
            <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span className="body-small font-medium text-text-primary">
                {t('thresholds.gatewayPing')}
              </span>
              <Tooltip text={THRESHOLD_HELP.gatewayPing} side="top">
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
                <label className="caption text-text-muted" htmlFor="gateway-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="gateway-good"
                  type="number"
                  value={thresholds.gateway.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('gateway', 'good', Number(e.target.value))
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
                <label className="caption text-text-muted" htmlFor="gateway-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="gateway-warning"
                  type="number"
                  value={thresholds.gateway.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('gateway', 'warning', Number(e.target.value))
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

          {/* Wi-Fi Signal Thresholds */}
          <div
            className={cn(
              spacing.pad.sm,
              'bg-surface-base',
              radius.md,
              'border border-surface-border',
            )}
          >
            <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span className="body-small font-medium text-text-primary">
                {t('thresholds.wifiSignal')}
              </span>
              <Tooltip text={THRESHOLD_HELP.wifiSignal} side="top">
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
                <label className="caption text-text-muted" htmlFor="wifi-good">
                  {t('thresholds.goodGreater')}
                </label>
                <input
                  id="wifi-good"
                  type="number"
                  value={thresholds.wifi.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('wifi', 'good', Number(e.target.value))
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
                <label className="caption text-text-muted" htmlFor="wifi-warning">
                  {t('thresholds.warningGreater')}
                </label>
                <input
                  id="wifi-warning"
                  type="number"
                  value={thresholds.wifi.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('wifi', 'warning', Number(e.target.value))
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

          {/* Health Check Ping Thresholds */}
          <div
            className={cn(
              spacing.pad.sm,
              'bg-surface-base',
              radius.md,
              'border border-surface-border',
            )}
          >
            <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span className="body-small font-medium text-text-primary">
                {t('thresholds.healthPing')}
              </span>
              <Tooltip text={THRESHOLD_HELP.healthCheckPing} side="top">
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
                <label className="caption text-text-muted" htmlFor="ping-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="ping-good"
                  type="number"
                  value={thresholds.customPing.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customPing', 'good', Number(e.target.value))
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
                <label className="caption text-text-muted" htmlFor="ping-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="ping-warning"
                  type="number"
                  value={thresholds.customPing.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customPing', 'warning', Number(e.target.value))
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

          {/* Health Check TCP Thresholds */}
          <div
            className={cn(
              spacing.pad.sm,
              'bg-surface-base',
              radius.md,
              'border border-surface-border',
            )}
          >
            <div className={cn(layout.inline.tight, spacing.margin.bottom.inline)}>
              <span className="body-small font-medium text-text-primary">
                {t('thresholds.healthTcp')}
              </span>
              <Tooltip text={THRESHOLD_HELP.healthCheckTcp} side="top">
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
                <label className="caption text-text-muted" htmlFor="tcp-good">
                  {t('thresholds.goodLess')}
                </label>
                <input
                  id="tcp-good"
                  type="number"
                  value={thresholds.customTcp.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customTcp', 'good', Number(e.target.value))
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
                <label className="caption text-text-muted" htmlFor="tcp-warning">
                  {t('thresholds.warningLess')}
                </label>
                <input
                  id="tcp-warning"
                  type="number"
                  value={thresholds.customTcp.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    updateThreshold('customTcp', 'warning', Number(e.target.value))
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

          <ThresholdsHttpSection
            thresholds={thresholds}
            setThresholds={setThresholds}
            updateThreshold={updateThreshold}
          />
        </div>
      </CollapsibleSection>
    );
  },
);
