import { act, cleanup, fireEvent, screen } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, beforeEach, vi } from 'vitest';
import { Modal } from '../Modal';
import { Tooltip } from '../Tooltip';

const BUBBLE = { width: 100, height: 20 };
const TRIGGER = { top: 200, left: 300, width: 40, height: 30 };

/** The trigger's box, which the mocked getBoundingClientRect reports. */
let trigger = { ...TRIGGER };

export const moveTrigger = (to: Partial<typeof TRIGGER>) => {
  trigger = { ...trigger, ...to };
};

function rect({ top, left, width, height }: typeof TRIGGER): DOMRect {
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

/** Fake timers, a fixed viewport and fixed boxes for the trigger and the bubble. Call once per test file. */
export function installTooltipHarness() {
  beforeEach(() => {
    vi.useFakeTimers();
    trigger = { ...TRIGGER };
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
}

export const tip = () => (
  <Tooltip text="Delete device">
    <button type="button">x</button>
  </Tooltip>
);

export const bubble = () => screen.getByRole('tooltip', { hidden: true });
export const shown = () => !bubble().className.includes('invisible');
export const hover = (element: Element) => fireEvent.mouseOver(element);
/** The pointer leaves for the page; the bubble closes after its short grace period. */
export const leave = (element: Element) => {
  fireEvent.mouseOut(element);
  act(() => {
    vi.advanceTimersByTime(300);
  });
};

export function Host({ onClose }: { onClose: () => void }) {
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
