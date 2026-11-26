/**
 * HelpIcon: the "?" beside a label or a section heading whose meaning needs a
 * sentence. A button, so it is reachable and named; its text shows in a Tooltip
 * on hover and focus, and the button takes focus on a tap (Safari and Firefox on
 * macOS and iOS do not focus a button themselves), which is what opens the
 * tooltip on touch. It replaces stem's `HelpIcon` and niac's hover-only
 * `HelpTip`, which a keyboard could not reach.
 *
 * `label` is the accessible name ("Help: Timeout"); `text` is what the bubble
 * says. Anything the user must read to finish the form belongs in the field's
 * `hint`, not here. Put it in FormField's `labelAccessory` or FormSection's
 * `help`; it is not meant to sit inside a `<label>`, but it swallows its click
 * so it would not toggle the control beside it if it did.
 */
import { CircleHelp } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from './cx';
import { Tooltip } from './Tooltip';

export interface HelpIconProps {
  /** Accessible name of the button, already translated: "Help: Timeout". */
  label: string;
  /** What the tooltip says. */
  text: ReactNode;
  className?: string;
  'data-testid'?: string;
}

export function HelpIcon({ label, text, className, 'data-testid': testId }: HelpIconProps) {
  return (
    <Tooltip text={text}>
      {/* The render form, so a click does not dismiss the tooltip: here a tap is what shows it. */}
      {(description) => (
        <button
          type="button"
          aria-label={label}
          aria-describedby={description['aria-describedby']}
          data-testid={testId}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            event.currentTarget.focus();
          }}
          className={cx(
            'inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-text-muted hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary',
            className,
          )}
        >
          <CircleHelp className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      )}
    </Tooltip>
  );
}
