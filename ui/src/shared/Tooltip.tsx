/**
 * Tooltip: a short text on hover and keyboard focus, for a control whose
 * meaning is not in its label (an icon button), or the full value of one that
 * truncates. Not for anything the user must read: touch has no hover, so the
 * text must also live in the control's `aria-label` or the surrounding copy.
 *
 * WCAG 1.4.13 (content on hover or focus): the bubble is dismissable (Escape),
 * hoverable (the pointer can move onto it and it stays, across the small gap, by
 * a short close delay) and persistent (it stays until the pointer and focus
 * have both left, or Escape). Escape is swallowed here, so it closes the bubble
 * and not the drawer or dialog behind it, unless a dialog that does not contain
 * the trigger is on top.
 *
 * `text` is optional: with none there is no bubble and no description, so a
 * caller can wrap unconditionally and pass text only when there is some. The
 * child is one element (it gets `aria-describedby`) or a render prop, for a
 * trigger nested inside something else (an input in a wrapper) that must carry
 * the description itself:
 *
 *   <Tooltip text="Copy">{(description) => <input {...description} />}</Tooltip>
 *
 * The bubble is portalled to `document.body` and positioned with `fixed` from
 * the trigger's bounding box, so no `overflow-hidden` or scrolling ancestor can
 * clip it. It prefers the requested side, flips when that side has no room,
 * stays inside the viewport, and follows the trigger on scroll and resize.
 *
 * A disabled child fires no pointer events and cannot take focus, so it is
 * rendered `aria-disabled` instead (still in the tab order, its click
 * swallowed): a disabled action can still explain itself. Button styles
 * `aria-disabled` like `disabled`.
 */
import {
  cloneElement,
  isValidElement,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { createPortal } from 'react-dom';
import { cx } from './cx';

interface TargetProps {
  disabled?: boolean;
  'aria-disabled'?: boolean;
  onClick?: (event: MouseEvent<HTMLElement>) => void;
  'aria-describedby'?: string;
}

/** What a render-prop child spreads onto the element that carries the description. */
export interface TooltipDescription {
  'aria-describedby': string | undefined;
  /** Dismisses the bubble: an activated control no longer needs its explanation. */
  onClick: () => void;
}

export type TooltipSide = 'top' | 'bottom';

export interface TooltipProps {
  /** Nothing renders when this is empty. */
  text?: ReactNode;
  /** The preferred side; it flips when there is no room. */
  side?: TooltipSide;
  children: ReactElement<TargetProps> | ((description: TooltipDescription) => ReactNode);
}

const GAP = 6;
const MARGIN = 8;
const CLOSE_DELAY_MS = 120;

interface Placement {
  top: number;
  left: number;
}

function clamp(value: number, size: number, viewport: number): number {
  return Math.max(MARGIN, Math.min(value, viewport - size - MARGIN));
}

function place(trigger: DOMRect, bubble: DOMRect, side: TooltipSide): Placement {
  const fitsAbove = trigger.top - bubble.height - GAP >= MARGIN;
  const fitsBelow = trigger.bottom + bubble.height + GAP <= window.innerHeight - MARGIN;
  const above = side === 'top' ? fitsAbove || !fitsBelow : !fitsBelow && fitsAbove;
  const centred = trigger.left + trigger.width / 2 - bubble.width / 2;
  return {
    top: above ? trigger.top - bubble.height - GAP : trigger.bottom + GAP,
    left: clamp(centred, bubble.width, window.innerWidth),
  };
}

function hasContent(text: ReactNode): boolean {
  return text !== undefined && text !== null && text !== false && text !== '';
}

export function Tooltip({ text, side = 'top', children }: TooltipProps) {
  const id = useId();
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [placement, setPlacement] = useState<Placement | null>(null);
  const wrapperRef = useRef<HTMLSpanElement>(null);
  const bubbleRef = useRef<HTMLSpanElement>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const dismissedRef = useRef(false);
  const present = hasContent(text);
  const open = present && (hovered || focused) && !dismissed;
  const dismiss = () => {
    dismissedRef.current = true;
    setDismissed(true);
  };

  useEffect(() => () => clearTimeout(closeTimer.current), []);

  useLayoutEffect(() => {
    if (!open) {
      setPlacement(null);
      return;
    }
    const update = () => {
      const { current: wrapper } = wrapperRef;
      const { current: bubble } = bubbleRef;
      const trigger =
        wrapper?.querySelector<HTMLElement>(`[aria-describedby~="${id}"]`) ??
        wrapper?.firstElementChild;
      if (trigger && bubble) {
        setPlacement(place(trigger.getBoundingClientRect(), bubble.getBoundingClientRect(), side));
      }
    };
    update();
    window.addEventListener('scroll', update, true);
    window.addEventListener('resize', update);
    return () => {
      window.removeEventListener('scroll', update, true);
      window.removeEventListener('resize', update);
    };
  }, [open, side, id]);

  // Native listeners, not React props: the wrapper is a static element, and the
  // bubble is outside it in the DOM (a portal), so the pointer going from one to
  // the other has to be tracked on both.
  //
  // Whether the bubble is active lives in refs, written by the same native
  // events that set the state, and the Escape listener is attached and removed
  // in those handlers. Gating it on the rendered `open` state raced: React
  // renders a hover (a continuous event) later than the pointer, so an Escape
  // that arrived first found no listener and the bubble opened after it. And
  // `dismissed` is cleared only when pointer and focus have both left: clearing
  // it when the pointer's grace period ended reopened a bubble that Escape had
  // closed while focus was still on the trigger.
  useEffect(() => {
    const { current: wrapper } = wrapperRef;
    const bubble = present ? bubbleRef.current : null;
    if (!wrapper) {
      return;
    }
    const pointer = { over: false, focus: false };
    let listening = false;
    const onKey = (event: globalThis.KeyboardEvent) => {
      if (event.key !== 'Escape' || dismissedRef.current) {
        return;
      }
      dismissedRef.current = true;
      setDismissed(true);
      // WebKit sends no mouseleave when a drawer opens under a still pointer, so
      // a hovered bubble can sit beneath a modal; that Escape is the dialog's.
      const dialog =
        event.target instanceof Element ? event.target.closest('[aria-modal="true"]') : null;
      if (!dialog || dialog.contains(wrapper)) {
        event.stopPropagation();
      }
    };
    const sync = () => {
      const active = pointer.over || pointer.focus;
      if (active && !listening) {
        document.addEventListener('keydown', onKey, true);
      } else if (!active && listening) {
        document.removeEventListener('keydown', onKey, true);
      }
      listening = active;
      if (!active) {
        dismissedRef.current = false;
        setDismissed(false);
      }
    };
    const inside = (node: EventTarget | null) =>
      node instanceof Node && (wrapper.contains(node) || Boolean(bubble?.contains(node)));
    const over = () => {
      clearTimeout(closeTimer.current);
      pointer.over = true;
      sync();
      setHovered(true);
    };
    const out = (event: globalThis.MouseEvent) => {
      if (inside(event.relatedTarget)) {
        return;
      }
      clearTimeout(closeTimer.current);
      closeTimer.current = setTimeout(() => {
        pointer.over = false;
        sync();
        setHovered(false);
      }, CLOSE_DELAY_MS);
    };
    const focusIn = () => {
      pointer.focus = true;
      sync();
      setFocused(true);
    };
    const focusOut = (event: globalThis.FocusEvent) => {
      if (!inside(event.relatedTarget)) {
        pointer.focus = false;
        sync();
        setFocused(false);
      }
    };
    wrapper.addEventListener('focusin', focusIn);
    wrapper.addEventListener('focusout', focusOut);
    for (const element of [wrapper, bubble]) {
      element?.addEventListener('mouseover', over);
      element?.addEventListener('mouseout', out);
    }
    return () => {
      document.removeEventListener('keydown', onKey, true);
      wrapper.removeEventListener('focusin', focusIn);
      wrapper.removeEventListener('focusout', focusOut);
      for (const element of [wrapper, bubble]) {
        element?.removeEventListener('mouseover', over);
        element?.removeEventListener('mouseout', out);
      }
    };
  }, [present]);

  let content: ReactNode = null;
  if (typeof children === 'function') {
    content = children({ 'aria-describedby': present ? id : undefined, onClick: dismiss });
  } else if (!present || !isValidElement(children)) {
    content = children;
  } else {
    const { props } = children;
    const described = props['aria-describedby'] ? `${props['aria-describedby']} ${id}` : id;
    content = cloneElement(
      children,
      props.disabled === true
        ? {
            'aria-describedby': described,
            disabled: false,
            'aria-disabled': true,
            onClick: (event: MouseEvent<HTMLElement>) => {
              event.preventDefault();
              event.stopPropagation();
            },
          }
        : {
            'aria-describedby': described,
            onClick: (event: MouseEvent<HTMLElement>) => {
              props.onClick?.(event);
              dismiss();
            },
          },
    );
  }

  return (
    <span ref={wrapperRef} className="contents">
      {content}
      {present
        ? createPortal(
            <span
              ref={bubbleRef}
              id={id}
              role="tooltip"
              style={placement ? { top: placement.top, left: placement.left } : { top: 0, left: 0 }}
              className={cx(
                'fixed z-[100] w-max max-w-[min(20rem,calc(100vw-1rem))] whitespace-normal rounded-md bg-text-primary px-2 py-1 text-xs font-medium text-text-inverse shadow-lg',
                open && placement ? 'visible' : 'invisible',
              )}
            >
              {text}
            </span>,
            document.body,
          )
        : null}
    </span>
  );
}
