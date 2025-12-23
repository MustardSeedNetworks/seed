/**
 * HeatmapStats Component
 *
 * Purpose: Displays statistical metrics for heatmap data including
 * average, median, standard deviation, and sample count.
 *
 * Features:
 * - Calculates avg, median, std dev from sample data
 * - Displays metric-appropriate units
 * - Shows sample count for context
 * - Compact grid layout
 *
 * Usage:
 * ```typescript
 * <HeatmapStats
 *   samples={surveyData.samples}
 *   metric="rssi"
 * />
 * ```
 */

import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { HeatmapMetric, SamplePoint } from "../../hooks/useSurvey";
import { cn, radius, spacing } from "../../styles/theme";

interface HeatmapStatsProps {
  samples: SamplePoint[];
  metric: HeatmapMetric;
}

interface MetricStats {
  average: number;
  median: number;
  stdDev: number;
  min: number;
  max: number;
  count: number;
}

/**
 * Get metric unit for display
 */
function getMetricUnit(metric: HeatmapMetric): string {
  switch (metric) {
    case "rssi":
    case "noise":
      return "dBm";
    case "snr":
      return "dB";
    case "cochannel":
    case "adjacent":
    case "apDensity":
      return "APs";
    case "throughput":
      return "Mbps";
    case "latency":
      return "ms";
    case "channelUtil":
      return "%";
    case "ssidCount":
      return "SSIDs";
    default:
      return "";
  }
}

/**
 * Extract metric value from a sample point
 */
function extractMetricValue(
  sample: SamplePoint,
  metric: HeatmapMetric
): number | null {
  const data = sample.sampleData;
  if (!data) return null;

  // Handle passive samples (networks array)
  if ("networks" in data && data.networks) {
    const networks = data.networks;
    if (networks.length === 0) return null;

    switch (metric) {
      case "rssi": {
        // Best RSSI from all networks
        const rssiValues = networks
          .map((n) => n.rssi)
          .filter((r) => r !== undefined);
        return rssiValues.length > 0 ? Math.max(...rssiValues) : null;
      }
      case "snr": {
        // Best SNR from all networks
        const snrValues = networks
          .map((n) => n.snr)
          .filter((s): s is number => s !== undefined);
        return snrValues.length > 0 ? Math.max(...snrValues) : null;
      }
      case "noise": {
        // Noise floor (use from sample or first network)
        if ("noiseFloor" in data && data.noiseFloor !== undefined) {
          return data.noiseFloor;
        }
        const noiseValues = networks
          .map((n) => n.noiseFloor)
          .filter((n): n is number => n !== undefined);
        return noiseValues.length > 0 ? noiseValues[0] : null;
      }
      case "cochannel": {
        // Count networks on same channel as strongest
        const strongest = networks.reduce(
          (best, n) => (n.rssi > (best?.rssi ?? -999) ? n : best),
          networks[0]
        );
        if (!strongest) return null;
        return networks.filter(
          (n) => n.channel === strongest.channel && n.bssid !== strongest.bssid
        ).length;
      }
      case "adjacent": {
        // Count networks on adjacent channels
        const strongest = networks.reduce(
          (best, n) => (n.rssi > (best?.rssi ?? -999) ? n : best),
          networks[0]
        );
        if (!strongest) return null;
        const ch = strongest.channel;
        return networks.filter((n) => {
          const diff = Math.abs(n.channel - ch);
          return diff > 0 && diff <= 2 && n.bssid !== strongest.bssid;
        }).length;
      }
      case "apDensity":
        return networks.length;
      case "ssidCount": {
        const uniqueSSIDs = new Set(
          networks.map((n) => n.ssid).filter((s) => s && s !== "")
        );
        return uniqueSSIDs.size;
      }
      default:
        return null;
    }
  }

  // Handle active samples
  if ("rssi" in data && "dataRate" in data) {
    switch (metric) {
      case "rssi":
        return data.rssi;
      case "throughput":
        return data.dataRate;
      default:
        return null;
    }
  }

  // Handle throughput samples
  if ("downloadMbps" in data) {
    switch (metric) {
      case "rssi":
        return data.rssi;
      case "throughput":
        return (data.downloadMbps + data.uploadMbps) / 2;
      case "latency":
        return data.latency;
      default:
        return null;
    }
  }

  return null;
}

/**
 * Calculate statistics for a set of values
 */
function calculateStats(values: number[]): MetricStats {
  if (values.length === 0) {
    return { average: 0, median: 0, stdDev: 0, min: 0, max: 0, count: 0 };
  }

  const sorted = [...values].sort((a, b) => a - b);
  const count = sorted.length;
  const min = sorted[0];
  const max = sorted[count - 1];
  const sum = sorted.reduce((acc, v) => acc + v, 0);
  const average = sum / count;

  // Median - using at() for safe array access
  const mid = Math.floor(count / 2);
  const median =
    count % 2 === 0
      ? ((sorted.at(mid - 1) ?? 0) + (sorted.at(mid) ?? 0)) / 2
      : (sorted.at(mid) ?? 0);

  // Standard deviation
  const squaredDiffs = sorted.map((v) => Math.pow(v - average, 2));
  const avgSquaredDiff = squaredDiffs.reduce((acc, v) => acc + v, 0) / count;
  const stdDev = Math.sqrt(avgSquaredDiff);

  return { average, median, stdDev, min, max, count };
}

/**
 * HeatmapStats Component
 * Displays statistical analysis of heatmap data
 */
export function HeatmapStats({ samples, metric }: HeatmapStatsProps) {
  const { t } = useTranslation("survey");

  const stats = useMemo(() => {
    if (!metric || samples.length === 0) {
      return null;
    }

    const values: number[] = [];
    for (const sample of samples) {
      const value = extractMetricValue(sample, metric);
      if (value !== null && !isNaN(value)) {
        values.push(value);
      }
    }

    if (values.length === 0) return null;

    return calculateStats(values);
  }, [samples, metric]);

  if (!stats || stats.count === 0) return null;

  const unit = getMetricUnit(metric);

  return (
    <div
      className={cn(
        "bg-surface-raised border border-surface-border",
        radius.md,
        spacing.pad.sm,
        spacing.margin.top.tight
      )}
    >
      <div className={cn("grid grid-cols-2 gap-x-4 gap-y-1")}>
        <div className="flex justify-between">
          <span className="caption text-text-muted">
            {t("heatmapStats.average")}
          </span>
          <span className="caption font-medium">
            {stats.average.toFixed(1)} {unit}
          </span>
        </div>
        <div className="flex justify-between">
          <span className="caption text-text-muted">
            {t("heatmapStats.median")}
          </span>
          <span className="caption font-medium">
            {stats.median.toFixed(1)} {unit}
          </span>
        </div>
        <div className="flex justify-between">
          <span className="caption text-text-muted">
            {t("heatmapStats.stdDev")}
          </span>
          <span className="caption font-medium">
            {stats.stdDev.toFixed(1)} {unit}
          </span>
        </div>
        <div className="flex justify-between">
          <span className="caption text-text-muted">
            {t("heatmapStats.samples")}
          </span>
          <span className="caption font-medium">{stats.count}</span>
        </div>
      </div>
    </div>
  );
}
