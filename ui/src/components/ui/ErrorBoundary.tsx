/**
 * Error Boundary Component (UI Utility)
 *
 * A React Error Boundary class component for catching and handling errors in child component trees.
 * Provides graceful error handling with an optional custom fallback UI or a default error alert.
 *
 * @component
 * @example
 * ```tsx
 * <ErrorBoundary fallback={<CustomErrorUI />}>
 *   <YourComponent />
 * </ErrorBoundary>
 * ```
 *
 * @remarks
 * - Uses React's class component API (getDerivedStateFromError, componentDidCatch)
 * - Logs errors to console for debugging
 * - Provides a "Retry" button to attempt recovery by resetting state
 * - Styled with theme tokens (status-error color, border, radius)
 * - Displays an alert icon and error message when errors are caught
 * - Different from the main ErrorBoundary.tsx - this is the UI utility version
 */

import type { TFunction } from "i18next";
import { Component, type ErrorInfo, type ReactNode } from "react";
import { Translation } from "react-i18next";
import { LogComponents, logger } from "../../lib/logger";
import { button, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";

/**
 * Props for the ErrorBoundary component
 * @interface Props
 */
interface Props {
  /** Child components to be protected by error boundary */
  children: ReactNode;
  /** Optional custom fallback UI to display when error occurs */
  fallback?: ReactNode;
}

/**
 * State for the ErrorBoundary component
 * @interface State
 */
interface State {
  /** Flag indicating if an error has been caught */
  hasError: boolean;
  /** The caught error object, or null if no error */
  error: Error | null;
}

/**
 * Error Boundary Class Component
 *
 * Implements React's Error Boundary pattern to gracefully handle errors in the component tree.
 * - Catches JavaScript errors in child components
 * - Logs error information to console for debugging
 * - Displays error UI with retry functionality
 * - Supports custom fallback UI as alternative to default error display
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    // Initialize state: no error caught, error is null
    this.state = { hasError: false, error: null };
  }

  /**
   * React lifecycle method called when an error is thrown in a child component.
   * Updates component state to indicate error state and stores the error object.
   *
   * @param {Error} error - The error thrown by child component
   * @returns {State} Updated state with error flag and error object
   */
  static getDerivedStateFromError(error: Error): State {
    // Mark that an error has been caught in the component tree
    return { hasError: true, error };
  }

  /**
   * React lifecycle method called after an error has been caught.
   * Used for logging and error reporting to services.
   *
   * @param {Error} error - The error that was caught
   * @param {ErrorInfo} errorInfo - Object with componentStack property containing stack trace
   */
  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log error and component stack for debugging and monitoring
    logger.error(LogComponents.Ui, "ErrorBoundary caught an error", error, {
      componentStack: errorInfo.componentStack,
    });
  }

  /**
   * Handler to reset error state and attempt recovery.
   * Called when user clicks the "Retry" button in error UI.
   */
  handleRetry = (): void => {
    // Clear error state to allow components to re-render normally
    this.setState({ hasError: false, error: null });
  };

  /**
   * Render method that returns either error UI or child components.
   * If error is caught and no custom fallback provided, displays default error alert UI.
   * If custom fallback provided, displays that instead.
   * Otherwise renders child components normally.
   *
   * @returns {ReactNode} Error UI, custom fallback, or child components
   */
  render(): ReactNode {
    // If an error has been caught, display error UI
    if (this.state.hasError) {
      // Use custom fallback if provided by consumer
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Render default error alert UI with icon, message, and retry button
      return (
        <Translation ns="common">
          {(t: TFunction): ReactNode => (
            <div
              role="alert"
              class={cn("pad bg-status-error/10 border border-status-error/20", radius.lg)}
            >
              <div class={cn("flex items-start", spacing.gap.default)}>
                {/* Error icon SVG */}
                <svg
                  class={cn(iconTokens.size.md, "text-status-error shrink-0", spacing.micro.mt)}
                  fill="currentColor"
                  viewBox="0 0 20 20"
                  aria-hidden="true"
                >
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
                {/* Error content: heading, message, and retry button */}
                <div class={cn("flex-1", layout.stack.tight)}>
                  <h3 class="body-small font-medium text-status-error">
                    {t("errorBoundary.title")}
                  </h3>
                  <p class="body-small text-text-muted">
                    {/* Display caught error message or generic fallback message */}
                    {this.state.error?.message || t("errorBoundary.defaultMessage")}
                  </p>
                  {/* Retry button to attempt recovery by resetting error state */}
                  <button
                    type="button"
                    onClick={this.handleRetry}
                    class={cn(
                      button.size.sm,
                      "font-medium text-text-inverse bg-status-error",
                      radius.default,
                      "hover:bg-status-error/90 focus:outline-none focus:ring-2 focus:ring-status-error focus:ring-offset-2",
                    )}
                  >
                    {t("errorBoundary.tryAgain")}
                  </button>
                </div>
              </div>
            </div>
          )}
        </Translation>
      );
    }

    // If no error, render children normally
    return this.props.children;
  }
}
