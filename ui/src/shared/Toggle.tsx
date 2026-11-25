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
 * The track is 24px tall (WCAG 2.5.8). `id`, `name`, `data-testid` and any
 * `aria-*` land on the switch button, so a spec's `getByTestId` reaches the
 * control and a plain `<label htmlFor>` elsewhere can name it.
 */
import { type ButtonHTMLAttributes, type ReactNode, useId } from 'react';
import { cx } from './cx';

type NativeProps = Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  'onChange' | 'onClick' | 'role' | 'type' | 'className' | 'children' | 'aria-checked' | 'disabled'
>;

export interface ToggleProps extends NativeProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: ReactNode;
  /** One line under the label, linked as the switch's description. */
  description?: string;
  disabled?: boolean;
  /** Classes for the wrapper. */
  className?: string;
}

export function Toggle({
  checked,
  onChange,
  label,
  description,
  disabled,
  className,
  id: requestedId,
  ...rest
}: ToggleProps) {
  const generatedId = useId();
  const id = requestedId ?? generatedId;
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
        {...rest}
        id={id}
        type="button"
        role="switch"
        aria-checked={checked}
        aria-describedby={
          [rest['aria-describedby'], description ? descriptionId : undefined]
            .filter(Boolean)
            .join(' ') || undefined
        }
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
