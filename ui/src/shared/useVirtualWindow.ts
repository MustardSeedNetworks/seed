/**
 * The row window for a fixed-row-height scroller, ported from niac's
 * `useVirtualScroll`. It returns which slice of `count` rows to mount for the
 * current scroll offset and the handler that tracks it; the caller pads the
 * space above and below with spacer rows, so the markup stays a real table.
 */
import { type UIEvent, useState } from 'react';

export interface VirtualWindowOptions {
  count: number;
  rowHeight: number;
  /** Height of the scroll viewport in px. */
  viewportHeight: number;
  /** Rows mounted beyond each edge of the viewport. */
  overscan?: number;
}

export interface VirtualWindow {
  start: number;
  end: number;
  onScroll: (event: UIEvent<HTMLElement>) => void;
}

export function useVirtualWindow({
  count,
  rowHeight,
  viewportHeight,
  overscan = 5,
}: VirtualWindowOptions): VirtualWindow {
  const [scrollTop, setScrollTop] = useState(0);
  const first = Math.floor(scrollTop / rowHeight);
  const start = Math.max(0, Math.min(first - overscan, count - 1));
  const end = Math.min(count, first + Math.ceil(viewportHeight / rowHeight) + overscan);
  return {
    start,
    end,
    onScroll: (event) => setScrollTop(event.currentTarget.scrollTop),
  };
}
