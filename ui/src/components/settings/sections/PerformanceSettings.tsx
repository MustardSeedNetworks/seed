/**
 * Performance testing configuration: which tests run automatically, and the
 * speedtest.net server to use. The iperf3 (LAN speed) half lives in
 * `PerformanceIperfSection`, split out so this file could take the viewer
 * read-only gate (#2467) without growing past the size baseline.
 *
 * Everything this file renders is a write, but the gate is a fieldset around
 * the body rather than `readOnlyReason` on the CollapsibleSection: that
 * fieldset would reach into the iperf child and disable its "find hosts" read.
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
import type {
  CardSettings,
  IperfSettings,
  IperfSuggestion,
  SaveStatus,
  TestsSettings,
} from '../../../types/settings';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Gauge } from '../../ui/icons';
import { AutoSaveIndicator } from './AutoSaveIndicator';
import { PerformanceIperfSection } from './PerformanceIperfSection';

interface PerformanceSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  iperfSettings: IperfSettings;
  setIperfSettings: React.Dispatch<React.SetStateAction<IperfSettings>>;
  iperfStatus: SaveStatus;
  iperfSuggestions: IperfSuggestion[];
  iperfSuggestionsStatus: 'idle' | 'loading' | 'error';
  iperfSuggestionsError: string | null;
  fetchIperfSuggestions: () => void;
  /** Card settings for FAB auto-run configuration */
  cardSettings: CardSettings;
  /** Update card settings (triggers auto-save to profile) */
  updateCardSettings: (updates: Partial<CardSettings>) => void;
}

/**
 * Settings section for speed test and iPerf performance testing configuration.
 * Memoized to prevent unnecessary re-renders when parent state changes.
 */
export const PerformanceSettings: React.NamedExoticComponent<PerformanceSettingsProps> = memo(
  function performanceSettings({
    testsSettings,
    setTestsSettings,
    iperfSettings,
    setIperfSettings,
    iperfStatus,
    iperfSuggestions,
    iperfSuggestionsStatus,
    iperfSuggestionsError,
    fetchIperfSuggestions,
    cardSettings,
    updateCardSettings,
  }: PerformanceSettingsProps) {
    const { t } = useTranslation('settings');
    const { canWrite } = useRole();
    // Not `readOnlyReason` on the CollapsibleSection: that fieldset would cover
    // the whole body, and the iperf child carries a read (find hosts) a viewer
    // keeps. The write groups here take their own fieldset instead.
    const readOnlyReason = canWrite ? undefined : t('common.readOnly');

    return (
      <CollapsibleSection
        data-testid="performance-settings-section"
        title={
          <div className={layout.inline.default}>
            <Gauge className={iconTokens.size.sm} />
            <span>{t('sections.performance')}</span>
            <AutoSaveIndicator status={iperfStatus} />
          </div>
        }
        defaultOpen={false}
      >
        <div className="stack">
          <fieldset disabled={!canWrite} title={readOnlyReason} className="stack min-w-0">
            {/* Enable/Disable Toggles */}
            <div className="stack-sm">
              <label
                className={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  'bg-surface-base',
                  radius.default,
                  'border border-surface-border',
                )}
              >
                <div>
                  <span className="body-small text-text-primary font-medium">
                    {t('performance.enableSpeedtest')}
                  </span>
                  <p className="caption text-text-muted">{t('performance.speedtestDesc')}</p>
                </div>
                <input
                  type="checkbox"
                  checked={testsSettings.runSpeedtest}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      runSpeedtest: e.target.checked,
                    }))
                  }
                  className={iconTokens.size.sm}
                />
              </label>
              <label
                className={cn(
                  layout.flex.between,
                  spacing.pad.sm,
                  'bg-surface-base',
                  radius.default,
                  'border border-surface-border',
                )}
              >
                <div>
                  <span className="body-small text-text-primary font-medium">
                    {t('performance.enableIperf')}
                  </span>
                  <p className="caption text-text-muted">{t('performance.iperfDesc')}</p>
                </div>
                <input
                  type="checkbox"
                  checked={testsSettings.runIperf}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      runIperf: e.target.checked,
                    }))
                  }
                  className={iconTokens.size.sm}
                />
              </label>
            </div>

            {/* Auto-Run on Link Up (FAB button) */}
            <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
              <span className="caption text-text-muted font-medium">
                {t('performance.autoRunOnLink')}
              </span>
              <p className="caption text-text-muted mt-tight">
                {t('performance.autoRunOnLinkDesc')}
              </p>
              <div className={cn(spacing.margin.top.inline, 'stack-sm')}>
                <label
                  className={cn(
                    layout.flex.between,
                    spacing.pad.sm,
                    'bg-surface-base',
                    radius.default,
                    'border border-surface-border',
                  )}
                >
                  <span className="body-small text-text-primary">{t('performance.speedtest')}</span>
                  <input
                    type="checkbox"
                    checked={cardSettings.performance.speedtest.autoRunOnLink}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      updateCardSettings({
                        performance: {
                          ...cardSettings.performance,
                          speedtest: {
                            ...cardSettings.performance.speedtest,
                            autoRunOnLink: e.target.checked,
                          },
                        },
                      })
                    }
                    className={iconTokens.size.sm}
                  />
                </label>
                <label
                  className={cn(
                    layout.flex.between,
                    spacing.pad.sm,
                    'bg-surface-base',
                    radius.default,
                    'border border-surface-border',
                  )}
                >
                  <span className="body-small text-text-primary">{t('performance.iperf')}</span>
                  <input
                    type="checkbox"
                    checked={cardSettings.performance.iperf.autoRunOnLink}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                      updateCardSettings({
                        performance: {
                          ...cardSettings.performance,
                          iperf: {
                            ...cardSettings.performance.iperf,
                            autoRunOnLink: e.target.checked,
                          },
                        },
                      })
                    }
                    className={iconTokens.size.sm}
                  />
                </label>
              </div>
            </div>

            {/* Internet Speed (Speedtest) Subsection */}
            <div className={cn('border-t border-surface-border', spacing.padding.top.heading)}>
              <h4
                className={cn(
                  'body-small font-semibold text-text-primary',
                  spacing.margin.bottom.inline,
                  'uppercase tracking-wide',
                )}
              >
                {t('performance.internetSpeed')}
              </h4>
              <div className="stack">
                <div>
                  <label
                    htmlFor="speedtest-server-id"
                    className="caption text-text-muted font-medium"
                  >
                    {t('performance.serverId')}
                  </label>
                  <input
                    id="speedtest-server-id"
                    type="text"
                    value={testsSettings.speedtest.serverId}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setTestsSettings((prev) => ({
                        ...prev,
                        speedtest: {
                          ...prev.speedtest,
                          serverId: e.target.value,
                        },
                      }))
                    }
                    placeholder={t('performance.autoClosestServer')}
                    className={cn(
                      inputTokens.base,
                      inputTokens.state.default,
                      inputTokens.size.md,
                      'w-full',
                      spacing.margin.top.tight,
                      'body-small',
                    )}
                  />
                  <div className={cn(layout.flex.between, spacing.margin.top.tight)}>
                    <p className="caption text-text-muted">{t('performance.autoSelectDesc')}</p>
                    <button
                      type="button"
                      onClick={(): void =>
                        setTestsSettings((prev) => ({
                          ...prev,
                          speedtest: { ...prev.speedtest, serverId: '' },
                        }))
                      }
                      className="caption text-brand-primary hover:underline"
                    >
                      {t('performance.resetToAuto')}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </fieldset>

          <PerformanceIperfSection
            iperfSettings={iperfSettings}
            setIperfSettings={setIperfSettings}
            iperfSuggestions={iperfSuggestions}
            iperfSuggestionsStatus={iperfSuggestionsStatus}
            iperfSuggestionsError={iperfSuggestionsError}
            fetchIperfSuggestions={fetchIperfSuggestions}
          />
        </div>
      </CollapsibleSection>
    );
  },
);
