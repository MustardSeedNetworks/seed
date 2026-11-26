/**
 * EmptyState: what a list, table or module shows when there is nothing to show.
 *
 * Based on stem's ModuleEmptyState (an icon, one line saying what is missing,
 * one line saying why, at most one action that resolves it).
 *
 * The tone is neutral on purpose. No data is not a verdict: it is never green,
 * and it never reads "all clear" or "no problems". Say what is missing and
 * what the operator can do ("No devices yet. Add one"), not that things are
 * fine. If a page is healthy, say so with a StatusPill on data the page has.
 *
 * It has no frame of its own. Put it inside a Card or a DataTable, so a page
 * never carries a bordered empty box inside another bordered box.
 */
import type { ReactNode } from 'react';
import { cx } from './cx';

export interface EmptyStateProps {
  /** A decorative icon, about 24px. The title carries the meaning. */
  icon?: ReactNode;
  /** What is missing, e.g. "No devices yet". */
  title: string;
  /** Why, or what to do about it. One sentence. */
  hint?: string;
  /** The single action that resolves it. Not two. */
  action?: ReactNode;
  className?: string;
}

export function EmptyState({ icon, title, hint, action, className }: EmptyStateProps) {
  return (
    <div
      className={cx('flex flex-col items-center gap-2 px-4 py-10 text-center', className)}
      data-testid="empty-state"
    >
      {icon ? (
        <span className="text-text-muted" aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <p className="text-sm font-medium text-text-primary">{title}</p>
      {hint ? <p className="max-w-md text-sm text-text-muted">{hint}</p> : null}
      {action ? <div className="mt-1">{action}</div> : null}
    </div>
  );
}
