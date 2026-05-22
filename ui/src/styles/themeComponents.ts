/**
 * theme_components.ts — variant tokens for the core interactive components:
 * button, input, card, badge, toast, alert, modal, and section. Re-exported
 * through theme.ts.
 */

/**
 * Button variants - consistent button styling across the app
 */
export const button = {
  base: 'inline-flex items-center justify-center gap-2 rounded font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed',

  variant: {
    // Seed anchor is seed-500 (#4caf50) — needs DARK text (white fails AA).
    // text-zinc-900 is constant across modes (text-text-inverse flips and
    // would fail one of them). Opacity hover avoids the lighten-to-accent
    // trap. Per Phase 7 of the 2026-05-22 brand audit.
    primary: 'bg-brand-primary text-zinc-900 hover:bg-brand-primary/90',
    secondary: 'border border-surface-border bg-surface-raised hover:bg-surface-hover',
    ghost: 'hover:bg-surface-hover',
    // Status danger/success buttons also fail AA with text-inverse in some
    // mode; use a constant.
    danger: 'bg-status-error text-white hover:bg-status-error/90',
    success: 'bg-status-success text-zinc-900 hover:bg-status-success/90',
  },

  size: {
    xs: 'px-2 py-1 text-xs', // Tiny buttons, badges
    sm: 'px-3 py-1.5 text-sm', // Small buttons
    md: 'px-4 py-2 text-base', // Default buttons
    lg: 'px-6 py-3 text-lg', // Large CTAs
  },
} as const;

/**
 * Input variants - consistent form input styling
 */
export const input = {
  base: 'w-full rounded border bg-surface-raised text-text-primary transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary disabled:opacity-50 disabled:cursor-not-allowed',

  state: {
    default: 'border-surface-border',
    error: 'border-status-error',
    success: 'border-status-success',
  },

  size: {
    sm: 'px-2 py-1.5 text-sm', // Compact inputs
    md: 'px-2.5 py-2 text-sm', // Default inputs (most common)
    lg: 'px-3 py-2.5 text-base', // Large inputs
  },
} as const;

/**
 * Card variants - consistent card styling
 */
export const card = {
  base: 'rounded-lg border bg-surface-raised',

  variant: {
    default: 'border-surface-border',
    elevated: 'border-surface-border shadow-lg',
    interactive:
      'border-surface-border hover:border-brand-primary cursor-pointer transition-colors',
  },

  padding: {
    none: '',
    sm: 'p-3',
    md: 'p-4',
    lg: 'p-6',
  },
} as const;

/**
 * Badge/Chip variants - for status indicators
 */
export const badge = {
  base: 'inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium',

  variant: {
    default: 'bg-surface-hover text-text-primary',
    success: 'bg-status-success/10 text-status-success',
    warning: 'bg-status-warning/10 text-status-warning',
    error: 'bg-status-error/10 text-status-error',
    info: 'bg-status-info/10 text-status-info',
    primary: 'bg-brand-primary/10 text-brand-primary',
  },
} as const;

/**
 * Toast/Notification variants - for non-modal notifications
 */
export const toast = {
  container: 'px-4 py-3 shadow-lg',
  animation: 'animate-slide-in',
} as const;

/**
 * Alert/Banner variants - for inline messages
 * Use these for error/warning/info banners instead of hardcoded colors
 */
export const alert = {
  base: 'px-4 py-3 rounded-lg border',

  variant: {
    error:
      'bg-status-error/10 border-status-error/20 text-status-error dark:bg-status-error/15 dark:border-status-error/30',
    warning:
      'bg-status-warning/10 border-status-warning/20 text-status-warning dark:bg-status-warning/15 dark:border-status-warning/30',
    success:
      'bg-status-success/10 border-status-success/20 text-status-success dark:bg-status-success/15 dark:border-status-success/30',
    info: 'bg-status-info/10 border-status-info/20 text-status-info dark:bg-status-info/15 dark:border-status-info/30',
  },
} as const;

/**
 * Modal/Dialog variants
 */
export const modal = {
  overlay: 'fixed inset-0 z-50 flex items-center justify-center p-4',
  backdrop: 'absolute inset-0 bg-black/50 backdrop-blur-sm',
  content:
    'bg-surface-raised border border-surface-border rounded-lg shadow-xl max-h-modal overflow-y-auto',

  size: {
    sm: 'max-w-md w-full',
    md: 'max-w-2xl w-full',
    lg: 'max-w-4xl w-full',
    xl: 'max-w-6xl w-full',
    full: 'max-w-7xl w-full',
  },

  padding: {
    sm: 'pad', // 16px - uses semantic token
    md: 'pad-lg', // 24px - uses semantic token
    lg: 'pad-xl', // 32px - uses semantic token
  },
} as const;

/**
 * Section/Container variants
 */
export const section = {
  container: 'mx-auto px-4',

  width: {
    sm: 'max-w-3xl',
    md: 'max-w-5xl',
    lg: 'max-w-7xl',
    xl: 'max-w-8xl',
    full: 'max-w-full',
  },

  spacing: {
    tight: 'space-y-2',
    default: 'space-y-4',
    comfortable: 'space-y-6',
    spacious: 'space-y-8',
  },
} as const;
