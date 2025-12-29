/**
 * StatusBadge Component
 *
 * Purpose: Displays system status with visual indicators (icon or dot) using a centralized
 * status configuration system. Provides consistent color-coding and symbols across the UI
 * for success, warning, error, unknown, and loading states.
 *
 * Key Features:
 * - Variants: "icon" (large symbol) or "dot" (small indicator)
 * - Sizes: "sm" (small), "md" (medium)
 * - Status types: success (green), warning (yellow), error (red), unknown (gray), loading (spinner)
 * - Centralized statusConfig mapping all visual properties for each status
 * - Fully accessible with proper ARIA labels and semantics
 *
 * Usage:
 * ```typescript
 * // Icon variant (large status indicator)
 * <StatusBadge status="success" variant="icon" />
 *
 * // Dot variant (compact indicator)
 * <StatusBadge status="warning" variant="dot" size="sm" />
 *
 * // With custom styling
 * <StatusBadge status="error" variant="icon" className="custom-class" />
 * ```
 *
 * Dependencies: React, theme utilities (cn, icon tokens, radius, layout)
 * State: None - purely presentational component
 */

import { cn, layout, radius } from "../../styles/theme";
import { getSizeConfig, getStatusConfig, type Status } from "./statusConfig";

// Re-export Status type for convenience (types don't affect react-refresh)
export type { Status };

interface StatusBadgeProps {
  status: Status;
  variant?: "icon" | "dot";
  size?: "sm" | "md";
  className?: string;
}

/**
 * StatusBadge - Unified status indicator component
 *
 * Variants:
 * - icon: Shows checkmark/triangle/X icon (default)
 * - dot: Shows small colored dot
 *
 * Usage:
 * <StatusBadge status="success" />           // Icon badge
 * <StatusBadge status="warning" size="sm" /> // Small icon
 * <StatusBadge status="error" variant="dot" /> // Dot indicator
 */
export function StatusBadge({
  status,
  variant = "icon",
  size = "sm",
  className = "",
}: StatusBadgeProps) {
  const config = getStatusConfig(status);
  const sizes = getSizeConfig(size);

  if (variant === "dot") {
    return (
      <span
        className={cn(
          "inline-block",
          sizes.dot,
          radius.full,
          config.bgColor.replace("/10", ""),
          className,
        )}
        aria-label={config.label}
      />
    );
  }

  return (
    <span
      className={cn(
        layout.flex.center,
        "inline-flex",
        radius.full,
        config.color,
        config.bgColor,
        sizes.padding,
        className,
      )}
      aria-label={config.label}
    >
      <span className={sizes.icon}>{config.icon}</span>
    </span>
  );
}

// Re-export Status type for convenience
export type { Status as StatusType };
