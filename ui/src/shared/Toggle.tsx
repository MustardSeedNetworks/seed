/**
 * Toggle: an on/off switch with a track you can see in both themes.
 *
 * stem's toggle painted its off track as an elevated surface, which in dark
 * mode is the page: the knob floated on nothing. The off track here is the
 * sunken surface with the 3:1 `surface-border` edge, and the on track is the
 * product hue, so state reads from the track alone, not from the knob's side.
 *
 * A real button with `role="switch"` and `aria-checked`, named by its label
 * (a <label> names any labelable element, a button included), so a click on
 * the text toggles it and a screen reader announces "<label>, switch, on".
 */
import { useId } from 'react';
import { cx } from './cx';

export interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  /** One line under the label, linked as the switch's description. */
  description?: string;
  disabled?: boolean;
  className?: string;
}

export function Toggle({
  checked,
  onChange,
  label,
  description,
  disabled,
  className,
}: ToggleProps) {
  const id = useId();
  const descriptionId = `${id}-description`;
  return (
    <div className={cx('flex items-start justify-between gap-4', className)}>
      <div className="min-w-0">
        <label
          htmlFor={id}
          className={cx(
            'text-sm font-medium',
            disabled ? 'cursor-not-allowed text-text-disabled' : 'cursor-pointer text-text-primary',
          )}
        >
          {label}
        </label>
        {description ? (
          <p id={descriptionId} className="mt-0.5 text-xs text-text-muted">
            {description}
          </p>
        ) : null}
      </div>
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={checked}
        aria-describedby={description ? descriptionId : undefined}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        data-state={checked ? 'on' : 'off'}
        className={cx(
          'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:opacity-60',
          checked
            ? 'border-brand-primary bg-brand-primary'
            : 'border-surface-border bg-surface-sunken',
        )}
      >
        <span
          aria-hidden="true"
          className={cx(
            'inline-block h-4 w-4 rounded-full transition-transform',
            checked ? 'translate-x-6 bg-knob' : 'translate-x-1 bg-text-muted',
          )}
        />
      </button>
    </div>
  );
}
