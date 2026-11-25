import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Button } from './Button';
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
  vi.restoreAllMocks();
});

const tip = () => (
  <Tooltip text="Delete device">
    <button type="button">x</button>
  </Tooltip>
);

const bubble = () => screen.getByRole('tooltip', { hidden: true });
const shown = () => !bubble().className.includes('invisible');

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
    fireEvent.mouseEnter(button);
    expect(shown()).toBe(true);
    fireEvent.mouseLeave(button);
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
      fireEvent.mouseEnter(screen.getByRole('button'));
      expect(position()).toEqual({ top: `${200 - 20 - 6}px`, left: `${300 + 20 - 50}px` });
    });

    it('flips below when there is no room above', () => {
      trigger = { ...trigger, top: 4 };
      render(tip());
      fireEvent.mouseEnter(screen.getByRole('button'));
      expect(position().top).toBe(`${4 + 30 + 6}px`);
    });

    it('flips above when it prefers below and there is no room under', () => {
      trigger = { ...trigger, top: 780 };
      render(
        <Tooltip text="Tip" side="bottom">
          <button type="button">x</button>
        </Tooltip>,
      );
      fireEvent.mouseEnter(screen.getByRole('button'));
      expect(position().top).toBe(`${780 - 20 - 6}px`);
    });

    it('stays inside the viewport horizontally', () => {
      trigger = { ...trigger, left: 0, width: 10 };
      render(tip());
      fireEvent.mouseEnter(screen.getByRole('button'));
      expect(position().left).toBe('8px');
    });

    it('follows the trigger when an ancestor scrolls or the window resizes while open', () => {
      render(tip());
      fireEvent.mouseEnter(screen.getByRole('button'));
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
      fireEvent.mouseEnter(button);
      expect(shown()).toBe(true);
      fireEvent.mouseLeave(button);
      expect(shown()).toBe(false);
      button.focus();
      fireEvent.focus(button);
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
});
