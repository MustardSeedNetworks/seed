/**
 * Whether an element's content outruns its box, kept current as it changes. A
 * scroller that overflows must be reachable from the keyboard (axe
 * `scrollable-region-focusable`): either it holds something tabbable, or it is
 * itself focusable.
 *
 * - `scrolls`: the content overflows, so the caller gives the region
 *   `tabIndex={0}` and a name (DataTable does, always, so a keyboard user can pan
 *   it with the arrow keys even when it holds buttons).
 * - `needsFocus`, only with `{ trackTabbables: true }`: it overflows AND holds
 *   nothing tabbable, the minimum a modal body needs (a long read-only panel).
 *
 * Observers attach once on mount: a ResizeObserver on the element and its direct
 * children (a child growing is what changes `scrollHeight` when the element's own
 * box is fixed), and a MutationObserver that re-attaches when the children are
 * replaced. Nothing runs per render or per scroll tick. jsdom has no layout, so
 * both flags are false there unless a test stubs the sizes.
 */
import { type RefObject, useEffect, useRef, useState } from 'react';
import { tabbables } from './useDialog';

export interface Scrollable<T extends HTMLElement> {
  ref: RefObject<T | null>;
  scrolls: boolean;
  needsFocus: boolean;
}

export function useScrollable<T extends HTMLElement>({
  trackTabbables = false,
}: {
  trackTabbables?: boolean;
} = {}): Scrollable<T> {
  const ref = useRef<T>(null);
  const [state, setState] = useState({ scrolls: false, needsFocus: false });

  useEffect(() => {
    const { current: element } = ref;
    if (!element) {
      return;
    }
    const measure = () => {
      const scrolls =
        element.scrollWidth > element.clientWidth || element.scrollHeight > element.clientHeight;
      const needsFocus = trackTabbables && scrolls && tabbables(element).length === 0;
      setState((previous) =>
        previous.scrolls === scrolls && previous.needsFocus === needsFocus
          ? previous
          : { scrolls, needsFocus },
      );
    };
    const resize = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(measure);
    const watchChildren = () => {
      resize?.disconnect();
      resize?.observe(element);
      for (const child of element.children) {
        resize?.observe(child);
      }
    };
    const mutation = new MutationObserver(() => {
      watchChildren();
      measure();
    });
    mutation.observe(element, { childList: true });
    watchChildren();
    measure();
    return () => {
      resize?.disconnect();
      mutation.disconnect();
    };
  }, [trackTabbables]);

  return { ref, ...state };
}
