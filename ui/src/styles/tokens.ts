/**
 * tokens.ts — runtime-readable design-token VALUES.
 *
 * The single source of truth for token VALUES is index.css (@theme + :root /
 * .dark). This module READS those CSS custom properties at runtime; it never
 * hardcodes hex. Use it ONLY where Tailwind utility classes cannot reach:
 *   - <canvas> drawing (ctx.fillStyle / strokeStyle need a color string)
 *   - generated HTML/PDF reports (a standalone document, no Tailwind classes)
 *
 * Everywhere else, use the utility classes / TS class-token objects in
 * styles/theme. This keeps a single derivation direction (CSS → here), so a
 * token value is defined in exactly one place and can never drift.
 */

/**
 * CSS custom-property names, grouped by tier. The values live in index.css;
 * these are just the addresses. Extend as canvas/report consumers need more.
 */
export const tokenVar = {
  // Brand
  brandPrimary: '--color-brand-primary',
  brandGold: '--color-brand-gold',
  // Text
  textPrimary: '--color-text-primary',
  textMuted: '--color-text-muted',
  textAccent: '--color-text-accent', // seed-600 strong green — AA on light
  textInverse: '--color-text-inverse',
  // Surfaces
  surfaceBase: '--color-surface-base',
  surfaceRaised: '--color-surface-raised',
  surfaceBorder: '--color-surface-border',
  // Status
  statusSuccess: '--color-status-success',
  statusWarning: '--color-status-warning',
  statusError: '--color-status-error',
  statusInfo: '--color-status-info',
  // On-color foregrounds
  onBrand: '--color-on-brand',
  onDanger: '--color-on-danger',
  // Categorical data-viz palette
  cat1: '--color-cat-1',
  cat2: '--color-cat-2',
  cat3: '--color-cat-3',
  cat4: '--color-cat-4',
  cat5: '--color-cat-5',
  cat6: '--color-cat-6',
  cat7: '--color-cat-7',
  cat8: '--color-cat-8',
} as const;

export type TokenName = keyof typeof tokenVar;

/**
 * Read a resolved CSS custom-property value from :root for the active theme.
 * Returns '' when no DOM is available (SSR / tests without a document).
 */
export function cssVar(name: string): string {
  if (typeof document === 'undefined') {
    return '';
  }
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

/**
 * Read a design token's current value.
 * @example token('brandPrimary') // '#4caf50'
 */
export function token(name: TokenName): string {
  return cssVar(tokenVar[name]);
}

/**
 * Batch-read several tokens with a single getComputedStyle call — preferred in
 * canvas render loops where many colors are needed per frame.
 */
export function readTokens<K extends TokenName>(names: readonly K[]): Record<K, string> {
  if (typeof document === 'undefined') {
    return Object.fromEntries(names.map((name) => [name, ''])) as Record<K, string>;
  }
  const styles = getComputedStyle(document.documentElement);
  return Object.fromEntries(
    names.map((name) => [name, styles.getPropertyValue(tokenVar[name]).trim()]),
  ) as Record<K, string>;
}
