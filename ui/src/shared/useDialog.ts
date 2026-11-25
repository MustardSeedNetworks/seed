/**
 * The focus handling every modal surface shares (RightDrawer, Modal): focus
 * moves in on open (to the element marked `data-autofocus`, else the first
 * tabbable one, else the panel), Tab stays inside, Escape closes, focus returns
 * to the opener on close, and the page behind cannot scroll while it is open.
 * It folds in stem's and niac's `useFocusTrap` so these files import no product
 * code.
 *
 * Focus moves in a layout effect, so it is inside the dialog from the commit
 * that shows it: no frame, and no passive-effect gap an e2e can land in.
 *
 * The opener is the element that had focus when the dialog opened. Safari and
 * Firefox on macOS do not focus a button on click, so `document.activeElement`
 * is the body there; the hook then falls back to the target of the click that
 * opened the dialog (recorded in the capture phase, one task long), and a caller
 * that opens from somewhere else (a timer, a route change) passes
 * `returnFocusRef`, which wins over both.
 *
 * "Tabbable" excludes `tabindex="-1"`, so a roving-tabindex tab list as the
 * last stop does not hide the real last stop from the wrap, and a panel with
 * nothing tabbable still traps: focus stays on the panel.
 */
import { type KeyboardEvent, type RefObject, useEffect, useLayoutEffect, useRef } from 'react';

const TABBABLE = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]',
]
  .map((selector) => `${selector}:not([tabindex^="-"])`)
  .join(', ');

const FOCUSABLE_ANCESTOR =
  'button, a[href], input, select, textarea, summary, [role="button"], [tabindex]';

/** Elements Tab can reach inside `root`, in document order; hidden subtrees do not count. */
export function tabbables(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(TABBABLE)).filter(
    (element) => !element.closest('[hidden], [inert]'),
  );
}

export interface DialogOptions {
  /** Where focus goes on close, when the opener cannot be found from focus or the click. */
  returnFocusRef?: RefObject<HTMLElement | null>;
}

export interface Dialog {
  panelRef: RefObject<HTMLDivElement | null>;
  onKeyDown: (event: KeyboardEvent<HTMLDivElement>) => void;
}

function leavesPanel(
  active: Element | null,
  panel: HTMLElement,
  first: HTMLElement,
  last: HTMLElement,
  backwards: boolean,
): boolean {
  if (!active || active === panel || !panel.contains(active)) {
    return true;
  }
  if (backwards) {
    return (
      active === first ||
      Boolean(active.compareDocumentPosition(first) & Node.DOCUMENT_POSITION_FOLLOWING)
    );
  }
  return (
    active === last ||
    Boolean(active.compareDocumentPosition(last) & Node.DOCUMENT_POSITION_PRECEDING)
  );
}

export function useDialog(
  open: boolean,
  onClose: () => void,
  { returnFocusRef }: DialogOptions = {},
): Dialog {
  const panelRef = useRef<HTMLDivElement>(null);
  const clicked = useRef<Element | null>(null);

  useEffect(() => {
    let reset: ReturnType<typeof setTimeout> | undefined;
    const remember = (event: Event) => {
      clicked.current = event.target instanceof Element ? event.target : null;
      clearTimeout(reset);
      reset = setTimeout(() => {
        clicked.current = null;
      }, 0);
    };
    document.addEventListener('click', remember, true);
    return () => {
      clearTimeout(reset);
      document.removeEventListener('click', remember, true);
    };
  }, []);

  useLayoutEffect(() => {
    if (!open) {
      return;
    }
    const { activeElement } = document;
    const focused =
      activeElement instanceof HTMLElement && activeElement !== document.body
        ? activeElement
        : null;
    const opener =
      returnFocusRef?.current ??
      focused ??
      clicked.current?.closest<HTMLElement>(FOCUSABLE_ANCESTOR);
    const { overflow } = document.body.style;
    document.body.style.overflow = 'hidden';
    const { current: panel } = panelRef;
    const target =
      panel?.querySelector<HTMLElement>('[data-autofocus]') ??
      (panel ? tabbables(panel)[0] : undefined) ??
      panel;
    target?.focus();
    return () => {
      document.body.style.overflow = overflow;
      if (opener?.isConnected) {
        opener.focus();
      }
    };
  }, [open, returnFocusRef]);

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape') {
      event.stopPropagation();
      onClose();
      return;
    }
    if (event.key !== 'Tab') {
      return;
    }
    const panel = event.currentTarget;
    const items = tabbables(panel);
    const [first] = items;
    const last = items.at(-1);
    if (!first || !last) {
      event.preventDefault();
      panel.focus();
      return;
    }
    if (leavesPanel(document.activeElement, panel, first, last, event.shiftKey)) {
      event.preventDefault();
      (event.shiftKey ? last : first).focus();
    }
  };

  return { panelRef, onKeyDown };
}
