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
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import type {
  FloorPlan,
  SamplePoint,
  Survey,
  SurveyConfig,
  SurveyType,
} from '../../hooks/useSurvey';
import {
  button,
  cn,
  icon as iconTokens,
  layout,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';
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
    <div className={cn('lg:col-span-1', spacing.stack.default)}>
      {/* Setup checklist to guide users before starting a survey */}
      {survey.status === 'created' && (
        <div className={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <h2 className="heading-3">{t('setup.checklist')}</h2>
            <span className="caption text-text-muted">
              {completedSetupSteps}/{setupSteps.length}
            </span>
          </div>
          <div className="stack-sm">
            {setupSteps.map((step) => (
              <div
                key={step.key}
                className={cn(
                  'flex items-center justify-between',
                  spacing.pad.xs,
                  radius.sm,
                  step.done ? 'bg-surface-hover' : 'bg-transparent',
                )}
              >
                <div className={layout.inline.default}>
                  {step.done ? (
                    <CheckCircle className={cn(iconTokens.size.sm, statusColor.text.success)} />
                  ) : (
                    <Clock className={cn(iconTokens.size.sm, 'text-text-muted')} />
                  )}
                  <span className="body-small">{step.label}</span>
                </div>
                {step.done ? <span className="caption text-status-success">✓</span> : null}
              </div>
            ))}
          </div>
        </div>
      )}
      {/* Survey Settings Panel - only show when survey hasn't started */}
      {survey.status === 'created' && (
        <div className={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
          <h2 className={cn('heading-3', spacing.margin.bottom.content)}>{t('settings.title')}</h2>
          <div className="stack">
            {/* Survey Type */}
            <div>
              <label
                htmlFor="survey-type-select"
                className={cn('body-small text-text-muted block', spacing.margin.bottom.tight)}
              >
                {t('settings.surveyType')}
              </label>
              <select
                id="survey-type-select"
                value={editSurveyType}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setEditSurveyType(e.target.value as SurveyType)
                }
                className={cn(
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
                    htmlFor="survey-iperf-server"
                    className={cn('body-small text-text-muted block', spacing.margin.bottom.tight)}
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
                    className={cn(
                      'w-full',
                      button.size.md,
                      'border border-surface-border',
                      radius.md,
                      'bg-surface-base text-text-primary',
                    )}
                  />
                  <p className={cn('caption text-text-muted', spacing.margin.top.tight)}>
                    {t('settings.iperfServerHint')}
                  </p>
                </div>

                <div>
                  <label
                    htmlFor="survey-test-duration"
                    className={cn('body-small text-text-muted block', spacing.margin.bottom.tight)}
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
                    className={cn(
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
              className={cn(
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
              className={cn(
                'caption text-text-muted border-t border-surface-border',
                spacing.padding.top.section,
                spacing.margin.top.inline,
              )}
            >
              <p className={cn('font-medium', spacing.margin.bottom.inline)}>
                {t('settings.typesDescription')}
              </p>
              <ul className={cn('list-disc list-inside', spacing.stack.xs)}>
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
      <div className={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
        <h2 className={cn('heading-3', spacing.margin.bottom.content)}>
          {t('samples.title')} ({currentSamples.length})
        </h2>
        <div className="stack-sm max-h-[70vh] overflow-y-auto">
          {currentSamples.length === 0 ? (
            <p className={cn('body-small text-center', spacing.pad.lg)}>
              {t('samples.noSamples')}{' '}
              {survey.status === 'in_progress'
                ? t('samples.clickToStart')
                : t('samples.startToBegin')}
            </p>
          ) : (
            currentSamples.map((sample, idx) => (
              <div
                key={sample.timestamp}
                className={cn('border border-surface-border', radius.md, 'pad-sm body-small')}
              >
                <div
                  className={cn('flex items-center justify-between', spacing.margin.bottom.inline)}
                >
                  <span className="font-semibold">#{idx + 1}</span>
                  <span className="caption">{new Date(sample.timestamp).toLocaleTimeString()}</span>
                </div>
                <div className="caption stack-xs">
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
