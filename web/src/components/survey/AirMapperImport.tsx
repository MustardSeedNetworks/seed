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

import { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
  parseAirMapperFile,
  importAirMapperViaBackend,
  getAirMapperSummary,
  type AirMapperData,
  type AirMapperParseResult,
} from "../../utils/airmapper";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { logger, LogComponents } from "../../lib/logger";
import {
  Upload,
  FileArchive,
  AlertTriangle,
  Check,
  X,
  MapPin,
  Radio,
  Users,
} from "lucide-react";
import {
  cn,
  radius,
  spacing,
  layout,
  button,
  icon as iconTokens,
} from "../../styles/theme";

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
export function AirMapperImport({ onImport, onCancel }: AirMapperImportProps) {
  const { t } = useTranslation("survey");

  // State
  const [isDragging, setIsDragging] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [parseResult, setParseResult] = useState<AirMapperParseResult | null>(
    null
  );
  const [importOptions, setImportOptions] = useState<ImportOptions>({
    importFloorPlan: true,
    importCalibration: true,
    importLocations: true,
    importViews: false,
  });

  // Handle file selection - try backend API first, then fallback to client-side
  const handleFile = useCallback(
    async (file: File) => {
      if (!file.name.toLowerCase().endsWith(".amp")) {
        setParseResult({
          success: false,
          error: t("import.invalidFormat"),
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
          logger.warn(
            LogComponents.SURVEY,
            "Backend parsing failed, falling back to client-side",
            {
              error: backendResult.error,
            }
          );
          const buffer = await file.arrayBuffer();
          const result = await parseAirMapperFile(buffer);
          setParseResult(result);
        }
      } catch {
        // Final fallback - try client-side parsing directly
        try {
          const buffer = await file.arrayBuffer();
          const result = await parseAirMapperFile(buffer);
          setParseResult(result);
        } catch (clientErr) {
          setParseResult({
            success: false,
            error:
              clientErr instanceof Error
                ? clientErr.message
                : t("import.parseFailed"),
            warnings: [],
          });
        }
      } finally {
        setIsLoading(false);
      }
    },
    [t]
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

      const files = e.dataTransfer.files;
      if (files.length > 0) {
        handleFile(files[0]);
      }
    },
    [handleFile]
  );

  // File input handler
  const handleFileInput = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files;
      if (files && files.length > 0) {
        handleFile(files[0]);
      }
    },
    [handleFile]
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
  const renderDropZone = () => (
    <div
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      className={cn(
        "border-2 border-dashed",
        isDragging
          ? "border-brand-primary bg-brand-primary/10"
          : "border-surface-border",
        radius.lg,
        spacing.pad.lg,
        "text-center transition-colors"
      )}
    >
      {isLoading ? (
        <div className={cn(layout.stack.default, "items-center")}>
          <div className="animate-spin">
            <FileArchive
              className={cn(iconTokens.size.lg, "text-text-muted")}
            />
          </div>
          <p className="body-small text-text-muted">{t("import.parsing")}</p>
        </div>
      ) : (
        <div className={cn(layout.stack.default, "items-center")}>
          <Upload className={cn(iconTokens.size.lg, "text-text-muted")} />
          <div>
            <p className="body-small font-medium">{t("import.dropPrompt")}</p>
            <p className="caption text-text-muted">{t("import.orClick")}</p>
          </div>
          <label
            className={cn(
              button.size.sm,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "hover:bg-brand-primary/90 cursor-pointer"
            )}
          >
            {t("import.selectFile")}
            <input
              type="file"
              accept=".amp"
              onChange={handleFileInput}
              className="hidden"
            />
          </label>
          <p className="caption text-text-muted">
            {t("import.supportedFormat")}
          </p>
        </div>
      )}
    </div>
  );

  // Render error state
  const renderError = () => (
    <div className={layout.stack.default}>
      <div
        className={cn(
          "bg-status-error/10 border border-status-error/20",
          radius.md,
          spacing.pad.default
        )}
      >
        <div className={cn(layout.inline.default, "text-status-error")}>
          <AlertTriangle className={iconTokens.size.sm} />
          <span className="body-small font-medium">{t("import.error")}</span>
        </div>
        <p
          className={cn(
            "body-small text-text-primary",
            spacing.margin.top.tight
          )}
        >
          {parseResult?.error}
        </p>
      </div>
      <div className={cn(layout.inline.default, "justify-end")}>
        <button
          onClick={handleReset}
          className={cn(
            button.size.sm,
            "border border-surface-border",
            radius.md,
            "hover:bg-surface-hover"
          )}
        >
          {t("import.tryAnother")}
        </button>
        <button
          onClick={onCancel}
          className={cn(
            button.size.sm,
            "border border-surface-border",
            radius.md,
            "hover:bg-surface-hover"
          )}
        >
          {t("buttons.cancel")}
        </button>
      </div>
    </div>
  );

  // Render preview
  const renderPreview = () => {
    if (!parseResult?.data) return null;

    const summary = getAirMapperSummary(parseResult.data);

    return (
      <div className={layout.stack.default}>
        {/* Warnings */}
        {parseResult.warnings.length > 0 && (
          <div
            className={cn(
              "bg-status-warning/10 border border-status-warning/20",
              radius.md,
              spacing.pad.sm
            )}
          >
            <div className={cn(layout.inline.default, "text-status-warning")}>
              <AlertTriangle className={iconTokens.size.sm} />
              <span className="caption font-medium">
                {t("import.warnings")}
              </span>
            </div>
            <ul
              className={cn(
                "caption text-text-muted",
                spacing.margin.top.tight,
                "list-disc list-inside"
              )}
            >
              {parseResult.warnings.map((w, i) => (
                <li key={i}>{w}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Preview info */}
        <div className={cn("bg-surface-base", radius.md, spacing.pad.default)}>
          <h4
            className={cn(
              "body-small font-medium",
              spacing.margin.bottom.content
            )}
          >
            {summary.surveyName}
          </h4>

          <div className="grid grid-cols-2 gap-2 caption">
            <div className="text-text-muted">{t("import.device")}:</div>
            <div>{summary.deviceInfo}</div>

            <div className="text-text-muted">{t("import.surveyPoints")}:</div>
            <div>{summary.pointCount}</div>

            <div className="text-text-muted">{t("import.facilitySize")}:</div>
            <div>{summary.facilitySize}</div>

            <div className="text-text-muted">{t("import.propagation")}:</div>
            <div>{summary.propagation}</div>
          </div>

          {/* Location counts */}
          <div
            className={cn(
              layout.inline.default,
              spacing.margin.top.content,
              "flex-wrap"
            )}
          >
            <div className={cn(layout.inline.default, "caption")}>
              <MapPin className="w-3 h-3 text-green-500" />
              <span>{summary.apCount} APs</span>
            </div>
            <div className={cn(layout.inline.default, "caption")}>
              <Users className="w-3 h-3 text-blue-500" />
              <span>
                {summary.clientCount} {t("import.clients")}
              </span>
            </div>
            {summary.hasBothModes && (
              <div className={cn(layout.inline.default, "caption")}>
                <Radio className="w-3 h-3 text-purple-500" />
                <span>{t("import.passiveAndActive")}</span>
              </div>
            )}
          </div>

          {/* Floor plan preview */}
          {parseResult.data.floorPlanImage && (
            <div className={spacing.margin.top.content}>
              <img
                src={parseResult.data.floorPlanImage}
                alt="Floor plan preview"
                className={cn(
                  "max-h-40",
                  radius.md,
                  "border border-surface-border"
                )}
              />
            </div>
          )}
        </div>

        {/* Import options */}
        <div
          className={cn(
            "border border-surface-border",
            radius.md,
            spacing.pad.sm
          )}
        >
          <h4
            className={cn("caption font-medium", spacing.margin.bottom.inline)}
          >
            {t("import.options")}
          </h4>
          <div className={layout.stack.default}>
            <label className={cn(layout.inline.default, "cursor-pointer")}>
              <input
                type="checkbox"
                checked={importOptions.importFloorPlan}
                onChange={(e) =>
                  setImportOptions((prev) => ({
                    ...prev,
                    importFloorPlan: e.target.checked,
                  }))
                }
                className="w-4 h-4 accent-brand-primary"
                disabled={!parseResult.data.floorPlanImage}
              />
              <span className="body-small">{t("import.optionFloorPlan")}</span>
            </label>
            <label className={cn(layout.inline.default, "cursor-pointer")}>
              <input
                type="checkbox"
                checked={importOptions.importCalibration}
                onChange={(e) =>
                  setImportOptions((prev) => ({
                    ...prev,
                    importCalibration: e.target.checked,
                  }))
                }
                className="w-4 h-4 accent-brand-primary"
              />
              <span className="body-small">
                {t("import.optionCalibration")}
              </span>
            </label>
            <label className={cn(layout.inline.default, "cursor-pointer")}>
              <input
                type="checkbox"
                checked={importOptions.importLocations}
                onChange={(e) =>
                  setImportOptions((prev) => ({
                    ...prev,
                    importLocations: e.target.checked,
                  }))
                }
                className="w-4 h-4 accent-brand-primary"
              />
              <span className="body-small">{t("import.optionLocations")}</span>
            </label>
          </div>
        </div>

        {/* Actions */}
        <div className={cn(layout.inline.default, "justify-end")}>
          <button
            onClick={handleReset}
            className={cn(
              button.size.sm,
              "border border-surface-border",
              radius.md,
              "hover:bg-surface-hover",
              layout.inline.default
            )}
          >
            <X className={iconTokens.size.sm} />
            {t("import.tryAnother")}
          </button>
          <button
            onClick={onCancel}
            className={cn(
              button.size.sm,
              "border border-surface-border",
              radius.md,
              "hover:bg-surface-hover"
            )}
          >
            {t("buttons.cancel")}
          </button>
          <button
            onClick={handleConfirmImport}
            className={cn(
              button.size.sm,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "hover:bg-brand-primary/90",
              layout.inline.default
            )}
          >
            <Check className={iconTokens.size.sm} />
            {t("import.confirm")}
          </button>
        </div>
      </div>
    );
  };

  return (
    <div
      className={cn(
        "bg-surface-raised",
        radius.md,
        "border border-surface-border",
        spacing.pad.default
      )}
    >
      <div className={cn(layout.inline.default, spacing.margin.bottom.content)}>
        <FileArchive className={iconTokens.size.sm} />
        <h3 className="heading-3">{t("import.title")}</h3>
      </div>

      <p
        className={cn(
          "body-small text-text-muted",
          spacing.margin.bottom.content
        )}
      >
        {t("import.description")}
      </p>

      {!parseResult && renderDropZone()}
      {parseResult && !parseResult.success && renderError()}
      {parseResult?.success && renderPreview()}
    </div>
  );
}
