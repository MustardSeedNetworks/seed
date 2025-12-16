# NetScope UI Style Guide

This guide documents the shared design tokens, components, and patterns so new UI matches the
current system.

## Tokens (Tailwind config)

- Fonts: `font-display` and `font-body` map to Inter; `font-mono` to JetBrains Mono.
- Colors: Defined as CSS variables in `src/index.css` and exposed in `tailwind.config.js` under
  `brand`, `surface`, `text`, `status`. Use these instead of raw hex.
- Spacing: Prefer Tailwind spacing scale; primary control spacing uses `p-2.5`, `py-2.5`, `px-2.5`.
  Section gaps: `space-y-3/4/6` depending on depth.
- Max width: wrap main content in `.content-max` or `max-w-7xl` with `px-3 sm:px-4 lg:px-6`.

## Components

- **Card** (`src/components/ui/Card.tsx`): Supports status badges
  (success/warning/error/unknown/loading), keyboard activation, focus rings. Use for dashboard
  tiles.
- **Status badge**: Inline-flex with icon + color + `aria-label`. Use the pattern from Card and
  ConnectionStatus; avoid raw colored dots.
- **Buttons**: Use Tailwind utility combos: primary
  `bg-brand-primary text-text-inverse hover:bg-brand-accent focus:ring-2 focus:ring-brand-primary focus:ring-offset-2`.
  For toggles, use bordered neutral state.
- **Inputs**: Standard padding `px-2.5 py-2`, border `border-surface-border`, background
  `bg-surface-base`, text `text-text-primary`. Labels are `text-xs text-text-muted font-medium`.
  Helper text uses `text-xs text-text-muted mt-1`.
- **Selects/Checkboxes**: Match input padding; checkbox size `w-4 h-4`.
- **Sections**: Labels in settings use
  `text-xs uppercase tracking-wide text-text-muted font-semibold`; subsections separated with
  `border-t` and `pt-3` when stacked.
- **Badges/Chips**: Reuse `inline-flex items-center gap-1 px-2 py-0.5 rounded` with surface/status
  colors; add `aria-label` where conveying status.
- **FAB**: Uses shared focus ring and disables while running; spinner for loading.

## Layout

- Header/Main wrapped in `max-w-7xl` with consistent gutters. Avoid full-width stretching of cards
  on large screens.
- Grid: `grid gap-3 sm:gap-4` with responsive columns; prefer 1/2/3/4 columns breakpoints already
  used in `App.tsx`.

## Accessibility

- All interactive elements: focus-visible ring
  (`focus-visible:ring-2 focus-visible:ring-brand-primary focus-visible:ring-offset-2`).
- Status indicators: include `aria-label` describing state (e.g., "Status: success").
- Inputs associated with labels; use `aria-live` on async status text where appropriate (e.g., save
  indicators).

## Dark/Light

- Use tokenized colors; avoid hard-coded grays. Surfaces: `surface.base/raised/hover/border`; text:
  `text.primary/secondary/muted`.

## Patterns to follow

- Spacing rhythm: inside cards/forms use `space-y-3`; between sections use `space-y-4/6` and
  `border-t pt-3` when stacking.
- Error/success messaging: `text-xs` with status color; avoid inline `alert` boxes unless critical.
- Avoid ad-hoc icons; use Heroicons/Lucide SVGs with size classes (`w-4 h-4`, `w-5 h-5`) and
  currentColor.

## Adding new UI

1. Start from existing components in `src/components/ui/`.
2. Use tokens and utility combos above; no raw colors/pixels unless added as tokens.
3. Add tests for any new UI component (vitest + Testing Library).
4. For new statuses, extend the status badge map and reuse it consistently.

Keeping to these conventions should make future additions consistent without extra design overhead.
