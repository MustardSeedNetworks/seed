/**
 * ScaleCalibrationPanel Component
 *
 * Purpose: Provides floor plan scale calibration and signal propagation settings
 * for WiFi surveys. Supports multiple calibration methods and environment presets.
 *
 * Key Features:
 * - Dimension entry: User inputs facility length/width to calculate scale
 * - Two-point calibration: Click two points and enter known distance
 * - Environment presets: Quick selection for common facility types
 * - Propagation slider: Adjustable signal propagation radius
 * - Smart suggestions: Calculates recommended samples based on area
 *
 * Usage:
 * ```typescript
 * <ScaleCalibrationPanel
 *   floorPlan={survey.floorPlan}
 *   onUpdate={(updates) => handleFloorPlanUpdate(updates)}
 *   onStartCalibration={() => setCalibrationMode(true)}
 * />
 * ```
 */

import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { FloorPlan, ScaleSource } from "../../hooks/useSurvey";
import { Ruler, Building, Sliders } from "lucide-react";
import { useSettings } from "../../contexts/useSettings";
import {
  cn,
  radius,
  spacing,
  layout,
  button,
  icon as iconTokens,
  input as inputTokens,
} from "../../styles/theme";

/** Environment preset configuration */
interface EnvironmentPreset {
  id: string;
  labelKey: string;
  propagationMin: number;
  propagationMax: number;
  propagationDefault: number;
  descriptionKey: string;
}

/** Available environment presets */
const ENVIRONMENT_PRESETS: EnvironmentPreset[] = [
  {
    id: "dense_office",
    labelKey: "environments.denseOffice",
    propagationMin: 5,
    propagationMax: 8,
    propagationDefault: 6,
    descriptionKey: "environments.denseOfficeDesc",
  },
  {
    id: "open_office",
    labelKey: "environments.openOffice",
    propagationMin: 8,
    propagationMax: 12,
    propagationDefault: 10,
    descriptionKey: "environments.openOfficeDesc",
  },
  {
    id: "warehouse",
    labelKey: "environments.warehouse",
    propagationMin: 15,
    propagationMax: 25,
    propagationDefault: 20,
    descriptionKey: "environments.warehouseDesc",
  },
  {
    id: "retail",
    labelKey: "environments.retail",
    propagationMin: 10,
    propagationMax: 15,
    propagationDefault: 12,
    descriptionKey: "environments.retailDesc",
  },
  {
    id: "healthcare",
    labelKey: "environments.healthcare",
    propagationMin: 6,
    propagationMax: 10,
    propagationDefault: 8,
    descriptionKey: "environments.healthcareDesc",
  },
  {
    id: "education",
    labelKey: "environments.education",
    propagationMin: 10,
    propagationMax: 15,
    propagationDefault: 12,
    descriptionKey: "environments.educationDesc",
  },
];

interface ScaleCalibrationPanelProps {
  floorPlan: FloorPlan;
  onUpdate: (updates: Partial<FloorPlan>) => void;
  onStartCalibration: () => void;
  isCalibrating?: boolean;
}

/**
 * ScaleCalibrationPanel provides floor plan scale calibration
 * and signal propagation configuration.
 */
export function ScaleCalibrationPanel({
  floorPlan,
  onUpdate,
  onStartCalibration,
  isCalibrating = false,
}: ScaleCalibrationPanelProps) {
  const { t } = useTranslation("survey");
  const { displayOptions } = useSettings();

  // Use global unit system setting - SAE = feet, Metric = meters
  const isMetric = displayOptions.unitSystem === "metric";

  // Dimension entry state
  const [dimensionMode, setDimensionMode] = useState<"length" | "width">(
    "length"
  );
  const [dimensionValue, setDimensionValue] = useState("");

  // Propagation state - initialize from floorPlan prop
  const [selectedPreset, setSelectedPreset] = useState<string | null>(null);
  const [propagation, setPropagation] = useState(floorPlan.propagationM || 10);

  // Unit conversion helpers
  const metersToDisplay = (meters: number): number =>
    isMetric ? meters : meters * 3.281;
  const displayToMeters = (display: number): number =>
    isMetric ? display : display * 0.3048;
  const formatDistance = (meters: number, decimals = 1): string =>
    `${metersToDisplay(meters).toFixed(decimals)} ${isMetric ? "m" : "ft"}`;
  const formatArea = (sqMeters: number): string => {
    if (isMetric) return `${sqMeters.toFixed(0)} m²`;
    return `${(sqMeters * 10.764).toFixed(0)} ft²`;
  };

  // Calculate facility dimensions from current scale
  const facilityWidthM = floorPlan.width * floorPlan.scaleM;
  const facilityHeightM = floorPlan.height * floorPlan.scaleM;
  const facilityAreaM2 = facilityWidthM * facilityHeightM;

  // Calculate recommended samples based on propagation
  // Use floorPlan.propagationM if available, otherwise use local state
  const effectivePropagation = floorPlan.propagationM || propagation;
  const coverageAreaPerSample =
    Math.PI * effectivePropagation * effectivePropagation;
  const recommendedSamples = Math.max(
    1,
    Math.ceil(facilityAreaM2 / coverageAreaPerSample)
  );

  // Handle dimension entry to calculate scale
  const handleDimensionSubmit = () => {
    const value = parseFloat(dimensionValue);
    if (isNaN(value) || value <= 0) return;

    // Convert to meters using global unit setting
    const valueM = displayToMeters(value);

    // Calculate scale based on which dimension was entered
    const pixelDimension =
      dimensionMode === "length" ? floorPlan.width : floorPlan.height;
    const newScaleM = valueM / pixelDimension;

    onUpdate({
      scaleM: newScaleM,
      scaleSource: "dimensions" as ScaleSource,
    });

    setDimensionValue("");
  };

  // Handle environment preset selection
  const handlePresetSelect = (preset: EnvironmentPreset) => {
    setSelectedPreset(preset.id);
    setPropagation(preset.propagationDefault);
    onUpdate({ propagationM: preset.propagationDefault });
  };

  // Handle propagation slider change
  const handlePropagationChange = (value: number) => {
    setPropagation(value);
    onUpdate({ propagationM: value });
  };

  // Get scale source display text
  const getScaleSourceText = () => {
    switch (floorPlan.scaleSource) {
      case "auto":
        return t("scalePanel.sourceAuto");
      case "dimensions":
        return t("scalePanel.sourceDimensions");
      case "calibration":
        return t("scalePanel.sourceCalibration");
      case "imported":
        return t("scalePanel.sourceImported");
      default:
        return t("scalePanel.sourceDefault");
    }
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
      <h3 className={cn("heading-3", spacing.margin.bottom.content)}>
        {t("scalePanel.title")}
      </h3>

      {/* Current Scale Info */}
      <div
        className={cn(
          "bg-surface-base",
          radius.md,
          spacing.pad.sm,
          spacing.margin.bottom.content
        )}
      >
        <div className={cn(layout.flex.between, "body-small")}>
          <span className="text-text-muted">
            {t("scalePanel.currentScale")}:
          </span>
          <span className="font-medium">
            {floorPlan.scaleM.toFixed(4)} m/px
          </span>
        </div>
        <div
          className={cn(
            layout.flex.between,
            "body-small",
            spacing.margin.top.tight
          )}
        >
          <span className="text-text-muted">{t("scalePanel.source")}:</span>
          <span className="font-medium">{getScaleSourceText()}</span>
        </div>
        <div
          className={cn(
            layout.flex.between,
            "body-small",
            spacing.margin.top.tight
          )}
        >
          <span className="text-text-muted">
            {t("scalePanel.facilitySize")}:
          </span>
          <span className="font-medium">
            {formatDistance(facilityWidthM)} × {formatDistance(facilityHeightM)}{" "}
            ({formatArea(facilityAreaM2)})
          </span>
        </div>
      </div>

      {/* Calibration Methods */}
      <div className={spacing.stack.default}>
        {/* Method 1: Enter Dimensions */}
        <div
          className={cn(
            "border border-surface-border",
            radius.md,
            spacing.pad.sm
          )}
        >
          <div
            className={cn(layout.inline.default, spacing.margin.bottom.inline)}
          >
            <Building className={iconTokens.size.sm} />
            <span className="body-small font-medium">
              {t("scalePanel.enterDimensions")}
            </span>
          </div>
          <div className={layout.inline.default}>
            <select
              value={dimensionMode}
              onChange={(e) =>
                setDimensionMode(e.target.value as "length" | "width")
              }
              className={cn(
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.sm
              )}
            >
              <option value="length">{t("scalePanel.length")}</option>
              <option value="width">{t("scalePanel.width")}</option>
            </select>
            <input
              type="number"
              step="0.1"
              min="0"
              value={dimensionValue}
              onChange={(e) => setDimensionValue(e.target.value)}
              placeholder={t("scalePanel.enterValue")}
              className={cn(
                "flex-1",
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.sm
              )}
            />
            <span className="body-small text-text-muted min-w-8">
              {isMetric ? "m" : "ft"}
            </span>
            <button
              onClick={handleDimensionSubmit}
              disabled={!dimensionValue}
              className={cn(
                button.size.sm,
                "bg-brand-primary text-text-inverse",
                radius.md,
                "hover:bg-brand-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
              )}
            >
              {t("scalePanel.apply")}
            </button>
          </div>
        </div>

        {/* Method 2: Two-Point Calibration */}
        <div
          className={cn(
            "border border-surface-border",
            radius.md,
            spacing.pad.sm
          )}
        >
          <div
            className={cn(layout.inline.default, spacing.margin.bottom.inline)}
          >
            <Ruler className={iconTokens.size.sm} />
            <span className="body-small font-medium">
              {t("scalePanel.measureDistance")}
            </span>
          </div>
          <p
            className={cn(
              "caption text-text-muted",
              spacing.margin.bottom.inline
            )}
          >
            {t("scalePanel.measureInstructions")}
          </p>
          <button
            onClick={onStartCalibration}
            disabled={isCalibrating}
            className={cn(
              button.size.sm,
              "border border-surface-border",
              radius.md,
              "hover:bg-surface-hover disabled:opacity-50"
            )}
          >
            {isCalibrating
              ? t("scalePanel.calibrating")
              : t("scalePanel.startMeasurement")}
          </button>
        </div>
      </div>

      {/* Signal Propagation Section */}
      <div
        className={cn(
          "border-t border-surface-border",
          spacing.margin.top.content,
          "pt-4"
        )}
      >
        <div
          className={cn(layout.inline.default, spacing.margin.bottom.content)}
        >
          <Sliders className={iconTokens.size.sm} />
          <span className="body-small font-medium">
            {t("scalePanel.signalPropagation")}
          </span>
        </div>

        {/* Environment Presets */}
        <div className={spacing.margin.bottom.content}>
          <label
            className={cn(
              "caption text-text-muted block",
              spacing.margin.bottom.tight
            )}
          >
            {t("scalePanel.environmentType")}
          </label>
          <div className="flex flex-wrap gap-2">
            {ENVIRONMENT_PRESETS.map((preset) => (
              <button
                key={preset.id}
                onClick={() => handlePresetSelect(preset)}
                className={cn(
                  button.size.xs,
                  radius.md,
                  "border",
                  selectedPreset === preset.id
                    ? "bg-brand-primary text-text-inverse border-brand-primary"
                    : "border-surface-border hover:bg-surface-hover"
                )}
              >
                {t(preset.labelKey as never)}
              </button>
            ))}
          </div>
        </div>

        {/* Propagation Slider */}
        <div>
          <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
            <label className="caption text-text-muted">
              {t("scalePanel.propagationRadius")}
            </label>
            <span className="body-small font-medium">
              {formatDistance(propagation)}
            </span>
          </div>
          <input
            type="range"
            min="3"
            max="30"
            step="1"
            value={propagation}
            onChange={(e) => handlePropagationChange(parseInt(e.target.value))}
            className="w-full"
          />
          <div
            className={cn(
              layout.flex.between,
              "caption text-text-muted",
              spacing.margin.top.tight
            )}
          >
            <span>{formatDistance(3, 0)}</span>
            <span>{formatDistance(30, 0)}</span>
          </div>
        </div>

        {/* Sample Recommendation */}
        <div
          className={cn(
            "bg-status-info/10 border border-status-info/20",
            radius.md,
            spacing.pad.sm,
            spacing.margin.top.content
          )}
        >
          <div className="body-small text-status-info">
            {t("scalePanel.recommendedSamples", { count: recommendedSamples })}
          </div>
          <div
            className={cn("caption text-text-muted", spacing.margin.top.tight)}
          >
            {t("scalePanel.coveragePerSample", {
              area: formatArea(coverageAreaPerSample),
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
