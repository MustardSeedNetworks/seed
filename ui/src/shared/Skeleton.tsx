/**
 * Skeleton: a placeholder bar for content that is on its way. Decorative: the
 * region that holds it announces the loading state (DataTable does, with
 * `aria-busy` and a status line). Pulse only when the user allows motion.
 */
import { cx } from './cx';

export interface SkeletonProps {
  /** Size it with utilities, e.g. `h-4 w-24`. Defaults to a full-width text line. */
  className?: string;
}

export function Skeleton({ className }: SkeletonProps) {
  return (
    <span
      aria-hidden="true"
      className={cx(
        'block h-4 w-full rounded bg-surface-sunken motion-safe:animate-pulse',
        className,
      )}
    />
  );
}
