/**
 * Toast.constants.tsx - Toast Notification Styling Constants
 *
 * Purpose: Centralized styling configuration for toast notifications.
 * Defines colors, icons, and CSS classes for different toast types (success, error, warning, info).
 *
 * Key Features:
 * - Type styles: CSS classes for each toast type with appropriate background and text colors
 * - Toast icons: SVG icon definitions for success, error, warning, and info notifications
 * - Icon size: Consistent sizing using theme icon tokens
 * - Color mapping: Maps toast types to semantic colors (success/error/warning/brand)
 *
 * Usage:
 * ```typescript
 * import { typeStyles, icons, ToastType } from './Toast.constants';
 *
 * const className = typeStyles['success'];
 * const icon = icons['success'];
 * ```
 *
 * Dependencies: theme icon tokens
 * Exported Types: ToastType ('success' | 'error' | 'warning' | 'info')
 */

import { icon } from "../../styles/theme";

export type ToastType = "success" | "error" | "warning" | "info";

export const typeStyles = {
  success: "bg-status-success text-text-inverse",
  error: "bg-status-error text-text-inverse",
  warning: "bg-status-warning text-text-inverse",
  info: "bg-brand-primary text-text-inverse",
};

// Icon size for toast notifications
const toastIconSize = icon.size.md; // w-5 h-5

export const icons = {
  success: (
    <svg className={toastIconSize} fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
        clipRule="evenodd"
      />
    </svg>
  ),
  error: (
    <svg className={toastIconSize} fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
        clipRule="evenodd"
      />
    </svg>
  ),
  warning: (
    <svg className={toastIconSize} fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
        clipRule="evenodd"
      />
    </svg>
  ),
  info: (
    <svg className={toastIconSize} fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
        clipRule="evenodd"
      />
    </svg>
  ),
};
