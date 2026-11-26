import type React from 'react';
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Tooltip } from '../../../shared/Tooltip';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../../styles/theme';
import type { ThresholdSettings } from '../../../types/generated/settings-response';
import type { SaveStatus } from '../../../types/settings';
import { Info, SlidersHorizontal } from '../../ui/Icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';
import { ThresholdsHttpSection } from './ThresholdsHttpSection';

interface ThresholdsSettingsProps {
  thresholds: ThresholdSettings;
  setThresholds: React.Dispatch<React.SetStateAction<ThresholdSettings>>;
  thresholdsStatus: SaveStatus;
}

/**
 * Settings section for configuring alert thresholds across metrics.
 */
export function ThresholdsSettings({
  thresholds,
  setThresholds,
  thresholdsStatus,
}: ThresholdsSettingsProps) {
  const { t } = useTranslation(['settings', 'help']);
  const { canWrite } = useRole();
  const readOnlyReason = canWrite ? undefined : t('common.readOnly');

  // Type-safe threshold category getter
  function getThresholdCategory(
    prev: ThresholdSettings,
    category: keyof Omit<ThresholdSettings, 'httpTimings'>,
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
    category: keyof Omit<ThresholdSettings, 'httpTimings'>,
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
            <Tooltip text={t('help:thresholds.dnsLookup')} side="top">
              <button
                type="button"
                aria-label={t('help:thresholds.dnsLookup')}
                className="inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
              >
                <Info
                  className={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </button>
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <FormField id="dns-good" label={t('thresholds.goodLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.dns.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('dns', 'good', Number(e.target.value))
                  }
                />
              )}
            </FormField>
            <FormField id="dns-warning" label={t('thresholds.warningLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.dns.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('dns', 'warning', Number(e.target.value))
                  }
                />
              )}
            </FormField>
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
            <Tooltip text={t('help:thresholds.gatewayPing')} side="top">
              <button
                type="button"
                aria-label={t('help:thresholds.gatewayPing')}
                className="inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
              >
                <Info
                  className={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </button>
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <FormField id="gateway-good" label={t('thresholds.goodLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.gateway.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('gateway', 'good', Number(e.target.value))
                  }
                />
              )}
            </FormField>
            <FormField id="gateway-warning" label={t('thresholds.warningLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.gateway.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('gateway', 'warning', Number(e.target.value))
                  }
                />
              )}
            </FormField>
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
            <Tooltip text={t('help:thresholds.wifiSignal')} side="top">
              <button
                type="button"
                aria-label={t('help:thresholds.wifiSignal')}
                className="inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
              >
                <Info
                  className={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </button>
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <FormField id="wifi-good" label={t('thresholds.goodGreater')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.wifi.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('wifi', 'good', Number(e.target.value))
                  }
                />
              )}
            </FormField>
            <FormField id="wifi-warning" label={t('thresholds.warningGreater')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.wifi.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('wifi', 'warning', Number(e.target.value))
                  }
                />
              )}
            </FormField>
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
            <Tooltip text={t('help:thresholds.healthCheckPing')} side="top">
              <button
                type="button"
                aria-label={t('help:thresholds.healthCheckPing')}
                className="inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
              >
                <Info
                  className={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </button>
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <FormField id="ping-good" label={t('thresholds.goodLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.customPing.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('customPing', 'good', Number(e.target.value))
                  }
                />
              )}
            </FormField>
            <FormField id="ping-warning" label={t('thresholds.warningLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.customPing.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('customPing', 'warning', Number(e.target.value))
                  }
                />
              )}
            </FormField>
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
            <Tooltip text={t('help:thresholds.healthCheckTcp')} side="top">
              <button
                type="button"
                aria-label={t('help:thresholds.healthCheckTcp')}
                className="inline-flex rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
              >
                <Info
                  className={cn(
                    iconTokens.size.xs,
                    'text-text-muted hover:text-text-secondary cursor-help',
                  )}
                />
              </button>
            </Tooltip>
          </div>
          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <FormField id="tcp-good" label={t('thresholds.goodLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.customTcp.good}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('customTcp', 'good', Number(e.target.value))
                  }
                />
              )}
            </FormField>
            <FormField id="tcp-warning" label={t('thresholds.warningLess')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  value={thresholds.customTcp.warning}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    updateThreshold('customTcp', 'warning', Number(e.target.value))
                  }
                />
              )}
            </FormField>
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
}
