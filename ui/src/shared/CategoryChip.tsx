/**
 * CategoryChip: a neutral label for a category (device type, protocol, role,
 * tag). It takes no colour prop on purpose: a category is not a status, and
 * the status hues are reserved for StatusPill. If a category needs telling
 * apart, use its icon or its text.
 */
import type { ReactNode } from 'react';
import { cx } from './cx';

export interface CategoryChipProps {
  label: string;
  /** Decorative, about 14px. */
  icon?: ReactNode;
  className?: string;
}

export function CategoryChip({ label, icon, className }: CategoryChipProps) {
  return (
    <span
      className={cx(
        'inline-flex items-center gap-1.5 rounded-md border border-hairline bg-surface-hover px-2 py-0.5 text-xs font-medium text-text-secondary',
        className,
      )}
    >
      {icon ? <span aria-hidden="true">{icon}</span> : null}
      {label}
    </span>
  );
}
