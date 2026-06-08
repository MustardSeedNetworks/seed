/**
 * SeedLogo — the single canonical Seed brand mark (M3).
 *
 * Replaces the two previously-divergent marks (the topbar's inline seed-glyph
 * SVG and the sidebar's generic `Activity`-in-gradient icon) with one component.
 * The seed glyph is the canonical product mark.
 *
 * - Default (plain): a bare glyph; pass `className` for size + color. The topbar
 *   uses this and tints it by connection status.
 * - `badge`: the glyph inside the brand gradient box (the sidebar mark).
 */

import type { JSX } from 'react';
import { cn } from '../../styles/theme';

// Seed/leaf-in-circle glyph — the canonical brand mark.
const SEED_GLYPH_PATH =
  'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.94-.49-7-3.85-7-7.93s3.05-7.44 7-7.93v15.86zm2-15.86c1.03.13 2 .45 2.87.93H13v-.93zM13 7h5.24c.25.31.48.65.68 1H13V7zm0 3h6.74c.08.33.15.66.19 1H13v-1zm0 9.93V19h2.87c-.87.48-1.84.8-2.87.93zM18.24 17H13v-1h5.92c-.2.35-.43.69-.68 1zm1.5-3H13v-1h6.93c-.04.34-.11.67-.19 1z';

interface SeedLogoProps {
  /** Plain glyph: classes applied directly to the `<svg>` (size + color). */
  className?: string;
  /** Render inside the brand gradient badge (the sidebar mark). */
  badge?: boolean;
  /** Badge box classes (size + shadow). Only used when `badge`. */
  badgeClassName?: string;
  /** Glyph classes inside the badge (size). Only used when `badge`. */
  glyphClassName?: string;
}

export function SeedLogo({
  className,
  badge = false,
  badgeClassName,
  glyphClassName,
}: SeedLogoProps): JSX.Element {
  if (badge) {
    return (
      <div
        className={cn(
          'rounded-lg bg-gradient-to-br from-brand-primary to-brand-accent flex-center',
          badgeClassName,
        )}
      >
        <svg
          viewBox="0 0 24 24"
          fill="currentColor"
          aria-hidden="true"
          className={cn('text-text-inverse', glyphClassName)}
        >
          <path d={SEED_GLYPH_PATH} />
        </svg>
      </div>
    );
  }

  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true" className={className}>
      <path d={SEED_GLYPH_PATH} />
    </svg>
  );
}
