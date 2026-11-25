import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Modal, type ModalProps } from './Modal';

afterEach(cleanup);

type Extra = Partial<
  Pick<
    ModalProps,
    | 'open'
    | 'onClose'
    | 'busy'
    | 'closeOnBackdrop'
    | 'closeOnEscape'
    | 'showCloseButton'
    | 'size'
    | 'header'
  >
>;

const modal = (props: Extra = {}, body?: React.ReactNode) => (
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

function OpenerPage() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        opener
      </button>
      <Modal open={open} onClose={() => setOpen(false)} title="T" closeLabel="Close">
        <input aria-label="Name" />
      </Modal>
    </>
  );
}

describe('Modal, covering seed and niac usage', () => {
  it('closeOnBackdrop={false} ignores a scrim click but still closes on Escape and the button', () => {
    const onClose = vi.fn();
    render(modal({ onClose, closeOnBackdrop: false }));
    fireEvent.click(screen.getByTestId('modal-scrim'));
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it('closeOnEscape={false} ignores Escape and does not let it reach the page behind', () => {
    const onClose = vi.fn();
    const behind = vi.fn();
    render(<fieldset onKeyDown={behind}>{modal({ onClose, closeOnEscape: false })}</fieldset>);
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
    expect(behind).not.toHaveBeenCalled();
  });

  it('showCloseButton={false} leaves no close button', () => {
    render(modal({ showCloseButton: false }));
    expect(screen.queryByRole('button', { name: 'Close' })).toBeNull();
  });

  it.each([
    ['sm', 'max-w-sm'],
    ['md', 'max-w-md'],
    ['lg', 'max-w-lg'],
    ['3xl', 'max-w-3xl'],
    ['full', 'max-w-4xl'],
  ] as const)('size %s is %s, always full width up to the cap', (size, cap) => {
    render(modal({ size }));
    const { className } = screen.getByTestId('modal');
    expect(className).toContain(cap);
    expect(className).toContain('w-full');
    expect(className).toContain('max-h-full');
  });

  it('pins the footer outside the scrolling body, and renders header content beside the title', () => {
    render(modal({ header: <span data-testid="step">Step 2 of 3</span> }));
    const footer = screen.getByRole('button', { name: 'Save' }).parentElement;
    expect(footer?.className).toContain('shrink-0');
    expect(footer?.className).toContain('justify-end');
    expect(screen.getByTestId('step').parentElement).toBe(
      screen.getByRole('heading', { name: 'Rename' }).parentElement,
    );
  });

  it('is named by labelledBy or ariaLabel when it has no title', () => {
    const { rerender } = render(
      <Modal open onClose={() => {}} closeLabel="Close" labelledBy="own-heading">
        <h3 id="own-heading">Own heading</h3>
      </Modal>,
    );
    expect(screen.getByRole('dialog', { name: 'Own heading' })).toBeTruthy();
    rerender(
      <Modal open onClose={() => {}} closeLabel="Close" ariaLabel="Preview">
        <p>x</p>
      </Modal>,
    );
    expect(screen.getByRole('dialog', { name: 'Preview' })).toBeTruthy();
    expect(screen.queryByRole('heading')).toBeNull();
  });

  it('has no header bar at all without a title, header or close button', () => {
    render(
      <Modal open onClose={() => {}} closeLabel="Close" ariaLabel="Bare" showCloseButton={false}>
        <p>x</p>
      </Modal>,
    );
    expect(screen.getByRole('dialog').firstElementChild?.className).toContain('overflow-y-auto');
  });

  it('puts focus inside from the commit and hands it back to a click that did not focus its opener', () => {
    render(<OpenerPage />);
    const opener = screen.getByRole('button', { name: 'opener' });
    fireEvent.click(opener);
    expect(screen.getByRole('dialog').contains(document.activeElement)).toBe(true);
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(document.activeElement).toBe(opener);
  });

  it('makes a scrolling body with nothing tabbable focusable, and leaves a tabbable one alone', () => {
    const sizes = vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(900);
    const height = vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(300);
    const { unmount } = render(modal({}, <p>Long read-only text</p>));
    const body = screen.getByText('Long read-only text').parentElement;
    expect(body?.getAttribute('tabindex')).toBe('0');
    expect(body?.getAttribute('aria-label')).toBe('Rename');
    unmount();
    render(modal());
    expect(screen.getByLabelText('Name').parentElement?.hasAttribute('tabindex')).toBe(false);
    sizes.mockRestore();
    height.mockRestore();
  });
});
