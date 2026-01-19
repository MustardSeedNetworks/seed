/**
 * Icon Configuration
 *
 * Size presets and utilities for consistent icon sizing.
 */

// Icon size presets for consistency
export const ICON_SIZES = {
  xs: 'w-3 h-3',
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
  xl: 'w-8 h-8',
} as const;

export type IconSize = keyof typeof ICON_SIZES;
