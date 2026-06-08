/**
 * fab.test.tsx - Floating Action Button Tests
 *
 * The FAB triggers a run via the testRunStore and reflects the store's run
 * status (idle / running / partial). These tests pin: clicking starts a run,
 * the button disables while running, a re-click is ignored, the 60s backstop
 * surfaces a partial outcome, and a clean completion settles back to idle.
 *
 * Test Framework: Vitest with React Testing Library and fake timers.
 */

import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useTestRunStore } from '../../stores/testRunStore';
import { Fab } from './fab';

describe('Fab', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useTestRunStore.getState().reset();
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

  it('starts a run when clicked', () => {
    const before = useTestRunStore.getState().startSignal;
    render(<Fab />);

    fireEvent.click(screen.getByRole('button'));

    expect(useTestRunStore.getState().startSignal).toBe(before + 1);
    expect(useTestRunStore.getState().status).toBe('running');
  });

  it('shows spinner and disables while running', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    expect(button).not.toBeDisabled();

    fireEvent.click(button);

    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('data-run-status', 'running');
    expect(button.querySelector('.animate-spin')).toBeInTheDocument();
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

  it('reflects a partial outcome settled on the store (C2)', () => {
    render(<Fab />);

    const button = screen.getByRole('button');
    fireEvent.click(button);
    const { runId } = useTestRunStore.getState();

    act(() => {
      useTestRunStore.getState().settlePartial(runId);
    });

    expect(button).not.toBeDisabled();
    expect(button).toHaveAttribute('data-run-status', 'partial');
  });

  it('settles to idle on a complete run and clears a prior warning', () => {
    render(<Fab />);

    const button = screen.getByRole('button');

    // First run times out -> partial.
    fireEvent.click(button);
    act(() => {
      vi.advanceTimersByTime(60000);
    });
    expect(button).toHaveAttribute('data-run-status', 'partial');

    // A fresh run with no card tests completes cleanly and clears the warning.
    fireEvent.click(button);
    act(() => {
      useTestRunStore.getState().awaitTests([]);
    });
    expect(button).toHaveAttribute('data-run-status', 'idle');
    expect(button).toHaveAttribute('aria-label', 'Run All Tests');
  });

  it('does not start a second run while one is running', () => {
    render(<Fab />);

    const button = screen.getByRole('button');

    fireEvent.click(button);
    const afterFirst = useTestRunStore.getState().startSignal;

    // Second click while running - should not start again.
    fireEvent.click(button);
    expect(useTestRunStore.getState().startSignal).toBe(afterFirst);
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
