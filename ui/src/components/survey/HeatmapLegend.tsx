/**
 * HeatmapLegend Component
 *
 * Purpose: Displays a color gradient legend for heatmap visualizations showing
 * metric name, unit, min/max values, and color scale.
 *
 * Features:
 * - Color gradient bar matching heatmap colors
 * - Min/max value labels
 * - Metric name and unit display
 * - Automatic color direction based on metric type
 *
 * Usage:
 * ```typescript
 * <HeatmapLegend
 *   metric="rssi"
 *   minValue={-85}
 *   maxValue={-40}
 * />
 * ```
 */

import type React from "react";
import { useTranslation } from "react-i18next";
import type { HeatmapMetric } from "../../hooks/useSurvey";
import { cn, layout, radius, spacing } from "../../styles/theme";

interface HeatmapLegendProps {
  metric: HeatmapMetric;
  minValue: number;
  maxValue: number;
}

/**
 * Get metric metadata (name, unit, color direction)
 */
function getMetricInfo(
  metric: HeatmapMetric,
  t: (key: string) => string,
): { name: string; unit: string; higherIsBetter: boolean } {
  switch (metric) {
    case "rssi":
      return { name: t("heatmaps.rssi"), unit: "dBm", higherIsBetter: true };
    case "snr":
      return { name: t("heatmaps.snr"), unit: "dB", higherIsBetter: true };
    case "noise":
      return { name: t("heatmaps.noise"), unit: "dBm", higherIsBetter: false };
    case "cochannel":
      return {
        name: t("heatmaps.cochannel"),
        unit: "APs",
        higherIsBetter: false,
      };
    case "adjacent":
      return {
        name: t("heatmaps.adjacent"),
        unit: "APs",
        higherIsBetter: false,
      };
    case "throughput":
      return {
        name: t("heatmaps.throughput"),
        unit: "Mbps",
        higherIsBetter: true,
      };
    case "latency":
      return { name: t("heatmaps.latency"), unit: "ms", higherIsBetter: false };
    case "channelUtil":
      return {
        name: t("heatmaps.channelUtil"),
        unit: "%",
        higherIsBetter: false,
      };
    case "apDensity":
      return {
        name: t("heatmaps.apDensity"),
        unit: "APs",
        higherIsBetter: false,
      };
    case "ssidCount":
      return {
        name: t("heatmaps.ssidCount"),
        unit: "SSIDs",
        higherIsBetter: false,
      };
    default:
      return { name: "Unknown", unit: "", higherIsBetter: true };
  }
}

/**
 * HeatmapLegend Component
 * Displays color gradient legend for heatmap visualizations
 */
export function HeatmapLegend({
  metric,
  minValue,
  maxValue,
}: HeatmapLegendProps): React.ReactElement | null {
  const { t } = useTranslation("survey");

  if (!metric) {
    return null;
  }

  const info = getMetricInfo(metric, t);

  // Build gradient based on metric type
  // For interference metrics (cochannel, adjacent), use blue->purple
  // For others, use green->yellow->red (reversed if higher is better)
  const isInterference = metric === "cochannel" || metric === "adjacent";
  let gradient: string;

  if (isInterference) {
    // Blue (good) to purple (bad)
    gradient = "linear-gradient(to right, rgb(100, 150, 255), rgb(200, 50, 255))";
  } else if (info.higherIsBetter) {
    // Higher is better: green (high) to red (low)
    gradient = "linear-gradient(to right, rgb(255, 0, 0), rgb(255, 255, 0), rgb(0, 255, 0))";
  } else {
    // Lower is better: green (low) to red (high)
    gradient = "linear-gradient(to right, rgb(0, 255, 0), rgb(255, 255, 0), rgb(255, 0, 0))";
  }

  return (
    <div class={cn("bg-surface-raised border border-surface-border", radius.md, spacing.pad.sm)}>
      <div class={cn(layout.flex.between, spacing.margin.bottom.tight)}>
        <span class="body-small font-medium">{info.name}</span>
      </div>

      {/* Color gradient bar */}
      <div
        class={cn("relative h-6", radius.default, "overflow-hidden", spacing.margin.bottom.tight)}
      >
        <div
          class="absolute inset-0"
          style={{
            background: gradient,
          }}
        />
      </div>

      {/* Min/max labels */}
      <div class={layout.flex.between}>
        <span class="caption text-text-muted">
          {minValue.toFixed(1)} {info.unit}
        </span>
        <span class="caption text-text-muted">
          {maxValue.toFixed(1)} {info.unit}
        </span>
      </div>
    </div>
  );
}
