/**
 * RadioGroup: one choice from a few, as native radio inputs in a labelled group.
 *
 * Native on purpose: the browser supplies the arrow-key movement inside the
 * group, the single tab stop, type-to-select and the platform's screen-reader
 * behaviour, none of which a hand-rolled `role="radio"` gets right for free.
 * Each button is a 24px box (WCAG 2.5.8) and its label is part of the click
 * target.
 *
 * The group is a FormField `group`: a `<fieldset>` named by the `label` as its
 * legend, with `hint` and `error` linked by `aria-describedby` and the error
 * announced. Every radio carries `aria-invalid` while the group has an error.
 *
 * Controlled: `value` is the chosen option's value (`''` for none) and
 * `onValueChange(value)` reports a change. `name` ties the radios together and
 * is what a form submits; it is required so two groups on a page can never
 * share one by accident.
 *
 * Use a Select past about five options, a SegmentedControl when the choice
 * changes a view in place rather than a value to save, and Toggle for on/off.
 */

import type { ReactNode } from 'react';
import { useId } from 'react';
import { checkboxControlClass } from './Checkbox';
import { cx } from './cx';
import { FormField } from './FormField';

export interface RadioOption {
  value: string;
  label: ReactNode;
  /** One line under the label, linked as the radio's description. */
  description?: string;
  disabled?: boolean;
  'data-testid'?: string;
}

export interface RadioGroupProps {
  /** The group's question, rendered as the fieldset's legend. */
  label: ReactNode;
  name: string;
  options: readonly RadioOption[];
  value: string;
  onValueChange: (value: string) => void;
  hint?: ReactNode;
  error?: ReactNode;
  required?: boolean;
  disabled?: boolean;
  /** `horizontal` lays the options in a wrapping row; the default is a column. */
  orientation?: 'vertical' | 'horizontal';
  id?: string;
  className?: string;
  'data-testid'?: string;
}

export function RadioGroup({
  label,
  name,
  options,
  value,
  onValueChange,
  hint,
  error,
  required = false,
  disabled = false,
  orientation = 'vertical',
  id: requestedId,
  className,
  'data-testid': testId,
}: RadioGroupProps) {
  const generatedId = useId();
  const id = requestedId ?? generatedId;
  return (
    <FormField
      group
      id={id}
      label={label}
      {...(hint === undefined ? {} : { hint })}
      {...(error === undefined ? {} : { error })}
      required={required}
      {...(className === undefined ? {} : { className })}
    >
      <div
        data-testid={testId}
        className={cx(
          'flex gap-x-6 gap-y-2',
          orientation === 'vertical' ? 'flex-col' : 'flex-row flex-wrap',
        )}
      >
        {options.map((option, index) => {
          const optionId = `${id}-${String(index)}`;
          const descriptionId = `${optionId}-description`;
          const off = disabled || option.disabled === true;
          return (
            <div key={option.value} className="flex items-start gap-3">
              <input
                id={optionId}
                type="radio"
                name={name}
                value={option.value}
                checked={option.value === value}
                disabled={off}
                required={required}
                aria-invalid={error ? true : undefined}
                aria-describedby={option.description ? descriptionId : undefined}
                data-testid={option['data-testid']}
                onChange={() => onValueChange(option.value)}
                className={checkboxControlClass}
              />
              <div className="min-w-0">
                <label
                  htmlFor={optionId}
                  className={cx(
                    'block min-h-6 text-sm font-medium leading-6',
                    off
                      ? 'cursor-not-allowed text-text-disabled'
                      : 'cursor-pointer text-text-primary',
                  )}
                >
                  {option.label}
                </label>
                {option.description ? (
                  <p id={descriptionId} className="text-xs text-text-muted">
                    {option.description}
                  </p>
                ) : null}
              </div>
            </div>
          );
        })}
      </div>
    </FormField>
  );
}
