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

import type React from 'react';
import { memo, useMemo } from 'react';
import { cn, gauge, radius as radiusTokens, spacing } from '../../styles/theme';

interface SpeedGaugeProps {
  value: number; // Current speed in Mbps
  maxValue?: number; // Max scale value
  label?: string; // "Download" or "Upload"
  unit?: string; // "Mbps" or "Gbps"
  isRunning?: boolean; // Show animated pulsing when test is running
  size?: 'sm' | 'md' | 'lg';
}

// Calculate gauge color based on speed percentage using CSS variables
function getGaugeColor(percentage: number): string {
  return gauge.getColor(percentage);
}

// Helper for gauge padding bottom class
function getPaddingBottomClass(size: 'sm' | 'md' | 'lg'): string {
  switch (size) {
    case 'sm':
      return spacing.micro.pb;
    case 'md':
      return spacing.micro.pbCompact;
    case 'lg':
      return spacing.micro.pbCompactMd;
    default: {
      const _exhaustive: never = size;
      return spacing.micro.pbCompact;
    }
  }
}

// Helper for gauge font size class
function getFontSizeClass(size: 'sm' | 'md' | 'lg'): string {
  switch (size) {
    case 'sm':
      return 'text-[12px]';
    case 'md':
      return 'text-[14px]';
    case 'lg':
      return 'text-[16px]';
    default: {
      const _exhaustive: never = size;
      return 'text-[14px]';
    }
  }
}

export const SpeedGauge: React.MemoExoticComponent<typeof SpeedGaugeComponent> =
  memo(SpeedGaugeComponent);

function SpeedGaugeComponent({
  value,
  maxValue = 1000,
  label,
  unit = 'Mbps',
  isRunning = false,
  size = 'md',
}: SpeedGaugeProps): React.JSX.Element {
  // Calculate display value (auto-convert to Gbps if > 1000 Mbps)
  const displayValue: { value: string; unit: string } = useMemo(() => {
    if (value >= 1000) {
      return {
        value: (value / 1000).toFixed(2),
        unit: 'Gbps',
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
      case 'sm':
        return { width: 100, height: 60, strokeWidth: 8, fontSize: 14 };
      case 'lg':
        return { width: 180, height: 110, strokeWidth: 14, fontSize: 24 };
      default:
        return { width: 140, height: 85, strokeWidth: 12, fontSize: 18 };
    }
  }, [size]);

  // Gauge calculations
  const percentage = Math.min((value / maxValue) * 100, 100);
  const gaugeColor = getGaugeColor(percentage);

  // SVG arc calculations
  const arcRadius = (sizeConfig.width - sizeConfig.strokeWidth) / 2;
  const centerX = sizeConfig.width / 2;
  const centerY = sizeConfig.height - 10;

  // Arc goes from -150 degrees to -30 degrees (180 degree sweep)
  const startAngle = -150;
  const endAngle = -30;
  const angleRange = endAngle - startAngle;
  const currentAngle = startAngle + (percentage / 100) * angleRange;

  // Convert angle to radians
  const toRadians = (angle: number): number => (angle * Math.PI) / 180;

  // Calculate arc path
  const startX = centerX + arcRadius * Math.cos(toRadians(startAngle));
  const startY = centerY + arcRadius * Math.sin(toRadians(startAngle));
  const endX = centerX + arcRadius * Math.cos(toRadians(endAngle));
  const endY = centerY + arcRadius * Math.sin(toRadians(endAngle));
  const currentX = centerX + arcRadius * Math.cos(toRadians(currentAngle));
  const currentY = centerY + arcRadius * Math.sin(toRadians(currentAngle));

  // Background arc (full gauge)
  const bgArcPath = `M ${startX} ${startY} A ${arcRadius} ${arcRadius} 0 0 1 ${endX} ${endY}`;

  // Value arc (filled portion)
  const valueArcPath = `M ${startX} ${startY} A ${arcRadius} ${arcRadius} 0 0 1 ${currentX} ${currentY}`;

  // Tick marks
  const ticks = [0, 25, 50, 75, 100];

  return (
    <div class="flex flex-col items-center">
      {label ? (
        <p
          class={cn(
            'caption text-text-muted',
            spacing.margin.bottom.tight,
            'uppercase tracking-wider',
          )}
        >
          {label}
        </p>
      ) : null}
      <div class="relative">
        <svg
          width={sizeConfig.width}
          height={sizeConfig.height}
          viewBox={`0 0 ${sizeConfig.width} ${sizeConfig.height}`}
          class={isRunning ? 'animate-pulse' : ''}
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
            class="text-surface-border"
          />

          {/* Value arc with gradient */}
          {value > 0 ? (
            <path
              d={valueArcPath}
              fill="none"
              stroke={gaugeColor}
              strokeWidth={sizeConfig.strokeWidth}
              strokeLinecap="round"
              class="transition-all duration-500 ease-out"
            />
          ) : null}

          {/* Tick marks */}
          {ticks.map((tick) => {
            const tickAngle = startAngle + (tick / 100) * angleRange;
            const innerRadius = arcRadius - sizeConfig.strokeWidth / 2 - 4;
            const outerRadius = arcRadius - sizeConfig.strokeWidth / 2 - 8;
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
                class="text-text-muted/50"
              />
            );
          })}

          {/* Needle indicator */}
          {value > 0 ? (
            <circle
              cx={currentX}
              cy={currentY}
              r={4}
              fill={gaugeColor}
              class="transition-all duration-500 ease-out"
            />
          ) : null}
        </svg>

        {/* Center value display */}
        <div
          class={cn(
            'absolute inset-0 flex flex-col items-center justify-end',
            getPaddingBottomClass(size),
          )}
        >
          <span
            class={cn('font-mono font-bold text-text-primary tabular-nums', getFontSizeClass(size))}
          >
            {isRunning && value === 0 ? '—' : displayValue.value}
          </span>
          <span class={cn('caption text-text-muted', spacing.micro.mtNeg)}>
            {displayValue.unit}
          </span>{' '}
          {/* Negative margin for tight visual alignment between value and unit */}
        </div>
      </div>
    </div>
  );
}

// Simple progress ring for general test progress
interface ProgressRingProps {
  progress: number; // 0-100
  size?: number;
  strokeWidth?: number;
  label?: string;
}

export const ProgressRing: React.MemoExoticComponent<typeof ProgressRingComponent> =
  memo(ProgressRingComponent);

function ProgressRingComponent({
  progress,
  size = 48,
  strokeWidth = 4,
  label,
}: ProgressRingProps): React.JSX.Element {
  const ringRadius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * ringRadius;
  const offset = circumference - (progress / 100) * circumference;

  return (
    <div class="flex flex-col items-center">
      <div class="relative">
        <svg width={size} height={size} class="-rotate-90" aria-hidden="true">
          {/* Background circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={ringRadius}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            class="text-surface-border"
          />
          {/* Progress circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={ringRadius}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            class="text-brand-primary transition-all duration-300 ease-out"
          />
        </svg>
        {/* Center percentage */}
        <div class="absolute inset-0 flex items-center justify-center">
          <span class="caption font-medium text-text-primary tabular-nums">
            {Math.round(progress)}%
          </span>
        </div>
      </div>
      {label ? (
        <span class={cn('caption text-text-muted', spacing.margin.top.tight, 'text-center')}>
          {label}
        </span>
      ) : null}
    </div>
  );
}

// Animated pulsing dot for "in progress" indicator
interface PulsingDotProps {
  color?: 'primary' | 'success' | 'warning' | 'error';
  size?: 'sm' | 'md';
}

export const PulsingDot: React.MemoExoticComponent<typeof PulsingDotComponent> =
  memo(PulsingDotComponent);

function PulsingDotComponent({
  color = 'primary',
  size = 'md',
}: PulsingDotProps): React.JSX.Element {
  // Type-safe color class getter
  function getColorClass(c: PulsingDotProps['color']): string {
    switch (c) {
      case 'primary':
        return 'bg-brand-primary';
      case 'success':
        return 'bg-status-success';
      case 'warning':
        return 'bg-status-warning';
      case 'error':
        return 'bg-status-error';
      default:
        return 'bg-brand-primary';
    }
  }

  // Type-safe size class getter
  function getDotSizeClass(s: PulsingDotProps['size']): string {
    return s === 'sm' ? 'w-2 h-2' : 'w-3 h-3';
  }

  const colorClass = getColorClass(color);
  const sizeClass = getDotSizeClass(size);

  return (
    <span class="relative flex">
      <span
        class={cn(
          'animate-ping absolute inline-flex h-full w-full',
          radiusTokens.full,
          'opacity-75',
          colorClass,
        )}
      />
      <span class={cn('relative inline-flex', radiusTokens.full, sizeClass, colorClass)} />
    </span>
  );
}
