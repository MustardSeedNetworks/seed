/**
 * Toast Context
 *
 * Provides the React Context for toast notifications.
 * Separated from Toast.tsx for react-refresh compliance.
 */

import { type Context, createContext } from "react";

/**
 * Toast notification type variants
 */
export type ToastType = "success" | "error" | "warning" | "info";

/**
 * Context API value for toast management
 */
export interface ToastContextType {
  /** Add new toast notification */
  addToast: (message: string, type?: ToastType, duration?: number) => void;
  /** Remove specific toast by ID */
  removeToast: (id: string) => void;
}

/** React Context for toast notifications */
export const ToastContext: Context<ToastContextType | null> =
  createContext<ToastContextType | null>(null);
