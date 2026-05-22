/**
 * SurveyView floor-plan panel.
 *
 * Left-side `lg:col-span-2` panel that owns the floor-plan canvas,
 * the heatmap controls header, the inline calibration mode UI, and
 * the empty-state upload/import widget when no floor plan is loaded.
 *
 * The heatmap metric *selector* lives in its own sibling component
 * (SurveyViewHeatmapSelector) and is rendered inside this panel.
 */

import { FileArchive, Ruler } from 'lucide-react';
import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import type { FloorPlan, HeatmapMetric, SamplePoint, Survey } from '../../hooks/useSurvey';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { Upload } from '../ui/icons';
import type { CalibrationPoint } from './FloorPlanCanvas';
import { FloorPlanCanvas } from './FloorPlanCanvas';
import { HeatmapLegend } from './HeatmapLegend';
import { HeatmapStats } from './HeatmapStats';
import { SurveyViewHeatmapSelector } from './SurveyViewHeatmapSelector';
import { calculateMetricRange } from './surveyViewHelpers';

interface WiFiStatusForFloorPlan {
  canScan: boolean;
}

interface SurveyViewFloorPlanPanelProps {
  survey: Survey;
  currentFloorPlan: FloorPlan | null | undefined;
  currentSamples: SamplePoint[];
  heatmapMetric: HeatmapMetric;
  setHeatmapMetric: (metric: HeatmapMetric) => void;
  calibrationMode: boolean;
  setCalibrationMode: (mode: boolean) => void;
  calibrationPoints: CalibrationPoint[];
  setCalibrationPoints: (points: CalibrationPoint[]) => void;
  calibrationDistance: string;
  setCalibrationDistance: (value: string) => void;
  useSae: boolean;
  handleSaveCalibration: () => Promise<void> | void;
  handleCancelCalibration: () => void;
  handlePointClick: (x: number, y: number) => Promise<void> | void;
  handleCalibrationClick: (x: number, y: number) => void;
  handleFloorPlanUpload: (file: File) => Promise<void>;
  sampling: boolean;
  wifiStatus: WiFiStatusForFloorPlan | null;
  uploadingFloorPlan: boolean;
  setShowImport: (show: boolean) => void;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Panel renders a multi-state floor-plan area; mirrors original inline structure
export function SurveyViewFloorPlanPanel({
  survey,
  currentFloorPlan,
  currentSamples,
  heatmapMetric,
  setHeatmapMetric,
  calibrationMode,
  setCalibrationMode,
  calibrationPoints,
  setCalibrationPoints,
  calibrationDistance,
  setCalibrationDistance,
  useSae,
  handleSaveCalibration,
  handleCancelCalibration,
  handlePointClick,
  handleCalibrationClick,
  handleFloorPlanUpload,
  sampling,
  wifiStatus,
  uploadingFloorPlan,
  setShowImport,
}: SurveyViewFloorPlanPanelProps): JSX.Element {
  const { t } = useTranslation('survey');

  return (
    <div className="lg:col-span-2">
      <div className={cn('bg-surface-raised', radius.md, 'border border-surface-border pad')}>
        <div className={cn(layout.flex.between, spacing.margin.bottom.content)}>
          <h2 className="heading-3">{t('floorPlan.title')}</h2>
          {heatmapMetric !== null && (
            <button
              type="button"
              onClick={() => setHeatmapMetric(null)}
              className={cn(
                button.size.sm,
                'body-small bg-brand-primary text-text-inverse',
                radius.md,
                'hover:bg-brand-primary/90',
              )}
            >
              {t('buttons.hideHeatmap')}
            </button>
          )}
        </div>

        <SurveyViewHeatmapSelector
          heatmapMetric={heatmapMetric}
          setHeatmapMetric={setHeatmapMetric}
          sampleCount={currentSamples.length}
          surveyType={survey.surveyType}
        />

        {currentFloorPlan ? (
          <div>
            {/* Calibration panel */}
            {calibrationMode ? (
              <div
                className={cn(
                  'bg-status-warning/10 border border-status-warning/20',
                  spacing.pad.sm,
                  radius.md,
                  spacing.margin.bottom.content,
                )}
              >
                <div
                  className={cn('font-medium text-status-warning', spacing.margin.bottom.inline)}
                >
                  📐 {t('calibration.title')}
                </div>
                <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
                  {t('calibration.instructions')}
                </p>
                <div className="stack-sm">
                  <div className={layout.inline.default}>
                    <span className="body-small text-text-muted w-20">
                      {t('calibration.pointA')}:
                    </span>
                    {calibrationPoints[0] ? (
                      <span className="body-small font-medium">
                        ({calibrationPoints[0].x}, {calibrationPoints[0].y})
                      </span>
                    ) : (
                      <span className="body-small text-text-muted italic">
                        {t('calibration.clickFloorPlan')}
                      </span>
                    )}
                  </div>
                  <div className={layout.inline.default}>
                    <span className="body-small text-text-muted w-20">
                      {t('calibration.pointB')}:
                    </span>
                    {calibrationPoints[1] ? (
                      <span className="body-small font-medium">
                        ({calibrationPoints[1].x}, {calibrationPoints[1].y})
                      </span>
                    ) : (
                      <span className="body-small text-text-muted italic">
                        {t('calibration.clickFloorPlan')}
                      </span>
                    )}
                  </div>
                  {calibrationPoints.length === 2 && (
                    <div className={layout.inline.default}>
                      <span className="body-small text-text-muted w-20">
                        {t('calibration.pixelDistance')}:
                      </span>
                      <span className="body-small font-medium">
                        {Math.sqrt(
                          (calibrationPoints[1].x - calibrationPoints[0].x) ** 2 +
                            (calibrationPoints[1].y - calibrationPoints[0].y) ** 2,
                        ).toFixed(0)}{' '}
                        px
                      </span>
                    </div>
                  )}
                  <div className={cn(layout.inline.default, spacing.margin.top.inline)}>
                    <label
                      htmlFor="calibration-distance"
                      className="body-small text-text-muted w-20"
                    >
                      {t('calibration.distance')}:
                    </label>
                    <input
                      id="calibration-distance"
                      type="number"
                      step="0.1"
                      min="0"
                      value={calibrationDistance}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void => setCalibrationDistance(e.target.value)}
                      placeholder={
                        useSae ? t('calibration.enterFeet') : t('calibration.enterMeters')
                      }
                      className={cn(
                        'flex-1',
                        button.size.sm,
                        'border border-surface-border',
                        radius.md,
                        'bg-surface-base text-text-primary',
                      )}
                    />
                    <span className="body-small text-text-muted">
                      {useSae ? t('calibration.feet') : t('calibration.meters')}
                    </span>
                  </div>
                  <div className={cn(layout.inline.default, spacing.margin.top.inline)}>
                    <button
                      type="button"
                      onClick={handleSaveCalibration}
                      disabled={calibrationPoints.length !== 2 || !calibrationDistance}
                      className={cn(
                        button.size.sm,
                        'bg-brand-primary text-text-inverse',
                        radius.md,
                        'hover:bg-brand-primary/90 disabled:opacity-50 disabled:cursor-not-allowed',
                      )}
                    >
                      {t('buttons.saveScale')}
                    </button>
                    <button
                      type="button"
                      onClick={handleCancelCalibration}
                      className={cn(
                        button.size.sm,
                        'border border-surface-border',
                        radius.md,
                        'hover:bg-surface-hover',
                      )}
                    >
                      {t('buttons.cancel')}
                    </button>
                    <button
                      type="button"
                      onClick={() => setCalibrationPoints([])}
                      className={cn(
                        button.size.sm,
                        'border border-surface-border',
                        radius.md,
                        'hover:bg-surface-hover',
                      )}
                    >
                      {t('buttons.resetPoints')}
                    </button>
                  </div>
                </div>
              </div>
            ) : null}

            {/* Calibrate button and current scale info */}
            {!calibrationMode && currentFloorPlan && (
              <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
                <div className="body-small text-text-muted">
                  {t('floorPlan.scale')}: {currentFloorPlan.scaleM.toFixed(3)} m/px
                  {survey.status === 'in_progress' && ` • ${t('floorPlan.clickToMeasure')}`}
                </div>
                <button
                  type="button"
                  onClick={() => setCalibrationMode(true)}
                  className={cn(
                    button.size.sm,
                    'body-small border border-surface-border',
                    radius.md,
                    'hover:bg-surface-hover',
                    layout.inline.tight,
                  )}
                >
                  <Ruler className={iconTokens.size.sm} />
                  {t('buttons.calibrateScale')}
                </button>
              </div>
            )}

            <FloorPlanCanvas
              floorPlan={
                currentFloorPlan ?? {
                  id: '',
                  name: '',
                  imageUrl: '',
                  width: 0,
                  height: 0,
                  scale: 1,
                }
              }
              samples={currentSamples}
              // #727: render imported AP placements so they're not silently dropped.
              apLocations={survey.apLocations}
              onPointClick={handlePointClick}
              interactive={
                survey.status === 'in_progress' &&
                !sampling &&
                !calibrationMode &&
                wifiStatus?.canScan === true
              }
              heatmapMetric={heatmapMetric}
              calibrationMode={calibrationMode}
              calibrationPoints={calibrationPoints}
              onCalibrationClick={handleCalibrationClick}
            />

            {/* Heatmap Legend and Stats - show when heatmap is active */}
            {heatmapMetric !== null && currentSamples.length > 0 && (
              <div className={spacing.margin.top.content}>
                <HeatmapLegend
                  metric={heatmapMetric}
                  minValue={calculateMetricRange(currentSamples, heatmapMetric).min}
                  maxValue={calculateMetricRange(currentSamples, heatmapMetric).max}
                />
                <HeatmapStats samples={currentSamples} metric={heatmapMetric} />
              </div>
            )}
          </div>
        ) : (
          <div
            className={cn(
              'border-2 border-dashed border-surface-border',
              radius.md,
              'pad-lg text-center',
            )}
          >
            <Upload
              className={cn(
                iconTokens.size.xl,
                'mx-auto text-text-muted',
                spacing.margin.bottom.content,
              )}
            />
            <p className={cn('text-text-muted', spacing.margin.bottom.content)}>
              {t('floorPlan.uploadPrompt')}
            </p>
            <label
              className={cn(
                'inline-block',
                button.size.md,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'cursor-pointer hover:bg-brand-primary/90',
              )}
            >
              {uploadingFloorPlan ? t('floorPlan.uploading') : t('floorPlan.chooseFile')}
              <input
                type="file"
                accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
                className="hidden"
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void => {
                  const file = (e.target as HTMLInputElement).files?.[0];
                  if (file) {
                    handleFloorPlanUpload(file).catch(() => undefined);
                  }
                  // Reset input so same file can be selected again if needed
                  (e.target as HTMLInputElement).value = '';
                }}
                disabled={uploadingFloorPlan}
              />
            </label>
            <p className={cn('caption text-text-muted', spacing.margin.top.inline)}>
              {t('floorPlan.supportedFormats')}
            </p>
            <div
              className={cn(
                spacing.margin.top.content,
                'border-t border-surface-border',
                spacing.padding.top.section,
              )}
            >
              <p className={cn('caption text-text-muted', spacing.margin.bottom.inline)}>
                {t('import.description')}
              </p>
              <button
                type="button"
                onClick={() => setShowImport(true)}
                className={cn(
                  button.size.sm,
                  'border border-surface-border',
                  radius.md,
                  'hover:bg-surface-hover',
                  layout.inline.default,
                )}
              >
                <FileArchive className={iconTokens.size.sm} />
                {t('import.button')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
