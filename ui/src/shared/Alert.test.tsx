import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Alert, type AlertTone } from './Alert';

afterEach(cleanup);

describe('Alert', () => {
  it.each<[AlertTone, string]>([
    ['error', 'alert'],
    ['warning', 'alert'],
    ['success', 'status'],
    ['info', 'status'],
  ])('%s announces as %s', (tone, role) => {
    render(<Alert tone={tone}>Message</Alert>);
    expect(screen.getByRole(role).textContent).toContain('Message');
  });

  it('takes its wash from the status hue and its icon from the -strong token', () => {
    render(<Alert tone="error">Message</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert.className).toContain('bg-status-error/10');
    expect(alert.querySelector('svg')?.getAttribute('class')).toContain('text-status-error-strong');
    expect(alert.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
  });

  it('shows a title when given', () => {
    render(
      <Alert tone="info" title="Heads up">
        Message
      </Alert>,
    );
    expect(screen.getByText('Heads up')).toBeTruthy();
  });

  it('dismisses only when asked to', () => {
    const onDismiss = vi.fn();
    const { rerender } = render(<Alert tone="info">Message</Alert>);
    expect(screen.queryByRole('button')).toBeNull();
    rerender(
      <Alert tone="info" onDismiss={onDismiss} dismissLabel="Dismiss">
        Message
      </Alert>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it('wraps long text instead of widening the page', () => {
    render(<Alert tone="info">Message</Alert>);
    expect(screen.getByText('Message').parentElement?.className).toContain('min-w-0');
  });
});
