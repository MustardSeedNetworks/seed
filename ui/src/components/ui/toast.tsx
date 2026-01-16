/**
 * Toast Notification System
 *
 * Provides non-modal notifications for user feedback (success, error, warning, info).
 *
 * Features:
 * - Multiple toast types with color coding (success, error, warning, info)
 * - Auto-dismiss with configurable duration (default 5s)
 * - React Context API for global access
 * - Toast queue for multiple simultaneous notifications
 * - Smooth animations (fade in/out)
 * - Accessible with ARIA labels
 * - Custom hooks for easy integration
 *
 * Usage:
 * ```tsx
 * // Wrap app with provider:
 * <ToastProvider>
 *   <App />
 * </ToastProvider>
 *
 * // Use in components:
 * const { addToast } = useToast();
 * addToast('Operation completed', 'success', 3000);
 * addToast('An error occurred', 'error');
 * ```
 *
 * Toast notifications appear in the bottom-right corner and automatically
 * dismiss after the specified duration.
 */

import type React from "react";
import { type ReactNode, useCallback, useEffect, useState } from "react";
import {
  cn,
  icon as iconTokens,
  layout,
  radius,
  spacing,
  toast as toastTokens,
} from "../../styles/theme";
import { icons, typeStyles } from "./Toast.constants.tsx";
import { ToastContext, type ToastType } from "./toast-context";

/**
 * Individual toast notification
 */
interface Toast {
  id: string; // Unique identifier
  message: string; // Notification message text
  type: ToastType; // Type (success, error, warning, info)
  duration?: number; // Auto-dismiss time in ms (0 = manual dismiss)
}

/**
 * Props for ToastProvider component
 */
interface ToastProviderProps {
  /** Child components to wrap */
  children: ReactNode;
}

/**
 * Toast Provider - wraps app to provide toast notifications globally
 */
export function ToastProvider({ children }: ToastProviderProps): React.JSX.Element {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback((message: string, type: ToastType = "info", duration = 5000) => {
    const id = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    setToasts((prev) => [...prev, { id, message, type, duration }]);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ addToast, removeToast }}>
      {children}
      <toastContainer toasts={toasts} removeToast={removeToast} />
    </ToastContext.Provider>
  );
}

interface ToastContainerProps {
  toasts: Toast[];
  removeToast: (id: string) => void;
}

function _toastContainer({ toasts, removeToast }: ToastContainerProps): React.JSX.Element {
  const handleClose = (id: string): void => removeToast(id);
  return (
    // biome-ignore lint/a11y/useSemanticElements: Region role with aria-live is the correct pattern for toast notifications
    <div
      role="region"
      aria-live="polite"
      aria-label="Notifications"
      class={cn("fixed bottom-20 right-4 z-50 max-w-sm", layout.stack.default)}
    >
      {toasts.map((toast) => (
        <toastItem key={toast.id} toast={toast} onClose={handleClose.bind(null, toast.id)} />
      ))}
    </div>
  );
}

interface ToastItemProps {
  toast: Toast;
  onClose: () => void;
}

function _toastItem({ toast, onClose }: ToastItemProps): React.JSX.Element {
  useEffect((): undefined | (() => void) => {
    if (toast.duration && toast.duration > 0) {
      const timer = setTimeout(onClose, toast.duration);
      return (): void => {
        clearTimeout(timer);
      };
    }
  }, [toast.duration, onClose]);

  return (
    <div
      role="alert"
      class={cn(
        layout.inline.comfortable,
        toastTokens.container,
        toastTokens.animation,
        radius.lg,
        typeStyles[toast.type],
      )}
      aria-label={`Notification: ${toast.type}`}
    >
      {icons[toast.type]}
      <p class="body-small font-medium flex-1">{toast.message}</p>
      <button
        type="button"
        onClick={onClose}
        class={cn(
          spacing.iconBtn.sm,
          "hover:bg-surface-hover/50 focus:outline-none focus:ring-2 focus:ring-surface-border",
          radius.default,
        )}
        aria-label="Dismiss notification"
      >
        <svg class={iconTokens.size.sm} fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
          <path
            fillRule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
    </div>
  );
}
