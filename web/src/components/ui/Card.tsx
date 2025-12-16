/**
 * Card Component
 *
 * Base card container used throughout the application for displaying information.
 *
 * Features:
 * - Status badge with color coding (success, warning, error, loading, unknown)
 * - Header with title, optional subtitle, icon, and custom actions
 * - Keyboard accessibility (Enter/Space to activate)
 * - Optional click handler for interactive cards
 * - Responsive padding (mobile and desktop)
 * - Smooth transitions and hover effects
 * - Focus ring for keyboard navigation
 *
 * Usage:
 * ```tsx
 * <Card
 *   title="Link Status"
 *   subtitle="eth0"
 *   status="success"
 *   icon={<CableIcon />}
 *   onClick={() => handleCardClick()}
 * >
 *   <CardRow label="Speed" value="1000 Mbps" />
 *   <CardDivider />
 *   <CardRow label="Duplex" value="Full" />
 * </Card>
 * ```
 */

import { ReactNode } from "react";
import { StatusBadge } from "./StatusBadge";
import { Status, getStatusConfig } from "./statusConfig";
import { cn, card, layout, icon as iconTokens, spacing } from "../../styles/theme";

// Re-export Status type for convenience (types don't affect react-refresh)
export type { Status };

// Type-safe size class getter
function getSizeClass(size: "sm" | "md" | "lg") {
  switch (size) {
    case "sm":
      return "body-small";
    case "md":
      return "body font-medium leading-snug";
    case "lg":
      return "body-large font-semibold leading-snug";
  }
}

/**
 * Props for the Card component
 */
interface CardProps {
  /** Card title (required) */
  title: string;
  /** Optional subtitle or description */
  subtitle?: string;
  /** Status indicator color (success, warning, error, loading, unknown) */
  status: Status;
  /** Card content */
  children: ReactNode;
  /** Additional CSS classes */
  className?: string;
  /** Callback when card is clicked */
  onClick?: () => void;
  /** Optional icon displayed in header */
  icon?: ReactNode;
  /** Optional action element in header (right side) */
  headerAction?: ReactNode;
}

/**
 * Card component - displays information in a styled container with status badge.
 */
export function Card({
  title,
  subtitle,
  status,
  children,
  className = "",
  onClick,
  icon,
  headerAction,
}: CardProps) {
  // Card is interactive if click handler is provided
  const isInteractive = typeof onClick === "function";

  /**
   * Handle keyboard activation (Enter/Space) for interactive cards.
   * Provides accessibility for keyboard navigation.
   */
  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (!isInteractive) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onClick?.();
    }
  };

  const interactiveProps = isInteractive ? { role: "button" as const, tabIndex: 0 } : {};

  return (
    <div
      className={cn(
        card.base,
        card.variant.default,
        `${spacing.pad.sm} sm:${spacing.pad.default} transition-all hover:border-brand-primary/40 touch-manipulation focus-visible:ring-2 focus-visible:ring-brand-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface-base outline-none`,
        isInteractive && "cursor-pointer active:scale-[0.98]",
        className
      )}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      {...interactiveProps}
    >
      <div className={layout.flex.between}>
        <div className={layout.inline.default}>
          {icon && (
            <span className={cn("text-text-muted shrink-0", iconTokens.size.md)}>{icon}</span>
          )}
          <div className={layout.flex.col}>
            <h3 className="heading-4 font-display">{title}</h3>
            {subtitle && <p className="caption leading-tight">{subtitle}</p>}
          </div>
        </div>
        <div className={layout.inline.default}>
          {headerAction}
          <StatusBadge status={status} size="md" />
        </div>
      </div>
      <div className="mt-2 sm:mt-3">{children}</div>
    </div>
  );
}

interface CardValueProps {
  label?: string;
  value: string | number;
  unit?: string;
  size?: "sm" | "md" | "lg";
  status?: Status;
  mono?: boolean;
  allowWrap?: boolean;
}

/**
 * Displays a prominent value with optional label, unit, and status indicator.
 */
export function CardValue({
  label,
  value,
  unit,
  size = "md",
  status,
  mono = false,
  allowWrap = false,
}: CardValueProps) {
  const statusColorClass = status ? getStatusConfig(status).color : "text-text-primary";
  const textMods = [
    statusColorClass,
    mono ? "font-mono tabular-nums" : "",
    allowWrap ? "break-all whitespace-pre-wrap" : "",
  ]
    .filter(Boolean)
    .join(" ");

  const statusIcon = status ? getStatusConfig(status).icon : null;

  return (
    <div>
      {label && <p className="caption mb-1">{label}</p>}
      <p className={cn(getSizeClass(size), textMods, layout.inline.tight)} data-testid="card-value">
        {statusIcon && (
          <span className={cn(layout.flex.center, iconTokens.size.xs, "shrink-0 text-current")}>
            {statusIcon}
          </span>
        )}
        <span className={cn(layout.inline.tight, "items-baseline")}>
          <span>{value}</span>
          {unit && <span className="body-small font-normal text-text-muted">{unit}</span>}
        </span>
      </p>
    </div>
  );
}

interface CardRowProps {
  label: string;
  value: string | number;
  status?: Status;
  wrap?: boolean;
  mono?: boolean;
  align?: "left" | "right";
}

/**
 * Displays a label-value pair in a horizontal row with optional status indicator.
 */
export function CardRow({
  label,
  value,
  status,
  wrap = false,
  mono = false,
  align = "right",
}: CardRowProps) {
  const resolvedStatus = status ? getStatusConfig(status) : null;
  const statusIcon = resolvedStatus?.icon ?? null;
  const justifyClass = align === "right" ? "justify-end" : "justify-start";

  return (
    <div
      className={cn(
        "flex justify-between py-1",
        layout.inline.default,
        wrap ? "items-start" : "items-center"
      )}
    >
      <span className="body-small shrink-0">{label}</span>
      <span
        className={cn(
          "body-small font-medium",
          layout.inline.tight,
          justifyClass,
          align === "right" ? "text-right" : "text-left",
          wrap ? "break-all whitespace-pre-wrap" : "truncate",
          mono && "font-mono tabular-nums",
          resolvedStatus?.color ?? "text-text-primary"
        )}
        title={String(value)}
        data-testid="card-row-value"
      >
        {statusIcon && (
          <span className={cn(iconTokens.size.xs, "shrink-0 text-current")}>{statusIcon}</span>
        )}
        <span>{value}</span>
      </span>
    </div>
  );
}

interface CardDividerProps {
  className?: string;
}

/**
 * Horizontal divider line for separating card sections.
 */
export function CardDivider({ className = "" }: CardDividerProps) {
  return <hr className={cn("border-surface-border my-3", className)} />;
}
