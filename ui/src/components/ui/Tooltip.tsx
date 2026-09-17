import {
  type ButtonHTMLAttributes,
  cloneElement,
  type FC,
  isValidElement,
  type ReactElement,
  type ReactNode,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { createPortal } from 'react-dom';

type Description = { 'aria-describedby': string; onClick: () => void };

export interface TooltipProps {
  text?: ReactNode;
  side?: 'top' | 'bottom' | 'left' | 'right';
  children: ReactNode | ((description: Description) => ReactNode);
  className?: string;
}

// Nested inputs use the render form to put the description on the focusable control.
function describedChild(
  children: TooltipProps['children'],
  id: string,
  dismiss: () => void,
): ReactNode {
  if (typeof children === 'function') return children({ 'aria-describedby': id, onClick: dismiss });
  if (!isValidElement(children)) return children;
  const child = children as ReactElement<ButtonHTMLAttributes<HTMLButtonElement>>;
  const existing = child.props['aria-describedby'];
  const description = { 'aria-describedby': existing ? `${existing} ${id}` : id };
  if (child.type !== 'button' || !child.props.disabled) {
    return cloneElement(child, {
      ...description,
      onClick: (event) => {
        child.props.onClick?.(event);
        dismiss();
      },
    });
  }
  // Keep an unavailable action in the tab order so its reason can be read.
  return cloneElement(child, {
    ...description,
    disabled: false,
    'aria-disabled': true,
    className: `${child.props.className ?? ''} aria-disabled:opacity-50 aria-disabled:cursor-not-allowed`,
    onClick: (event) => {
      event.preventDefault();
      event.stopPropagation();
    },
  });
}

export const Tooltip: FC<TooltipProps> = ({ text, side = 'top', children, className = '' }) => {
  const id = useId();
  const hasText = text !== null && text !== undefined && text !== '';
  const wrapperRef = useRef<HTMLSpanElement>(null);
  const bubbleRef = useRef<HTMLSpanElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const open = (hovered || focused) && !dismissed;

  useEffect(() => {
    if (!(open && hasText)) return;
    const dismiss = (event: KeyboardEvent): void => {
      if (event.key !== 'Escape') return;
      setDismissed(true);
      event.stopPropagation();
    };
    document.addEventListener('keydown', dismiss, true);
    return () => document.removeEventListener('keydown', dismiss, true);
  }, [open, hasText]);

  useLayoutEffect(() => {
    if (!open || !hasText) return;
    const reposition = () => {
      const trigger = wrapperRef.current?.querySelector<HTMLElement>(`[aria-describedby~="${id}"]`);
      const bubble = bubbleRef.current;
      if (!trigger || !bubble) return;
      const anchor = trigger.getBoundingClientRect();
      const bounds = bubble.getBoundingClientRect();
      let top = anchor.top - bounds.height;
      let left = anchor.left + (anchor.width - bounds.width) / 2;
      if (side === 'bottom') top = anchor.bottom;
      if (side === 'left' || side === 'right') {
        top = anchor.top + (anchor.height - bounds.height) / 2;
        left = side === 'left' ? anchor.left - bounds.width : anchor.right;
      }
      // Keep the bubble inside the viewport, including the collapsed sidebar.
      setPosition({
        top: Math.max(0, Math.min(top, window.innerHeight - bounds.height)),
        left: Math.max(0, Math.min(left, window.innerWidth - bounds.width)),
      });
    };
    reposition();
    window.addEventListener('resize', reposition);
    window.addEventListener('scroll', reposition, true);
    return () => {
      window.removeEventListener('resize', reposition);
      window.removeEventListener('scroll', reposition, true);
    };
  }, [hasText, id, open, side, text]);

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: events bubble from the described focusable trigger
    <span
      ref={wrapperRef}
      className={`contents ${className}`}
      onMouseEnter={() => {
        setHovered(true);
        setDismissed(false);
      }}
      onMouseLeave={() => setHovered(false)}
      onFocus={() => {
        setFocused(true);
        setDismissed(false);
      }}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setFocused(false);
      }}
    >
      {hasText
        ? describedChild(children, id, () => setDismissed(true))
        : typeof children === 'function'
          ? children({ 'aria-describedby': '', onClick: () => setDismissed(true) })
          : children}
      {hasText
        ? createPortal(
            <span
              ref={bubbleRef}
              id={id}
              role="tooltip"
              hidden={!open}
              style={position}
              className="fixed z-[60] w-max max-w-[min(20rem,100vw)] whitespace-normal rounded-md bg-surface-raised px-cell py-compact text-xs text-text-primary ring-1 ring-surface-border"
            >
              {text}
            </span>,
            document.body,
          )
        : null}
    </span>
  );
};
