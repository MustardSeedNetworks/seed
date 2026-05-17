/**
 * SurveyView side panel.
 *
 * Right-hand column shown alongside the floor-plan area: setup
 * checklist, survey-type / iperf settings (pre-start), scale
 * calibration panel, survey configuration panel, and the running list
 * of samples. Pulled out so SurveyView.tsx can shrink under the
 * file-size budget.
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import type {
  FloorPlan,
  SamplePoint,
  Survey,
  SurveyConfig,
  SurveyType,
} from '../../hooks/useSurvey';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { CheckCircle, Clock } from '../ui/icons';
import { ScaleCalibrationPanel } from './ScaleCalibrationPanel';
import { SurveyConfigPanel } from './SurveyConfigPanel';
import { renderSampleData } from './surveyViewHelpers';

interface WiFiStatusForPanel {
  availableAdapters?: string[];
  currentInterface?: string;
}

interface SetupStep {
  key: string;
  label: string;
  done: boolean | string | undefined | null;
}

interface SurveyViewSidePanelProps {
  survey: Survey;
  currentSamples: SamplePoint[];
  currentFloorPlan: FloorPlan | null | undefined;
  setupSteps: SetupStep[];
  completedSetupSteps: number;
  editSurveyType: SurveyType;
  setEditSurveyType: (type: SurveyType) => void;
  editIperfServer: string;
  setEditIperfServer: (value: string) => void;
  editTestDuration: number;
  setEditTestDuration: (value: number) => void;
  savingSettings: boolean;
  handleSaveSettings: () => Promise<void> | void;
  handleFloorPlanUpdate: (updates: Partial<FloorPlan>) => Promise<void>;
  setCalibrationMode: (mode: boolean) => void;
  calibrationMode: boolean;
  wifiStatus: WiFiStatusForPanel | null;
  handleConfigUpdate: (configUpdates: Partial<SurveyConfig>) => Promise<void>;
  handleSurveyTypeChange: (newType: SurveyType) => void;
  handleIperfSettingsChange: (server: string, duration: number) => void;
}

export function SurveyViewSidePanel({
  survey,
  currentSamples,
  currentFloorPlan,
  setupSteps,
  completedSetupSteps,
  editSurveyType,
  setEditSurveyType,
  editIperfServer,
  setEditIperfServer,
  editTestDuration,
  setEditTestDuration,
  savingSettings,
  handleSaveSettings,
  handleFloorPlanUpdate,
  setCalibrationMode,
  calibrationMode,
  wifiStatus,
  handleConfigUpdate,
  handleSurveyTypeChange,
  handleIperfSettingsChange,
}: SurveyViewSidePanelProps): JSX.Element {
  const { t } = useTranslation('survey');

  return (
    <div class={cn('lg:col-span-1', spacing.stack.default)}>
      {/* Setup checklist to guide users before starting a survey */}
      {survey.status === 'created' && (
        <div class={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
          <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <h2 class="heading-3">{t('setup.checklist')}</h2>
            <span class="caption text-text-muted">
              {completedSetupSteps}/{setupSteps.length}
            </span>
          </div>
          <div class="stack-sm">
            {setupSteps.map((step) => (
              <div
                key={step.key}
                class={cn(
                  'flex items-center justify-between',
                  spacing.pad.xs,
                  radius.sm,
                  step.done ? 'bg-surface-hover' : 'bg-transparent',
                )}
              >
                <div class={layout.inline.default}>
                  {step.done ? (
                    <CheckCircle class={cn(iconTokens.size.sm, 'text-status-success')} />
                  ) : (
                    <Clock class={cn(iconTokens.size.sm, 'text-text-muted')} />
                  )}
                  <span class="body-small">{step.label}</span>
                </div>
                {step.done ? <span class="caption text-status-success">✓</span> : null}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Survey Settings Panel - only show when survey hasn't started */}
      {survey.status === 'created' && (
        <div class={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
          <h2 class={cn('heading-3', spacing.margin.bottom.content)}>{t('settings.title')}</h2>
          <div class="stack">
            {/* Survey Type */}
            <div>
              <label
                for="survey-type-select"
                class={cn('body-small text-text-muted block', spacing.margin.bottom.tight)}
              >
                {t('settings.surveyType')}
              </label>
              <select
                id="survey-type-select"
                value={editSurveyType}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setEditSurveyType(e.target.value as SurveyType)
                }
                class={cn(
                  'w-full',
                  button.size.md,
                  'border border-surface-border',
                  radius.md,
                  'bg-surface-base text-text-primary',
                )}
              >
                <option value="passive">{t('settings.types.passive')}</option>
                <option value="active">{t('settings.types.active')}</option>
                <option value="throughput">{t('settings.types.throughput')}</option>
              </select>
            </div>

            {/* iperf Server - only show for throughput surveys */}
            {editSurveyType === 'throughput' && (
              <>
                <div>
                  <label
                    for="survey-iperf-server"
                    class={cn('body-small text-text-muted block', spacing.margin.bottom.tight)}
                  >
                    {t('settings.iperfServer')}
                  </label>
                  <input
                    id="survey-iperf-server"
                    type="text"
                    value={editIperfServer}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setEditIperfServer(e.target.value)
                    }
                    placeholder="hostname:5201"
                    class={cn(
                      'w-full',
                      button.size.md,
                      'border border-surface-border',
                      radius.md,
                      'bg-surface-base text-text-primary',
                    )}
                  />
                  <p class={cn('caption text-text-muted', spacing.margin.top.tight)}>
                    {t('settings.iperfServerHint')}
                  </p>
                </div>

                <div>
                  <label
                    for="survey-test-duration"
                    class={cn('body-small text-text-muted block', spacing.margin.bottom.tight)}
                  >
                    {t('settings.testDuration')}
                  </label>
                  <input
                    id="survey-test-duration"
                    type="number"
                    min="1"
                    max="60"
                    value={editTestDuration}
                    onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                      setEditTestDuration(Number.parseInt(e.target.value, 10) || 3)
                    }
                    class={cn(
                      'w-full',
                      button.size.md,
                      'border border-surface-border',
                      radius.md,
                      'bg-surface-base text-text-primary',
                    )}
                  />
                </div>
              </>
            )}

            {/* Save button */}
            <button
              type="button"
              onClick={handleSaveSettings}
              disabled={savingSettings}
              class={cn(
                'w-full',
                button.size.md,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'hover:bg-brand-primary/90 disabled:opacity-50',
              )}
            >
              {savingSettings ? t('buttons.saving') : t('buttons.saveSettings')}
            </button>

            {/* Survey type descriptions */}
            <div
              class={cn(
                'caption text-text-muted border-t border-surface-border',
                spacing.padding.top.section,
                spacing.margin.top.inline,
              )}
            >
              <p class={cn('font-medium', spacing.margin.bottom.inline)}>
                {t('settings.typesDescription')}
              </p>
              <ul class={cn('list-disc list-inside', spacing.stack.xs)}>
                <li>
                  <strong>Passive:</strong> {t('settings.passiveDesc')}
                </li>
                <li>
                  <strong>Active:</strong> {t('settings.activeDesc')}
                </li>
                <li>
                  <strong>Throughput:</strong> {t('settings.throughputDesc')}
                </li>
              </ul>
            </div>
          </div>
        </div>
      )}

      {/* Scale Calibration Panel - show when floor plan exists */}
      {currentFloorPlan ? (
        <ScaleCalibrationPanel
          floorPlan={currentFloorPlan}
          onUpdate={handleFloorPlanUpdate}
          onStartCalibration={(): void => setCalibrationMode(true)}
          isCalibrating={calibrationMode}
        />
      ) : null}

      {/* Survey Configuration Panel - show when floor plan exists */}
      {currentFloorPlan && wifiStatus ? (
        <SurveyConfigPanel
          config={survey.config}
          surveyType={editSurveyType}
          availableAdapters={wifiStatus.availableAdapters || []}
          currentInterface={wifiStatus.currentInterface || survey.interface}
          iperfServer={editIperfServer}
          testDuration={editTestDuration}
          onUpdate={handleConfigUpdate}
          onSurveyTypeChange={handleSurveyTypeChange}
          onIperfSettingsChange={handleIperfSettingsChange}
        />
      ) : null}

      {/* Sample list */}
      <div class={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
        <h2 class={cn('heading-3', spacing.margin.bottom.content)}>
          {t('samples.title')} ({currentSamples.length})
        </h2>
        <div class="stack-sm max-h-[70vh] overflow-y-auto">
          {currentSamples.length === 0 ? (
            <p class={cn('body-small text-center', spacing.pad.lg)}>
              {t('samples.noSamples')}{' '}
              {survey.status === 'in_progress'
                ? t('samples.clickToStart')
                : t('samples.startToBegin')}
            </p>
          ) : (
            currentSamples.map((sample, idx) => (
              <div
                key={sample.timestamp}
                class={cn('border border-surface-border', radius.md, 'pad-sm body-small')}
              >
                <div class={cn('flex items-center justify-between', spacing.margin.bottom.inline)}>
                  <span class="font-semibold">#{idx + 1}</span>
                  <span class="caption">{new Date(sample.timestamp).toLocaleTimeString()}</span>
                </div>
                <div class="caption stack-xs">
                  {renderSampleData(sample.sampleData, survey.surveyType)}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
