/**
 * =============================================================================
 * INPUT PRESETS - Consolidated Input Styling Patterns
 * =============================================================================
 *
 * This file provides pre-built input className combinations to reduce
 * repetition across settings and form components.
 *
 * USAGE:
 * import { INPUT_PRESETS, inputPreset } from '../styles/input-presets';
 *
 * // Direct usage with cn:
 * <input className={cn(INPUT_PRESETS.compact, "additional-class")} />
 *
 * // Using the helper function:
 * <input className={inputPreset("flexOne")} />
 * <input className={inputPreset("compact", "text-center")} />
 *
 * =============================================================================
 */

import { cn, input, spacing } from "./theme";

/**
 * Base input styling - combines the three most common tokens
 * This is the foundation all presets build upon
 */
const INPUT_BASE: string = cn(input.base, input.state.default, input.size.md);

/**
 * Input presets for common width and layout patterns
 *
 * Patterns identified from codebase analysis:
 * - full: Standard full-width input
 * - fullWithMargin: Full-width with top margin (form fields)
 * - flexOne: Flexible input that expands (host/value fields)
 * - compact: Fixed-width for short values like names (w-24 = 96px)
 * - narrow: Narrower fixed-width for ports (w-20 = 80px)
 * - tiny: Very narrow for counts/numbers (w-14 = 56px)
 * - medium: Medium fixed-width (w-28 = 112px)
 * - wide: Wider fixed-width (w-32 = 128px)
 * - number: Numeric input with centered text
 */
export const INPUT_PRESETS = {
  /** Full-width input - most common for single-field rows */
  full: cn(INPUT_BASE, "w-full"),

  /** Full-width with top margin - for labeled form fields */
  fullWithMargin: cn(INPUT_BASE, "w-full", spacing.margin.top.tight),

  /** Full-width with body-small typography */
  fullSmall: cn(INPUT_BASE, "w-full", spacing.margin.top.tight, "body-small"),

  /** Flexible width (flex-1) - expands to fill available space */
  flexOne: cn(INPUT_BASE, "flex-1"),

  /** Flexible with raised background - for nested/modal inputs */
  flexOneRaised: cn(INPUT_BASE, "flex-1", "bg-surface-raised"),

  /** Compact fixed-width (w-24 = 96px) - for short labels/names */
  compact: cn(INPUT_BASE, "w-24"),

  /** Narrow fixed-width (w-20 = 80px) - for port numbers */
  narrow: cn(INPUT_BASE, "w-20"),

  /** Narrow with raised background */
  narrowRaised: cn(INPUT_BASE, "w-20", "bg-surface-raised"),

  /** Tiny fixed-width (w-14 = 56px) - for counts, small numbers */
  tiny: cn(INPUT_BASE, "w-14"),

  /** Medium fixed-width (w-28 = 112px) */
  medium: cn(INPUT_BASE, "w-28"),

  /** Medium with raised background */
  mediumRaised: cn(INPUT_BASE, "w-28", "bg-surface-raised"),

  /** Wide fixed-width (w-32 = 128px) */
  wide: cn(INPUT_BASE, "w-32"),

  /** Wide with raised background */
  wideRaised: cn(INPUT_BASE, "w-32", "bg-surface-raised"),

  /** Numeric input - tiny with centered text */
  number: cn(INPUT_BASE, "w-14", "text-center"),

  /** Numeric input - narrow with centered text */
  numberWide: cn(INPUT_BASE, "w-20", "text-center"),

  /** Raised background variant - for nested forms/modals */
  raised: cn(INPUT_BASE, "bg-surface-raised"),

  /** Full-width raised variant */
  fullRaised: cn(INPUT_BASE, "w-full", "bg-surface-raised"),
} as const;

/**
 * Type for input preset keys
 */
export type InputPreset = keyof typeof INPUT_PRESETS;

/**
 * Helper function to get an input preset with optional additional classes
 *
 * @param preset - The preset key to use
 * @param additionalClasses - Optional additional Tailwind classes
 * @returns Combined className string
 *
 * @example
 * // Basic usage
 * <input className={inputPreset("compact")} />
 *
 * // With additional classes
 * <input className={inputPreset("number", "text-right")} />
 */
export function inputPreset(
  preset: InputPreset,
  ...additionalClasses: (string | undefined | null | boolean)[]
): string {
  return cn(INPUT_PRESETS[preset], ...additionalClasses);
}

/**
 * Re-export the base tokens for cases where custom combinations are needed
 */
export { INPUT_BASE };
