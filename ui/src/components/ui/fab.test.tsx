/**
 * FAB.test.tsx - Floating Action Button Tests
 *
 * Purpose: Test suite for the FAB (Floating Action Button) component testing
 * button rendering, event dispatching, loading state, and timeout handling.
 *
 * Key Test Areas:
 * - Rendering: FAB button displays with correct aria-label
 * - Event dispatch: clicking button dispatches runAllTests event
 * - Custom event: verifies CustomEvent structure and detail
 * - Loading state: shows spinner while tests are running
 * - Test completion: listens for testsComplete event
 * - Timeout handling: 60-second timeout clears loading state
 * - Visual feedback: spinner visible during test execution
 *
 * Test Framework: Vitest with React Testing Library and fake timers
 *
 * Usage:
 * ```bash
 * npm test -- FAB.test.tsx
 * ```
 *
 * Dependencies: vitest, @testing-library/react
 */

import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Fab } from './fab';

describe('Fab', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders the FAB button', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    expect(button).toBeInTheDocument();
    expect(button).toHaveAttribute('aria-label', 'Run All Tests');
  });

  it('dispatches runAllTests event when clicked', () => {
    const dispatchEventSpy = vi.spyOn(window, 'dispatchEvent');

    render(<Fab />);

    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(dispatchEventSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'runAllTests',
      }),
    );

    dispatchEventSpy.mockRestore();
  });

  it('shows spinner when running', () => {
    render(<Fab />);

    const button = screen.getByRole('button');

    // Initially not running
    expect(button).not.toBeDisabled();

    // Click to start running
    fireEvent.click(button);

    // Should be disabled while running
    expect(button).toBeDisabled();

    // Should show spinner (has animate-spin class)
    const spinner = button.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  it('resets after timeout', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(button).toBeDisabled();

    // Fast forward fallback timeout (60s)
    act(() => {
      vi.advanceTimersByTime(60000);
    });

    expect(button).not.toBeDisabled();
  });

  it('marks the run partial when the backstop timeout fires (no completion)', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    fireEvent.click(button);
    expect(button).toHaveAttribute('data-run-status', 'running');

    act(() => {
      vi.advanceTimersByTime(60000);
    });

    // Timed out without a completion signal: partial, never a clean done.
    expect(button).not.toBeDisabled();
    expect(button).toHaveAttribute('data-run-status', 'partial');
    expect(button).toHaveAttribute(
      'aria-label',
      'Some checks did not finish — tap to run all tests again',
    );
  });

  it('marks the run partial when testsComplete reports partial: true (C2)', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    fireEvent.click(button);
    act(() => {
      window.dispatchEvent(new CustomEvent('testsComplete', { detail: { partial: true } }));
    });

    expect(button).not.toBeDisabled();
    expect(button).toHaveAttribute('data-run-status', 'partial');
  });

  it('settles to idle on a complete (partial: false) run and clears a prior warning', () => {
    render(<Fab />);

    const button = screen.getByRole('button');

    // First run times out -> partial.
    fireEvent.click(button);
    act(() => {
      vi.advanceTimersByTime(60000);
    });
    expect(button).toHaveAttribute('data-run-status', 'partial');

    // A fresh run that completes cleanly clears the partial warning.
    fireEvent.click(button);
    act(() => {
      window.dispatchEvent(new CustomEvent('testsComplete', { detail: { partial: false } }));
    });
    expect(button).toHaveAttribute('data-run-status', 'idle');
    expect(button).toHaveAttribute('aria-label', 'Run All Tests');
  });

  it('does not dispatch multiple events while running', () => {
    const dispatchEventSpy = vi.spyOn(window, 'dispatchEvent');

    render(<Fab />);

    const button = screen.getByRole('button');

    // First click
    fireEvent.click(button);
    expect(dispatchEventSpy).toHaveBeenCalledTimes(1);

    // Second click while running - should not dispatch
    fireEvent.click(button);
    expect(dispatchEventSpy).toHaveBeenCalledTimes(1);

    dispatchEventSpy.mockRestore();
  });

  it('has correct accessibility attributes', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('title', 'Run All Tests');
    expect(button).toHaveAttribute('aria-label', 'Run All Tests');
  });

  it('renders with correct styling', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    // FAB uses radius.full for rounded corners and has shadow
    expect(button).toHaveClass('rounded-full');
    expect(button).toHaveClass('shadow-lg');
  });

  it('accepts custom className', () => {
    render(<Fab className="custom-class" />);

    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });
});
