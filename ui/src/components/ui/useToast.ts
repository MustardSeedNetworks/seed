/**
 * useToast Hook
 *
 * Hook to access toast notification functions.
 * Must be used within a ToastProvider.
 */

import { useContext } from 'react';
import { ToastContext, type ToastContextType } from './toast-context';

/**
 * Hook to access toast functions in any component.
 *
 * @returns Toast context value with addToast and removeToast
 * @throws Error if used outside ToastProvider
 */
export function useToast(): ToastContextType {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}
