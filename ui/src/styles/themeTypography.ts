/**
 * theme_typography.ts — semantic typography tokens. CSS utility classes are
 * defined in index.css; use these TS constants for programmatic styling.
 * Re-exported through theme.ts.
 */

export const typography = {
  // Semantic heading classes (match CSS utilities in index.css)
  // These are the preferred way to style headings
  heading: {
    h1: 'heading-1', // Page titles: 20px/24px bold
    h2: 'heading-2', // Section/modal titles: 20px/24px semibold
    h3: 'heading-3', // Card titles: 18px/20px semibold
    h4: 'heading-4', // Subsections: 16px/18px medium
    section: 'section-title', // Category labels: 12px uppercase muted
  },

  // Body text variants
  body: {
    large: 'body-large', // 18px primary
    default: 'body', // 16px primary (most common)
    small: 'body-small', // 14px secondary
    caption: 'caption', // 12px muted (metadata)
  },

  // Utility classes
  label: 'label', // Form labels: 14px medium
  code: 'code', // Monospace with background

  // Raw size classes (use sparingly - prefer semantic variants above)
  size: {
    xs: 'text-xs', // 12px
    sm: 'text-sm', // 14px
    base: 'text-base', // 16px
    lg: 'text-lg', // 18px
    xl: 'text-xl', // 20px
    '2xl': 'text-2xl', // 24px
    '3xl': 'text-3xl', // 30px
  },

  // Font weights
  weight: {
    normal: 'font-normal', // 400
    medium: 'font-medium', // 500
    semibold: 'font-semibold', // 600
    bold: 'font-bold', // 700
  },

  // Font families
  family: {
    body: 'font-body', // Inter
    display: 'font-display', // Inter (display variant)
    mono: 'font-mono', // JetBrains Mono
  },

  // Line heights
  leading: {
    tight: 'leading-tight', // 1.25 - headings
    snug: 'leading-snug', // 1.375 - subheadings
    normal: 'leading-normal', // 1.5 - default
    relaxed: 'leading-relaxed', // 1.625 - body text
  },
} as const;
