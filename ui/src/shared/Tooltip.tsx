/**
 * Tooltip: a short text on hover and keyboard focus, for a control whose
 * meaning is not in its label (an icon button). Not for anything the user must
 * read: touch has no hover, so the text must also live in the control's
 * `aria-label` or the surrounding copy.
 *
 * The bubble is portalled to `document.body` and positioned with `fixed` from
 * the trigger's bounding box, so no `overflow-hidden` or scrolling ancestor (a
 * DataTable with `maxHeight`, a RightDrawer body) can clip it. It prefers the
 * requested side, flips when that side has no room, stays inside the viewport
 * horizontally, and follows the trigger on scroll and resize while open.
 *
 * The child gets `aria-describedby` pointing at the bubble and the hover and
 * focus handlers. A disabled child fires no pointer events and cannot take
 * focus, so it is rendered `aria-disabled` instead (still in the tab order, its
 * click swallowed): a disabled action can still explain itself on hover and on
 * focus. Button styles `aria-disabled` like `disabled`. Escape dismisses the
 * bubble until the pointer or focus leaves.
 */
import {
  cloneElement,
  type FocusEvent,
  type MouseEvent,
  type ReactElement,
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
  onMouseEnter?: (event: MouseEvent<HTMLElement>) => void;
  onMouseLeave?: (event: MouseEvent<HTMLElement>) => void;
  onFocus?: (event: FocusEvent<HTMLElement>) => void;
  onBlur?: (event: FocusEvent<HTMLElement>) => void;
}

export interface TooltipProps {
  text: string;
  /** The preferred side; it flips when there is no room. */
  side?: 'top' | 'bottom';
  /** One element that takes `aria-describedby` and mouse, focus and key handlers: a button, a link, an input. */
  children: ReactElement<TargetProps>;
}

const GAP = 6;
const MARGIN = 8;

interface Placement {
  top: number;
  left: number;
}

function place(trigger: DOMRect, bubble: DOMRect, side: 'top' | 'bottom'): Placement {
  const fitsAbove = trigger.top - bubble.height - GAP >= MARGIN;
  const fitsBelow = trigger.bottom + bubble.height + GAP <= window.innerHeight - MARGIN;
  const above = side === 'top' ? fitsAbove || !fitsBelow : !fitsBelow && fitsAbove;
  const centred = trigger.left + trigger.width / 2 - bubble.width / 2;
  return {
    top: above ? trigger.top - bubble.height - GAP : trigger.bottom + GAP,
    left: Math.max(MARGIN, Math.min(centred, window.innerWidth - bubble.width - MARGIN)),
  };
}

export function Tooltip({ text, side = 'top', children }: TooltipProps) {
  const id = useId();
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [placement, setPlacement] = useState<Placement | null>(null);
  const triggerRef = useRef<HTMLSpanElement>(null);
  const bubbleRef = useRef<HTMLSpanElement>(null);
  const open = (hovered || focused) && !dismissed;
  const { props } = children;
  const disabled = props.disabled === true;

  useLayoutEffect(() => {
    if (!open) {
      setPlacement(null);
      return;
    }
    const update = () => {
      const { current: trigger } = triggerRef;
      const { current: bubble } = bubbleRef;
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
  }, [open, side]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const onKey = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        setDismissed(true);
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open]);

  const handlers = {
    onMouseEnter: (event: MouseEvent<HTMLElement>) => {
      props.onMouseEnter?.(event);
      setHovered(true);
    },
    onMouseLeave: (event: MouseEvent<HTMLElement>) => {
      props.onMouseLeave?.(event);
      setHovered(false);
      setDismissed(false);
    },
    onFocus: (event: FocusEvent<HTMLElement>) => {
      props.onFocus?.(event);
      setFocused(true);
    },
    onBlur: (event: FocusEvent<HTMLElement>) => {
      props.onBlur?.(event);
      setFocused(false);
      setDismissed(false);
    },
  };
  // A disabled control fires no pointer events and cannot take focus, so it is
  // kept enabled to the DOM and marked `aria-disabled` instead, with its click
  // swallowed. The child must style `aria-disabled` like `disabled` (Button does).
  const unavailable = {
    disabled: false,
    'aria-disabled': true,
    onClick: (event: MouseEvent<HTMLElement>) => {
      event.preventDefault();
      event.stopPropagation();
    },
  };
  const describedBy = props['aria-describedby'];
  const description = describedBy ? `${describedBy} ${id}` : id;

  const bubble = createPortal(
    <span
      ref={bubbleRef}
      id={id}
      role="tooltip"
      style={placement ? { top: placement.top, left: placement.left } : { top: 0, left: 0 }}
      className={cx(
        'pointer-events-none fixed z-[100] w-max max-w-[16rem] rounded-md bg-text-primary px-2 py-1 text-xs font-medium text-text-inverse shadow-lg',
        open && placement ? 'visible' : 'invisible',
      )}
    >
      {text}
    </span>,
    document.body,
  );

  return (
    <>
      <span ref={triggerRef} className="inline-flex">
        {cloneElement(children, {
          'aria-describedby': description,
          ...(disabled ? unavailable : {}),
          ...handlers,
        })}
      </span>
      {bubble}
    </>
  );
}
