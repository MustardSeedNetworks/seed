/**
 * SpeedGauge Component (and related gauge utilities)
 *
 * Purpose: Visual speedometer gauge for displaying internet speed test results.
 * Shows current speed with color-coded indicator (green→yellow→red) based on percentage.
 * Includes ProgressRing for circular progress indicators and PulsingDot for animated indicators.
 *
 * Key Features:
 * - SVG gauge: arc-based visual representation with animated stroke
 * - Auto-scaling: converts Mbps to Gbps when value exceeds 1000 Mbps
 * - Color-coded: green (good), yellow (medium), red (poor) based on percentage
 * - Three sizes: sm (100x60), md (140x85), lg (180x110)
 * - Running state: shows pulsing animation during active tests
 * - Customizable range: maxValue prop sets gauge scale
 * - ProgressRing: circular progress bar for percentage indicators
 * - PulsingDot: animated dot indicator for "in progress" states
 *
 * Usage:
 * ```typescript
 * // Speed gauge
 * <SpeedGauge
 *   value={250}
 *   maxValue={1000}
 *   label="Download"
 *   isRunning={false}
 *   size="md"
 * />
 *
 * // Progress ring (for percentages)
 * <ProgressRing percent={75} size={100} />
 *
 * // Pulsing dot (for active states)
 * <PulsingDot />
 * ```
 *
 * Dependencies: React, memo/useMemo hooks, theme utilities (gauge config)
 * State: Memoized calculations for display values and size configurations
 */

import { memo, useMemo } from "react";
import { cn, gauge, radius, spacing } from "../../styles/theme";

interface SpeedGaugeProps {
  value: number; // Current speed in Mbps
  maxValue?: number; // Max scale value
  label?: string; // "Download" or "Upload"
  unit?: string; // "Mbps" or "Gbps"
  isRunning?: boolean; // Show animated pulsing when test is running
  size?: "sm" | "md" | "lg";
}

// Calculate gauge color based on speed percentage using CSS variables
function getGaugeColor(percentage: number): string {
  return gauge.getColor(percentage);
}

export const SpeedGauge = memo(function SpeedGauge({
  value,
  maxValue = 1000,
  label,
  unit = "Mbps",
  isRunning = false,
  size = "md",
}: SpeedGaugeProps) {
  // Calculate display value (auto-convert to Gbps if > 1000 Mbps)
  const displayValue = useMemo(() => {
    if (value >= 1000) {
      return {
        value: (value / 1000).toFixed(2),
        unit: "Gbps",
      };
    }
    return {
      value: value.toFixed(1),
      unit: unit,
    };
  }, [value, unit]);

  // Size configurations
  const sizeConfig = useMemo(() => {
    switch (size) {
      case "sm":
        return { width: 100, height: 60, strokeWidth: 8, fontSize: 14 };
      case "lg":
        return { width: 180, height: 110, strokeWidth: 14, fontSize: 24 };
      default:
        return { width: 140, height: 85, strokeWidth: 12, fontSize: 18 };
    }
  }, [size]);

  // Gauge calculations
  const percentage = Math.min((value / maxValue) * 100, 100);
  const gaugeColor = getGaugeColor(percentage);

  // SVG arc calculations
  const radius = (sizeConfig.width - sizeConfig.strokeWidth) / 2;
  const centerX = sizeConfig.width / 2;
  const centerY = sizeConfig.height - 10;

  // Arc goes from -150 degrees to -30 degrees (180 degree sweep)
  const startAngle = -150;
  const endAngle = -30;
  const angleRange = endAngle - startAngle;
  const currentAngle = startAngle + (percentage / 100) * angleRange;

  // Convert angle to radians
  const toRadians = (angle: number) => (angle * Math.PI) / 180;

  // Calculate arc path
  const startX = centerX + radius * Math.cos(toRadians(startAngle));
  const startY = centerY + radius * Math.sin(toRadians(startAngle));
  const endX = centerX + radius * Math.cos(toRadians(endAngle));
  const endY = centerY + radius * Math.sin(toRadians(endAngle));
  const currentX = centerX + radius * Math.cos(toRadians(currentAngle));
  const currentY = centerY + radius * Math.sin(toRadians(currentAngle));

  // Background arc (full gauge)
  const bgArcPath = `M ${startX} ${startY} A ${radius} ${radius} 0 0 1 ${endX} ${endY}`;

  // Value arc (filled portion)
  const valueArcPath = `M ${startX} ${startY} A ${radius} ${radius} 0 0 1 ${currentX} ${currentY}`;

  // Tick marks
  const ticks = [0, 25, 50, 75, 100];

  return (
    <div className="flex flex-col items-center">
      {label && (
        <p
          className={cn(
            "caption text-text-muted",
            spacing.margin.bottom.tight,
            "uppercase tracking-wider",
          )}
        >
          {label}
        </p>
      )}
      <div className="relative">
        <svg
          width={sizeConfig.width}
          height={sizeConfig.height}
          viewBox={`0 0 ${sizeConfig.width} ${sizeConfig.height}`}
          className={isRunning ? "animate-pulse" : ""}
          role="img"
          aria-label={`Speed gauge showing ${value} ${unit}`}
        >
          {/* Background arc */}
          <path
            d={bgArcPath}
            fill="none"
            stroke="currentColor"
            strokeWidth={sizeConfig.strokeWidth}
            strokeLinecap="round"
            className="text-surface-border"
          />

          {/* Value arc with gradient */}
          {value > 0 && (
            <path
              d={valueArcPath}
              fill="none"
              stroke={gaugeColor}
              strokeWidth={sizeConfig.strokeWidth}
              strokeLinecap="round"
              className="transition-all duration-500 ease-out"
            />
          )}

          {/* Tick marks */}
          {ticks.map((tick) => {
            const tickAngle = startAngle + (tick / 100) * angleRange;
            const innerRadius = radius - sizeConfig.strokeWidth / 2 - 4;
            const outerRadius = radius - sizeConfig.strokeWidth / 2 - 8;
            const tickStartX = centerX + innerRadius * Math.cos(toRadians(tickAngle));
            const tickStartY = centerY + innerRadius * Math.sin(toRadians(tickAngle));
            const tickEndX = centerX + outerRadius * Math.cos(toRadians(tickAngle));
            const tickEndY = centerY + outerRadius * Math.sin(toRadians(tickAngle));
            return (
              <line
                key={tick}
                x1={tickStartX}
                y1={tickStartY}
                x2={tickEndX}
                y2={tickEndY}
                stroke="currentColor"
                strokeWidth={1.5}
                className="text-text-muted/50"
              />
            );
          })}

          {/* Needle indicator */}
          {value > 0 && (
            <circle
              cx={currentX}
              cy={currentY}
              r={4}
              fill={gaugeColor}
              className="transition-all duration-500 ease-out"
            />
          )}
        </svg>

        {/* Center value display */}
        <div
          className={cn(
            "absolute inset-0 flex flex-col items-center justify-end",
            size === "sm"
              ? spacing.micro.pb
              : size === "md"
                ? spacing.micro.pbCompact
                : spacing.micro.pbCompactMd, // Semantic tokens for gauge value positioning
          )}
        >
          <span
            className={cn(
              "font-mono font-bold text-text-primary tabular-nums",
              size === "sm" ? "text-[12px]" : size === "md" ? "text-[14px]" : "text-[16px]",
            )}
          >
            {isRunning && value === 0 ? "—" : displayValue.value}
          </span>
          <span className={cn("caption text-text-muted", spacing.micro.mtNeg)}>
            {displayValue.unit}
          </span>{" "}
          {/* Negative margin for tight visual alignment between value and unit */}
        </div>
      </div>
    </div>
  );
});

// Simple progress ring for general test progress
interface ProgressRingProps {
  progress: number; // 0-100
  size?: number;
  strokeWidth?: number;
  label?: string;
}

export const ProgressRing = memo(function ProgressRing({
  progress,
  size = 48,
  strokeWidth = 4,
  label,
}: ProgressRingProps) {
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (progress / 100) * circumference;

  return (
    <div className="flex flex-col items-center">
      <div className="relative">
        <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
          {/* Background circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            className="text-surface-border"
          />
          {/* Progress circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            className="text-brand-primary transition-all duration-300 ease-out"
          />
        </svg>
        {/* Center percentage */}
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="caption font-medium text-text-primary tabular-nums">
            {Math.round(progress)}%
          </span>
        </div>
      </div>
      {label && (
        <span className={cn("caption text-text-muted", spacing.margin.top.tight, "text-center")}>
          {label}
        </span>
      )}
    </div>
  );
});

// Animated pulsing dot for "in progress" indicator
interface PulsingDotProps {
  color?: "primary" | "success" | "warning" | "error";
  size?: "sm" | "md";
}

export const PulsingDot = memo(function PulsingDot({
  color = "primary",
  size = "md",
}: PulsingDotProps) {
  // Type-safe color class getter
  function getColorClass(c: PulsingDotProps["color"]) {
    switch (c) {
      case "primary":
        return "bg-brand-primary";
      case "success":
        return "bg-status-success";
      case "warning":
        return "bg-status-warning";
      case "error":
        return "bg-status-error";
      default:
        return "bg-brand-primary";
    }
  }

  // Type-safe size class getter
  function getSizeClass(s: PulsingDotProps["size"]) {
    return s === "sm" ? "w-2 h-2" : "w-3 h-3";
  }

  const colorClass = getColorClass(color);
  const sizeClass = getSizeClass(size);

  return (
    <span className="relative flex">
      <span
        className={cn(
          "animate-ping absolute inline-flex h-full w-full",
          radius.full,
          "opacity-75",
          colorClass,
        )}
      />
      <span className={cn("relative inline-flex", radius.full, sizeClass, colorClass)} />
    </span>
  );
});
