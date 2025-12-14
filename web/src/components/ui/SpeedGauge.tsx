import { memo, useMemo } from "react";
import { gauge } from "../../styles/theme";

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
        <p className="text-xs text-text-muted mb-1 uppercase tracking-wider">
          {label}
        </p>
      )}
      <div className="relative">
        <svg
          width={sizeConfig.width}
          height={sizeConfig.height}
          viewBox={`0 0 ${sizeConfig.width} ${sizeConfig.height}`}
          className={isRunning ? "animate-pulse" : ""}
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
            const tickStartX =
              centerX + innerRadius * Math.cos(toRadians(tickAngle));
            const tickStartY =
              centerY + innerRadius * Math.sin(toRadians(tickAngle));
            const tickEndX =
              centerX + outerRadius * Math.cos(toRadians(tickAngle));
            const tickEndY =
              centerY + outerRadius * Math.sin(toRadians(tickAngle));
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
          className="absolute inset-0 flex flex-col items-center justify-end pb-0"
          style={{ paddingBottom: size === "sm" ? "2px" : "4px" }}
        >
          <span
            className="font-mono font-bold text-text-primary tabular-nums"
            style={{ fontSize: sizeConfig.fontSize }}
          >
            {isRunning && value === 0 ? "—" : displayValue.value}
          </span>
          <span className="text-xs text-text-muted -mt-1">
            {displayValue.unit}
          </span>
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
        <svg width={size} height={size} className="-rotate-90">
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
          <span className="text-xs font-medium text-text-primary tabular-nums">
            {Math.round(progress)}%
          </span>
        </div>
      </div>
      {label && (
        <span className="text-xs text-text-muted mt-1 text-center">
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
  const colorClasses = {
    primary: "bg-brand-primary",
    success: "bg-status-success",
    warning: "bg-status-warning",
    error: "bg-status-error",
  };

  const sizeClasses = {
    sm: "w-2 h-2",
    md: "w-3 h-3",
  };

  return (
    <span className="relative flex">
      <span
        className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${colorClasses[color]}`}
      />
      <span
        className={`relative inline-flex rounded-full ${sizeClasses[size]} ${colorClasses[color]}`}
      />
    </span>
  );
});
