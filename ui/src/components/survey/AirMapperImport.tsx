/**
 * AirMapperImport Component
 *
 * Purpose: Import NetAlly AirMapper .amp survey files for analysis and comparison.
 * Provides drag-and-drop interface with preview of what will be imported.
 *
 * Key Features:
 * - Drag and drop or file picker for .amp files
 * - Preview imported data before confirming
 * - Import options: calibration only, floor plan, or full survey
 * - Shows summary statistics from imported file
 *
 * Usage:
 * ```typescript
 * <AirMapperImport
 *   onImport={(data, options) => handleImport(data, options)}
 *   onCancel={() => setShowImport(false)}
 * />
 * ```
 */

import { AlertTriangle, Check, FileArchive, MapPin, Radio, Upload, Users, X } from 'lucide-react';
import type React from 'react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { LogComponents, logger } from '../../lib/logger';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import {
  type AirMapperData,
  type AirMapperParseResult,
  getAirMapperSummary,
  importAirMapperViaBackend,
  parseAirMapperFile,
} from '../../utils/airmapper';

/** Import options */
export interface ImportOptions {
  importFloorPlan: boolean;
  importCalibration: boolean;
  importLocations: boolean;
  importViews: boolean;
}

interface AirMapperImportProps {
  onImport: (data: AirMapperData, options: ImportOptions) => void;
  onCancel: () => void;
}

/**
 * AirMapperImport provides a UI for importing .amp survey files
 */
export function AirMapperImport({ onImport, onCancel }: AirMapperImportProps): React.ReactElement {
  const { t } = useTranslation('survey');

  // State
  const [isDragging, setIsDragging] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [parseResult, setParseResult] = useState<AirMapperParseResult | null>(null);
  const [importOptions, setImportOptions] = useState<ImportOptions>({
    importFloorPlan: true,
    importCalibration: true,
    importLocations: true,
    importViews: false,
  });

  // Handle file selection - try backend API first, then fallback to client-side
  const handleFile = useCallback(
    async (file: File) => {
      if (!file.name.toLowerCase().endsWith('.amp')) {
        setParseResult({
          success: false,
          error: t('import.invalidFormat'),
          warnings: [],
        });
        return;
      }

      setIsLoading(true);
      try {
        // Try backend API first (faster for large files)
        // Pass empty headers - cookie auth is handled via credentials: 'include' in the utility
        const backendResult = await importAirMapperViaBackend(file, {});

        if (backendResult.success) {
          setParseResult(backendResult);
        } else {
          // Fallback to client-side parsing if backend fails
          logger.warn(LogComponents.Survey, 'Backend parsing failed, falling back to client-side', {
            error: backendResult.error,
          });
          // biome-ignore lint/nursery/useAwaitThenable: File.arrayBuffer() returns a Promise
          const buffer = await file.arrayBuffer();
          const result = await parseAirMapperFile(buffer);
          setParseResult(result);
        }
      } catch {
        // Final fallback - try client-side parsing directly
        try {
          // biome-ignore lint/nursery/useAwaitThenable: File.arrayBuffer() returns a Promise
          const buffer = await file.arrayBuffer();
          const result = await parseAirMapperFile(buffer);
          setParseResult(result);
        } catch (clientErr) {
          setParseResult({
            success: false,
            error: clientErr instanceof Error ? clientErr.message : t('import.parseFailed'),
            warnings: [],
          });
        }
      } finally {
        setIsLoading(false);
      }
    },
    [t],
  );

  // Drag handlers
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);

      const { files } = e.dataTransfer;
      if (files.length > 0) {
        handleFile(files[0]).catch(() => undefined);
      }
    },
    [handleFile],
  );

  // File input handler
  const handleFileInput = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const { files } = e.target;
      if (files && files.length > 0) {
        handleFile(files[0]).catch(() => undefined);
      }
    },
    [handleFile],
  );

  // Handle import confirmation
  const handleConfirmImport = useCallback(() => {
    if (parseResult?.success && parseResult.data) {
      onImport(parseResult.data, importOptions);
    }
  }, [parseResult, importOptions, onImport]);

  // Reset to try another file
  const handleReset = useCallback(() => {
    setParseResult(null);
  }, []);

  // Render drop zone
  const renderDropZone = (): React.ReactElement => (
    // biome-ignore lint/a11y/useSemanticElements: Drop zone with drag/drop events - no semantic HTML alternative
    <div
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      class={cn(
        'border-2 border-dashed',
        isDragging ? 'border-brand-primary bg-brand-primary/10' : 'border-surface-border',
        radius.lg,
        spacing.pad.lg,
        'text-center transition-colors',
      )}
      role="region"
      aria-label={t('import.dropZone', 'File drop zone')}
    >
      {isLoading ? (
        <div class={cn(layout.stack.default, 'items-center')}>
          <div class="animate-spin">
            <FileArchive class={cn(iconTokens.size.lg, 'text-text-muted')} />
          </div>
          <p class="body-small text-text-muted">{t('import.parsing')}</p>
        </div>
      ) : (
        <div class={cn(layout.stack.default, 'items-center')}>
          <Upload class={cn(iconTokens.size.lg, 'text-text-muted')} />
          <div>
            <p class="body-small font-medium">{t('import.dropPrompt')}</p>
            <p class="caption text-text-muted">{t('import.orClick')}</p>
          </div>
          <label
            class={cn(
              button.size.sm,
              'bg-brand-primary text-text-inverse',
              radius.md,
              'hover:bg-brand-primary/90 cursor-pointer',
            )}
          >
            {t('import.selectFile')}
            <input type="file" accept=".amp" onChange={handleFileInput} class="hidden" />
          </label>
          <p class="caption text-text-muted">{t('import.supportedFormat')}</p>
        </div>
      )}
    </div>
  );

  // Render error state
  const renderError = (): React.ReactElement => (
    <div class={layout.stack.default}>
      <div
        class={cn(
          'bg-status-error/10 border border-status-error/20',
          radius.md,
          spacing.pad.default,
        )}
      >
        <div class={cn(layout.inline.default, 'text-status-error')}>
          <AlertTriangle class={iconTokens.size.sm} />
          <span class="body-small font-medium">{t('import.error')}</span>
        </div>
        <p class={cn('body-small text-text-primary', spacing.margin.top.tight)}>
          {parseResult?.error}
        </p>
      </div>
      <div class={cn(layout.inline.default, 'justify-end')}>
        <button
          type="button"
          onClick={handleReset}
          class={cn(
            button.size.sm,
            'border border-surface-border',
            radius.md,
            'hover:bg-surface-hover',
          )}
        >
          {t('import.tryAnother')}
        </button>
        <button
          type="button"
          onClick={onCancel}
          class={cn(
            button.size.sm,
            'border border-surface-border',
            radius.md,
            'hover:bg-surface-hover',
          )}
        >
          {t('buttons.cancel')}
        </button>
      </div>
    </div>
  );

  // Render preview
  const renderPreview = (): React.ReactElement | null => {
    if (!parseResult?.data) {
      return null;
    }

    const summary = getAirMapperSummary(parseResult.data);

    return (
      <div class={layout.stack.default}>
        {/* Warnings */}
        {parseResult.warnings.length > 0 ? (
          <div
            class={cn(
              'bg-status-warning/10 border border-status-warning/20',
              radius.md,
              spacing.pad.sm,
            )}
          >
            <div class={cn(layout.inline.default, 'text-status-warning')}>
              <AlertTriangle class={iconTokens.size.sm} />
              <span class="caption font-medium">{t('import.warnings')}</span>
            </div>
            <ul
              class={cn(
                'caption text-text-muted',
                spacing.margin.top.tight,
                'list-disc list-inside',
              )}
            >
              {parseResult.warnings.map((w) => (
                <li key={w}>{w}</li>
              ))}
            </ul>
          </div>
        ) : null}

        {/* Preview info */}
        <div class={cn('bg-surface-base', radius.md, spacing.pad.default)}>
          <h4 class={cn('body-small font-medium', spacing.margin.bottom.content)}>
            {summary.surveyName}
          </h4>

          <div class="grid grid-cols-2 gap-2 caption">
            <div class="text-text-muted">{t('import.device')}:</div>
            <div>{summary.deviceInfo}</div>

            <div class="text-text-muted">{t('import.surveyPoints')}:</div>
            <div>{summary.pointCount}</div>

            <div class="text-text-muted">{t('import.facilitySize')}:</div>
            <div>{summary.facilitySize}</div>

            <div class="text-text-muted">{t('import.propagation')}:</div>
            <div>{summary.propagation}</div>
          </div>

          {/* Location counts */}
          <div class={cn(layout.inline.default, spacing.margin.top.content, 'flex-wrap')}>
            <div class={cn(layout.inline.default, 'caption')}>
              <MapPin class="w-3 h-3 text-green-500" />
              <span>{summary.apCount} APs</span>
            </div>
            <div class={cn(layout.inline.default, 'caption')}>
              <Users class="w-3 h-3 text-blue-500" />
              <span>
                {summary.clientCount} {t('import.clients')}
              </span>
            </div>
            {summary.hasBothModes ? (
              <div class={cn(layout.inline.default, 'caption')}>
                <Radio class="w-3 h-3 text-purple-500" />
                <span>{t('import.passiveAndActive')}</span>
              </div>
            ) : null}
          </div>

          {/* Floor plan preview */}
          {parseResult.data.floorPlanImage ? (
            <div class={spacing.margin.top.content}>
              <img
                src={parseResult.data.floorPlanImage}
                alt="Floor plan preview"
                class={cn('max-h-40', radius.md, 'border border-surface-border')}
              />
            </div>
          ) : null}
        </div>

        {/* Import options */}
        <div class={cn('border border-surface-border', radius.md, spacing.pad.sm)}>
          <h4 class={cn('caption font-medium', spacing.margin.bottom.inline)}>
            {t('import.options')}
          </h4>
          <div class={layout.stack.default}>
            <label class={cn(layout.inline.default, 'cursor-pointer')}>
              <input
                type="checkbox"
                checked={importOptions.importFloorPlan}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  setImportOptions((prev) => ({
                    ...prev,
                    importFloorPlan: e.target.checked,
                  }))
                }
                class="w-4 h-4 accent-brand-primary"
                disabled={!parseResult.data.floorPlanImage}
              />
              <span class="body-small">{t('import.optionFloorPlan')}</span>
            </label>
            <label class={cn(layout.inline.default, 'cursor-pointer')}>
              <input
                type="checkbox"
                checked={importOptions.importCalibration}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  setImportOptions((prev) => ({
                    ...prev,
                    importCalibration: e.target.checked,
                  }))
                }
                class="w-4 h-4 accent-brand-primary"
              />
              <span class="body-small">{t('import.optionCalibration')}</span>
            </label>
            <label class={cn(layout.inline.default, 'cursor-pointer')}>
              <input
                type="checkbox"
                checked={importOptions.importLocations}
                onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                  setImportOptions((prev) => ({
                    ...prev,
                    importLocations: e.target.checked,
                  }))
                }
                class="w-4 h-4 accent-brand-primary"
              />
              <span class="body-small">{t('import.optionLocations')}</span>
            </label>
          </div>
        </div>

        {/* Actions */}
        <div class={cn(layout.inline.default, 'justify-end')}>
          <button
            type="button"
            onClick={handleReset}
            class={cn(
              button.size.sm,
              'border border-surface-border',
              radius.md,
              'hover:bg-surface-hover',
              layout.inline.default,
            )}
          >
            <X class={iconTokens.size.sm} />
            {t('import.tryAnother')}
          </button>
          <button
            type="button"
            onClick={onCancel}
            class={cn(
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
            onClick={handleConfirmImport}
            class={cn(
              button.size.sm,
              'bg-brand-primary text-text-inverse',
              radius.md,
              'hover:bg-brand-primary/90',
              layout.inline.default,
            )}
          >
            <Check class={iconTokens.size.sm} />
            {t('import.confirm')}
          </button>
        </div>
      </div>
    );
  };

  return (
    <div
      class={cn(
        'bg-surface-raised',
        radius.md,
        'border border-surface-border',
        spacing.pad.default,
      )}
    >
      <div class={cn(layout.inline.default, spacing.margin.bottom.content)}>
        <FileArchive class={iconTokens.size.sm} />
        <h3 class="heading-3">{t('import.title')}</h3>
      </div>

      <p class={cn('body-small text-text-muted', spacing.margin.bottom.content)}>
        {t('import.description')}
      </p>

      {parseResult ? null : renderDropZone()}
      {parseResult && !parseResult.success ? renderError() : null}
      {parseResult?.success ? renderPreview() : null}
    </div>
  );
}
