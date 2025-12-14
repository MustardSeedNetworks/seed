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

export const spacing = {
  tight: "0.5", // 2px
  compact: "2", // 8px
  default: "3", // 12px
  comfortable: "4", // 16px
  spacious: "6", // 24px
  section: "8", // 32px
  major: "12", // 48px
} as const;

// ============================================================================
// TYPOGRAPHY
// ============================================================================

export const typography = {
  // Font sizes
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
    normal: "font-normal",
    medium: "font-medium",
    semibold: "font-semibold",
    bold: "font-bold",
  },
  // Font families
  family: {
    body: "font-body",
    display: "font-display",
    mono: "font-mono",
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
    sm: "px-3 py-1.5 text-sm",
    md: "px-4 py-2 text-base",
    lg: "px-6 py-3 text-lg",
  },
} as const;

/**
 * Input variants - consistent form input styling
 */
export const input = {
  base: "w-full rounded border bg-surface-raised px-3 py-2 text-text-primary transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed",

  state: {
    default: "border-surface-border",
    error: "border-status-error",
    success: "border-status-success",
  },

  size: {
    sm: "px-2 py-1 text-sm",
    md: "px-3 py-2 text-base",
    lg: "px-4 py-3 text-lg",
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
