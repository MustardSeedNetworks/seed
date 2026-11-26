/**
 * CollapsibleSection: a titled block the reader can fold away (advanced
 * options, a long reference, a settings group). The title is a button with
 * `aria-expanded` controlling the region. Do not hide anything the user needs to
 * finish the page's main task. Settings sections are tabs, not these.
 *
 * - `title` is a ReactNode: a plain string, or a heading with an icon or a
 *   link-free badge. The region is named by the button, so whatever the title
 *   renders names it.
 * - `count` appends "(n)", the size of what is folded.
 * - `status` (with its `statusLabel`) puts a StatusIndicator before the title:
 *   what the folded content says, visible while folded.
 * - `variant`: `default` is a bordered block that stands alone; `compact` has no
 *   border, for inside a Card (which must not gain a second box).
 * - `readOnlyReason`: set when the section is not writable for this user. The
 *   header stays a working button (a viewer can still open and read it), the
 *   reason shows in the header, and the body is a disabled `<fieldset>`, so every
 *   control in it is disabled at once. Only for a section whose every control
 *   writes; one that also carries a read action (Refresh, download) gates its
 *   controls individually instead.
 *
 * - `open` and `onOpenChange` control it (niac's sections fold together and
 *   remember state); without `open` it manages itself, from `defaultOpen`.
 *
 * The body stays mounted while folded (`hidden`), so form state survives.
 */
import { ChevronRight } from 'lucide-react';
import { type ReactNode, useId, useState } from 'react';
import { cx } from './cx';
import { type IndicatorTone, StatusIndicator } from './StatusPill';

interface BaseProps {
  title: ReactNode;
  children: ReactNode;
  defaultOpen?: boolean;
  /** Controls the fold; pair with `onOpenChange`. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** The size of what is folded, shown as "(n)" after the title. */
  count?: number;
  variant?: 'default' | 'compact';
  /** Why this section is read-only; leave undefined when it is writable. */
  readOnlyReason?: string;
  id?: string;
  className?: string;
  'data-testid'?: string;
}

type StatusProps =
  | { status?: undefined; statusLabel?: undefined }
  | {
      status: IndicatorTone;
      /** Accessible name of the indicator, already translated. */
      statusLabel: string;
    };

export type CollapsibleSectionProps = BaseProps & StatusProps;

export function CollapsibleSection({
  title,
  children,
  defaultOpen = false,
  open: controlledOpen,
  onOpenChange,
  count,
  status,
  statusLabel,
  variant = 'default',
  readOnlyReason,
  id: requestedId,
  className,
  'data-testid': testId,
}: CollapsibleSectionProps) {
  const [ownOpen, setOwnOpen] = useState(defaultOpen);
  const open = controlledOpen ?? ownOpen;
  const generatedId = useId();
  const id = requestedId ?? generatedId;
  const buttonId = `${id}-button`;
  const regionId = `${id}-region`;
  const compact = variant === 'compact';
  const readOnly = readOnlyReason !== undefined;
  return (
    <div
      id={requestedId}
      data-testid={testId}
      className={cx(!compact && 'overflow-hidden rounded-lg border border-hairline', className)}
    >
      <button
        id={buttonId}
        type="button"
        aria-expanded={open}
        aria-controls={regionId}
        onClick={() => {
          setOwnOpen(!open);
          onOpenChange?.(!open);
        }}
        className={cx(
          'flex min-h-9 w-full items-center gap-2 text-left text-sm font-semibold text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary',
          compact ? 'rounded-lg' : 'bg-surface-base px-3',
        )}
      >
        <ChevronRight
          className={cx(
            'h-4 w-4 shrink-0 text-text-muted transition-transform',
            open && 'rotate-90',
          )}
          aria-hidden="true"
        />
        {status === undefined ? null : <StatusIndicator tone={status} label={statusLabel ?? ''} />}
        <span className="min-w-0 flex-1">
          {title}
          {count === undefined ? null : (
            <>
              {' '}
              <span className="font-normal text-text-muted">({count})</span>
            </>
          )}
        </span>
        {readOnly ? (
          <span className="shrink-0 text-xs font-normal text-text-muted">{readOnlyReason}</span>
        ) : null}
      </button>
      <section
        id={regionId}
        aria-labelledby={buttonId}
        hidden={!open}
        className={cx(compact ? 'pl-6 pt-2' : 'border-t border-hairline p-3')}
      >
        {readOnly ? (
          <fieldset disabled className="m-0 min-w-0 border-0 p-0">
            {children}
          </fieldset>
        ) : (
          children
        )}
      </section>
    </div>
  );
}
