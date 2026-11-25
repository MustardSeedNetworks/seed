import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { type ReactNode, type RefObject, useLayoutEffect, useRef, useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { tabbables, useDialog } from './useDialog';

afterEach(cleanup);

interface HostProps {
  children?: ReactNode;
  returnFocusRef?: RefObject<HTMLElement | null>;
  onSeen?: (active: Element | null) => void;
}

/** A trigger and a dialog. fireEvent.click does not focus the button, as in WebKit. */
function Host({ children, returnFocusRef, onSeen }: HostProps) {
  const [open, setOpen] = useState(false);
  const dialog = useDialog(open, () => setOpen(false), returnFocusRef ? { returnFocusRef } : {});
  // Declared after useDialog, so it runs after the dialog's own layout effect and
  // before any passive effect: what it sees is where focus is at commit.
  useLayoutEffect(() => {
    if (open) {
      onSeen?.(document.activeElement);
    }
  }, [open, onSeen]);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        open
      </button>
      <button type="button">elsewhere</button>
      {open ? (
        <div
          ref={dialog.panelRef}
          role="dialog"
          aria-label="d"
          tabIndex={-1}
          onKeyDown={dialog.onKeyDown}
        >
          {children}
        </div>
      ) : null}
    </>
  );
}

const open = () => fireEvent.click(screen.getByRole('button', { name: 'open' }));
const panel = () => screen.getByRole('dialog');

function IconOpener() {
  const [isOpen, setOpen] = useState(false);
  const dialog = useDialog(isOpen, () => setOpen(false));
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        <svg data-testid="icon" aria-hidden="true" />
      </button>
      {isOpen ? (
        <div
          ref={dialog.panelRef}
          role="dialog"
          aria-label="d"
          tabIndex={-1}
          onKeyDown={dialog.onKeyDown}
        >
          <button type="button">in</button>
        </div>
      ) : null}
    </>
  );
}

function WithRef() {
  const target = useRef<HTMLButtonElement>(null);
  return (
    <>
      <button type="button" ref={target}>
        explicit
      </button>
      <Host returnFocusRef={target} />
    </>
  );
}

function Vanishing() {
  const [shown, setShown] = useState(true);
  const [isOpen, setOpen] = useState(false);
  const dialog = useDialog(isOpen, () => setOpen(false));
  return (
    <>
      {shown ? (
        <button type="button" onClick={() => setOpen(true)}>
          open
        </button>
      ) : null}
      {isOpen ? (
        <div
          ref={dialog.panelRef}
          role="dialog"
          aria-label="d"
          tabIndex={-1}
          onKeyDown={dialog.onKeyDown}
        >
          <button
            type="button"
            onClick={() => {
              setShown(false);
              setOpen(false);
            }}
          >
            drop
          </button>
        </div>
      ) : null}
    </>
  );
}

function RovingTabs() {
  return (
    <div role="tablist">
      <button type="button" role="tab" tabIndex={0}>
        active
      </button>
      <button type="button" role="tab" tabIndex={-1}>
        two
      </button>
      <button type="button" role="tab" tabIndex={-1}>
        three
      </button>
    </div>
  );
}

describe('useDialog focus timing', () => {
  it('has focus inside the dialog in the commit that shows it, before any passive effect', () => {
    let seen: Element | null = null;
    render(
      <Host onSeen={(active) => (seen = active)}>
        <input aria-label="first" />
        <button type="button">second</button>
      </Host>,
    );
    open();
    expect((seen as Element | null)?.getAttribute('aria-label')).toBe('first');
  });

  it('focuses the data-autofocus element over the first tabbable one', () => {
    render(
      <Host>
        <button type="button">first</button>
        <button type="button" data-autofocus="">
          chosen
        </button>
      </Host>,
    );
    open();
    expect(document.activeElement?.textContent).toBe('chosen');
  });

  it('focuses the panel itself when nothing inside is tabbable', () => {
    render(
      <Host>
        <p>Read only</p>
      </Host>,
    );
    open();
    expect(document.activeElement).toBe(panel());
  });

  it('locks page scroll while open and restores it on close', () => {
    document.body.style.overflow = 'auto';
    render(<Host />);
    open();
    expect(document.body.style.overflow).toBe('hidden');
    fireEvent.keyDown(panel(), { key: 'Escape' });
    expect(document.body.style.overflow).toBe('auto');
    document.body.style.overflow = '';
  });
});

describe('useDialog returns focus to the opener', () => {
  it('when the opener had focus (keyboard)', () => {
    render(<Host />);
    const opener = screen.getByRole('button', { name: 'open' });
    act(() => {
      opener.focus();
    });
    open();
    expect(document.activeElement).toBe(panel());
    fireEvent.keyDown(panel(), { key: 'Escape' });
    expect(document.activeElement).toBe(opener);
  });

  it('when a click did not focus it, as in Safari and Firefox on macOS', () => {
    render(<Host />);
    const opener = screen.getByRole('button', { name: 'open' });
    expect(document.activeElement).toBe(document.body);
    open();
    expect(document.activeElement).not.toBe(opener);
    fireEvent.keyDown(panel(), { key: 'Escape' });
    expect(document.activeElement).toBe(opener);
  });

  it('from an icon inside the clicked button', () => {
    render(<IconOpener />);
    fireEvent.click(screen.getByTestId('icon'));
    fireEvent.keyDown(panel(), { key: 'Escape' });
    expect(document.activeElement).toBe(screen.getByRole('button'));
  });

  it('to returnFocusRef when one is given, ahead of the click target', () => {
    render(<WithRef />);
    open();
    fireEvent.keyDown(panel(), { key: 'Escape' });
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'explicit' }));
  });

  it('does not throw when the opener has left the page before the dialog closes', () => {
    render(<Vanishing />);
    fireEvent.click(screen.getByRole('button', { name: 'open' }));
    fireEvent.click(screen.getByRole('button', { name: 'drop' }));
    expect(screen.queryByRole('dialog')).toBeNull();
  });
});

describe('useDialog keyboard', () => {
  it('closes on Escape without letting it reach ancestors', () => {
    const outer = vi.fn();
    render(
      <fieldset onKeyDown={outer}>
        <Host />
      </fieldset>,
    );
    open();
    fireEvent.keyDown(panel(), { key: 'Escape' });
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(outer).not.toHaveBeenCalled();
  });

  it('wraps Tab from the last tabbable to the first, and Shift+Tab back', () => {
    render(
      <Host>
        <button type="button">a</button>
        <button type="button">b</button>
      </Host>,
    );
    open();
    const a = screen.getByRole('button', { name: 'a' });
    const b = screen.getByRole('button', { name: 'b' });
    act(() => b.focus());
    expect(fireEvent.keyDown(b, { key: 'Tab' })).toBe(false);
    expect(document.activeElement).toBe(a);
    expect(fireEvent.keyDown(a, { key: 'Tab', shiftKey: true })).toBe(false);
    expect(document.activeElement).toBe(b);
  });

  it('leaves Tab alone between stops', () => {
    render(
      <Host>
        <button type="button">a</button>
        <button type="button">b</button>
      </Host>,
    );
    open();
    expect(fireEvent.keyDown(screen.getByRole('button', { name: 'a' }), { key: 'Tab' })).toBe(true);
  });

  describe('tabindex="-1" is not tabbable', () => {
    it('does not count roving-tabindex tabs, so the wrap fires from the one real stop', () => {
      render(
        <Host>
          <RovingTabs />
        </Host>,
      );
      open();
      const active = screen.getByRole('tab', { name: 'active' });
      expect(tabbables(panel())).toEqual([active]);
      act(() => active.focus());
      expect(fireEvent.keyDown(active, { key: 'Tab' })).toBe(false);
      expect(document.activeElement).toBe(active);
    });

    it('does not let Tab leave from a -1 element after the last stop', () => {
      render(
        <Host>
          <button type="button">only</button>
          <RovingTabs />
          <button type="button" tabIndex={-1}>
            skipped
          </button>
        </Host>,
      );
      open();
      const skipped = screen.getByRole('button', { name: 'skipped' });
      act(() => skipped.focus());
      expect(fireEvent.keyDown(skipped, { key: 'Tab' })).toBe(false);
      expect(document.activeElement).toBe(screen.getByRole('button', { name: 'only' }));
    });

    it('does not let Shift+Tab leave from the panel itself', () => {
      render(
        <Host>
          <button type="button">a</button>
          <button type="button">b</button>
        </Host>,
      );
      open();
      act(() => panel().focus());
      expect(fireEvent.keyDown(panel(), { key: 'Tab', shiftKey: true })).toBe(false);
      expect(document.activeElement).toBe(screen.getByRole('button', { name: 'b' }));
    });

    it('ignores elements inside a hidden subtree', () => {
      render(
        <Host>
          <button type="button">shown</button>
          <section hidden>
            <button type="button">folded</button>
          </section>
        </Host>,
      );
      open();
      expect(tabbables(panel()).map((element) => element.textContent)).toEqual(['shown']);
    });
  });

  describe('a panel with nothing tabbable', () => {
    it('still traps: Tab and Shift+Tab are cancelled and focus stays on the panel', () => {
      render(
        <Host>
          <p>Nothing to press</p>
        </Host>,
      );
      open();
      expect(fireEvent.keyDown(panel(), { key: 'Tab' })).toBe(false);
      expect(document.activeElement).toBe(panel());
      expect(fireEvent.keyDown(panel(), { key: 'Tab', shiftKey: true })).toBe(false);
      expect(document.activeElement).toBe(panel());
    });

    it('pulls focus back onto the panel if it had strayed', () => {
      render(
        <Host>
          <p>Nothing to press</p>
        </Host>,
      );
      open();
      act(() => screen.getByRole('button', { name: 'elsewhere' }).focus());
      fireEvent.keyDown(panel(), { key: 'Tab' });
      expect(document.activeElement).toBe(panel());
    });
  });
});
