/**
 * FormSection: a titled group of fields inside one form (stem's config forms
 * group "Frame", "Timing", "Thresholds"; seed's settings do the same). A named
 * group, so a screen reader announces the section a field belongs to; the title
 * is an h3 under the page's h1 and the Card's h2. `help` sits beside the title
 * (a HelpIcon), `description` is one line under it. Fields inside are still
 * FormFields, which cap their own width.
 */
import { type ReactNode, useId } from 'react';
import { cx } from './cx';

export interface FormSectionProps {
  title: string;
  /** Beside the title: a HelpIcon. */
  help?: ReactNode;
  /** One line under the title. */
  description?: string;
  children: ReactNode;
  className?: string;
}

export function FormSection({ title, help, description, children, className }: FormSectionProps) {
  const titleId = useId();
  return (
    <section aria-labelledby={titleId} className={cx('space-y-3', className)}>
      <div>
        <div className="flex items-center gap-1.5">
          <h3
            id={titleId}
            className="text-xs font-semibold uppercase tracking-wide text-text-muted"
          >
            {title}
          </h3>
          {help}
        </div>
        {description ? <p className="mt-1 text-xs text-text-muted">{description}</p> : null}
      </div>
      {children}
    </section>
  );
}
