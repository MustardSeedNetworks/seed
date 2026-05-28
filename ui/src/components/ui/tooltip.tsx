/**
 * Tooltip primitive — shared design across seed / stem / niac.
 *
 * Minimal CSS-only tooltip with proper a11y wiring. Use native `title=` for
 * plain strings; reach for this primitive when the tooltip holds formatted,
 * multi-line, or linked content. The wrapper exposes `aria-describedby`
 * pointing at the bubble so screen readers announce it; hover OR keyboard
 * focus on the trigger reveals it.
 *
 * Behavior/API is kept consistent with the stem and niac copies (each repo
 * owns its own file; no master). Visuals use this repo's theme tokens.
 */
import type React from 'react';
import { type ReactNode, useId, useState } from 'react';
import { border, cn, radius, spacing } from '../../styles/theme';

export interface TooltipProps {
  /** Hover/focus content. If omitted, the wrapper renders children unchanged. */
  text?: ReactNode;
  /** Where to place the bubble relative to the trigger. Defaults to "top". */
  side?: 'top' | 'bottom' | 'left' | 'right';
  /** Trigger element(s). */
  children: ReactNode;
  /** Optional class on the wrapper. */
  className?: string;
}

const sideClass: Record<NonNullable<TooltipProps['side']>, string> = {
  top: cn('bottom-full left-1/2 -translate-x-1/2', spacing.margin.bottom.inline),
  bottom: cn('top-full left-1/2 -translate-x-1/2', spacing.margin.top.inline),
  left: cn('right-full top-1/2 -translate-y-1/2', spacing.margin.right.inline),
  right: cn('left-full top-1/2 -translate-y-1/2', spacing.margin.left.inline),
};

/**
 * Hover/focus-triggered tooltip that enriches an element with extra context.
 */
export function Tooltip({
  text,
  side = 'top',
  children,
  className = '',
}: TooltipProps): React.JSX.Element {
  const id = useId();
  const [open, setOpen] = useState(false);

  if (text === undefined || text === null || text === '') {
    return <>{children}</>;
  }

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: hover-only enrichment; a11y comes from aria-describedby below
    <span
      className={cn('relative inline-flex', className)}
      onMouseEnter={(): void => setOpen(true)}
      onMouseLeave={(): void => setOpen(false)}
      onFocus={(): void => setOpen(true)}
      onBlur={(): void => setOpen(false)}
    >
      <span aria-describedby={id} className="inline-flex">
        {children}
      </span>
      <span
        id={id}
        role="tooltip"
        className={cn(
          'pointer-events-none absolute z-50 max-w-xs whitespace-normal shadow-lg transition-opacity duration-100',
          spacing.cell.px,
          spacing.compact.pyMd,
          radius.default,
          border.card,
          'bg-surface-raised text-text-primary caption',
          sideClass[side],
          open ? 'opacity-100' : 'opacity-0',
        )}
      >
        {text}
      </span>
    </span>
  );
}
