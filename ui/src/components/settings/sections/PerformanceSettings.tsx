import { Tooltip } from '../../../shared/Tooltip';
import { Info } from '../../ui/Icons';
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
import { useTranslation } from 'react-i18next';
import { useRole } from '../../../contexts/RoleContext';
import { CollapsibleSection } from '../../../shared/CollapsibleSection';
import { FormField } from '../../../shared/FormField';
import { Input } from '../../../shared/Input';
import { Toggle } from '../../../shared/Toggle';
import { cn, icon as iconTokens, layout, spacing } from '../../../styles/theme';
import type { IperfSuggestion } from '../../../types/generated/iperf-suggestion';
import type {
  CardSettings,
  IperfSettings,
  SaveStatus,
  TestsSettings,
} from '../../../types/settings';
import { Gauge } from '../../ui/Icons';
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
 */
export function PerformanceSettings({
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
              {/* Enable/Disable Toggles */}
              <div className="stack-sm">
                <Toggle
                  label={t('performance.enableSpeedtest')}
                  description={t('performance.speedtestDesc')}
                  checked={testsSettings.runSpeedtest}
                  onChange={(checked: boolean): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      runSpeedtest: checked,
                    }))
                  }
                  className="rounded-lg border border-surface-border bg-surface-base p-3"
                />
                <Toggle
                  label={t('performance.enableIperf')}
                  description={t('performance.iperfDesc')}
                  checked={testsSettings.runIperf}
                  onChange={(checked: boolean): void =>
                    setTestsSettings((prev) => ({
                      ...prev,
                      runIperf: checked,
                    }))
                  }
                  className="rounded-lg border border-surface-border bg-surface-base p-3"
                />
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
                  <Toggle
                    label={t('performance.speedtest')}
                    checked={cardSettings.performance.speedtest.autoRunOnLink}
                    onChange={(checked: boolean): void =>
                      updateCardSettings({
                        performance: {
                          ...cardSettings.performance,
                          speedtest: {
                            ...cardSettings.performance.speedtest,
                            autoRunOnLink: checked,
                          },
                        },
                      })
                    }
                    className="rounded-lg border border-surface-border bg-surface-base p-3"
                  />
                  <Toggle
                    label={t('performance.iperf')}
                    checked={cardSettings.performance.iperf.autoRunOnLink}
                    onChange={(checked: boolean): void =>
                      updateCardSettings({
                        performance: {
                          ...cardSettings.performance,
                          iperf: {
                            ...cardSettings.performance.iperf,
                            autoRunOnLink: checked,
                          },
                        },
                      })
                    }
                    className="rounded-lg border border-surface-border bg-surface-base p-3"
                  />
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
                    <FormField
                      id="speedtest-server-id"
                      label={t('performance.serverId')}
                      hint={t('performance.autoSelectDesc')}
                    >
                      {(control) => (
                        <Input
                          {...control}
                          type="text"
                          value={testsSettings.speedtest.serverId}
                          onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                            setTestsSettings((prev) => ({
                              ...prev,
                              speedtest: {
                                ...prev.speedtest,
                                serverId: e.target.value,
                              },
                            }))
                          }
                          placeholder={t('performance.autoClosestServer')}
                        />
                      )}
                    </FormField>
                    <button
                      type="button"
                      onClick={(): void =>
                        setTestsSettings((prev) => ({
                          ...prev,
                          speedtest: { ...prev.speedtest, serverId: '' },
                        }))
                      }
                      className={cn(
                        'caption text-brand-primary hover:underline',
                        spacing.margin.top.tight,
                      )}
                    >
                      {t('performance.resetToAuto')}
                    </button>
                  </div>
                </div>
              </div>
            </fieldset>
          )}
        </Tooltip>

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
}
