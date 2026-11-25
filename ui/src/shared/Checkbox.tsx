/**
 * Checkbox: a native checkbox with its label. For an on/off setting that takes
 * effect at once use Toggle; use Checkbox for a choice that waits for Save or
 * for picking several items. `onChange` gets the new boolean, as Toggle's does.
 *
 * The box is 24px (WCAG 2.5.8 target size), and the label beside it is part of
 * the target. `id`, `name`, `data-testid` and any `aria-*` land on the input, so
 * a spec's `getByTestId` and a form's `name` reach the control; `ref` is the
 * input's. For react-hook-form use `Controller` (`checked={field.value}`,
 * `onChange={field.onChange}`), whose `onChange` takes the boolean.
 */
import type { InputHTMLAttributes, ReactNode, Ref } from 'react';
import { useId } from 'react';
import { cx } from './cx';

/** The 24px native box, shared with DataTable's row and select-all checkboxes. */
export const checkboxControlClass =
  'h-6 w-6 shrink-0 accent-brand-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed';

type NativeProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  'type' | 'checked' | 'onChange' | 'className' | 'children'
>;

export interface CheckboxProps extends NativeProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: ReactNode;
  description?: string;
  /** Classes for the wrapper; the input has none to override. */
  className?: string;
  ref?: Ref<HTMLInputElement>;
}

export function Checkbox({
  checked,
  onChange,
  label,
  description,
  disabled,
  className,
  id: requestedId,
  ref,
  ...rest
}: CheckboxProps) {
  const generatedId = useId();
  const id = requestedId ?? generatedId;
  const descriptionId = `${id}-description`;
  return (
    <div className={cx('flex items-start gap-3', className)}>
      <input
        {...rest}
        ref={ref}
        id={id}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        aria-describedby={
          [rest['aria-describedby'], description ? descriptionId : undefined]
            .filter(Boolean)
            .join(' ') || undefined
        }
        onChange={(event) => onChange(event.target.checked)}
        className={checkboxControlClass}
      />
      <div className="min-w-0">
        <label
          htmlFor={id}
          className={cx(
            'block min-h-6 text-sm font-medium leading-6',
            disabled ? 'cursor-not-allowed text-text-disabled' : 'cursor-pointer text-text-primary',
          )}
        >
          {label}
        </label>
        {description ? (
          <p id={descriptionId} className="text-xs text-text-muted">
            {description}
          </p>
        ) : null}
      </div>
    </div>
  );
}
