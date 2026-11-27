import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Button } from './Button';
import { Modal } from './Modal';
import { Tooltip } from './Tooltip';

const BUBBLE = { width: 100, height: 20 };
let trigger = { top: 200, left: 300, width: 40, height: 30 };

function rect({ top, left, width, height }: typeof trigger): DOMRect {
  return {
    top,
    left,
    width,
    height,
    bottom: top + height,
    right: left + width,
    x: left,
    y: top,
    toJSON: () => ({}),
  };
}

beforeEach(() => {
  vi.useFakeTimers();
  trigger = { top: 200, left: 300, width: 40, height: 30 };
  window.innerWidth = 1000;
  window.innerHeight = 800;
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (
    this: HTMLElement,
  ) {
    return this.getAttribute('role') === 'tooltip'
      ? rect({ top: 0, left: 0, ...BUBBLE })
      : rect(trigger);
  });
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

const tip = () => (
  <Tooltip text="Delete device">
    <button type="button">x</button>
  </Tooltip>
);

const bubble = () => screen.getByRole('tooltip', { hidden: true });
const shown = () => !bubble().className.includes('invisible');
const hover = (element: Element) => fireEvent.mouseOver(element);
/** The pointer leaves for the page; the bubble closes after its short grace period. */
const leave = (element: Element) => {
  fireEvent.mouseOut(element);
  act(() => {
    vi.advanceTimersByTime(300);
  });
};

function Host({ onClose }: { onClose: () => void }) {
  const [open, setOpen] = useState(true);
  return (
    <Modal
      open={open}
      onClose={() => {
        onClose();
        setOpen(false);
      }}
      title="Devices"
      closeLabel="Close"
    >
      <Tooltip text="Export">
        <button type="button">export</button>
      </Tooltip>
      <button type="button">other</button>
    </Modal>
  );
}

describe('Tooltip', () => {
  it('describes its child with the bubble, which is in the DOM from the start', () => {
    render(tip());
    expect(screen.getByRole('button', { name: 'x' }).getAttribute('aria-describedby')).toBe(
      bubble().id,
    );
    expect(bubble().textContent).toBe('Delete device');
    expect(shown()).toBe(false);
  });

  it('shows on hover and on keyboard focus, and hides again', () => {
    render(tip());
    const button = screen.getByRole('button');
    hover(button);
    expect(shown()).toBe(true);
    leave(button);
    expect(shown()).toBe(false);
    fireEvent.focus(button);
    expect(shown()).toBe(true);
    fireEvent.blur(button);
    expect(shown()).toBe(false);
  });

  it('is dismissed by Escape until the pointer or focus leaves', () => {
    render(tip());
    const button = screen.getByRole('button');
    fireEvent.focus(button);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(shown()).toBe(false);
    fireEvent.blur(button);
    fireEvent.focus(button);
    expect(shown()).toBe(true);
  });

  it("keeps the child's own handlers and description", () => {
    const onFocus = vi.fn();
    render(
      <Tooltip text="Tip">
        <button type="button" aria-describedby="other" onFocus={onFocus}>
          x
        </button>
      </Tooltip>,
    );
    const button = screen.getByRole('button');
    fireEvent.focus(button);
    expect(onFocus).toHaveBeenCalledOnce();
    expect(button.getAttribute('aria-describedby')).toMatch(/^other /);
  });

  describe('WCAG 1.4.13: hoverable, persistent, dismissable', () => {
    it('does not set pointer-events-none, so the bubble can be hovered', () => {
      render(tip());
      expect(bubble().className).not.toContain('pointer-events-none');
    });

    it('stays open while the pointer travels from the trigger onto the bubble', () => {
      render(tip());
      const button = screen.getByRole('button');
      hover(button);
      fireEvent.mouseOut(button);
      act(() => {
        vi.advanceTimersByTime(60);
      });
      hover(bubble());
      act(() => {
        vi.advanceTimersByTime(1000);
      });
      expect(shown()).toBe(true);
    });

    it('stays open for as long as the pointer is on the bubble, and closes once it leaves both', () => {
      render(tip());
      hover(screen.getByRole('button'));
      hover(bubble());
      leave(bubble());
      expect(shown()).toBe(false);
    });

    it('does not close when the pointer moves between the trigger and the bubble directly', () => {
      render(tip());
      const button = screen.getByRole('button');
      hover(button);
      fireEvent.mouseOut(button, { relatedTarget: bubble() });
      act(() => {
        vi.advanceTimersByTime(1000);
      });
      expect(shown()).toBe(true);
    });

    it('is dismissed by Escape while the pointer rests on the bubble', () => {
      render(tip());
      hover(screen.getByRole('button'));
      hover(bubble());
      fireEvent.keyDown(bubble(), { key: 'Escape' });
      expect(shown()).toBe(false);
    });

    it('swallows Escape, so it closes the bubble and not the dialog behind it', () => {
      const onDialogKey = vi.fn();
      render(
        <div role="dialog" aria-modal="true" onKeyDown={onDialogKey}>
          {tip()}
        </div>,
      );
      const button = screen.getByRole('button');
      fireEvent.focus(button);
      fireEvent.keyDown(button, { key: 'Escape' });
      expect(shown()).toBe(false);
      expect(onDialogKey).not.toHaveBeenCalled();
    });

    it('leaves Escape to a dialog that opened over a still-hovered bubble', () => {
      const onDialogKey = vi.fn();
      render(
        <>
          {tip()}
          <div role="dialog" aria-modal="true" onKeyDown={onDialogKey}>
            <button type="button" data-testid="inside">
              ok
            </button>
          </div>
        </>,
      );
      hover(screen.getByRole('button', { name: 'x' }));
      fireEvent.keyDown(screen.getByTestId('inside'), { key: 'Escape' });
      expect(onDialogKey).toHaveBeenCalledOnce();
    });
  });

  describe('optional text and render-prop child', () => {
    it.each([undefined, null, '', false])('renders no bubble and no description for %j', (text) => {
      render(
        <Tooltip text={text}>
          <button type="button">x</button>
        </Tooltip>,
      );
      expect(screen.queryByRole('tooltip', { hidden: true })).toBeNull();
      expect(screen.getByRole('button').getAttribute('aria-describedby')).toBeNull();
      hover(screen.getByRole('button'));
      expect(screen.queryByRole('tooltip', { hidden: true })).toBeNull();
    });

    it('accepts rich text', () => {
      render(
        <Tooltip text={<strong>Full value</strong>}>
          <button type="button">x</button>
        </Tooltip>,
      );
      expect(bubble().querySelector('strong')?.textContent).toBe('Full value');
    });

    it('hands a render-prop child the description to put on its own control', () => {
      render(
        <Tooltip text="Hostname">
          {(description) => (
            <label>
              Host
              <input {...description} />
            </label>
          )}
        </Tooltip>,
      );
      const input = screen.getByLabelText('Host');
      expect(input.getAttribute('aria-describedby')).toBe(bubble().id);
      fireEvent.focus(input);
      expect(shown()).toBe(true);
    });

    it('dismisses through the render-prop onClick', () => {
      render(
        <Tooltip text="Hostname">
          {(description) => (
            <button type="button" {...description}>
              go
            </button>
          )}
        </Tooltip>,
      );
      const button = screen.getByRole('button');
      fireEvent.focus(button);
      fireEvent.click(button);
      expect(shown()).toBe(false);
    });

    it('gives a render-prop child no description when there is no text', () => {
      render(<Tooltip>{(description) => <input aria-label="x" {...description} />}</Tooltip>);
      expect(screen.getByLabelText('x').getAttribute('aria-describedby')).toBeNull();
    });

    it('hides on click of an enabled child, and still runs its onClick', () => {
      const onClick = vi.fn();
      render(
        <Tooltip text="Tip">
          <button type="button" onClick={onClick}>
            x
          </button>
        </Tooltip>,
      );
      const button = screen.getByRole('button');
      fireEvent.focus(button);
      fireEvent.click(button);
      expect(onClick).toHaveBeenCalledOnce();
      expect(shown()).toBe(false);
    });
  });

  describe('portal', () => {
    it('renders the bubble in document.body, outside an overflow-hidden ancestor', () => {
      render(
        <div data-testid="clip" style={{ overflow: 'hidden' }} className="overflow-hidden">
          {tip()}
        </div>,
      );
      expect(screen.getByTestId('clip').contains(bubble())).toBe(false);
      expect(bubble().parentElement).toBe(document.body);
      expect(bubble().className).toContain('fixed');
    });

    it('removes the bubble when the tooltip unmounts', () => {
      const { unmount } = render(tip());
      unmount();
      expect(screen.queryByRole('tooltip', { hidden: true })).toBeNull();
    });
  });

  describe('placement', () => {
    const position = () => ({ top: bubble().style.top, left: bubble().style.left });

    it('sits above the trigger, centred on it, when it has room', () => {
      render(tip());
      hover(screen.getByRole('button'));
      expect(position()).toEqual({ top: `${200 - 20 - 6}px`, left: `${300 + 20 - 50}px` });
    });

    it('flips below when there is no room above', () => {
      trigger = { ...trigger, top: 4 };
      render(tip());
      hover(screen.getByRole('button'));
      expect(position().top).toBe(`${4 + 30 + 6}px`);
    });

    it('flips above when it prefers below and there is no room under', () => {
      trigger = { ...trigger, top: 780 };
      render(
        <Tooltip text="Tip" side="bottom">
          <button type="button">x</button>
        </Tooltip>,
      );
      hover(screen.getByRole('button'));
      expect(position().top).toBe(`${780 - 20 - 6}px`);
    });

    it('stays inside the viewport horizontally', () => {
      trigger = { ...trigger, left: 0, width: 10 };
      render(tip());
      hover(screen.getByRole('button'));
      expect(position().left).toBe('8px');
    });

    it('follows the trigger when an ancestor scrolls or the window resizes while open', () => {
      render(tip());
      hover(screen.getByRole('button'));
      trigger = { ...trigger, top: 400 };
      act(() => {
        fireEvent.scroll(document.body);
      });
      expect(position().top).toBe(`${400 - 20 - 6}px`);
      trigger = { ...trigger, top: 500 };
      act(() => {
        fireEvent(window, new Event('resize'));
      });
      expect(position().top).toBe(`${500 - 20 - 6}px`);
    });
  });

  describe('disabled trigger', () => {
    const disabledTip = (onClick = () => {}) => (
      <Tooltip text="Needs operator role">
        <Button variant="primary" disabled onClick={onClick}>
          Save
        </Button>
      </Tooltip>
    );

    it('still shows on hover and on focus, because it stays focusable', () => {
      render(disabledTip());
      const button = screen.getByRole('button', { name: 'Save' });
      hover(button);
      expect(shown()).toBe(true);
      leave(button);
      expect(shown()).toBe(false);
      act(() => {
        button.focus();
      });
      expect(shown()).toBe(true);
    });

    it('is exposed as disabled to assistive technology, with its description, and keeps its look', () => {
      render(disabledTip());
      const button = screen.getByRole('button', { name: 'Save' }) as HTMLButtonElement;
      expect(button.disabled).toBe(false);
      expect(button.getAttribute('aria-disabled')).toBe('true');
      expect(button.getAttribute('aria-describedby')).toBe(bubble().id);
      expect(button.className).toContain('aria-disabled:bg-surface-sunken');
      expect(button.className).toContain('aria-disabled:cursor-not-allowed');
    });

    it('swallows the click', () => {
      const onClick = vi.fn();
      render(disabledTip(onClick));
      fireEvent.click(screen.getByRole('button', { name: 'Save' }));
      expect(onClick).not.toHaveBeenCalled();
    });

    it('leaves an enabled button untouched', () => {
      const onClick = vi.fn();
      render(
        <Tooltip text="Tip">
          <Button onClick={onClick}>Go</Button>
        </Tooltip>,
      );
      const button = screen.getByRole('button', { name: 'Go' });
      expect(button.getAttribute('aria-disabled')).toBeNull();
      fireEvent.click(button);
      expect(onClick).toHaveBeenCalledOnce();
    });
  });

  describe('Escape under load (the niac "Escape dismisses tooltip" flake)', () => {
    it('stays dismissed when the pointer grace period ends while focus is still on the trigger', () => {
      render(tip());
      const button = screen.getByRole('button');
      hover(button);
      fireEvent.focus(button);
      fireEvent.mouseOut(button);
      fireEvent.keyDown(document, { key: 'Escape' });
      expect(shown()).toBe(false);
      act(() => {
        vi.advanceTimersByTime(300);
      });
      expect(shown()).toBe(false);
      fireEvent.blur(button);
      fireEvent.focus(button);
      expect(shown()).toBe(true);
    });

    it('is dismissed by an Escape that arrives before React has rendered the hover', async () => {
      const environment = globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean };
      render(tip());
      const button = screen.getByRole('button');
      environment.IS_REACT_ACT_ENVIRONMENT = false;
      try {
        // Native events with no act(): the hover is queued, not yet committed,
        // as when a parallel worker's renderer is behind the pointer.
        button.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        await vi.advanceTimersByTimeAsync(500);
        expect(shown()).toBe(false);
      } finally {
        environment.IS_REACT_ACT_ENVIRONMENT = true;
      }
    });

    it('lets a second Escape through to the dialog once the bubble is closed', () => {
      const onKey = vi.fn();
      document.addEventListener('keydown', onKey);
      render(tip());
      fireEvent.focus(screen.getByRole('button'));
      fireEvent.keyDown(document, { key: 'Escape' });
      expect(onKey).not.toHaveBeenCalled();
      fireEvent.keyDown(document, { key: 'Escape' });
      expect(onKey).toHaveBeenCalledOnce();
      document.removeEventListener('keydown', onKey);
    });
  });
  describe('Escape in the window between focus or pointer leaving and the bubble hiding', () => {
    const focusOut = (from: Element, to: Element) =>
      from.dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: to }));
    const focusIn = (target: Element) =>
      target.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));
    const pressEscape = (target: Element) =>
      target.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }),
      );
    const keydownListeners = () => {
      const live = new Set<unknown>();
      const added = vi.spyOn(document, 'addEventListener');
      const removed = vi.spyOn(document, 'removeEventListener');
      return {
        count: () => {
          for (const [type, listener] of added.mock.calls) {
            if (type === 'keydown') live.add(listener);
          }
          for (const [type, listener] of removed.mock.calls) {
            if (type === 'keydown') live.delete(listener);
          }
          return live.size;
        },
      };
    };

    /** Everything inside one act(): React has not rendered the state the events queued. */
    const inWindow = (steps: () => void) => {
      act(steps);
    };

    it('closes only the bubble when Escape arrives while it is still on screen', () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => exportButton.focus());
      expect(shown()).toBe(true);
      inWindow(() => {
        focusOut(exportButton, other);
        expect(shown()).toBe(true);
        pressEscape(other);
      });
      expect(onClose).not.toHaveBeenCalled();
      expect(shown()).toBe(false);
    });

    it('lets Escape reach the dialog once the bubble has fully hidden', () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => exportButton.focus());
      act(() => focusOut(exportButton, other));
      expect(shown()).toBe(false);
      act(() => pressEscape(other));
      expect(onClose).toHaveBeenCalledOnce();
    });

    it('does not leave the tooltip dismissed after an Escape in the window', () => {
      render(<Host onClose={vi.fn()} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => exportButton.focus());
      inWindow(() => {
        focusOut(exportButton, other);
        pressEscape(other);
      });
      act(() => focusIn(exportButton));
      expect(shown()).toBe(true);
    });

    it('keeps the listener and the bubble when focus returns inside the window', () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => exportButton.focus());
      inWindow(() => {
        focusOut(exportButton, other);
        focusIn(exportButton);
      });
      expect(shown()).toBe(true);
      act(() => pressEscape(exportButton));
      expect(onClose).not.toHaveBeenCalled();
      expect(shown()).toBe(false);
    });

    it('keeps the listener and the bubble when the pointer returns inside the window', () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      hover(exportButton);
      fireEvent.mouseOut(exportButton);
      act(() => {
        vi.advanceTimersByTime(300);
        hover(exportButton);
        expect(shown()).toBe(true);
        pressEscape(exportButton);
      });
      expect(onClose).not.toHaveBeenCalled();
    });

    it('re-hovering inside the window keeps the bubble open', () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      hover(exportButton);
      inWindow(() => {
        fireEvent.mouseOut(exportButton);
        vi.advanceTimersByTime(300);
        hover(exportButton);
      });
      expect(shown()).toBe(true);
      act(() => pressEscape(exportButton));
      expect(onClose).not.toHaveBeenCalled();
      expect(shown()).toBe(false);
    });

    it('leaves no keydown listener on document when unmounted inside the window', () => {
      const listeners = keydownListeners();
      const { unmount } = render(<Host onClose={vi.fn()} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => exportButton.focus());
      const live = listeners.count();
      expect(live).toBeGreaterThan(0);
      act(() => {
        focusOut(exportButton, other);
        unmount();
      });
      expect(listeners.count()).toBeLessThan(live);
    });

    it("releases the listener when the bubble was never shown, so a later Escape is the dialog's", () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => {
        focusIn(exportButton);
        focusOut(exportButton, other);
      });
      act(() => pressEscape(other));
      expect(onClose).toHaveBeenCalledOnce();
    });

    it('releases the listener when Escape had already hidden the bubble before focus left', () => {
      const onClose = vi.fn();
      render(<Host onClose={onClose} />);
      const exportButton = screen.getByText('export');
      const other = screen.getByText('other');
      act(() => exportButton.focus());
      act(() => pressEscape(exportButton));
      act(() => focusOut(exportButton, other));
      act(() => pressEscape(other));
      expect(onClose).toHaveBeenCalledOnce();
    });
  });
});
