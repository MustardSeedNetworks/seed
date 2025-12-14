/**
 * LuminetIQ Design System
 *
 * Centralized design tokens and utilities for consistent UI
 */

// ============================================================================
// SPACING SCALE
// ============================================================================
// Use Tailwind's spacing scale: 1 unit = 0.25rem (4px)
// Common values:
// - 0.5 = 2px (tight spacing)
// - 1 = 4px (minimal)
// - 2 = 8px (compact)
// - 3 = 12px (default)
// - 4 = 16px (comfortable)
// - 6 = 24px (spacious)
// - 8 = 32px (section separation)
// - 12 = 48px (major sections)

/**
 * Spacing scale - based on 4px grid
 * Use these semantic spacing utilities for consistency.
 * CSS utility classes are defined in index.css (@layer components)
 */
export const spacing = {
  // Raw Tailwind values (for reference)
  values: {
    xs: "1", // 4px
    sm: "2", // 8px
    default: "3", // 12px
    md: "4", // 16px
    lg: "6", // 24px
    xl: "8", // 32px
    "2xl": "12", // 48px
  },

  // Semantic CSS utility classes (preferred - use these)
  stack: {
    xs: "stack-xs", // 4px vertical
    sm: "stack-sm", // 8px vertical
    default: "stack", // 12px vertical
    lg: "stack-lg", // 16px vertical
    xl: "stack-xl", // 24px vertical
  },

  section: {
    default: "section-gap", // 24px between sections
    lg: "section-gap-lg", // 32px for page-level
  },

  gap: {
    tight: "gap-tight", // 4px
    compact: "gap-compact", // 8px
    default: "gap-default", // 12px
    comfortable: "gap-comfortable", // 16px
    spacious: "gap-spacious", // 24px
  },

  inline: {
    xs: "inline-gap-xs", // 4px
    sm: "inline-gap-sm", // 6px
    default: "inline-gap", // 8px
    lg: "inline-gap-lg", // 12px
  },

  pad: {
    sm: "pad-sm", // 12px
    default: "pad", // 16px
    lg: "pad-lg", // 24px
  },
} as const;

// ============================================================================
// TYPOGRAPHY
// ============================================================================
// CSS utility classes are defined in index.css (@layer components)
// Use these TypeScript constants for programmatic styling or documentation

export const typography = {
  // Semantic heading classes (match CSS utilities in index.css)
  // These are the preferred way to style headings
  heading: {
    h1: "heading-1", // Page titles: 24px/30px bold
    h2: "heading-2", // Section/modal titles: 20px/24px semibold
    h3: "heading-3", // Card titles: 18px/20px semibold
    h4: "heading-4", // Subsections: 16px/18px medium
    section: "section-title", // Category labels: 12px uppercase muted
  },

  // Body text variants
  body: {
    large: "body-large", // 18px primary
    default: "body", // 16px primary (most common)
    small: "body-small", // 14px secondary
    caption: "caption", // 12px muted (metadata)
  },

  // Utility classes
  label: "label", // Form labels: 14px medium
  code: "code", // Monospace with background

  // Raw size classes (use sparingly - prefer semantic variants above)
  size: {
    xs: "text-xs", // 12px
    sm: "text-sm", // 14px
    base: "text-base", // 16px
    lg: "text-lg", // 18px
    xl: "text-xl", // 20px
    "2xl": "text-2xl", // 24px
    "3xl": "text-3xl", // 30px
  },

  // Font weights
  weight: {
    normal: "font-normal", // 400
    medium: "font-medium", // 500
    semibold: "font-semibold", // 600
    bold: "font-bold", // 700
  },

  // Font families
  family: {
    body: "font-body", // Inter
    display: "font-display", // Inter (display variant)
    mono: "font-mono", // JetBrains Mono
  },

  // Line heights
  leading: {
    tight: "leading-tight", // 1.25 - headings
    snug: "leading-snug", // 1.375 - subheadings
    normal: "leading-normal", // 1.5 - default
    relaxed: "leading-relaxed", // 1.625 - body text
  },
} as const;

// ============================================================================
// COMPONENT VARIANTS
// ============================================================================

/**
 * Button variants - consistent button styling across the app
 */
export const button = {
  base: "inline-flex items-center justify-center gap-2 rounded font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",

  variant: {
    primary: "bg-brand-primary text-text-inverse hover:bg-brand-accent",
    secondary:
      "border border-surface-border bg-surface-raised hover:bg-surface-hover",
    ghost: "hover:bg-surface-hover",
    danger: "bg-status-error text-text-inverse hover:opacity-90",
    success: "bg-status-success text-text-inverse hover:opacity-90",
  },

  size: {
    xs: "px-2 py-1 text-xs", // Tiny buttons, badges
    sm: "px-3 py-1.5 text-sm", // Small buttons
    md: "px-4 py-2 text-base", // Default buttons
    lg: "px-6 py-3 text-lg", // Large CTAs
  },
} as const;

/**
 * Input variants - consistent form input styling
 */
export const input = {
  base: "w-full rounded border bg-surface-raised text-text-primary transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",

  state: {
    default: "border-surface-border",
    error: "border-status-error",
    success: "border-status-success",
  },

  size: {
    sm: "px-2 py-1.5 text-sm", // Compact inputs
    md: "px-2.5 py-2 text-sm", // Default inputs (most common)
    lg: "px-3 py-2.5 text-base", // Large inputs
  },
} as const;

/**
 * Card variants - consistent card styling
 */
export const card = {
  base: "rounded-lg border bg-surface-raised",

  variant: {
    default: "border-surface-border",
    elevated: "border-surface-border shadow-lg",
    interactive:
      "border-surface-border hover:border-brand-primary cursor-pointer transition-colors",
  },

  padding: {
    none: "",
    sm: "p-3",
    md: "p-4",
    lg: "p-6",
  },
} as const;

/**
 * Badge/Chip variants - for status indicators
 */
export const badge = {
  base: "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium",

  variant: {
    default: "bg-surface-hover text-text-primary",
    success: "bg-status-success/10 text-status-success",
    warning: "bg-status-warning/10 text-status-warning",
    error: "bg-status-error/10 text-status-error",
    info: "bg-status-info/10 text-status-info",
    primary: "bg-brand-primary/10 text-brand-primary",
  },
} as const;

/**
 * Alert/Banner variants - for inline messages
 * Use these for error/warning/info banners instead of hardcoded colors
 */
export const alert = {
  base: "px-4 py-3 rounded border",

  variant: {
    error:
      "bg-status-error/10 border-status-error/20 text-status-error dark:bg-status-error/15 dark:border-status-error/30",
    warning:
      "bg-status-warning/10 border-status-warning/20 text-status-warning dark:bg-status-warning/15 dark:border-status-warning/30",
    success:
      "bg-status-success/10 border-status-success/20 text-status-success dark:bg-status-success/15 dark:border-status-success/30",
    info: "bg-status-info/10 border-status-info/20 text-status-info dark:bg-status-info/15 dark:border-status-info/30",
  },
} as const;

/**
 * Discovery method colors - for network discovery badges
 * Use dark: variants for proper dark mode support
 */
export const discoveryMethod = {
  arp: "bg-blue-500/20 text-blue-600 dark:text-blue-400",
  icmp: "bg-cyan-500/20 text-cyan-600 dark:text-cyan-400",
  lldp: "bg-green-500/20 text-green-600 dark:text-green-400",
  cdp: "bg-orange-500/20 text-orange-600 dark:text-orange-400",
  snmp: "bg-purple-500/20 text-purple-600 dark:text-purple-400",
  edp: "bg-teal-500/20 text-teal-600 dark:text-teal-400",
} as const;

/**
 * Progress bar colors - for timing/performance visualization
 */
export const progressBar = {
  http: "bg-blue-500 dark:bg-blue-400",
  tcp: "bg-amber-500 dark:bg-amber-400",
  success: "bg-green-500 dark:bg-green-400",
} as const;

/**
 * Sizing tokens - for consistent heights/widths
 * Avoids arbitrary values like h-[500px] or w-[28rem]
 */
export const sizing = {
  // Modal/drawer heights (viewport-relative)
  height: {
    modal: "max-h-modal", // 85vh - defined in @theme
    drawer: "h-full", // Full height for side drawers
    panel: "max-h-[70vh]", // Shorter panels
  },

  // Fixed widths for drawers, panels
  width: {
    drawer: "w-80", // 320px - standard drawer
    drawerWide: "w-96", // 384px - wide drawer
    panel: "w-72", // 288px - side panels
    dropdown: "w-64", // 256px - dropdown menus
  },

  // Min/max constraints
  minHeight: {
    card: "min-h-[120px]", // Minimum card height
    section: "min-h-[200px]", // Minimum section height
  },
} as const;

/**
 * Modal/Dialog variants
 */
export const modal = {
  overlay:
    "fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4",
  content:
    "bg-surface-raised border border-surface-border rounded-lg shadow-xl max-h-modal overflow-y-auto",

  size: {
    sm: "max-w-md w-full",
    md: "max-w-2xl w-full",
    lg: "max-w-4xl w-full",
    xl: "max-w-6xl w-full",
    full: "max-w-7xl w-full",
  },

  padding: {
    sm: "p-4",
    md: "p-6",
    lg: "p-8",
  },
} as const;

/**
 * Section/Container variants
 */
export const section = {
  container: "mx-auto px-4",

  width: {
    sm: "max-w-3xl",
    md: "max-w-5xl",
    lg: "max-w-7xl",
    xl: "max-w-8xl",
    full: "max-w-full",
  },

  spacing: {
    tight: "space-y-2",
    default: "space-y-4",
    comfortable: "space-y-6",
    spacious: "space-y-8",
  },
} as const;

/**
 * Status indicator variants - for connection status, health, etc.
 */
export const status = {
  dot: "inline-block w-2 h-2 rounded-full",

  color: {
    success: "bg-status-success",
    warning: "bg-status-warning",
    error: "bg-status-error",
    info: "bg-status-info",
    inactive: "bg-surface-border",
  },

  withLabel: "inline-flex items-center gap-2",
} as const;

/**
 * Severity colors - for CVE/vulnerability ratings (industry standard)
 * Critical = Red, High = Orange, Medium = Yellow, Low = Green
 */
export const severity = {
  critical: {
    bg: "bg-status-error/15",
    text: "text-status-error",
    border: "border-status-error/30",
    dot: "bg-status-error",
  },
  high: {
    bg: "bg-orange-500/15 dark:bg-orange-400/15",
    text: "text-orange-600 dark:text-orange-400",
    border: "border-orange-500/30 dark:border-orange-400/30",
    dot: "bg-orange-500 dark:bg-orange-400",
  },
  medium: {
    bg: "bg-status-warning/15",
    text: "text-status-warning",
    border: "border-status-warning/30",
    dot: "bg-status-warning",
  },
  low: {
    bg: "bg-status-success/15",
    text: "text-status-success",
    border: "border-status-success/30",
    dot: "bg-status-success",
  },
  info: {
    bg: "bg-status-info/15",
    text: "text-status-info",
    border: "border-status-info/30",
    dot: "bg-status-info",
  },
} as const;

/**
 * Timing/phase colors - for HTTP timing bars, performance metrics
 * Following industry conventions for network timing visualization
 */
export const timing = {
  dns: {
    bg: "bg-blue-500 dark:bg-blue-400",
    text: "text-blue-600 dark:text-blue-400",
  },
  tcp: {
    bg: "bg-cyan-500 dark:bg-cyan-400",
    text: "text-cyan-600 dark:text-cyan-400",
  },
  tls: {
    bg: "bg-purple-500 dark:bg-purple-400",
    text: "text-purple-600 dark:text-purple-400",
  },
  wait: {
    bg: "bg-amber-500 dark:bg-amber-400",
    text: "text-amber-600 dark:text-amber-400",
  },
  download: {
    bg: "bg-green-500 dark:bg-green-400",
    text: "text-green-600 dark:text-green-400",
  },
} as const;

/**
 * Category colors - for device types, network segments
 */
export const category = {
  router: "text-blue-500 dark:text-blue-400",
  server: "text-purple-500 dark:text-purple-400",
  workstation: "text-green-500 dark:text-green-400",
  printer: "text-orange-500 dark:text-orange-400",
  mobile: "text-cyan-500 dark:text-cyan-400",
  network: "text-teal-500 dark:text-teal-400",
  unknown: "text-text-muted",
} as const;

/**
 * Gauge colors - for speed gauges, progress indicators
 * Returns CSS variable-compatible color based on percentage
 */
export const gauge = {
  getColor: (percentage: number): string => {
    if (percentage < 25) return "var(--color-status-error)";
    if (percentage < 50) return "var(--color-status-warning)";
    if (percentage < 75) return "var(--gauge-amber, #eab308)";
    return "var(--color-status-success)";
  },
  // Tailwind class equivalents for non-SVG usage
  class: {
    critical: "text-status-error",
    warning: "text-status-warning",
    caution: "text-amber-500 dark:text-amber-400",
    good: "text-status-success",
  },
} as const;

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Combine class names, filtering out falsy values
 */
export function cn(
  ...classes: (string | boolean | undefined | null)[]
): string {
  return classes.filter(Boolean).join(" ");
}

/**
 * Build a button class string
 */
export function buttonClass(
  variant: keyof typeof button.variant = "primary",
  size: keyof typeof button.size = "md",
  className?: string,
): string {
  return cn(button.base, button.variant[variant], button.size[size], className);
}

/**
 * Build an input class string
 */
export function inputClass(
  state: keyof typeof input.state = "default",
  size: keyof typeof input.size = "md",
  className?: string,
): string {
  return cn(input.base, input.state[state], input.size[size], className);
}

/**
 * Build a card class string
 */
export function cardClass(
  variant: keyof typeof card.variant = "default",
  padding: keyof typeof card.padding = "md",
  className?: string,
): string {
  return cn(card.base, card.variant[variant], card.padding[padding], className);
}

/**
 * Build a badge class string
 */
export function badgeClass(
  variant: keyof typeof badge.variant = "default",
  className?: string,
): string {
  return cn(badge.base, badge.variant[variant], className);
}

/**
 * Build a modal class string
 */
export function modalClass(
  size: keyof typeof modal.size = "md",
  padding: keyof typeof modal.padding = "md",
  className?: string,
): string {
  return cn(modal.content, modal.size[size], modal.padding[padding], className);
}
