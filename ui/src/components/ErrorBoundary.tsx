/**
 * Error Boundary Component
 *
 * React Error Boundary that catches JavaScript errors anywhere in the component tree
 * and displays a fallback UI instead of crashing the entire application.
 *
 * Features:
 * - Catches errors in child components during rendering
 * - Logs errors for debugging
 * - Displays user-friendly error message
 * - Provides "Try Again" button to retry rendering
 * - Optional custom fallback UI
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary fallback={<CustomError />}>
 *   <YourComponent />
 * </ErrorBoundary>
 * ```
 *
 * Limitations:
 * - Does NOT catch errors in event handlers (use try/catch)
 * - Does NOT catch errors in async code
 * - Does NOT catch errors during server-side rendering
 * - Does NOT catch errors in the Error Boundary itself
 *
 * The Error Boundary wraps the entire App in main.tsx to provide global crash protection.
 */

import type { TFunction } from "i18next";
import type React from "react";
import { Component, type ReactNode } from "react";
import { Translation } from "react-i18next";
import { LogComponents, logger } from "../lib/logger";
import { button, cn, radius, spacing } from "../styles/theme";

/**
 * Props for ErrorBoundary component
 */
interface Props {
  /** Child components to render and protect */
  children: ReactNode;
  /** Optional custom fallback UI to display on error */
  fallback?: ReactNode;
}

/**
 * Internal state for ErrorBoundary
 */
interface State {
  hasError: boolean; // True if an error has been caught
  error: Error | null; // The caught error object
}

/**
 * ErrorBoundary Class Component
 *
 * React class components with getDerivedStateFromError and componentDidCatch
 * are required for error boundary functionality.
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    // Initialize state with no error
    this.state = { hasError: false, error: null };
  }

  /**
   * Update state when an error is caught.
   * Called after render phase, before commit phase.
   *
   * @param error - The error that was thrown
   * @returns New state to render error UI
   */
  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  /**
   * Log error details for debugging.
   * Called after render phase, allows side effects.
   *
   * @param error - The error that was thrown
   * @param errorInfo - Additional error information (component stack trace)
   */
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    logger.error(LogComponents.App, "ErrorBoundary caught an error", error, {
      componentStack: errorInfo.componentStack,
    });
  }

  /**
   * Attempt to recover from error by retrying render.
   * Clears error state so children are re-rendered.
   */
  handleRetry = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    // If error was caught, display error UI
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return (
        <Translation ns="common">
          {(t: TFunction): JSX.Element => (
            <div
              class={cn(
                "min-h-screen bg-surface-base flex items-center justify-center",
                spacing.pad.default,
              )}
            >
              <div class="w-full max-w-md text-center">
                <div class={cn("text-4xl", spacing.margin.bottom.content)}>
                  <span class="text-status-error">!</span>
                </div>
                <h1 class={cn("heading-2", spacing.margin.bottom.inline)}>
                  {t("errorBoundary.title")}
                </h1>
                <p class={cn("body-small", spacing.margin.bottom.content)}>
                  {this.state.error?.message || t("errorBoundary.defaultMessage")}
                </p>
                <button
                  type="button"
                  onClick={this.handleRetry}
                  class={cn(
                    button.size.md,
                    "bg-brand-primary text-text-inverse",
                    radius.md,
                    "hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary",
                  )}
                >
                  {t("errorBoundary.tryAgain")}
                </button>
              </div>
            </div>
          )}
        </Translation>
      );
    }

    return this.props.children;
  }
}
