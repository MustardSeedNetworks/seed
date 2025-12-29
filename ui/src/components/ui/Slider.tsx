/**
 * Slider Component
 *
 * A customizable range slider for numeric input with visual feedback and labels.
 * Designed for configuring timing, performance, and threshold settings.
 *
 * Features:
 * - Visual slider with draggable thumb
 * - Shows current value with optional custom formatter
 * - Supports min, max, step constraints
 * - Optional end labels (e.g., "Slower ◄────► Faster")
 * - Filled track shows visual progress
 * - Keyboard accessible (arrow keys, Page Up/Down, Home/End)
 * - Touch-friendly for mobile devices
 * - Disabled state support
 *
 * Usage:
 * ```tsx
 * // Simple slider
 * <Slider
 *   value={500}
 *   onChange={(val) => setInterval(val)}
 *   min={100}
 *   max={1000}
 *   step={100}
 *   label="Scan Timeout"
 *   formatValue={(v) => `${v}ms`}
 * />
 *
 * // Slider with end labels
 * <Slider
 *   value={20}
 *   onChange={(val) => setWorkers(val)}
 *   min={5}
 *   max={100}
 *   step={5}
 *   label="Worker Threads"
 *   leftLabel="Slower"
 *   rightLabel="Faster"
 * />
 * ```
 */

import type React from "react";
import { memo, useCallback, useRef, useState } from "react";
import { cn, layout, radius, spacing } from "../../styles/theme";

interface SliderProps {
  /** Current slider value */
  value: number;
  /** Callback when value changes */
  onChange: (value: number) => void;
  /** Minimum value */
  min: number;
  /** Maximum value */
  max: number;
  /** Step increment */
  step: number;
  /** Label displayed above slider */
  label?: string;
  /** Label displayed at left end (e.g., "Slower") */
  leftLabel?: string;
  /** Label displayed at right end (e.g., "Faster") */
  rightLabel?: string;
  /** Custom formatter for value display (e.g., v => `${v}ms`) */
  formatValue?: (value: number) => string;
  /** Disable slider interaction */
  disabled?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Range slider component for numeric input with visual feedback.
 * Supports keyboard navigation and custom value formatting.
 */
export const Slider = memo(function Slider({
  value,
  onChange,
  min,
  max,
  step,
  label,
  leftLabel,
  rightLabel,
  formatValue,
  disabled = false,
  className,
}: SliderProps) {
  const sliderRef = useRef<HTMLInputElement>(null);
  const [isDragging, setIsDragging] = useState(false);

  // Calculate fill percentage for visual track
  const percentage = ((value - min) / (max - min)) * 100;

  // Default formatter shows raw value
  const displayValue = formatValue ? formatValue(value) : String(value);

  /**
   * Handle input change from slider interaction
   */
  const handleChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = Number(event.target.value);
      onChange(newValue);
    },
    [onChange],
  );

  /**
   * Handle keyboard shortcuts for quick adjustments
   * - Arrow keys: increment/decrement by step
   * - Page Up/Down: increment/decrement by 10 steps
   * - Home/End: jump to min/max
   */
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLInputElement>) => {
      let newValue = value;
      const largeStep = step * 10;

      switch (event.key) {
        case "PageUp":
          event.preventDefault();
          newValue = Math.min(max, value + largeStep);
          break;
        case "PageDown":
          event.preventDefault();
          newValue = Math.max(min, value - largeStep);
          break;
        case "Home":
          event.preventDefault();
          newValue = min;
          break;
        case "End":
          event.preventDefault();
          newValue = max;
          break;
        default:
          return; // Let browser handle arrow keys
      }

      onChange(newValue);
    },
    [value, min, max, step, onChange],
  );

  return (
    <div className={cn("w-full", className)}>
      {/* Label and current value */}
      {label && (
        <div className={cn(layout.flex.between, spacing.margin.bottom.tight)}>
          <label
            htmlFor={`slider-${label.replace(/\s+/g, "-").toLowerCase()}`}
            className="label text-text-primary"
          >
            {label}
          </label>
          <span
            className="body-small font-medium text-brand-primary font-mono tabular-nums"
            aria-live="polite"
          >
            {displayValue}
          </span>
        </div>
      )}

      {/* Slider container with track and thumb */}
      <div className={spacing.margin.bottom.inline}>
        <div className="relative">
          {/* Background track */}
          <div
            className={cn(
              "absolute inset-0 h-2 top-1/2 -translate-y-1/2",
              radius.full,
              "bg-surface-hover",
              disabled && "opacity-50",
            )}
          />

          {/* Filled portion (progress) */}
          <div
            className={cn(
              "absolute h-2 top-1/2 -translate-y-1/2",
              radius.full,
              "bg-brand-primary transition-all",
              disabled && "opacity-50",
            )}
            style={{
              width: `${percentage}%`,
            }}
          />

          {/* Native range input (styled to show only thumb) */}
          <input
            ref={sliderRef}
            id={label ? `slider-${label.replace(/\s+/g, "-").toLowerCase()}` : undefined}
            type="range"
            min={min}
            max={max}
            step={step}
            value={value}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            onMouseDown={() => setIsDragging(true)}
            onMouseUp={() => setIsDragging(false)}
            onTouchStart={() => setIsDragging(true)}
            onTouchEnd={() => setIsDragging(false)}
            disabled={disabled}
            className={cn(
              "relative w-full h-2 appearance-none bg-transparent cursor-pointer",
              "focus:outline-none",
              // Thumb styling - webkit browsers
              "[&::-webkit-slider-thumb]:appearance-none",
              "[&::-webkit-slider-thumb]:w-5",
              "[&::-webkit-slider-thumb]:h-5",
              "[&::-webkit-slider-thumb]:rounded-full",
              "[&::-webkit-slider-thumb]:bg-brand-primary",
              "[&::-webkit-slider-thumb]:border-2",
              "[&::-webkit-slider-thumb]:border-surface-base",
              "[&::-webkit-slider-thumb]:shadow-md",
              "[&::-webkit-slider-thumb]:transition-all",
              "[&::-webkit-slider-thumb]:cursor-pointer",
              "[&::-webkit-slider-thumb]:hover:scale-110",
              "[&::-webkit-slider-thumb]:active:scale-125",
              // Thumb styling - Firefox
              "[&::-moz-range-thumb]:appearance-none",
              "[&::-moz-range-thumb]:w-5",
              "[&::-moz-range-thumb]:h-5",
              "[&::-moz-range-thumb]:rounded-full",
              "[&::-moz-range-thumb]:bg-brand-primary",
              "[&::-moz-range-thumb]:border-2",
              "[&::-moz-range-thumb]:border-surface-base",
              "[&::-moz-range-thumb]:shadow-md",
              "[&::-moz-range-thumb]:transition-all",
              "[&::-moz-range-thumb]:cursor-pointer",
              "[&::-moz-range-thumb]:hover:scale-110",
              "[&::-moz-range-thumb]:active:scale-125",
              "[&::-moz-range-thumb]:border-none",
              // Focus ring
              "focus-visible:ring-2",
              "focus-visible:ring-brand-primary",
              "focus-visible:ring-offset-2",
              "focus-visible:ring-offset-surface-base",
              radius.full,
              // Disabled state
              disabled && "cursor-not-allowed",
              disabled && "[&::-webkit-slider-thumb]:cursor-not-allowed",
              disabled && "[&::-moz-range-thumb]:cursor-not-allowed",
              disabled && "[&::-webkit-slider-thumb]:bg-text-muted",
              disabled && "[&::-moz-range-thumb]:bg-text-muted",
              disabled && "[&::-webkit-slider-thumb]:hover:scale-100",
              disabled && "[&::-moz-range-thumb]:hover:scale-100",
              // Active state visual feedback
              isDragging && "[&::-webkit-slider-thumb]:scale-125",
              isDragging && "[&::-moz-range-thumb]:scale-125",
            )}
            aria-label={label || "Slider"}
            aria-valuemin={min}
            aria-valuemax={max}
            aria-valuenow={value}
            aria-valuetext={displayValue}
          />
        </div>
      </div>

      {/* End labels (e.g., "Slower ◄────► Faster") */}
      {(leftLabel || rightLabel) && (
        <div className={cn(layout.flex.between, "caption text-text-muted")}>
          <span>{leftLabel || ""}</span>
          <span>{rightLabel || ""}</span>
        </div>
      )}
    </div>
  );
});
