/**
 * Select: a native <select>, themed.
 *
 * Native on purpose: the platform's picker is the one that works with a phone
 * keyboard, a screen reader and type-ahead. What it fixes is seed's un-themed
 * selects: the face takes the surface tokens and the border the 3:1
 * `surface-border`, and the chevron is drawn here, not left to the browser.
 * The popup's own colours follow `color-scheme`, which the shared theme sets
 * per mode.
 *
 * `onValueChange(value)` is the way to listen in app code. The native `onChange`
 * (the event) is also passed through, for react-hook-form's `register`, which
 * reads `event.target`; use one or the other. `ref`, `name`, `id`, `data-testid`
 * and `aria-*` land on the `<select>`. Pair it with a FormField, which supplies
 * `id` and the aria props.
 */
import { ChevronDown } from 'lucide-react';
import type { Ref, SelectHTMLAttributes } from 'react';
import { cx } from './cx';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'children'> {
  options: readonly SelectOption[];
  /** The chosen option's value, as seed's callers pass it. */
  onValueChange?: (value: string) => void;
  /** A disabled first option shown while nothing is chosen. */
  placeholder?: string;
  ref?: Ref<HTMLSelectElement>;
}

export function Select({
  options,
  placeholder,
  onValueChange,
  onChange,
  className,
  ref,
  ...rest
}: SelectProps) {
  return (
    <div className="relative w-full">
      <select
        ref={ref}
        className={cx(
          'min-h-10 w-full min-w-0 appearance-none truncate rounded-lg border border-surface-border bg-surface-raised py-2 pl-3 pr-9 text-sm text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:bg-surface-sunken disabled:text-text-disabled aria-[invalid=true]:border-status-error',
          className,
        )}
        onChange={(event) => {
          onChange?.(event);
          onValueChange?.(event.target.value);
        }}
        {...rest}
      >
        {placeholder ? (
          <option value="" disabled>
            {placeholder}
          </option>
        ) : null}
        {options.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>
            {option.label}
          </option>
        ))}
      </select>
      <ChevronDown
        className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted"
        aria-hidden="true"
      />
    </div>
  );
}
