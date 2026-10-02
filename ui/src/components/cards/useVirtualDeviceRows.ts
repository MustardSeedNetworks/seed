/**
 * Virtualised rows for the discovery table (#461).
 *
 * Measured on Chromium: mounting every row blocked the main thread for 219 ms
 * at 250 devices, 777 ms at 1,000 and 3.8 s at 5,000. Up to the threshold
 * every row stays in the DOM, so find-in-page and a screen reader's table
 * navigation see the whole list; above it only the rows in view are mounted.
 */

import { useVirtualizer } from '@tanstack/react-virtual';
import type React from 'react';
import { useEffect, useRef } from 'react';

const VIRTUALIZE_ABOVE = 200;
// A full-width row as rendered (the discovery badges wrap to two lines).
const ROW_HEIGHT_ESTIMATE = 61;

function rowControls(row: Element): HTMLElement[] {
  return Array.from(row.querySelectorAll<HTMLElement>('button:not([disabled])')).filter(
    (control) => control.offsetParent !== null,
  );
}

interface PendingFocus {
  index: number;
  backwards: boolean;
}

export interface VirtualDeviceRows<T> {
  scrollRef: React.RefObject<HTMLDivElement | null>;
  rows: { index: number; item: T }[];
  padTop: number;
  padBottom: number;
  measureRef: ((element: HTMLTableSectionElement | null) => void) | undefined;
  onKeyDown: (event: React.KeyboardEvent<HTMLElement>) => void;
}

export function useVirtualDeviceRows<T>(
  items: T[],
  keyOf: (item: T) => string,
): VirtualDeviceRows<T> {
  const scrollRef = useRef<HTMLDivElement>(null);
  const pendingFocus = useRef<PendingFocus | null>(null);
  const virtualize = items.length > VIRTUALIZE_ABOVE;
  const virtualizer = useVirtualizer({
    count: virtualize ? items.length : 0,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT_ESTIMATE,
    overscan: 10,
    // The index always comes from `count`; the lookup is only ever in range.
    getItemKey: (index) => {
      const item = items[index];
      return item === undefined ? index : keyOf(item);
    },
  });

  const virtualRows = virtualize ? virtualizer.getVirtualItems() : [];
  const rows = virtualize
    ? virtualRows.flatMap((row) => {
        const item = items[row.index];
        return item === undefined ? [] : [{ index: row.index, item }];
      })
    : items.map((item, index) => ({ index, item }));
  const padTop = virtualRows[0]?.start ?? 0;
  const padBottom = virtualize
    ? virtualizer.getTotalSize() - (virtualRows[virtualRows.length - 1]?.end ?? 0)
    : 0;

  // Hand focus to a row once the virtualiser has mounted it. A row with no
  // enabled control passes the move on to its neighbour.
  useEffect(() => {
    const pending = pendingFocus.current;
    const row = pending && scrollRef.current?.querySelector(`tbody[data-index="${pending.index}"]`);
    if (!(pending && row)) {
      return;
    }
    const controls = rowControls(row);
    const target = pending.backwards ? controls.at(-1) : controls[0];
    if (target) {
      pendingFocus.current = null;
      target.focus();
      return;
    }
    const next = pending.index + (pending.backwards ? -1 : 1);
    pendingFocus.current = next >= 0 && next < items.length ? { ...pending, index: next } : null;
    if (pendingFocus.current) {
      virtualizer.scrollToIndex(next);
    }
  });

  // Tab from the last control of the last mounted row would leave the table,
  // and the dialog's focus trap would wrap it to the top, because the next row
  // is not in the DOM yet. How far a fast Tab gets then depends on render
  // timing (it did on WebKit under load). Mount the next row and move to it.
  const onKeyDown = (event: React.KeyboardEvent<HTMLElement>): void => {
    if (!virtualize || event.key !== 'Tab' || !(event.target instanceof HTMLElement)) {
      return;
    }
    const row = event.target.closest<HTMLElement>('tbody[data-index]');
    if (!row) {
      return;
    }
    const controls = rowControls(row);
    const edge = event.shiftKey ? controls[0] : controls.at(-1);
    const next = Number(row.dataset.index) + (event.shiftKey ? -1 : 1);
    const mounted = scrollRef.current?.querySelector(`tbody[data-index="${next}"]`);
    if (event.target !== edge || next < 0 || next >= items.length || mounted) {
      return;
    }
    event.preventDefault();
    pendingFocus.current = { index: next, backwards: event.shiftKey };
    virtualizer.scrollToIndex(next);
  };

  return {
    scrollRef,
    rows,
    padTop,
    padBottom,
    measureRef: virtualize ? virtualizer.measureElement : undefined,
    onKeyDown,
  };
}
