/**
 * Status Configuration
 *
 * Centralized configuration for status indicators (success, warning, error, etc.).
 * Used by StatusBadge, Card, and other components that need consistent status styling.
 */

import type { ReactNode } from "react";
import { icon as iconTokens, spacing } from "../../styles/theme";

export type Status = "success" | "warning" | "error" | "unknown" | "loading";

// Centralized status configuration - icons, colors, and labels
export const statusConfig: Record<
  Status,
  { icon: ReactNode; color: string; bgColor: string; label: string }
> = {
  success: {
    icon: (
      <svg
        className="w-full h-full"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-10.707a1 1 0 00-1.414-1.414L9 9.172 7.707 7.879a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: "text-status-success",
    bgColor: "bg-status-success/10",
    label: "Status: success",
  },
  warning: {
    icon: (
      <svg
        className="w-full h-full"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M8.257 3.099c.765-1.36 2.72-1.36 3.485 0l6.518 11.6c.75 1.334-.214 3.001-1.742 3.001H3.48c-1.528 0-2.492-1.667-1.742-3.001l6.52-11.6zM11 14a1 1 0 11-2 0 1 1 0 012 0zm-1-2a1 1 0 01-1-1V8a1 1 0 112 0v3a1 1 0 01-1 1z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: "text-status-warning",
    bgColor: "bg-status-warning/10",
    label: "Status: warning",
  },
  error: {
    icon: (
      <svg
        className="w-full h-full"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1.293-5.293a1 1 0 011.414 0L10 12.586l.879-.879a1 1 0 111.414 1.414L11.414 14l.879.879a1 1 0 01-1.414 1.414L10 15.414l-.879.879a1 1 0 11-1.414-1.414L8.586 14l-.879-.879a1 1 0 010-1.414z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: "text-status-error",
    bgColor: "bg-status-error/10",
    label: "Status: error",
  },
  unknown: {
    icon: (
      <svg
        className="w-full h-full"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
        aria-hidden="true"
      >
        <path d="M9 7a1 1 0 012 0c0 1.5-2 1.5-2 3h2c0-1.5 2-1.5 2-3a3 3 0 10-6 0h2z" />
        <circle cx="10" cy="14" r="1" />
      </svg>
    ),
    color: "text-text-muted",
    bgColor: "bg-surface-hover",
    label: "Status: unknown",
  },
  loading: {
    icon: (
      <svg
        className="w-full h-full animate-spin"
        viewBox="0 0 20 20"
        fill="none"
        aria-hidden="true"
      >
        <circle
          className="opacity-25"
          cx="10"
          cy="10"
          r="8"
          stroke="currentColor"
          strokeWidth="3"
        />
        <path className="opacity-75" fill="currentColor" d="M18 10a8 8 0 00-8-8v4a4 4 0 014 4h4z" />
      </svg>
    ),
    color: "text-status-info",
    bgColor: "bg-status-info/10",
    label: "Status: loading",
  },
};

// Size configurations using design tokens
const sizeConfig = {
  sm: {
    icon: iconTokens.size.sm, // w-4 h-4
    dot: "w-2 h-2",
    padding: spacing.badge.xs, // 2px - extra small badge padding
  },
  md: {
    icon: iconTokens.size.md, // w-5 h-5
    dot: "w-2.5 h-2.5",
    padding: spacing.badge.sm, // 4px - small badge padding
  },
} as const;

type SizeKey = keyof typeof sizeConfig;

/**
 * Type-safe getter for status configuration.
 * @param status - The status type to get configuration for
 * @returns The status configuration object with icon, color, bgColor, and label
 */
export function getStatusConfig(status: Status) {
  switch (status) {
    case "success":
      return statusConfig.success;
    case "warning":
      return statusConfig.warning;
    case "error":
      return statusConfig.error;
    case "unknown":
      return statusConfig.unknown;
    case "loading":
      return statusConfig.loading;
  }
}

/**
 * Type-safe getter for size configuration.
 * @param size - The size key ("sm" or "md")
 * @returns The size configuration object with icon, dot, and padding values
 */
export function getSizeConfig(size: SizeKey) {
  if (size === "sm") return sizeConfig.sm;
  return sizeConfig.md;
}
