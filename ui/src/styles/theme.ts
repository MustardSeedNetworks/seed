import { twMerge } from 'tailwind-merge';

/**
 * =============================================================================
 * THE SEED DESIGN SYSTEM - Mustard Seed Networks
 * =============================================================================
 *
 * Centralized design tokens and utilities for consistent UI across the app.
 *
 * ARCHITECTURE:
 * 1. CSS Variables (index.css) - Core color tokens for light/dark modes
 * 2. This file (theme.ts) - Barrel that re-exports the per-domain token
 *    modules + the cn() class-name helper. For component STYLING use the
 *    components in components/ui/* (<Button>, <Card>, <Input>); for ad-hoc
 *    composition use the token objects directly (cn(button.base, ...)).
 * 3. Tailwind Classes - CSS-first configuration using @theme directive
 *
 * Token modules:
 *  - theme_spacing.ts    — margin / padding / gap / stack tokens
 *  - theme_typography.ts — heading / body / size / weight / family / leading
 *  - theme_components.ts — button, input, card, badge, toast, alert, modal, section
 *  - theme_colors.ts     — status, severity, timing, category, moduleColor, brand, gauge
 *  - theme_layout.ts     — sizing, icon, radius, border, layout
 *
 * BRAND COLORS:
 * - Primary: Seed Green (#2d7a3e / #81c784 dark) - Actions, links, focus states
 * - Accent: Lighter Seed Green (#4caf50 / #a5d6a7 dark) - Hover states
 * - Gold: Mustard Gold (#d4a017 / #fbbf24 dark) - Special highlights, premium
 *
 * USAGE:
 * import { spacing, button, cn, moduleColor } from '../styles/theme';
 * <button className={cn(button.base, button.variant.primary)}>Action</button>
 *
 * =============================================================================
 */

// biome-ignore lint/performance/noBarrelFile: theme.ts is the public design-token surface used by ~90 components; per-domain re-exports keep existing call sites working unchanged.
export {
  brand,
  category,
  discoveryMethod,
  gauge,
  moduleColor,
  progressBar,
  severity,
  status,
  timing,
} from './themeColors';
export { alert, badge, button, card, input, modal, section, toast } from './themeComponents';
export { border, icon, layout, radius, sizing } from './themeLayout';
// Re-export domain token modules so existing call sites keep working.
export { spacing } from './themeSpacing';
export { typography } from './themeTypography';

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Combine class names with Tailwind class conflict resolution.
 * Uses tailwind-merge to properly handle conflicting Tailwind classes
 * (e.g., z-50 vs z-20, p-4 vs p-2 will resolve to the last value).
 */
export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return twMerge(classes.filter(Boolean).join(' '));
}
