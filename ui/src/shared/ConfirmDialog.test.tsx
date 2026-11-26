import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ConfirmDialog } from './ConfirmDialog';

afterEach(cleanup);

const dialog = (props: Partial<React.ComponentProps<typeof ConfirmDialog>> = {}) => (
  <ConfirmDialog
    open
    title="Delete device"
    message="This cannot be undone."
    confirmLabel="Delete"
    cancelLabel="Cancel"
    closeLabel="Close"
    onConfirm={() => {}}
    onCancel={() => {}}
    {...props}
  />
);

describe('ConfirmDialog', () => {
  it('states what will happen and offers confirm and cancel', () => {
    render(dialog());
    expect(screen.getByRole('dialog', { name: 'Delete device' })).toBeTruthy();
    expect(screen.getByText('This cannot be undone.')).toBeTruthy();
  });

  it('confirms with the filled destructive button by default, and with primary on request', () => {
    const { rerender } = render(dialog());
    expect(screen.getByRole('button', { name: 'Delete' }).getAttribute('data-variant')).toBe(
      'destructive',
    );
    rerender(dialog({ variant: 'primary' }));
    expect(screen.getByRole('button', { name: 'Delete' }).getAttribute('data-variant')).toBe(
      'primary',
    );
  });

  it('focuses Cancel first, so Enter does not delete', () => {
    render(dialog());
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Cancel' }));
  });

  it('calls onConfirm and onCancel from their buttons, and onCancel from Escape', () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    render(dialog({ onConfirm, onCancel }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(onConfirm).toHaveBeenCalledOnce();
    expect(onCancel).toHaveBeenCalledTimes(2);
  });

  it('while confirming, spins the confirm button and locks everything else', () => {
    const onCancel = vi.fn();
    render(dialog({ confirming: true, onCancel }));
    expect(screen.getByRole('button', { name: 'Delete' }).getAttribute('aria-busy')).toBe('true');
    expect((screen.getByRole('button', { name: 'Cancel' }) as HTMLButtonElement).disabled).toBe(
      true,
    );
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(onCancel).not.toHaveBeenCalled();
  });
});
