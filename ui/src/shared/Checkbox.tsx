/**
 * Checkbox: a native checkbox with its label. For an on/off setting that takes
 * effect at once use Toggle; use Checkbox for a choice that waits for Save or
 * for picking several items. `onChange` gets the new boolean, as Toggle's does.
 */
import { type ReactNode, useId } from 'react';
import { cx } from './cx';

export interface CheckboxProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: ReactNode;
  description?: string;
  disabled?: boolean;
  className?: string;
}

export function Checkbox({
  checked,
  onChange,
  label,
  description,
  disabled,
  className,
}: CheckboxProps) {
  const id = useId();
  const descriptionId = `${id}-description`;
  return (
    <div className={cx('flex items-start gap-3', className)}>
      <input
        id={id}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        aria-describedby={description ? descriptionId : undefined}
        onChange={(event) => onChange(event.target.checked)}
        className="mt-0.5 h-4 w-4 shrink-0 accent-brand-primary disabled:cursor-not-allowed"
      />
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
    </div>
  );
}
