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
 * <Fab />
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

import type React from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { cn, icon as iconTokens, layout, radius } from '../../styles/theme';

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
export function Fab({ className = '' }: FabProps): React.JSX.Element {
  const [isRunning, setIsRunning] = useState(false);
  // partial = the previous run finished without every check reporting (a timeout
  // or a card that never completed). Surfaced distinctly so partial results are
  // never presented as a clean completion (the C2 correctness fix).
  const [partial, setPartial] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Settle the run when testsComplete fires: success clears any partial flag;
  // a partial completion records it so the button can warn the operator.
  useEffect(() => {
    const handleTestsComplete = (event: Event): void => {
      const detail = (event as CustomEvent<{ partial?: boolean }>).detail;
      setIsRunning(false);
      setPartial(Boolean(detail?.partial));
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
    };

    window.addEventListener('testsComplete', handleTestsComplete);
    return (): void => {
      window.removeEventListener('testsComplete', handleTestsComplete);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const handleClick = useCallback((): void => {
    if (isRunning) {
      return;
    }
    setIsRunning(true);
    setPartial(false); // a fresh run clears the prior partial warning

    window.dispatchEvent(new CustomEvent('runAllTests'));

    // Backstop: if no testsComplete arrives (e.g. the orchestrator never ran),
    // stop the spinner and mark the run partial — never silently "done".
    timeoutRef.current = setTimeout(() => {
      setIsRunning(false);
      setPartial(true);
    }, 60000);
  }, [isRunning]);

  const runStatus = isRunning ? 'running' : partial ? 'partial' : 'idle';
  const label = partial
    ? 'Some checks did not finish — tap to run all tests again'
    : 'Run All Tests';

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isRunning}
      className={cn(
        'w-14 h-14 bg-brand-primary text-on-brand shadow-lg hover:bg-brand-accent active:scale-95 transition-all touch-manipulation focus:outline-none focus:ring-4 focus:ring-brand-primary/50 focus:ring-offset-2 focus:ring-offset-surface-base',
        layout.flex.center,
        radius.full,
        isRunning && 'opacity-75 cursor-not-allowed',
        className,
      )}
      title={label}
      aria-label={label}
      // aria-busy + data-testid let E2E specs synchronise on the
      // "running" → "idle" transition without racing the animate-spin
      // class on the SVG. See seed#1168 / E2E_CONVENTIONS. data-run-status
      // additionally exposes the partial outcome (idle | running | partial).
      aria-busy={isRunning}
      data-testid="fab-run-all-tests"
      data-running={isRunning ? 'true' : 'false'}
      data-run-status={runStatus}
    >
      {isRunning ? (
        <svg
          className={cn(iconTokens.size.lg, 'animate-spin')}
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
      ) : partial ? (
        // Partial run: a warning triangle distinguishes "some checks did not
        // finish" from a clean completion, so partial results are never read as
        // final (C2). A re-click clears it and starts a fresh run.
        <svg
          className={iconTokens.size.lg}
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" />
          <path d="M12 9v4" />
          <path d="M12 17h.01" />
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
