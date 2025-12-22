/**
 * Tooltip Component
 *
 * Purpose: Displays contextual help text on hover or focus. Provides accessible tooltips
 * for explaining UI elements with customizable position (top/bottom).
 *
 * Key Features:
 * - Position control: top (default) or bottom positioning
 * - Hover and focus triggers: shows on mouseEnter or focus events
 * - Accessible: uses role="tooltip" for screen readers
 * - Max-width constraint: prevents long text from wrapping excessively
 * - Smooth positioning: uses CSS transforms for centering
 * - Theme-aware: uses surface-raised background and text-primary color
 * - Z-layer management: uses z-50 to appear above other content
 *
 * Usage:
 * ```typescript
 * <Tooltip content="Click here to start scanning">
 *   <button>Start</button>
 * </Tooltip>
 *
 * <Tooltip content="CPU usage %" position="bottom">
 *   <div>{cpuPercent}%</div>
 * </Tooltip>
 * ```
 *
 * Dependencies: React, theme utilities (cn, radius, border)
 * State: Manages show/hide state on hover and focus
 */

import { ReactNode, useState } from "react";
import { cn, radius, border, spacing } from "../../styles/theme";

interface TooltipProps {
  content: string;
  children: ReactNode;
  position?: "top" | "bottom";
}

/**
 * Hover-triggered tooltip that displays additional information for an element.
 */
export function Tooltip({ content, children, position = "top" }: TooltipProps) {
  const [show, setShow] = useState(false);

  const positionClasses =
    position === "top"
      ? cn(
          "bottom-full left-1/2 -translate-x-1/2",
          spacing.margin.bottom.inline
        )
      : cn("top-full left-1/2 -translate-x-1/2", spacing.margin.top.inline);

  return (
    <div className="relative inline-flex items-center">
      <div
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onFocus={() => setShow(true)}
        onBlur={() => setShow(false)}
        className="cursor-help"
      >
        {children}
      </div>
      {show && (
        <div
          className={cn(
            "absolute z-50 shadow-lg max-w-xs",
            spacing.cell.px,
            spacing.compact.pyMd,
            positionClasses,
            radius.default,
            border.card,
            "bg-surface-raised text-text-primary caption"
          )}
          role="tooltip"
        >
          {content}
        </div>
      )}
    </div>
  );
}
