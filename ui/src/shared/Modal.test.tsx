import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Modal } from './Modal';

afterEach(cleanup);

const modal = (props: Partial<React.ComponentProps<typeof Modal>> = {}, body?: React.ReactNode) => (
  <Modal
    open
    onClose={() => {}}
    title="Rename"
    closeLabel="Close"
    footer={<button type="button">Save</button>}
    {...props}
  >
    {body ?? <input aria-label="Name" />}
  </Modal>
);

describe('Modal', () => {
  it('renders nothing closed, and a named modal dialog open', () => {
    const { rerender } = render(modal({ open: false }));
    expect(screen.queryByRole('dialog')).toBeNull();
    rerender(modal());
    expect(screen.getByRole('dialog', { name: 'Rename' }).getAttribute('aria-modal')).toBe('true');
  });

  it('puts focus inside, wraps Tab, and closes on Escape and the scrim', () => {
    const onClose = vi.fn();
    render(modal({ onClose }));
    const dialog = screen.getByRole('dialog');
    expect(dialog.contains(document.activeElement)).toBe(true);
    screen.getByRole('button', { name: 'Save' }).focus();
    fireEvent.keyDown(dialog, { key: 'Tab' });
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Close' }));
    fireEvent.keyDown(dialog, { key: 'Escape' });
    fireEvent.click(screen.getByTestId('modal-scrim'));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it('focuses the element marked data-autofocus first', () => {
    render(
      modal(
        {},
        <button type="button" data-autofocus="">
          Safe
        </button>,
      ),
    );
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Safe' }));
  });

  it('cannot be dismissed while busy', () => {
    const onClose = vi.fn();
    render(modal({ onClose, busy: true }));
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    fireEvent.click(screen.getByTestId('modal-scrim'));
    expect((screen.getByRole('button', { name: 'Close' }) as HTMLButtonElement).disabled).toBe(
      true,
    );
    expect(onClose).not.toHaveBeenCalled();
  });

  it('fits a 390px viewport: gutter, full width up to a cap, scrolling body', () => {
    render(modal());
    expect(screen.getByTestId('modal').className).toContain('max-w-lg');
    expect(screen.getByTestId('modal').className).toContain('w-full');
    expect(screen.getByTestId('modal').parentElement?.className).toContain('p-4');
    expect(screen.getByLabelText('Name').parentElement?.className).toContain('overflow-y-auto');
  });
});
