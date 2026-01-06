/**
 * Floating Action Button (FAB) Component
 *
 * Fixed-position button in the bottom-right corner for triggering quick actions.
 *
 * Features:
 * - Fixed positioning (bottom-right corner)
 * - Loading spinner animation while tests are running
 * - Dispatches 'runAllTests' custom event
 * - Fallback 60-second timeout if event never completes
 * - Disabled state during test execution
 * - Keyboard accessible with focus ring
 * - Touch-friendly sizing (56x56 pixels)
 *
 * Usage:
 * ```tsx
 * // In app layout:
 * <FAB />
 *
 * // Listen for test completion:
 * window.addEventListener('testsComplete', () => {
 *   // Handle completion
 * });
 * ```
 *
 * The FAB is rendered at the root App level and provides quick access
 * to running all network diagnostics without opening settings.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { cn, icon as iconTokens, layout, radius } from "../../styles/theme";

/**
 * Props for FAB component
 */
interface FabProps {
  /** Additional CSS classes */
  className?: string;
}

/**
 * Floating Action Button - triggers all diagnostic tests
 */
export function FAB({ className = "" }: FabProps) {
  const [isRunning, setIsRunning] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Stop spinner when testsComplete fires
  useEffect(() => {
    const handleTestsComplete = () => {
      setIsRunning(false);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
    };

    window.addEventListener("testsComplete", handleTestsComplete);
    return () => {
      window.removeEventListener("testsComplete", handleTestsComplete);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const handleClick = useCallback(() => {
    if (isRunning) return;
    setIsRunning(true);

    window.dispatchEvent(new CustomEvent("runAllTests"));

    // Fallback timeout in case testsComplete never fires
    timeoutRef.current = setTimeout(() => {
      setIsRunning(false);
    }, 60000);
  }, [isRunning]);

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isRunning}
      className={cn(
        "w-14 h-14 bg-brand-primary text-text-inverse shadow-lg hover:bg-brand-accent active:scale-95 transition-all touch-manipulation focus:outline-none focus:ring-4 focus:ring-brand-primary/50 focus:ring-offset-2 focus:ring-offset-surface-base",
        layout.flex.center,
        radius.full,
        isRunning && "opacity-75 cursor-not-allowed",
        className,
      )}
      title="Run All Tests"
      aria-label="Run All Tests"
    >
      {isRunning ? (
        <svg
          className={cn(iconTokens.size.lg, "animate-spin")}
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
      ) : (
        <svg
          className={iconTokens.size.lg}
          fill="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path d="M8 5v14l11-7z" />
        </svg>
      )}
    </button>
  );
}
