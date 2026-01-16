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

import type React from "react";
import { type ReactNode, useState } from "react";
import { border, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";
import type { Status } from "./Card";
import { StatusBadge } from "./StatusBadge";

interface CollapsibleSectionProps {
  title: ReactNode;
  defaultOpen?: boolean;
  children: ReactNode;
  /** Number of items to display in header, e.g., "Server Results (2)" */
  count?: number;
  /** Status indicator to show next to title */
  status?: Status;
  /** Use compact styling for inside cards */
  variant?: "default" | "compact";
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
  variant = "default",
}: CollapsibleSectionProps): React.JSX.Element {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  const isCompact = variant === "compact";

  return (
    <section
      class={cn(
        !isCompact && border.card,
        !isCompact && radius.lg,
        !isCompact && "overflow-hidden",
      )}
    >
      <button
        type="button"
        onClick={(): void => setIsOpen(!isOpen)}
        class={cn(
          "w-full transition-colors",
          layout.flex.between,
          isCompact
            ? cn(spacing.chip.md, "hover:bg-surface-hover/50", radius.default)
            : cn(spacing.pad.sm, "bg-surface-base hover:bg-surface-hover"),
        )}
      >
        <div class={layout.inline.default}>
          <svg
            class={cn(
              iconTokens.size.xs,
              "text-text-muted transition-transform duration-200",
              isOpen && "rotate-90",
            )}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          {status ? <StatusBadge status={status} size="sm" /> : null}
          <span class={cn("font-medium text-text-primary", isCompact ? "caption" : "body-small")}>
            {title}
            {count !== undefined ? (
              <span class={cn("text-text-muted", spacing.margin.left.tight)}>({count})</span>
            ) : null}
          </span>
        </div>
      </button>
      {isOpen ? (
        <div
          class={cn(
            isCompact
              ? cn(spacing.indent, spacing.padding.bottom.inline, "stack-xs")
              : cn(spacing.pad.sm, "border-t border-surface-border bg-surface-raised stack"),
          )}
        >
          {children}
        </div>
      ) : null}
    </section>
  );
}
