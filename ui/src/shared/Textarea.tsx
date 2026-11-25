/** Textarea: the multi-line Input. Same theming; use it inside a FormField. */
import type { Ref, TextareaHTMLAttributes } from 'react';
import { cx } from './cx';
import { controlClass } from './Input';

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  ref?: Ref<HTMLTextAreaElement>;
}

export function Textarea({ className, rows = 4, ref, ...rest }: TextareaProps) {
  return (
    <textarea ref={ref} rows={rows} className={cx(controlClass, 'resize-y', className)} {...rest} />
  );
}
