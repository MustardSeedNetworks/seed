/**
 * FormField: a label, one control, an optional hint and an optional error.
 *
 * The control is a render prop so the association cannot be forgotten: the
 * field generates the id, wires `aria-describedby` to the hint and error, and
 * sets `aria-invalid` and `aria-required`; the caller spreads them onto the
 * control. (niac's field made `htmlFor` required for the same reason: an
 * optional association is one a new field silently forgets.)
 *
 *   <FormField label="Name" hint="Shown in the rail">
 *     {(control) => <Input {...control} value={name} onChange={...} />}
 *   </FormField>
 *
 * - `label` is a ReactNode (a name with a unit, a name with an icon). Anything
 *   interactive (a help button) does not go in it: a `<label>` may not contain
 *   another labelable element. Put it in `labelAccessory`, which sits beside the
 *   label on the same line.
 * - `required` appends a visible `*` (hidden from assistive technology) and sets
 *   `aria-required` on the control.
 * - `hint` and `error` are linked to the control; an error is announced
 *   (`role="alert"`) and sets `aria-invalid`.
 * - `children` may be a plain node when the control cannot take a render prop (a
 *   Checkbox group, a custom picker); give the field an `id` and put the same id
 *   on the control. The hint and error are then `${id}-hint` and `${id}-error`,
 *   for `aria-describedby`.
 *
 * With react-hook-form, `register("name")` spreads onto Input, Select and
 * Textarea (they pass `ref`, `name`, `onChange` and `onBlur` through); put the
 * message from `formState.errors` in `error`.
 *
 * The width is capped. stem's fields stretched to 1000px, which puts the label
 * and its input a screen apart. `md` is 28rem; `lg` is 42rem for a path or a
 * URL. There is no full-width option; a control that needs the page's width
 * (a log, an editor) is not a form field.
 */
import { type ReactNode, useId } from 'react';
import { cx } from './cx';

export interface FormControlProps {
  id: string;
  'aria-describedby': string | undefined;
  'aria-invalid': true | undefined;
  'aria-required': true | undefined;
}

export interface FormFieldProps {
  label: ReactNode;
  /** Beside the label, outside it: a help button or a Tooltip trigger. */
  labelAccessory?: ReactNode;
  /** Appends a visible marker and sets aria-required on the control. */
  required?: boolean;
  hint?: ReactNode;
  error?: ReactNode;
  size?: 'md' | 'lg';
  /** The control's id. Generated when omitted; needed with a plain-node child. */
  id?: string;
  className?: string;
  children: ReactNode | ((control: FormControlProps) => ReactNode);
}

const widths = { md: 'max-w-md', lg: 'max-w-2xl' } as const;

export function FormField({
  label,
  labelAccessory,
  required,
  hint,
  error,
  size = 'md',
  id: requestedId,
  className,
  children,
}: FormFieldProps) {
  const generatedId = useId();
  const id = requestedId ?? generatedId;
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const describedBy = [hint ? hintId : null, error ? errorId : null].filter(Boolean).join(' ');
  const control =
    typeof children === 'function'
      ? children({
          id,
          'aria-describedby': describedBy || undefined,
          'aria-invalid': error ? true : undefined,
          'aria-required': required ? true : undefined,
        })
      : children;
  return (
    <div className={cx('w-full', widths[size], className)}>
      <div className="mb-1.5 flex items-center gap-1.5">
        <label htmlFor={id} className="text-sm font-medium text-text-secondary">
          {label}
          {required ? (
            <span className="ml-0.5 text-status-error-strong" aria-hidden="true">
              *
            </span>
          ) : null}
        </label>
        {labelAccessory}
      </div>
      {control}
      {hint ? (
        <p id={hintId} className="mt-1.5 text-xs text-text-muted">
          {hint}
        </p>
      ) : null}
      {error ? (
        <p
          id={errorId}
          role="alert"
          className="mt-1.5 text-xs font-medium text-status-error-strong"
        >
          {error}
        </p>
      ) : null}
    </div>
  );
}
