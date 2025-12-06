import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { FAB } from './FAB';

describe('FAB', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders the FAB button', () => {
    render(<FAB />);

    const button = screen.getByRole('button');
    expect(button).toBeInTheDocument();
    expect(button).toHaveAttribute('aria-label', 'Run All Tests');
  });

  it('dispatches runAllTests event when clicked', () => {
    const dispatchEventSpy = vi.spyOn(window, 'dispatchEvent');

    render(<FAB />);

    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(dispatchEventSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'runAllTests',
      })
    );

    dispatchEventSpy.mockRestore();
  });

  it('shows spinner when running', () => {
    render(<FAB />);

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
    render(<FAB />);

    const button = screen.getByRole('button');
    fireEvent.click(button);

    expect(button).toBeDisabled();

    // Fast forward fallback timeout (60s)
    act(() => {
      vi.advanceTimersByTime(60000);
    });

    expect(button).not.toBeDisabled();
  });

  it('does not dispatch multiple events while running', () => {
    const dispatchEventSpy = vi.spyOn(window, 'dispatchEvent');

    render(<FAB />);

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
    render(<FAB />);

    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('title', 'Run All Tests');
    expect(button).toHaveAttribute('aria-label', 'Run All Tests');
  });

  it('renders with correct styling', () => {
    const { container } = render(<FAB />);

    const fabButton = container.firstChild;
    expect(fabButton).toHaveClass('fixed');
    expect(fabButton).toHaveClass('rounded-full');
  });

  it('accepts custom className', () => {
    render(<FAB className="custom-class" />);

    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });
});
