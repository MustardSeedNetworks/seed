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

import { Component, ReactNode } from "react";
import { radius } from "../styles/theme";

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
    console.error("ErrorBoundary caught an error:", error, errorInfo);
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
        <div className="min-h-screen bg-surface-base flex items-center justify-center p-4">
          <div className="w-full max-w-md text-center">
            <div className="text-4xl mb-4">
              <span className="text-status-error">!</span>
            </div>
            <h1 className="heading-2 mb-2">Something went wrong</h1>
            <p className="body-small mb-4">
              {this.state.error?.message || "An unexpected error occurred"}
            </p>
            <button
              onClick={this.handleRetry}
              className={`px-4 py-2 bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary`}
            >
              Try Again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
