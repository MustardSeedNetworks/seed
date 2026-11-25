import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RightDrawer } from './RightDrawer';

afterEach(cleanup);

function Page({ open, onClose = () => {} }: { open: boolean; onClose?: () => void }) {
  return (
    <>
      <button type="button">opener</button>
      <RightDrawer
        open={open}
        onClose={onClose}
        title="Help"
        closeLabel="Close"
        footer={<button type="button">Done</button>}
      >
        <input aria-label="Search" />
      </RightDrawer>
    </>
  );
}

describe('RightDrawer', () => {
  it('renders nothing while closed', () => {
    render(<Page open={false} />);
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('is a modal dialog named by its title', () => {
    render(<Page open />);
    const dialog = screen.getByRole('dialog', { name: 'Help' });
    expect(dialog.getAttribute('aria-modal')).toBe('true');
  });

  it('moves focus into the drawer on open', () => {
    render(<Page open />);
    expect(screen.getByRole('dialog').contains(document.activeElement)).toBe(true);
  });

  it('closes on Escape, on the close button and on the scrim', () => {
    const onClose = vi.fn();
    render(<Page open onClose={onClose} />);
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    fireEvent.click(screen.getByTestId('right-drawer-scrim'));
    expect(onClose).toHaveBeenCalledTimes(3);
  });

  it('keeps Tab inside: last wraps to first, and Shift+Tab on first wraps to last', () => {
    render(<Page open />);
    const dialog = screen.getByRole('dialog');
    const close = screen.getByRole('button', { name: 'Close' });
    const done = screen.getByRole('button', { name: 'Done' });

    done.focus();
    fireEvent.keyDown(dialog, { key: 'Tab' });
    expect(document.activeElement).toBe(close);

    close.focus();
    fireEvent.keyDown(dialog, { key: 'Tab', shiftKey: true });
    expect(document.activeElement).toBe(done);
  });

  it('returns focus to the opener on close and releases the page scroll lock', () => {
    const { rerender } = render(<Page open={false} />);
    const opener = screen.getByRole('button', { name: 'opener' });
    opener.focus();
    rerender(<Page open />);
    expect(document.body.style.overflow).toBe('hidden');
    rerender(<Page open={false} />);
    expect(document.activeElement).toBe(opener);
    expect(document.body.style.overflow).toBe('');
  });

  it('is the full screen width at 390px and a fixed column from sm up', () => {
    render(<Page open />);
    const { className } = screen.getByRole('dialog');
    expect(className).toContain('w-full');
    expect(className).toContain('max-w-full');
    expect(className).toContain('sm:w-[32.5rem]');
  });

  it('scrolls its body while the header and footer stay put', () => {
    render(<Page open />);
    expect(screen.getByLabelText('Search').parentElement?.className).toContain('overflow-y-auto');
    expect(screen.getByRole('button', { name: 'Done' }).parentElement?.className).toContain(
      'shrink-0',
    );
  });
});
