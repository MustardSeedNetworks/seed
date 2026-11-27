import { act, fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Host, hover, installTooltipHarness, shown, tip } from './test/tooltipTestUtils';

installTooltipHarness();

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
