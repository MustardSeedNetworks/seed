/**
 * CollapsibleSection Component
 *
 * Purpose: Collapsible/accordion section for organizing content within cards and modals.
 * Allows hiding/showing detailed information to reduce visual clutter.
 *
 * Key Features:
 * - Two variants: "default" (standalone with border) and "compact" (inside cards)
 * - Toggle control: click header to expand/collapse with smooth animation
 * - Status indicators: optional status badge next to title
 * - Item count: displays "(count)" next to title
 * - Customizable title: can be string or React node for complex headers
 * - Default open: optional defaultOpen prop to start expanded
 * - Semantic HTML: uses <section> and <button> for accessibility
 * - Keyboard support: button can be activated with Enter/Space
 *
 * Usage:
 * ```typescript
 * // Default variant (with border)
 * <CollapsibleSection title="Advanced Options" defaultOpen={false}>
 *   <p>Hidden by default, click to expand</p>
 * </CollapsibleSection>
 *
 * // Compact variant (inside card)
 * <CollapsibleSection
 *   title="Server Results"
 *   count={3}
 *   status="success"
 *   variant="compact"
 * >
 *   <div>Results here</div>
 * </CollapsibleSection>
 * ```
 *
 * Dependencies: React hooks, theme utilities, StatusBadge
 * State: Manages isOpen state with useState
 */

import type React from 'react';
import { type ReactNode, useState } from 'react';
import { border, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import type { Status } from './card';
import { StatusBadge } from './StatusBadge';

interface CollapsibleSectionProps {
  title: ReactNode;
  defaultOpen?: boolean;
  children: ReactNode;
  /** Number of items to display in header, e.g., "Server Results (2)" */
  count?: number;
  /** Status indicator to show next to title */
  status?: Status;
  /** Use compact styling for inside cards */
  variant?: 'default' | 'compact';
  /**
   * Why this section is read-only, or `undefined` when it is writable (#1254).
   *
   * When set, the body renders inside a disabled `<fieldset>` and the reason
   * appears in the header. The header button stays outside that fieldset, so a
   * viewer can still open the section and read it — the failure of both the
   * `RequireRole` wrapper this replaces (which showed a viewer nothing) and of
   * wrapping the whole section (which would lock the section shut).
   *
   * Only for a section whose every control writes. A section that also carries
   * a read action — a Refresh, a download — gates its controls individually on
   * `useRole().canWrite` instead, the way ApiTokensSettings does, so the read
   * stays available.
   */
  readOnlyReason?: string;
  /**
   * Stable test selector for E2E. Landed on the root `<section>` so the
   * section is queryable whether collapsed or expanded — call sites that
   * previously relied on `getByText(/title-substring/i)` were i18n-unsafe
   * and matched the wrong node when a sibling tab or breadcrumb shared the
   * substring (settings.spec.ts:41-58 was the canonical victim).
   */
  'data-testid'?: string;
}

/**
 * Expandable section with animated collapse/expand toggle and optional count badge.
 */
export function CollapsibleSection({
  title,
  defaultOpen = false,
  children,
  count,
  status,
  variant = 'default',
  readOnlyReason,
  'data-testid': dataTestId,
}: CollapsibleSectionProps): React.JSX.Element {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  const isCompact = variant === 'compact';
  const bodyClass = cn(
    isCompact
      ? cn(spacing.indent, spacing.padding.bottom.inline, 'stack-xs')
      : cn(spacing.pad.sm, 'border-t border-surface-border bg-surface-raised stack'),
  );

  return (
    <section
      data-testid={dataTestId}
      className={cn(
        !isCompact && border.card,
        !isCompact && radius.lg,
        !isCompact && 'overflow-hidden',
      )}
    >
      <button
        type="button"
        onClick={(): void => setIsOpen(!isOpen)}
        className={cn(
          'w-full transition-colors',
          layout.flex.between,
          isCompact
            ? cn(spacing.chip.md, 'hover:bg-surface-hover/50', radius.default)
            : cn(spacing.pad.sm, 'bg-surface-base hover:bg-surface-hover'),
        )}
      >
        <div className={layout.inline.default}>
          <svg
            className={cn(
              iconTokens.size.xs,
              'text-text-muted transition-transform duration-200',
              isOpen && 'rotate-90',
            )}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          {status ? <StatusBadge status={status} size="sm" /> : null}
          <span
            className={cn('font-medium text-text-primary', isCompact ? 'caption' : 'body-small')}
          >
            {title}
            {count !== undefined ? (
              <span className={cn('text-text-muted', spacing.margin.left.tight)}>({count})</span>
            ) : null}
          </span>
        </div>
        {readOnlyReason !== undefined ? (
          <span className={cn('caption text-text-muted', spacing.margin.left.tight)}>
            {readOnlyReason}
          </span>
        ) : null}
      </button>
      {isOpen && readOnlyReason === undefined ? <div className={bodyClass}>{children}</div> : null}
      {isOpen && readOnlyReason !== undefined ? (
        // min-w-0 undoes the UA `min-inline-size: min-content` on fieldset,
        // which otherwise makes a long child overflow its flex parent. Nothing
        // else is needed: Tailwind preflight already zeroes the UA border,
        // margin and padding, and `border-0` here would defeat the body's own
        // `border-t` through tailwind-merge.
        <fieldset disabled className={cn(bodyClass, 'min-w-0')}>
          {children}
        </fieldset>
      ) : null}
    </section>
  );
}
