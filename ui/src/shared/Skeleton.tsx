/**
 * Skeleton: a placeholder for content that is on its way. Decorative: the
 * region that holds it announces the loading state (DataTable and Card do, with
 * `aria-busy` and a status line). Pulse only when the user allows motion.
 *
 * - `variant`: `text` (a line, rounded; default), `rectangular` (a block with
 *   the card radius) or `circular` (an avatar or icon).
 * - `width` and `height`: a number is px, a string is any CSS length; without
 *   them a text line is full width and 16px tall. Size it with utilities in
 *   `className` instead where the token scale is enough.
 * - `lines`: a stack of text lines whose last is 75% wide, the shape of a
 *   paragraph.
 */
import { cx } from './cx';

export interface SkeletonProps {
  variant?: 'text' | 'circular' | 'rectangular';
  width?: number | string;
  height?: number | string;
  /** More than one renders a paragraph: `lines` bars, the last shorter. */
  lines?: number;
  /** Size it with utilities, e.g. `h-4 w-24`. */
  className?: string;
}

const variants = {
  text: 'h-4 rounded',
  circular: 'rounded-full',
  rectangular: 'rounded-lg',
} as const;

function length(value: number | string | undefined): string | undefined {
  return typeof value === 'number' ? `${value}px` : value;
}

const LINE_KEYS = ['one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight'] as const;

export function Skeleton({ variant = 'text', width, height, lines = 1, className }: SkeletonProps) {
  const bar = (key: string, last: boolean) => (
    <span
      key={key}
      aria-hidden="true"
      className={cx(
        'block bg-surface-sunken motion-safe:animate-pulse',
        variant === 'text' ? 'h-4' : '',
        variant === 'text' ? 'rounded' : variants[variant],
        width === undefined && variant === 'text' ? 'w-full' : '',
        lines === 1 ? className : undefined,
      )}
      style={{
        width: last && lines > 1 ? '75%' : length(width),
        height: length(height),
      }}
    />
  );
  if (lines <= 1) {
    return bar('only', false);
  }
  const count = Math.min(lines, LINE_KEYS.length);
  return (
    <span aria-hidden="true" className={cx('flex flex-col gap-2', className)}>
      {LINE_KEYS.slice(0, count).map((key, index) => bar(key, index === count - 1))}
    </span>
  );
}
