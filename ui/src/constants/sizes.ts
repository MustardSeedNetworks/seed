/**
 * Icon and text size tokens shared by sidebar/breadcrumbs/layout.
 * Mirrors the niac UI primitives so seed/stem/niac stay aligned.
 */
export const iconSizes = {
  xs: 'h-3 w-3',
  sm: 'h-3.5 w-3.5',
  md: 'h-4 w-4',
  lg: 'h-5 w-5',
  xl: 'h-6 w-6',
  '2xl': 'h-8 w-8',
  '3xl': 'h-12 w-12',
} as const;

export type IconSize = keyof typeof iconSizes;
