/**
 * Card: the single container for a block of page content.
 *
 * One radius and one hairline border, both from the shared `.panel` class, and
 * flat: no shadow, no gradient. A Card does not nest. A Card inside a Card is
 * the audit's "nested versus flat" finding, and it draws a grid of boxes where
 * the eye should read content. Inside a Card use a heading, a divider
 * (`border-t border-hairline`) or spacing. A DataTable and an EmptyState also
 * go inside a Card or stand alone; neither is a Card.
 *
 * `title` is an h2. A page's single h1 is the PageHeader's.
 */
import type { ReactNode } from 'react';
import { cx } from './cx';

export interface CardProps {
  title?: string;
  /** Right of the title: a StatusPill, a count or one button. */
  headerAction?: ReactNode;
  children: ReactNode;
  className?: string;
  'data-testid'?: string;
}

export function Card({
  title,
  headerAction,
  children,
  className,
  'data-testid': testId,
}: CardProps) {
  return (
    <section className={cx('panel min-w-0 p-4 sm:p-5', className)} data-testid={testId}>
      {title || headerAction ? (
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          {title ? (
            <h2 className="text-base font-semibold text-text-primary">{title}</h2>
          ) : (
            <span />
          )}
          {headerAction}
        </div>
      ) : null}
      {children}
    </section>
  );
}
