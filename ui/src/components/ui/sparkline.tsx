/**
 * Sparkline Component
 *
 * Purpose: Compact SVG line chart for displaying trends in health check data.
 * Shows 24h availability or latency trends with color-coded indicators.
 *
 * Key Features:
 * - Pure SVG rendering for crisp display at any size
 * - Smooth line interpolation with optional area fill
 * - Color-coded based on trend direction or threshold
 * - Three sizes: sm (60x24), md (80x32), lg (120x40)
 * - Animated path drawing on mount
 * - Supports availability (0-100%) and latency (ms) data
 * - Tooltip on hover showing exact values
 *
 * Usage:
 * ```typescript
 * // Availability sparkline
 * <Sparkline
 *   data={[99.5, 99.8, 100, 99.2, 98.5, 99.9, 100]}
 *   type="availability"
 *   size="md"
 * />
 *
 * // Latency sparkline
 * <Sparkline
 *   data={[45, 52, 48, 55, 42, 47, 50]}
 *   type="latency"
 *   threshold={100}
 *   size="md"
 * />
 * ```
 *
 * Dependencies: React, memo/useMemo hooks, theme utilities
 * State: Memoized calculations for path generation and scaling
 */

import type React from "react";
import { memo, useMemo } from "react";
import { cn, radius } from "../../styles/theme";

export type SparklineType = "availability" | "latency" | "score";

interface SparklineProps {
  /** Data points to display (most recent last) */
  data: number[];
  /** Type of data for coloring and scaling */
  type?: SparklineType;
  /** Size variant */
  size?: "sm" | "md" | "lg";
  /** Custom threshold for warning/error coloring (latency mode) */
  threshold?: number;
  /** Show area fill under the line */
  showArea?: boolean;
  /** Custom CSS classes */
  className?: string;
  /** Label for accessibility */
  label?: string;
}

// Size configurations
const sizeConfigs = {
  sm: { width: 60, height: 24, strokeWidth: 1.5 },
  md: { width: 80, height: 32, strokeWidth: 2 },
  lg: { width: 120, height: 40, strokeWidth: 2 },
};

// Get color based on value and type
function getSparklineColor(value: number, type: SparklineType, threshold?: number): string {
  if (type === "availability" || type === "score") {
    // Higher is better
    if (value >= 99) {
      return "var(--color-status-success)";
    }
    if (value >= 90) {
      return "var(--color-status-warning)";
    }
    return "var(--color-status-error)";
  }

  // Latency - lower is better
  const effectiveThreshold = threshold ?? 100;
  const ratio = value / effectiveThreshold;
  if (ratio <= 0.5) {
    return "var(--color-status-success)";
  }
  if (ratio <= 1.0) {
    return "var(--color-status-warning)";
  }
  return "var(--color-status-error)";
}

// Generate SVG path from data points
function generatePath(
  data: number[],
  width: number,
  height: number,
  padding: number,
  minValue: number,
  maxValue: number,
): string {
  if (data.length < 2) {
    return "";
  }

  const effectiveWidth = width - padding * 2;
  const effectiveHeight = height - padding * 2;
  const range = maxValue - minValue || 1;

  const points = data.map((value, index) => {
    const x = padding + (index / (data.length - 1)) * effectiveWidth;
    const normalizedValue = (value - minValue) / range;
    const y = height - padding - normalizedValue * effectiveHeight;
    return { x, y };
  });

  // Create smooth curve using quadratic bezier
  let path = `M ${points[0].x} ${points[0].y}`;

  for (let i = 0; i < points.length - 1; i++) {
    const current = points[i];
    const next = points[i + 1];
    const midX = (current.x + next.x) / 2;

    // Use quadratic bezier for smooth curves
    path += ` Q ${current.x} ${current.y} ${midX} ${(current.y + next.y) / 2}`;
  }

  // Connect to last point
  const last = points.at(-1);
  path += ` L ${last.x} ${last.y}`;

  return path;
}

// Generate area path (closed path for fill)
function generateAreaPath(
  linePath: string,
  width: number,
  height: number,
  padding: number,
): string {
  if (!linePath) {
    return "";
  }

  const baseY = height - padding;
  const startX = padding;
  const endX = width - padding;

  return `${linePath} L ${endX} ${baseY} L ${startX} ${baseY} Z`;
}

export const Sparkline: React.MemoExoticComponent<typeof SparklineComponent> =
  memo(SparklineComponent);

function SparklineComponent({
  data,
  type = "availability",
  size = "md",
  threshold,
  showArea = true,
  className,
  label,
}: SparklineProps): React.JSX.Element {
  const config = sizeConfigs[size];
  const padding = 2;

  // Calculate min/max for scaling
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex calculation with trend analysis
  const { minValue, maxValue, currentValue, trendDirection } = useMemo(() => {
    if (data.length === 0) {
      return {
        minValue: 0,
        maxValue: 100,
        currentValue: 0,
        trendDirection: "stable" as const,
      };
    }

    const [first] = data;
    let min = first;
    let max = first;
    let sum = 0;

    for (const value of data) {
      if (value < min) {
        min = value;
      }
      if (value > max) {
        max = value;
      }
      sum += value;
    }

    // For availability/score, use fixed 0-100 range for consistency
    if (type === "availability" || type === "score") {
      min = Math.min(min, 0);
      max = Math.max(max, 100);
    } else {
      // Add some padding to the range for latency
      const range = max - min;
      min = Math.max(0, min - range * 0.1);
      max += range * 0.1;
    }

    const avg = sum / data.length;
    const current = data.at(-1);

    // Determine trend direction
    let trend: "up" | "down" | "stable" = "stable";
    if (data.length >= 2) {
      const recentAvg =
        data.slice(-Math.min(3, data.length)).reduce((a, b) => a + b, 0) / Math.min(3, data.length);
      const olderAvg =
        data.slice(0, -Math.min(3, data.length)).reduce((a, b) => a + b, 0) /
        Math.max(1, data.length - Math.min(3, data.length));

      const diff = recentAvg - olderAvg;
      const thresholdPct = avg * 0.05; // 5% change threshold

      if (diff > thresholdPct) {
        trend = "up";
      } else if (diff < -thresholdPct) {
        trend = "down";
      }
    }

    return {
      minValue: min,
      maxValue: max,
      currentValue: current,
      trendDirection: trend,
    };
  }, [data, type]);

  // Generate paths
  const { linePath, areaPath } = useMemo(() => {
    const line = generatePath(data, config.width, config.height, padding, minValue, maxValue);
    const area = showArea ? generateAreaPath(line, config.width, config.height, padding) : "";
    return { linePath: line, areaPath: area };
  }, [data, config, minValue, maxValue, showArea]);

  // Get stroke color based on current value
  const strokeColor = getSparklineColor(currentValue, type, threshold);

  // Handle empty data
  if (data.length < 2) {
    return (
      <div
        role="img"
        class={cn("flex items-center justify-center text-text-muted", className)}
        style={{ width: config.width, height: config.height }}
        aria-label={label || "No data available"}
      >
        <span class="caption">—</span>
      </div>
    );
  }

  // Accessibility label
  const accessibilityLabel =
    label ||
    `${type} sparkline: current ${currentValue.toFixed(1)}${type === "latency" ? "ms" : "%"}, trend ${trendDirection}`;

  return (
    <div class={cn("relative inline-flex", className)}>
      <svg
        width={config.width}
        height={config.height}
        viewBox={`0 0 ${config.width} ${config.height}`}
        role="img"
        aria-label={accessibilityLabel}
        class="overflow-visible"
      >
        {/* Area fill (gradient from line color to transparent) */}
        {showArea && areaPath ? (
          <>
            <defs>
              <linearGradient id={`sparkline-gradient-${type}`} x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stopColor={strokeColor} stopOpacity="0.3" />
                <stop offset="100%" stopColor={strokeColor} stopOpacity="0.05" />
              </linearGradient>
            </defs>
            <path
              d={areaPath}
              fill={`url(#sparkline-gradient-${type})`}
              class="transition-all duration-500 ease-out"
            />
          </>
        ) : null}

        {/* Line */}
        <path
          d={linePath}
          fill="none"
          stroke={strokeColor}
          strokeWidth={config.strokeWidth}
          strokeLinecap="round"
          strokeLinejoin="round"
          class="transition-all duration-500 ease-out"
        />

        {/* Current value indicator dot */}
        <circle
          cx={config.width - padding}
          cy={
            config.height -
            padding -
            ((currentValue - minValue) / (maxValue - minValue || 1)) * (config.height - padding * 2)
          }
          r={config.strokeWidth + 1}
          fill={strokeColor}
          class="transition-all duration-300 ease-out"
        />
      </svg>
    </div>
  );
}

/**
 * SparklineWithLabel - Sparkline with inline label and current value
 */
interface SparklineWithLabelProps extends SparklineProps {
  /** Label to show before the sparkline */
  labelText?: string;
  /** Show current value after sparkline */
  showValue?: boolean;
  /** Unit suffix for value */
  unit?: string;
}

export const SparklineWithLabel: React.MemoExoticComponent<typeof SparklineWithLabelComponent> =
  memo(SparklineWithLabelComponent);

function SparklineWithLabelComponent({
  labelText,
  showValue = true,
  unit,
  data,
  type = "availability",
  ...props
}: SparklineWithLabelProps): React.JSX.Element {
  const currentValue = data.length > 0 ? data.at(-1) : 0;

  // Format value based on type
  const formattedValue = useMemo((): string => {
    if (type === "latency") {
      if (currentValue >= 1000) {
        return `${(currentValue / 1000).toFixed(1)}s`;
      }
      return `${Math.round(currentValue)}ms`;
    }
    return `${currentValue.toFixed(1)}%`;
  }, [currentValue, type]);

  const displayUnit = unit ?? (type === "latency" ? "" : "");

  return (
    <div class="inline-flex items-center gap-2">
      {labelText ? <span class="caption text-text-muted">{labelText}</span> : null}
      <Sparkline data={data} type={type} {...props} />
      {showValue ? (
        <span class="caption font-medium text-text-primary tabular-nums">
          {formattedValue}
          {displayUnit}
        </span>
      ) : null}
    </div>
  );
}

/**
 * HealthScoreBadge - Compact badge showing health score with color coding
 */
interface HealthScoreBadgeProps {
  /** Score from 0-100 */
  score: number;
  /** Size variant */
  size?: "sm" | "md" | "lg";
  /** Show numeric value */
  showValue?: boolean;
  /** Custom className */
  className?: string;
}

export const HealthScoreBadge: React.MemoExoticComponent<typeof HealthScoreBadgeComponent> =
  memo(HealthScoreBadgeComponent);

function HealthScoreBadgeComponent({
  score,
  size = "md",
  showValue = true,
  className,
}: HealthScoreBadgeProps): React.JSX.Element {
  // Determine status color
  const getStatusColor = (): string => {
    if (score >= 80) {
      return "bg-status-success/15 text-status-success border-status-success/30";
    }
    if (score >= 50) {
      return "bg-status-warning/15 text-status-warning border-status-warning/30";
    }
    return "bg-status-error/15 text-status-error border-status-error/30";
  };

  const getStatusLabel = (): string => {
    if (score >= 80) {
      return "Healthy";
    }
    if (score >= 50) {
      return "Degraded";
    }
    return "Critical";
  };

  const sizeClasses = {
    sm: "px-1.5 py-0.5 text-[10px]",
    md: "px-2 py-1 text-xs",
    lg: "px-3 py-1.5 text-sm",
  };

  return (
    <span
      class={cn(
        "inline-flex items-center gap-1 font-medium border",
        radius.md,
        sizeClasses[size],
        getStatusColor(),
        className,
      )}
      title={`Health Score: ${score.toFixed(0)}% - ${getStatusLabel()}`}
    >
      {showValue ? <span class="tabular-nums">{Math.round(score)}</span> : null}
      <span class={showValue ? "hidden sm:inline" : ""}>{getStatusLabel()}</span>
    </span>
  );
}
