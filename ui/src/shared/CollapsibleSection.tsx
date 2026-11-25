/**
 * CollapsibleSection: a titled block the reader can fold away (advanced
 * options, a long reference). The title is a button with `aria-expanded`
 * controlling the region. Do not hide anything the user needs to finish the
 * page's main task. Settings sections are tabs, not these.
 */
import { ChevronRight } from 'lucide-react';
import { type ReactNode, useId, useState } from 'react';
import { cx } from './cx';

export interface CollapsibleSectionProps {
  title: string;
  children: ReactNode;
  defaultOpen?: boolean;
  className?: string;
}

export function CollapsibleSection({
  title,
  children,
  defaultOpen = false,
  className,
}: CollapsibleSectionProps) {
  const [open, setOpen] = useState(defaultOpen);
  const regionId = useId();
  return (
    <div className={className}>
      <button
        type="button"
        aria-expanded={open}
        aria-controls={regionId}
        onClick={() => setOpen(!open)}
        className="flex min-h-9 w-full items-center gap-2 rounded-lg text-left text-sm font-semibold text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary"
      >
        <ChevronRight
          className={cx(
            'h-4 w-4 shrink-0 text-text-muted transition-transform',
            open && 'rotate-90',
          )}
          aria-hidden="true"
        />
        {title}
      </button>
      <section id={regionId} aria-label={title} hidden={!open} className="pt-2">
        {children}
      </section>
    </div>
  );
}
